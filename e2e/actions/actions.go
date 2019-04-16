// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package actions

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/kelseyhightower/envconfig"
	"github.com/sylabs/singularity/internal/pkg/test"
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
	tests := []struct {
		name   string
		image  string
		action string
		argv   []string
		Opts
		exit          int
		expectSuccess bool
	}{
		{"NoCommand", testenv.ImagePath, "run", []string{}, Opts{}, 0, true},
		{"true", testenv.ImagePath, "run", []string{"true"}, Opts{}, 0, true},
		{"false", testenv.ImagePath, "run", []string{"false"}, Opts{}, 1, false},
		{"ScifTestAppGood", testenv.ImagePath, "run", []string{}, Opts{App: "testapp"}, 0, true},
		{"ScifTestAppBad", testenv.ImagePath, "run", []string{}, Opts{App: "fakeapp"}, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			_, stderr, exitCode, err := ImageExec(t, testenv.CmdPath, tt.action, tt.Opts, tt.image, tt.argv)
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
		Opts
		exit          int
		expectSuccess bool
	}{
		{"NoCommand", testenv.ImagePath, "exec", []string{}, Opts{}, 1, false},
		{"true", testenv.ImagePath, "exec", []string{"true"}, Opts{}, 0, true},
		{"trueAbsPAth", testenv.ImagePath, "exec", []string{"/bin/true"}, Opts{}, 0, true},
		{"false", testenv.ImagePath, "exec", []string{"false"}, Opts{}, 1, false},
		{"falseAbsPath", testenv.ImagePath, "exec", []string{"/bin/false"}, Opts{}, 1, false},
		// Scif apps tests
		{"ScifTestAppGood", testenv.ImagePath, "exec", []string{"testapp.sh"}, Opts{App: "testapp"}, 0, true},
		{"ScifTestAppBad", testenv.ImagePath, "exec", []string{"testapp.sh"}, Opts{App: "fakeapp"}, 1, false},
		{"ScifTestfolderOrg", testenv.ImagePath, "exec", []string{"test", "-d", "/scif"}, Opts{}, 0, true},
		{"ScifTestfolderOrg", testenv.ImagePath, "exec", []string{"test", "-d", "/scif/apps"}, Opts{}, 0, true},
		{"ScifTestfolderOrg", testenv.ImagePath, "exec", []string{"test", "-d", "/scif/data"}, Opts{}, 0, true},
		{"ScifTestfolderOrg", testenv.ImagePath, "exec", []string{"test", "-d", "/scif/apps/foo"}, Opts{}, 0, true},
		{"ScifTestfolderOrg", testenv.ImagePath, "exec", []string{"test", "-d", "/scif/apps/bar"}, Opts{}, 0, true},
		// blocked by issue [scif-apps] Files created at install step fall into an unexpected path #2404
		{"ScifTestfolderOrg", testenv.ImagePath, "exec", []string{"test", "-f", "/scif/apps/foo/filefoo.exec"}, Opts{}, 0, true},
		{"ScifTestfolderOrg", testenv.ImagePath, "exec", []string{"test", "-f", "/scif/apps/bar/filebar.exec"}, Opts{}, 0, true},
		{"ScifTestfolderOrg", testenv.ImagePath, "exec", []string{"test", "-d", "/scif/data/foo/output"}, Opts{}, 0, true},
		{"ScifTestfolderOrg", testenv.ImagePath, "exec", []string{"test", "-d", "/scif/data/foo/input"}, Opts{}, 0, true},
		{"WorkdirContain", testenv.ImagePath, "exec", []string{"test", "-f", tmpfile.Name()}, Opts{Workdir: "testdata", Contain: true}, 0, false},
		{"Workdir", testenv.ImagePath, "exec", []string{"test", "-f", tmpfile.Name()}, Opts{Workdir: "testdata"}, 0, true},
		{"pwdGood", testenv.ImagePath, "exec", []string{"true"}, Opts{Pwd: "/etc"}, 0, true},
		{"home", testenv.ImagePath, "exec", []string{"test", "-f", tmpfile.Name()}, Opts{Home: pwd + "testdata"}, 0, true},
		{"homePath", testenv.ImagePath, "exec", []string{"test", "-f", "/home/" + testfile.Name()}, Opts{Home: "/tmp:/home"}, 0, true},
		{"homeTmp", testenv.ImagePath, "exec", []string{"true"}, Opts{Home: "/tmp"}, 0, true},
		{"homeTmpExplicit", testenv.ImagePath, "exec", []string{"true"}, Opts{Home: "/tmp:/home"}, 0, true},
		{"ScifTestAppGood", testenv.ImagePath, "exec", []string{"testapp.sh"}, Opts{App: "testapp"}, 0, true},
		{"ScifTestAppBad", testenv.ImagePath, "exec", []string{"testapp.sh"}, Opts{App: "fakeapp"}, 1, false},
		//
		{"userBind", testenv.ImagePath, "exec", []string{"test", "-f", "/var/tmp/" + testfile.Name()}, Opts{Binds: []string{"/tmp:/var/tmp"}}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			_, stderr, exitCode, err := ImageExec(t, testenv.CmdPath, tt.action, tt.Opts, tt.image, tt.argv)
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
		_, stderr, exitCode, err := ImageExec(t, testenv.CmdPath, "exec", Opts{NoHome: true}, pwd+"/container.img", []string{"ls", "-ld", "$HOME"})
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
	tests := []struct {
		binName string
		name    string
		argv    []string
		exit    int
	}{
		{"sh", "trueSTDIN", []string{"-c", fmt.Sprintf("echo hi | singularity exec %s grep hi", testenv.ImagePath)}, 0},
		{"sh", "falseSTDIN", []string{"-c", fmt.Sprintf("echo bye | singularity exec %s grep hi", testenv.ImagePath)}, 1},
		// Checking permissions
		{"sh", "permissions", []string{"-c", fmt.Sprintf("singularity exec %s id -u | grep `id -u`", testenv.ImagePath)}, 0},
		// testing run command properly hands arguments
		{"sh", "arguments", []string{"-c", fmt.Sprintf("singularity run %s foo | grep foo", testenv.ImagePath)}, 0},
		// Stdin to URI based image
		{"sh", "library", []string{"-c", "echo true | singularity shell library://busybox"}, 0},
		{"sh", "docker", []string{"-c", "echo true | singularity shell docker://busybox"}, 0},
		{"sh", "shub", []string{"-c", "echo true | singularity shell shub://singularityhub/busybox"}, 0},
		// Test apps
		{"sh", "appsFoo", []string{"-c", fmt.Sprintf("singularity run --app foo %s | grep 'FOO'", testenv.ImagePath)}, 0},
		// Test target pwd
		{"sh", "pwdPath", []string{"-c", fmt.Sprintf("singularity exec --pwd /etc %s pwd | egrep '^/etc'", testenv.ImagePath)}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			cmd := exec.Command(tt.binName, tt.argv...)
			if err := cmd.Start(); err != nil {
				t.Fatalf("cmd.Start: %v", err)
			}

			if err := cmd.Wait(); err != nil {
				exiterr, _ := err.(*exec.ExitError)
				status, _ := exiterr.Sys().(syscall.WaitStatus)
				if status.ExitStatus() != tt.exit {
					// The program has exited with an unexpected exit code
					{
						t.Fatalf("unexpected exit code '%v': for cmd %v", status.ExitStatus(), strings.Join(tt.argv, " "))
					}
				}
			}
		}))
	}
}

// RunFromURI tests min fuctionality for singularity run/exec URI://
func RunFromURI(t *testing.T) {
	runScript := "testdata/runscript.sh"
	bind := fmt.Sprintf("%s:/.singularity.d/runscript", runScript)

	runOpts := Opts{
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
		Opts
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
		{"trueDocker", "docker://busybox:latest", "exec", []string{"true"}, Opts{}, true},
		{"trueLibrary", "library://busybox:latest", "exec", []string{"true"}, Opts{}, true},
		{"trueShub", "shub://singularityhub/busybox", "exec", []string{"true"}, Opts{}, true},
		{"falseDocker", "docker://busybox:latest", "exec", []string{"false"}, Opts{}, false},
		{"falselibrary", "library://busybox:latest", "exec", []string{"false"}, Opts{}, false},
		{"falseShub", "shub://singularityhub/busybox", "exec", []string{"false"}, Opts{}, false},
		// exec from URI with user namespace enabled
		{"trueDockerUserns", "docker://busybox:latest", "exec", []string{"true"}, Opts{Userns: true}, true},
		{"trueLibraryUserns", "library://busybox:latest", "exec", []string{"true"}, Opts{Userns: true}, true},
		{"trueShubUserns", "shub://singularityhub/busybox", "exec", []string{"true"}, Opts{Userns: true}, true},
		{"falseDockerUserns", "docker://busybox:latest", "exec", []string{"false"}, Opts{Userns: true}, false},
		{"falselibraryUserns", "library://busybox:latest", "exec", []string{"false"}, Opts{Userns: true}, false},
		{"falseShubUserns", "shub://singularityhub/busybox", "exec", []string{"false"}, Opts{Userns: true}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			_, stderr, exitCode, err := ImageExec(t, testenv.CmdPath, tt.action, tt.Opts, tt.image, tt.argv)
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
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(squashfsImage)

	//  Create the overlay ext3 fs
	cmd = exec.Command("dd", "if=/dev/zero", "of=ext3_fs.img", "bs=1M", "count=768", "status=none")
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("mkfs.ext3", "-q", "-F", "ext3_fs.img")
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("ext3_fs.img")

	// create a file dir
	t.Run("overlay_create", test.WithPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := ImageExec(t, testenv.CmdPath, "exec", Opts{Overlay: []string{dir}}, testenv.ImagePath, []string{"touch", "/dir_overlay"})
		if exitCode != 0 {
			t.Log(stderr, err)
			t.Fatalf("unexpected failure running '%v'", strings.Join([]string{"test", "-f", "/dir_overlay"}, " "))
		}
	}))
	// look for the file dir
	t.Run("overlay_find", test.WithPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := ImageExec(t, testenv.CmdPath, "exec", Opts{Overlay: []string{dir}}, testenv.ImagePath, []string{"test", "-f", "/dir_overlay"})
		if exitCode != 0 {
			t.Log(stderr, err)
			t.Fatalf("unexpected failure running '%v'", strings.Join([]string{"test", "-f", "/dir_overlay"}, " "))
		}
	}))
	// create a file ext3
	t.Run("overlay_ext3_create", test.WithPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := ImageExec(t, testenv.CmdPath, "exec", Opts{Overlay: []string{"ext3_fs.img"}}, testenv.ImagePath, []string{"touch", "/ext3_overlay"})
		if exitCode != 0 {
			t.Log(stderr, err)
			t.Fatalf("unexpected failure running '%v'", strings.Join([]string{"test", "-f", "/ext3_overlay"}, " "))
		}
	}))
	// look for the file ext3
	t.Run("overlay_ext3_find", test.WithPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := ImageExec(t, testenv.CmdPath, "exec", Opts{Overlay: []string{"ext3_fs.img"}}, testenv.ImagePath, []string{"test", "-f", "/ext3_overlay"})
		if exitCode != 0 {
			t.Log(stderr, err)
			t.Fatalf("unexpected failure running '%v'", strings.Join([]string{"test", "-f", "/ext3_overlay"}, " "))
		}
	}))
	// look for the file squashFs
	t.Run("overlay_squashFS_find", test.WithPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := ImageExec(t, testenv.CmdPath, "exec", Opts{Overlay: []string{squashfsImage}}, testenv.ImagePath, []string{"test", "-f", fmt.Sprintf("/%s", tmpfile.Name())})
		if exitCode != 0 {
			t.Log(stderr, err)
			t.Fatalf("unexpected failure running '%v'", strings.Join([]string{"test", "-f", fmt.Sprintf("/%s", tmpfile.Name())}, " "))
		}
	}))
	// create a file multiple overlays
	t.Run("overlay_multiple_create", test.WithPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := ImageExec(t, testenv.CmdPath, "exec", Opts{Overlay: []string{"ext3_fs.img", squashfsImage}}, testenv.ImagePath, []string{"touch", "/multiple_overlay_fs"})
		if exitCode != 0 {
			t.Log(stderr, err)
			t.Fatalf("unexpected failure running '%v'", strings.Join([]string{"touch", "/multiple_overlay_fs"}, " "))
		}
	}))
	// look for the file with multiple overlays
	t.Run("overlay_multiple_find_ext3", test.WithPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := ImageExec(t, testenv.CmdPath, "exec", Opts{Overlay: []string{"ext3_fs.img", squashfsImage}}, testenv.ImagePath, []string{"test", "-f", "/multiple_overlay_fs"})
		if exitCode != 0 {
			t.Log(stderr, err)
			t.Fatalf("unexpected failure running '%v'", strings.Join([]string{"test", "-f", "multiple_overlay_fs"}, " "))
		}
	}))
	t.Run("overlay_multiple_find_squashfs", test.WithPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := ImageExec(t, testenv.CmdPath, "exec", Opts{Overlay: []string{"ext3_fs.img", squashfsImage}}, testenv.ImagePath, []string{"test", "-f", fmt.Sprintf("/%s", tmpfile.Name())})
		if exitCode != 0 {
			t.Log(stderr, err)
			t.Fatalf("unexpected failure running '%v'", strings.Join([]string{"test", "-f", fmt.Sprintf("%s", tmpfile.Name())}, " "))
		}
	}))
	// look for the file without root privs
	t.Run("overlay_noroot", test.WithoutPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := ImageExec(t, testenv.CmdPath, "exec", Opts{Overlay: []string{dir}}, testenv.ImagePath, []string{"test", "-f", "/foo_overlay"})
		if exitCode != 1 {
			t.Log(stderr, err)
			t.Fatalf("unexpected success running '%v'", strings.Join([]string{"test", "-f", "/foo_overlay"}, " "))
		}
	}))
	// look for the file without --overlay
	t.Run("overlay_noflag", test.WithPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := ImageExec(t, testenv.CmdPath, "exec", Opts{}, testenv.ImagePath, []string{"test", "-f", "/foo_overlay"})
		if exitCode != 1 {
			t.Log(stderr, err)
			t.Fatalf("unexpected success running '%v'", strings.Join([]string{"test", "-f", "/foo_overlay"}, " "))
		}
	}))
}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(t *testing.T) {
	err := envconfig.Process("E2E", &testenv)
	if err != nil {
		t.Fatal(err.Error())
	}

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
