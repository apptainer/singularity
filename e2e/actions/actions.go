// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package actions

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/test"
	"github.com/sylabs/singularity/internal/pkg/test/exec"
)

type testingEnv struct {
	// base env for running tests
	CmdPath     string `split_words:"true"`
	TestDir     string `split_words:"true"`
	RunDisabled bool   `default:"false"`
	//  base image for tests
	ImagePath string `split_words:"true"`
}

var testenv testingEnv

// run tests min fuctionality for singularity run
func actionRun(t *testing.T) {
	e2e.EnsureImage(t)

	tests := []struct {
		name   string
		image  string
		action string
		argv   []string
		e2e.ExecOpts
		exit          int
		expectSuccess bool
	}{
		{"NoCommand", testenv.ImagePath, "run", []string{}, e2e.ExecOpts{}, 0, true},
		{"true", testenv.ImagePath, "run", []string{"true"}, e2e.ExecOpts{}, 0, true},
		{"false", testenv.ImagePath, "run", []string{"false"}, e2e.ExecOpts{}, 1, false},
		{"ScifTestAppGood", testenv.ImagePath, "run", []string{}, e2e.ExecOpts{App: "testapp"}, 0, true},
		{"ScifTestAppBad", testenv.ImagePath, "run", []string{}, e2e.ExecOpts{App: "fakeapp"}, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			_, stderr, exitCode, err := e2e.ImageExec(t, testenv.CmdPath, tt.action, tt.ExecOpts, tt.image, tt.argv)
			if tt.expectSuccess && (exitCode != 0) {
				t.Log(stderr)
				t.Fatalf("unexpected failure running '%v': %v", strings.Join(tt.argv, " "), err)
			} else if !tt.expectSuccess && (exitCode != 1) {
				t.Log(stderr)
				t.Fatalf("unexpected success running '%v'", strings.Join(tt.argv, " "))
			}
		}))
	}
}

// exec tests min fuctionality for singularity exec
func actionExec(t *testing.T) {
	e2e.EnsureImage(t)

	// Create a temp testfile
	tmpfile, err := ioutil.TempFile("", "testSingularityExec.tmp")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	testfile, err := tmpfile.Stat()
	if err != nil {
		t.Fatal(err)
	}

	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		image  string
		action string
		argv   []string
		e2e.ExecOpts
		exit          int
		expectSuccess bool
	}{
		{"NoCommand", testenv.ImagePath, "exec", []string{}, e2e.ExecOpts{}, 1, false},
		{"true", testenv.ImagePath, "exec", []string{"true"}, e2e.ExecOpts{}, 0, true},
		{"trueAbsPAth", testenv.ImagePath, "exec", []string{"/bin/true"}, e2e.ExecOpts{}, 0, true},
		{"false", testenv.ImagePath, "exec", []string{"false"}, e2e.ExecOpts{}, 1, false},
		{"falseAbsPath", testenv.ImagePath, "exec", []string{"/bin/false"}, e2e.ExecOpts{}, 1, false},
		// Scif apps tests
		{"ScifTestAppGood", testenv.ImagePath, "exec", []string{"testapp.sh"}, e2e.ExecOpts{App: "testapp"}, 0, true},
		{"ScifTestAppBad", testenv.ImagePath, "exec", []string{"testapp.sh"}, e2e.ExecOpts{App: "fakeapp"}, 1, false},
		{"ScifTestfolderOrg", testenv.ImagePath, "exec", []string{"test", "-d", "/scif"}, e2e.ExecOpts{}, 0, true},
		{"ScifTestfolderOrg", testenv.ImagePath, "exec", []string{"test", "-d", "/scif/apps"}, e2e.ExecOpts{}, 0, true},
		{"ScifTestfolderOrg", testenv.ImagePath, "exec", []string{"test", "-d", "/scif/data"}, e2e.ExecOpts{}, 0, true},
		{"ScifTestfolderOrg", testenv.ImagePath, "exec", []string{"test", "-d", "/scif/apps/foo"}, e2e.ExecOpts{}, 0, true},
		{"ScifTestfolderOrg", testenv.ImagePath, "exec", []string{"test", "-d", "/scif/apps/bar"}, e2e.ExecOpts{}, 0, true},
		// blocked by issue [scif-apps] Files created at install step fall into an unexpected path #2404
		{"ScifTestfolderOrg", testenv.ImagePath, "exec", []string{"test", "-f", "/scif/apps/foo/filefoo.exec"}, e2e.ExecOpts{}, 0, true},
		{"ScifTestfolderOrg", testenv.ImagePath, "exec", []string{"test", "-f", "/scif/apps/bar/filebar.exec"}, e2e.ExecOpts{}, 0, true},
		{"ScifTestfolderOrg", testenv.ImagePath, "exec", []string{"test", "-d", "/scif/data/foo/output"}, e2e.ExecOpts{}, 0, true},
		{"ScifTestfolderOrg", testenv.ImagePath, "exec", []string{"test", "-d", "/scif/data/foo/input"}, e2e.ExecOpts{}, 0, true},
		{"WorkdirContain", testenv.ImagePath, "exec", []string{"test", "-f", tmpfile.Name()}, e2e.ExecOpts{Workdir: "testdata", Contain: true}, 0, false},
		{"Workdir", testenv.ImagePath, "exec", []string{"test", "-f", tmpfile.Name()}, e2e.ExecOpts{Workdir: "testdata"}, 0, true},
		{"pwdGood", testenv.ImagePath, "exec", []string{"true"}, e2e.ExecOpts{Pwd: "/etc"}, 0, true},
		{"home", testenv.ImagePath, "exec", []string{"test", "-f", tmpfile.Name()}, e2e.ExecOpts{Home: pwd + "testdata"}, 0, true},
		{"homePath", testenv.ImagePath, "exec", []string{"test", "-f", "/home/" + testfile.Name()}, e2e.ExecOpts{Home: "/tmp:/home"}, 0, true},
		{"homeTmp", testenv.ImagePath, "exec", []string{"true"}, e2e.ExecOpts{Home: "/tmp"}, 0, true},
		{"homeTmpExplicit", testenv.ImagePath, "exec", []string{"true"}, e2e.ExecOpts{Home: "/tmp:/home"}, 0, true},
		{"ScifTestAppGood", testenv.ImagePath, "exec", []string{"testapp.sh"}, e2e.ExecOpts{App: "testapp"}, 0, true},
		{"ScifTestAppBad", testenv.ImagePath, "exec", []string{"testapp.sh"}, e2e.ExecOpts{App: "fakeapp"}, 1, false},
		//
		{"userBind", testenv.ImagePath, "exec", []string{"test", "-f", "/var/tmp/" + testfile.Name()}, e2e.ExecOpts{Binds: []string{"/tmp:/var/tmp"}}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			_, stderr, exitCode, err := e2e.ImageExec(t, testenv.CmdPath, tt.action, tt.ExecOpts, tt.image, tt.argv)
			if tt.expectSuccess && (exitCode != 0) {
				t.Log(stderr)
				t.Fatalf("unexpected failure running '%v': %v", strings.Join(tt.argv, " "), err)
			} else if !tt.expectSuccess && (exitCode != 1) {
				t.Log(stderr)
				t.Fatalf("unexpected success running '%v'", strings.Join(tt.argv, " "))
			}
		}))
	}

	// test --no-home option
	err = os.Chdir("/tmp")
	if err != nil {
		t.Fatal(err)
	}
	t.Run("noHome", test.WithoutPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := e2e.ImageExec(t, testenv.CmdPath, "exec", e2e.ExecOpts{NoHome: true}, pwd+"/container.img", []string{"ls", "-ld", "$HOME"})
		if exitCode != 1 {
			t.Log(stderr, err)
			t.Fatalf("unexpected success running '%v'", strings.Join([]string{"ls", "-ld", "$HOME"}, " "))
		}
	}))
	// return to test SOURCEDIR
	err = os.Chdir(pwd)
	if err != nil {
		t.Fatal(err)
	}
}

// STDINPipe tests pipe stdin to singularity actions cmd
func STDINPipe(t *testing.T) {
	e2e.EnsureImage(t)

	tests := []struct {
		binName string
		name    string
		argv    []string
		exit    int
	}{
		{"sh", "trueSTDIN", []string{"-c", fmt.Sprintf("echo hi | %s exec %s grep hi", testenv.CmdPath, testenv.ImagePath)}, 0},
		{"sh", "falseSTDIN", []string{"-c", fmt.Sprintf("echo bye | %s exec %s grep hi", testenv.CmdPath, testenv.ImagePath)}, 1},
		// Checking permissions
		{"sh", "permissions", []string{"-c", fmt.Sprintf("%s exec %s id -u | grep `id -u`", testenv.CmdPath, testenv.ImagePath)}, 0},
		// testing run command properly hands arguments
		{"sh", "arguments", []string{"-c", fmt.Sprintf("%s run %s foo | grep foo", testenv.CmdPath, testenv.ImagePath)}, 0},
		// Stdin to URI based image
		{"sh", "library", []string{"-c", fmt.Sprintf("echo true | %s shell library://busybox", testenv.CmdPath)}, 0},
		{"sh", "docker", []string{"-c", fmt.Sprintf("echo true | %s shell docker://busybox", testenv.CmdPath)}, 0},
		{"sh", "shub", []string{"-c", fmt.Sprintf("echo true | %s shell shub://singularityhub/busybox", testenv.CmdPath)}, 0},
		// Test apps
		{"sh", "appsFoo", []string{"-c", fmt.Sprintf("%s run --app foo %s | grep 'FOO'", testenv.CmdPath, testenv.ImagePath)}, 0},
		// Test target pwd
		{"sh", "pwdPath", []string{"-c", fmt.Sprintf("%s exec --pwd /etc %s pwd | egrep '^/etc'", testenv.CmdPath, testenv.ImagePath)}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			cmd := exec.Command(tt.binName, tt.argv...)
			res := cmd.Run(t)

			if res.ExitCode != tt.exit {
				t.Fatalf("Unexpected exit code '%d' while running command.\n%s",
					res.ExitCode,
					res)
			}
		}))
	}
}

// RunFromURI tests min fuctionality for singularity run/exec URI://
func RunFromURI(t *testing.T) {
	runScript := "testdata/runscript.sh"
	bind := fmt.Sprintf("%s:/.singularity.d/runscript", runScript)

	runOpts := e2e.ExecOpts{
		Binds: []string{bind},
	}

	fi, err := os.Stat(runScript)
	if err != nil {
		t.Fatalf("can't find %s", runScript)
	}
	size := strconv.Itoa(int(fi.Size()))

	tests := []struct {
		name   string
		image  string
		action string
		argv   []string
		e2e.ExecOpts
		expectSuccess bool
	}{
		// Run from supported URI's and check the runscript call works
		{"RunFromDockerOK", "docker://busybox:latest", "run", []string{size}, runOpts, true},
		{"RunFromLibraryOK", "library://busybox:latest", "run", []string{size}, runOpts, true},
		{"RunFromShubOK", "shub://singularityhub/busybox", "run", []string{size}, runOpts, true},
		{"RunFromDockerKO", "docker://busybox:latest", "run", []string{"0"}, runOpts, false},
		{"RunFromLibraryKO", "library://busybox:latest", "run", []string{"0"}, runOpts, false},
		{"RunFromShubKO", "shub://singularityhub/busybox", "run", []string{"0"}, runOpts, false},
		// exec from a supported URI's and check the exit code
		{"trueDocker", "docker://busybox:latest", "exec", []string{"true"}, e2e.ExecOpts{}, true},
		{"trueLibrary", "library://busybox:latest", "exec", []string{"true"}, e2e.ExecOpts{}, true},
		{"trueShub", "shub://singularityhub/busybox", "exec", []string{"true"}, e2e.ExecOpts{}, true},
		{"falseDocker", "docker://busybox:latest", "exec", []string{"false"}, e2e.ExecOpts{}, false},
		{"falselibrary", "library://busybox:latest", "exec", []string{"false"}, e2e.ExecOpts{}, false},
		{"falseShub", "shub://singularityhub/busybox", "exec", []string{"false"}, e2e.ExecOpts{}, false},
		// exec from URI with user namespace enabled
		{"trueDockerUserns", "docker://busybox:latest", "exec", []string{"true"}, e2e.ExecOpts{Userns: true}, true},
		{"trueLibraryUserns", "library://busybox:latest", "exec", []string{"true"}, e2e.ExecOpts{Userns: true}, true},
		{"trueShubUserns", "shub://singularityhub/busybox", "exec", []string{"true"}, e2e.ExecOpts{Userns: true}, true},
		{"falseDockerUserns", "docker://busybox:latest", "exec", []string{"false"}, e2e.ExecOpts{Userns: true}, false},
		{"falselibraryUserns", "library://busybox:latest", "exec", []string{"false"}, e2e.ExecOpts{Userns: true}, false},
		{"falseShubUserns", "shub://singularityhub/busybox", "exec", []string{"false"}, e2e.ExecOpts{Userns: true}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			_, stderr, exitCode, err := e2e.ImageExec(t, testenv.CmdPath, tt.action, tt.ExecOpts, tt.image, tt.argv)
			if tt.expectSuccess && (exitCode != 0) {
				t.Log(stderr)
				t.Fatalf("unexpected failure running '%v': %v", strings.Join(tt.argv, " "), err)
			} else if !tt.expectSuccess && (exitCode != 1) {
				t.Log(stderr)
				t.Fatalf("unexpected success running '%v'", strings.Join(tt.argv, " "))
			}
		}))
	}
}

// PersistentOverlay test the --overlay function
func PersistentOverlay(t *testing.T) {
	e2e.EnsureImage(t)

	const squashfsImage = "squashfs.simg"
	//  Create the overlay dir
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := ioutil.TempDir(cwd, "overlay_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Create dirfs for squashfs
	squashDir, err := ioutil.TempDir(cwd, "overlay_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(squashDir)

	content := []byte("temporary file's content")
	tmpfile, err := ioutil.TempFile(squashDir, "bogus")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	cmd := exec.Command("mksquashfs", squashDir, squashfsImage, "-noappend", "-all-root")
	if res := cmd.Run(t); res.Error != nil {
		t.Fatalf("Unexpected error while running command.\n%s", res)
	}
	defer os.RemoveAll(squashfsImage)

	//  Create the overlay ext3 fs
	cmd = exec.Command("dd", "if=/dev/zero", "of=ext3_fs.img", "bs=1M", "count=768", "status=none")
	if res := cmd.Run(t); res.Error != nil {
		t.Fatalf("Unexpected error while running command.\n%s", res)
	}

	cmd = exec.Command("mkfs.ext3", "-q", "-F", "ext3_fs.img")
	if res := cmd.Run(t); res.Error != nil {
		t.Fatalf("Unexpected error while running command.\n%s", res)
	}

	defer os.Remove("ext3_fs.img")

	// create a file dir
	t.Run("overlay_create", test.WithPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := e2e.ImageExec(t, testenv.CmdPath, "exec", e2e.ExecOpts{Overlay: []string{dir}}, testenv.ImagePath, []string{"touch", "/dir_overlay"})
		if exitCode != 0 {
			t.Log(stderr, err)
			t.Fatalf("unexpected failure running '%v'", strings.Join([]string{"test", "-f", "/dir_overlay"}, " "))
		}
	}))
	// look for the file dir
	t.Run("overlay_find", test.WithPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := e2e.ImageExec(t, testenv.CmdPath, "exec", e2e.ExecOpts{Overlay: []string{dir}}, testenv.ImagePath, []string{"test", "-f", "/dir_overlay"})
		if exitCode != 0 {
			t.Log(stderr, err)
			t.Fatalf("unexpected failure running '%v'", strings.Join([]string{"test", "-f", "/dir_overlay"}, " "))
		}
	}))
	// create a file ext3
	t.Run("overlay_ext3_create", test.WithPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := e2e.ImageExec(t, testenv.CmdPath, "exec", e2e.ExecOpts{Overlay: []string{"ext3_fs.img"}}, testenv.ImagePath, []string{"touch", "/ext3_overlay"})
		if exitCode != 0 {
			t.Log(stderr, err)
			t.Fatalf("unexpected failure running '%v'", strings.Join([]string{"test", "-f", "/ext3_overlay"}, " "))
		}
	}))
	// look for the file ext3
	t.Run("overlay_ext3_find", test.WithPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := e2e.ImageExec(t, testenv.CmdPath, "exec", e2e.ExecOpts{Overlay: []string{"ext3_fs.img"}}, testenv.ImagePath, []string{"test", "-f", "/ext3_overlay"})
		if exitCode != 0 {
			t.Log(stderr, err)
			t.Fatalf("unexpected failure running '%v'", strings.Join([]string{"test", "-f", "/ext3_overlay"}, " "))
		}
	}))
	// look for the file squashFs
	t.Run("overlay_squashFS_find", test.WithPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := e2e.ImageExec(t, testenv.CmdPath, "exec", e2e.ExecOpts{Overlay: []string{squashfsImage}}, testenv.ImagePath, []string{"test", "-f", fmt.Sprintf("/%s", tmpfile.Name())})
		if exitCode != 0 {
			t.Log(stderr, err)
			t.Fatalf("unexpected failure running '%v'", strings.Join([]string{"test", "-f", fmt.Sprintf("/%s", tmpfile.Name())}, " "))
		}
	}))
	// create a file multiple overlays
	t.Run("overlay_multiple_create", test.WithPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := e2e.ImageExec(t, testenv.CmdPath, "exec", e2e.ExecOpts{Overlay: []string{"ext3_fs.img", squashfsImage}}, testenv.ImagePath, []string{"touch", "/multiple_overlay_fs"})
		if exitCode != 0 {
			t.Log(stderr, err)
			t.Fatalf("unexpected failure running '%v'", strings.Join([]string{"touch", "/multiple_overlay_fs"}, " "))
		}
	}))
	// look for the file with multiple overlays
	t.Run("overlay_multiple_find_ext3", test.WithPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := e2e.ImageExec(t, testenv.CmdPath, "exec", e2e.ExecOpts{Overlay: []string{"ext3_fs.img", squashfsImage}}, testenv.ImagePath, []string{"test", "-f", "/multiple_overlay_fs"})
		if exitCode != 0 {
			t.Log(stderr, err)
			t.Fatalf("unexpected failure running '%v'", strings.Join([]string{"test", "-f", "multiple_overlay_fs"}, " "))
		}
	}))
	t.Run("overlay_multiple_find_squashfs", test.WithPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := e2e.ImageExec(t, testenv.CmdPath, "exec", e2e.ExecOpts{Overlay: []string{"ext3_fs.img", squashfsImage}}, testenv.ImagePath, []string{"test", "-f", fmt.Sprintf("/%s", tmpfile.Name())})
		if exitCode != 0 {
			t.Log(stderr, err)
			t.Fatalf("unexpected failure running '%v'", strings.Join([]string{"test", "-f", tmpfile.Name()}, " "))
		}
	}))
	// look for the file without root privs
	t.Run("overlay_noroot", test.WithoutPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := e2e.ImageExec(t, testenv.CmdPath, "exec", e2e.ExecOpts{Overlay: []string{dir}}, testenv.ImagePath, []string{"test", "-f", "/foo_overlay"})
		if exitCode != 1 {
			t.Log(stderr, err)
			t.Fatalf("unexpected success running '%v'", strings.Join([]string{"test", "-f", "/foo_overlay"}, " "))
		}
	}))
	// look for the file without --overlay
	t.Run("overlay_noflag", test.WithPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := e2e.ImageExec(t, testenv.CmdPath, "exec", e2e.ExecOpts{}, testenv.ImagePath, []string{"test", "-f", "/foo_overlay"})
		if exitCode != 1 {
			t.Log(stderr, err)
			t.Fatalf("unexpected success running '%v'", strings.Join([]string{"test", "-f", "/foo_overlay"}, " "))
		}
	}))
}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(t *testing.T) {
	e2e.LoadEnv(t, &testenv)

	// singularity run
	t.Run("run", actionRun)
	// singularity exec
	t.Run("exec", actionExec)
	// stdin pipe
	t.Run("STDIN", STDINPipe)
	// action_URI
	t.Run("action_URI", RunFromURI)
	// Persistent Overlay
	t.Run("Persistent_Overlay", PersistentOverlay)
}
