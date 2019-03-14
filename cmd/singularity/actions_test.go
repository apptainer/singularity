// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

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

	"github.com/sylabs/singularity/internal/pkg/test"
)

//build base image for tests
const imagePath = "./container.sif"
const appsImage = "./appsImage.sif"

type opts struct {
	keepPrivs bool
	contain   bool
	noHome    bool
	userns    bool
	binds     []string
	security  []string
	dropCaps  string
	home      string
	workdir   string
	pwd       string
	app       string
	overlay   []string
}

// imageExec can be used to run/exec/shell a Singularity image
// it return the exitCode and err of the execution
func imageExec(t *testing.T, action string, opts opts, imagePath string, command []string) (stdout string, stderr string, exitCode int, err error) {
	// action can be run/exec/shell
	argv := []string{action}
	for _, bind := range opts.binds {
		argv = append(argv, "--bind", bind)
	}
	for _, sec := range opts.security {
		argv = append(argv, "--security", sec)
	}
	if opts.keepPrivs {
		argv = append(argv, "--keep-privs")
	}
	if opts.dropCaps != "" {
		argv = append(argv, "--drop-caps", opts.dropCaps)
	}
	if opts.contain {
		argv = append(argv, "--contain")
	}
	if opts.noHome {
		argv = append(argv, "--no-home")
	}
	if opts.home != "" {
		argv = append(argv, "--home", opts.home)
	}
	for _, fs := range opts.overlay {
		argv = append(argv, "--overlay", fs)
	}
	if opts.workdir != "" {
		argv = append(argv, "--workdir", opts.workdir)
	}
	if opts.pwd != "" {
		argv = append(argv, "--pwd", opts.pwd)
	}
	if opts.app != "" {
		argv = append(argv, "--app", opts.app)
	}
	if opts.userns {
		argv = append(argv, "-u")
	}
	argv = append(argv, imagePath)
	argv = append(argv, command...)

	var outbuf, errbuf bytes.Buffer
	cmd := exec.Command(cmdPath, argv...)

	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf

	if err := cmd.Start(); err != nil {
		t.Fatalf("cmd.Start: %v", err)
	}

	// retrieve exit code
	if err := cmd.Wait(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			// The program has exited with an exit code != 0
			exitCode = 1
		}
	}

	stdout = outbuf.String()
	stderr = errbuf.String()

	return
}

// testSingularityRun tests min fuctionality for singularity run
func testSingularityRun(t *testing.T) {
	tests := []struct {
		name   string
		image  string
		action string
		argv   []string
		opts
		exit          int
		expectSuccess bool
	}{
		{"NoCommand", imagePath, "run", []string{}, opts{}, 0, true},
		{"true", imagePath, "run", []string{"true"}, opts{}, 0, true},
		{"false", imagePath, "run", []string{"false"}, opts{}, 1, false},
		{"ScifTestAppGood", imagePath, "run", []string{}, opts{app: "testapp"}, 0, true},
		{"ScifTestAppBad", imagePath, "run", []string{}, opts{app: "fakeapp"}, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			_, stderr, exitCode, err := imageExec(t, tt.action, tt.opts, tt.image, tt.argv)
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

// testSingularityExec tests min fuctionality for singularity exec
func testSingularityExec(t *testing.T) {
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
		opts
		exit          int
		expectSuccess bool
	}{
		{"NoCommand", imagePath, "exec", []string{}, opts{}, 1, false},
		{"true", imagePath, "exec", []string{"true"}, opts{}, 0, true},
		{"trueAbsPAth", imagePath, "exec", []string{"/bin/true"}, opts{}, 0, true},
		{"false", imagePath, "exec", []string{"false"}, opts{}, 1, false},
		{"falseAbsPath", imagePath, "exec", []string{"/bin/false"}, opts{}, 1, false},
		// Scif apps tests
		{"ScifTestAppGood", imagePath, "exec", []string{"testapp.sh"}, opts{app: "testapp"}, 0, true},
		{"ScifTestAppBad", imagePath, "exec", []string{"testapp.sh"}, opts{app: "fakeapp"}, 1, false},
		{"ScifTestfolderOrg", appsImage, "exec", []string{"test", "-d", "/scif"}, opts{}, 0, true},
		{"ScifTestfolderOrg", appsImage, "exec", []string{"test", "-d", "/scif/apps"}, opts{}, 0, true},
		{"ScifTestfolderOrg", appsImage, "exec", []string{"test", "-d", "/scif/data"}, opts{}, 0, true},
		{"ScifTestfolderOrg", appsImage, "exec", []string{"test", "-d", "/scif/apps/foo"}, opts{}, 0, true},
		{"ScifTestfolderOrg", appsImage, "exec", []string{"test", "-d", "/scif/apps/bar"}, opts{}, 0, true},
		// blocked by issue [scif-apps] Files created at install step fall into an unexpected path #2404
		{"ScifTestfolderOrg", appsImage, "exec", []string{"test", "-f", "/scif/apps/foo/filefoo.exec"}, opts{}, 0, true},
		{"ScifTestfolderOrg", appsImage, "exec", []string{"test", "-f", "/scif/apps/bar/filebar.exec"}, opts{}, 0, true},
		{"ScifTestfolderOrg", appsImage, "exec", []string{"test", "-d", "/scif/data/foo/output"}, opts{}, 0, true},
		{"ScifTestfolderOrg", appsImage, "exec", []string{"test", "-d", "/scif/data/foo/input"}, opts{}, 0, true},
		{"WorkdirContain", imagePath, "exec", []string{"test", "-f", tmpfile.Name()}, opts{workdir: "testdata", contain: true}, 0, false},
		{"Workdir", imagePath, "exec", []string{"test", "-f", tmpfile.Name()}, opts{workdir: "testdata"}, 0, true},
		{"pwdGood", imagePath, "exec", []string{"true"}, opts{pwd: "/etc"}, 0, true},
		{"home", imagePath, "exec", []string{"test", "-f", tmpfile.Name()}, opts{home: pwd + "testdata"}, 0, true},
		{"homePath", imagePath, "exec", []string{"test", "-f", "/home/" + testfile.Name()}, opts{home: "/tmp:/home"}, 0, true},
		{"homeTmp", imagePath, "exec", []string{"true"}, opts{home: "/tmp"}, 0, true},
		{"homeTmpExplicit", imagePath, "exec", []string{"true"}, opts{home: "/tmp:/home"}, 0, true},
		{"ScifTestAppGood", imagePath, "exec", []string{"testapp.sh"}, opts{app: "testapp"}, 0, true},
		{"ScifTestAppBad", imagePath, "exec", []string{"testapp.sh"}, opts{app: "fakeapp"}, 1, false},
		//
		{"userBind", imagePath, "exec", []string{"test", "-f", "/var/tmp/" + testfile.Name()}, opts{binds: []string{"/tmp:/var/tmp"}}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			_, stderr, exitCode, err := imageExec(t, tt.action, tt.opts, tt.image, tt.argv)
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
		_, stderr, exitCode, err := imageExec(t, "exec", opts{noHome: true}, pwd+"/container.img", []string{"ls", "-ld", "$HOME"})
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

// testSTDINPipe tests pipe stdin to singularity actions cmd
func testSTDINPipe(t *testing.T) {
	tests := []struct {
		binName string
		name    string
		argv    []string
		exit    int
	}{
		{"sh", "trueSTDIN", []string{"-c", fmt.Sprintf("echo hi | singularity exec %s grep hi", imagePath)}, 0},
		{"sh", "falseSTDIN", []string{"-c", fmt.Sprintf("echo bye | singularity exec %s grep hi", imagePath)}, 1},
		// Checking permissions
		{"sh", "permissions", []string{"-c", fmt.Sprintf("singularity exec %s id -u | grep `id -u`", imagePath)}, 0},
		// testing run command properly hands arguments
		{"sh", "arguments", []string{"-c", fmt.Sprintf("singularity run %s foo | grep foo", imagePath)}, 0},
		// Stdin to URI based image
		{"sh", "library", []string{"-c", "echo true | singularity shell library://busybox"}, 0},
		{"sh", "docker", []string{"-c", "echo true | singularity shell docker://busybox"}, 0},
		{"sh", "shub", []string{"-c", "echo true | singularity shell shub://singularityhub/busybox"}, 0},
		// Test apps
		{"sh", "appsFoo", []string{"-c", fmt.Sprintf("singularity run --app foo %s | grep 'FOO'", appsImage)}, 0},
		// Test target pwd
		{"sh", "pwdPath", []string{"-c", fmt.Sprintf("singularity exec --pwd /etc %s pwd | egrep '^/etc'", imagePath)}, 0},
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

// testRunFromURI tests min fuctionality for singularity run/exec URI://
func testRunFromURI(t *testing.T) {
	runScript := "testdata/runscript.sh"
	bind := fmt.Sprintf("%s:/.singularity.d/runscript", runScript)

	runOpts := opts{
		binds: []string{bind},
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
		opts
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
		{"trueDocker", "docker://busybox:latest", "exec", []string{"true"}, opts{}, true},
		{"trueLibrary", "library://busybox:latest", "exec", []string{"true"}, opts{}, true},
		{"trueShub", "shub://singularityhub/busybox", "exec", []string{"true"}, opts{}, true},
		{"falseDocker", "docker://busybox:latest", "exec", []string{"false"}, opts{}, false},
		{"falselibrary", "library://busybox:latest", "exec", []string{"false"}, opts{}, false},
		{"falseShub", "shub://singularityhub/busybox", "exec", []string{"false"}, opts{}, false},
		// exec from URI with user namespace enabled
		{"trueDockerUserns", "docker://busybox:latest", "exec", []string{"true"}, opts{userns: true}, true},
		{"trueLibraryUserns", "library://busybox:latest", "exec", []string{"true"}, opts{userns: true}, true},
		{"trueShubUserns", "shub://singularityhub/busybox", "exec", []string{"true"}, opts{userns: true}, true},
		{"falseDockerUserns", "docker://busybox:latest", "exec", []string{"false"}, opts{userns: true}, false},
		{"falselibraryUserns", "library://busybox:latest", "exec", []string{"false"}, opts{userns: true}, false},
		{"falseShubUserns", "shub://singularityhub/busybox", "exec", []string{"false"}, opts{userns: true}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			_, stderr, exitCode, err := imageExec(t, tt.action, tt.opts, tt.image, tt.argv)
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

// testPersistentOverlay test the --overlay function
func testPersistentOverlay(t *testing.T) {
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
		_, stderr, exitCode, err := imageExec(t, "exec", opts{overlay: []string{dir}}, imagePath, []string{"touch", "/dir_overlay"})
		if exitCode != 0 {
			t.Log(stderr, err)
			t.Fatalf("unexpected failure running '%v'", strings.Join([]string{"test", "-f", "/dir_overlay"}, " "))
		}
	}))
	// look for the file dir
	t.Run("overlay_find", test.WithPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := imageExec(t, "exec", opts{overlay: []string{dir}}, imagePath, []string{"test", "-f", "/dir_overlay"})
		if exitCode != 0 {
			t.Log(stderr, err)
			t.Fatalf("unexpected failure running '%v'", strings.Join([]string{"test", "-f", "/dir_overlay"}, " "))
		}
	}))
	// create a file ext3
	t.Run("overlay_ext3_create", test.WithPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := imageExec(t, "exec", opts{overlay: []string{"ext3_fs.img"}}, imagePath, []string{"touch", "/ext3_overlay"})
		if exitCode != 0 {
			t.Log(stderr, err)
			t.Fatalf("unexpected failure running '%v'", strings.Join([]string{"test", "-f", "/ext3_overlay"}, " "))
		}
	}))
	// look for the file ext3
	t.Run("overlay_ext3_find", test.WithPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := imageExec(t, "exec", opts{overlay: []string{"ext3_fs.img"}}, imagePath, []string{"test", "-f", "/ext3_overlay"})
		if exitCode != 0 {
			t.Log(stderr, err)
			t.Fatalf("unexpected failure running '%v'", strings.Join([]string{"test", "-f", "/ext3_overlay"}, " "))
		}
	}))
	// look for the file squashFs
	t.Run("overlay_squashFS_find", test.WithPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := imageExec(t, "exec", opts{overlay: []string{squashfsImage}}, imagePath, []string{"test", "-f", fmt.Sprintf("/%s", tmpfile.Name())})
		if exitCode != 0 {
			t.Log(stderr, err)
			t.Fatalf("unexpected failure running '%v'", strings.Join([]string{"test", "-f", fmt.Sprintf("/%s", tmpfile.Name())}, " "))
		}
	}))
	// create a file multiple overlays
	t.Run("overlay_multiple_create", test.WithPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := imageExec(t, "exec", opts{overlay: []string{"ext3_fs.img", squashfsImage}}, imagePath, []string{"touch", "/multiple_overlay_fs"})
		if exitCode != 0 {
			t.Log(stderr, err)
			t.Fatalf("unexpected failure running '%v'", strings.Join([]string{"touch", "/multiple_overlay_fs"}, " "))
		}
	}))
	// look for the file with multiple overlays
	t.Run("overlay_multiple_find_ext3", test.WithPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := imageExec(t, "exec", opts{overlay: []string{"ext3_fs.img", squashfsImage}}, imagePath, []string{"test", "-f", "/multiple_overlay_fs"})
		if exitCode != 0 {
			t.Log(stderr, err)
			t.Fatalf("unexpected failure running '%v'", strings.Join([]string{"test", "-f", "multiple_overlay_fs"}, " "))
		}
	}))
	t.Run("overlay_multiple_find_squashfs", test.WithPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := imageExec(t, "exec", opts{overlay: []string{"ext3_fs.img", squashfsImage}}, imagePath, []string{"test", "-f", fmt.Sprintf("/%s", tmpfile.Name())})
		if exitCode != 0 {
			t.Log(stderr, err)
			t.Fatalf("unexpected failure running '%v'", strings.Join([]string{"test", "-f", fmt.Sprintf("/%s", tmpfile.Name())}, " "))
		}
	}))
	// look for the file without root privs
	t.Run("overlay_noroot", test.WithoutPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := imageExec(t, "exec", opts{overlay: []string{dir}}, imagePath, []string{"test", "-f", "/foo_overlay"})
		if exitCode != 1 {
			t.Log(stderr, err)
			t.Fatalf("unexpected success running '%v'", strings.Join([]string{"test", "-f", "/foo_overlay"}, " "))
		}
	}))
	// look for the file without --overlay
	t.Run("overlay_noflag", test.WithPrivilege(func(t *testing.T) {
		_, stderr, exitCode, err := imageExec(t, "exec", opts{}, imagePath, []string{"test", "-f", "/foo_overlay"})
		if exitCode != 1 {
			t.Log(stderr, err)
			t.Fatalf("unexpected success running '%v'", strings.Join([]string{"test", "-f", "/foo_overlay"}, " "))
		}
	}))
}

func TestSingularityActions(t *testing.T) {
	test.EnsurePrivilege(t)
	opts := buildOpts{
		force:   true,
		sandbox: false,
	}
	if b, err := imageBuild(opts, imagePath, "../../examples/busybox/Singularity"); err != nil {
		t.Log(string(b))
		t.Fatalf("unexpected failure: %v", err)
	}
	defer os.Remove(imagePath)
	if b, err := imageBuild(opts, appsImage, "../../examples/apps/Singularity"); err != nil {
		t.Log(string(b))
		t.Fatalf("unexpected failure: %v", err)
	}
	defer os.Remove(appsImage)

	// singularity run
	t.Run("run", testSingularityRun)
	// singularity exec
	t.Run("exec", testSingularityExec)
	// stdin pipe
	t.Run("STDIN", testSTDINPipe)
	// action_URI
	t.Run("action_URI", testRunFromURI)
	// Persistent Overlay
	t.Run("Persistent_Overlay", testPersistentOverlay)
}
