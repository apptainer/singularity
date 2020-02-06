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
	"path/filepath"
	"strconv"
	"testing"

	"github.com/pkg/errors"
	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/e2e/internal/testhelper"
	"github.com/sylabs/singularity/internal/pkg/test/tool/exec"
	"github.com/sylabs/singularity/internal/pkg/test/tool/require"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

type actionTests struct {
	env e2e.TestEnv
}

// run tests min fuctionality for singularity run
func (c actionTests) actionRun(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	tests := []struct {
		name string
		argv []string
		exit int
	}{
		{
			name: "NoCommand",
			argv: []string{c.env.ImagePath},
			exit: 0,
		},
		{
			name: "True",
			argv: []string{c.env.ImagePath, "true"},
			exit: 0,
		},
		{
			name: "False",
			argv: []string{c.env.ImagePath, "false"},
			exit: 1,
		},
		{
			name: "ScifTestAppGood",
			argv: []string{"--app", "testapp", c.env.ImagePath},
			exit: 0,
		},
		{
			name: "ScifTestAppBad",
			argv: []string{"--app", "fakeapp", c.env.ImagePath},
			exit: 1,
		},
	}

	for _, tt := range tests {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("run"),
			e2e.WithArgs(tt.argv...),
			e2e.ExpectExit(tt.exit),
		)
	}
}

// exec tests min fuctionality for singularity exec
func (c actionTests) actionExec(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	user := e2e.CurrentUser(t)

	// Create a temp testfile
	testdata, err := fs.MakeTmpDir(c.env.TestDir, "testdata", 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testdata)

	testdataTmp := filepath.Join(testdata, "tmp")
	if err := os.Mkdir(testdataTmp, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a temp testfile
	tmpfile, err := fs.MakeTmpFile(testdataTmp, "testSingularityExec.", 0644)
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	basename := filepath.Base(tmpfile.Name())
	tmpfilePath := filepath.Join("/tmp", basename)
	vartmpfilePath := filepath.Join("/var/tmp", basename)
	homePath := filepath.Join("/home", basename)

	tests := []struct {
		name string
		argv []string
		exit int
	}{
		{
			name: "NoCommand",
			argv: []string{c.env.ImagePath},
			exit: 1,
		},
		{
			name: "True",
			argv: []string{c.env.ImagePath, "true"},
			exit: 0,
		},
		{
			name: "TrueAbsPAth",
			argv: []string{c.env.ImagePath, "/bin/true"},
			exit: 0,
		},
		{
			name: "False",
			argv: []string{c.env.ImagePath, "false"},
			exit: 1,
		},
		{
			name: "FalseAbsPath",
			argv: []string{c.env.ImagePath, "/bin/false"},
			exit: 1,
		},
		// Scif apps tests
		{
			name: "ScifTestAppGood",
			argv: []string{"--app", "testapp", c.env.ImagePath, "testapp.sh"},
			exit: 0,
		},
		{
			name: "ScifTestAppBad",
			argv: []string{"--app", "fakeapp", c.env.ImagePath, "testapp.sh"},
			exit: 1,
		},
		{
			name: "ScifTestfolderOrg",
			argv: []string{c.env.ImagePath, "test", "-d", "/scif"},
			exit: 0,
		},
		{
			name: "ScifTestfolderOrg",
			argv: []string{c.env.ImagePath, "test", "-d", "/scif/apps"},
			exit: 0,
		},
		{
			name: "ScifTestfolderOrg",
			argv: []string{c.env.ImagePath, "test", "-d", "/scif/data"},
			exit: 0,
		},
		{
			name: "ScifTestfolderOrg",
			argv: []string{c.env.ImagePath, "test", "-d", "/scif/apps/foo"},
			exit: 0,
		},
		{
			name: "ScifTestfolderOrg",
			argv: []string{c.env.ImagePath, "test", "-d", "/scif/apps/bar"},
			exit: 0,
		},
		// blocked by issue [scif-apps] Files created at install step fall into an unexpected path #2404
		{
			name: "ScifTestfolderOrg",
			argv: []string{c.env.ImagePath, "test", "-f", "/scif/apps/foo/filefoo.exec"},
			exit: 0,
		},
		{
			name: "ScifTestfolderOrg",
			argv: []string{c.env.ImagePath, "test", "-f", "/scif/apps/bar/filebar.exec"},
			exit: 0,
		},
		{
			name: "ScifTestfolderOrg",
			argv: []string{c.env.ImagePath, "test", "-d", "/scif/data/foo/output"},
			exit: 0,
		},
		{
			name: "ScifTestfolderOrg",
			argv: []string{c.env.ImagePath, "test", "-d", "/scif/data/foo/input"},
			exit: 0,
		},
		{
			name: "ContainOnly",
			argv: []string{"--contain", c.env.ImagePath, "test", "-f", tmpfilePath},
			exit: 1,
		},
		{
			name: "WorkdirOnly",
			argv: []string{"--workdir", testdata, c.env.ImagePath, "test", "-f", tmpfilePath},
			exit: 1,
		},
		{
			name: "WorkdirContain",
			argv: []string{"--workdir", testdata, "--contain", c.env.ImagePath, "test", "-f", tmpfilePath},
			exit: 0,
		},
		{
			name: "PwdGood",
			argv: []string{"--pwd", "/etc", c.env.ImagePath, "true"},
			exit: 0,
		},
		{
			name: "Home",
			argv: []string{"--home", testdata, c.env.ImagePath, "test", "-f", tmpfile.Name()},
			exit: 0,
		},
		{
			name: "HomePath",
			argv: []string{"--home", testdataTmp + ":/home", c.env.ImagePath, "test", "-f", homePath},
			exit: 0,
		},
		{
			name: "HomeTmp",
			argv: []string{"--home", "/tmp", c.env.ImagePath, "true"},
			exit: 0,
		},
		{
			name: "HomeTmpExplicit",
			argv: []string{"--home", "/tmp:/home", c.env.ImagePath, "true"},
			exit: 0,
		},
		{
			name: "UserBindTmp",
			argv: []string{"--bind", testdataTmp + ":/tmp", c.env.ImagePath, "test", "-f", tmpfilePath},
			exit: 0,
		},
		{
			name: "UserBindVarTmp",
			argv: []string{"--bind", testdataTmp + ":/var/tmp", c.env.ImagePath, "test", "-f", vartmpfilePath},
			exit: 0,
		},
		{
			name: "NoHome",
			argv: []string{"--no-home", c.env.ImagePath, "ls", "-ld", user.Dir},
			exit: 1,
		},
	}

	for _, tt := range tests {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("exec"),
			e2e.WithDir("/tmp"),
			e2e.WithArgs(tt.argv...),
			e2e.ExpectExit(tt.exit),
		)
	}
}

// Shell interaction tests
func (c actionTests) actionShell(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	hostname, err := os.Hostname()
	err = errors.Wrap(err, "getting hostname")
	if err != nil {
		t.Fatalf("could not get hostname: %+v", err)
	}

	tests := []struct {
		name       string
		argv       []string
		consoleOps []e2e.SingularityConsoleOp
		exit       int
	}{
		{
			name: "ShellExit",
			argv: []string{c.env.ImagePath},
			consoleOps: []e2e.SingularityConsoleOp{
				// "cd /" to work around issue where a long
				// working directory name causes the test
				// to fail because the "Singularity" that
				// we are looking for is chopped from the
				// front.
				// TODO(mem): This test was added back in 491a71716013654acb2276e4b37c2e015d2dfe09
				e2e.ConsoleSendLine("cd /"),
				e2e.ConsoleExpect("Singularity"),
				e2e.ConsoleSendLine("exit"),
			},
			exit: 0,
		},
		{
			name: "ShellHostname",
			argv: []string{c.env.ImagePath},
			consoleOps: []e2e.SingularityConsoleOp{
				e2e.ConsoleSendLine("hostname"),
				e2e.ConsoleExpect(hostname),
				e2e.ConsoleSendLine("exit"),
			},
			exit: 0,
		},
		{
			name: "ShellBadCommand",
			argv: []string{c.env.ImagePath},
			consoleOps: []e2e.SingularityConsoleOp{
				e2e.ConsoleSendLine("_a_fake_command"),
				e2e.ConsoleSendLine("exit"),
			},
			exit: 127,
		},
	}

	for _, tt := range tests {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("shell"),
			e2e.WithArgs(tt.argv...),
			e2e.ConsoleRun(tt.consoleOps...),
			e2e.ExpectExit(tt.exit),
		)
	}
}

// STDPipe tests pipe stdin/stdout to singularity actions cmd
func (c actionTests) STDPipe(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	stdinTests := []struct {
		name    string
		command string
		argv    []string
		input   string
		exit    int
	}{
		{
			name:    "TrueSTDIN",
			command: "exec",
			argv:    []string{c.env.ImagePath, "grep", "hi"},
			input:   "hi",
			exit:    0,
		},
		{
			name:    "FalseSTDIN",
			command: "exec",
			argv:    []string{c.env.ImagePath, "grep", "hi"},
			input:   "bye",
			exit:    1,
		},
		{
			name:    "TrueLibrary",
			command: "shell",
			argv:    []string{"library://busybox"},
			input:   "true",
			exit:    0,
		},
		{
			name:    "FalseLibrary",
			command: "shell",
			argv:    []string{"library://busybox"},
			input:   "false",
			exit:    1,
		},
		{
			name:    "TrueDocker",
			command: "shell",
			argv:    []string{"docker://busybox"},
			input:   "true",
			exit:    0,
		},
		{
			name:    "FalseDocker",
			command: "shell",
			argv:    []string{"docker://busybox"},
			input:   "false",
			exit:    1,
		},
		// TODO(mem): reenable this; disabled while shub is down
		// {
		// 	name:    "TrueShub",
		// 	command: "shell",
		// 	argv:    []string{"shub://singularityhub/busybox"},
		// 	input:   "true",
		// 	exit:    0,
		// },
		// TODO(mem): reenable this; disabled while shub is down
		// {
		// 	name:    "FalseShub",
		// 	command: "shell",
		// 	argv:    []string{"shub://singularityhub/busybox"},
		// 	input:   "false",
		// 	exit:    1,
		// },
	}

	var input bytes.Buffer

	for _, tt := range stdinTests {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand(tt.command),
			e2e.WithArgs(tt.argv...),
			e2e.WithStdin(&input),
			e2e.PreRun(func(t *testing.T) {
				input.WriteString(tt.input)
			}),
			e2e.ExpectExit(tt.exit),
		)
		input.Reset()
	}

	user := e2e.CurrentUser(t)
	stdoutTests := []struct {
		name    string
		command string
		argv    []string
		output  string
		exit    int
	}{
		{
			name:    "AppsFoo",
			command: "run",
			argv:    []string{"--app", "foo", c.env.ImagePath},
			output:  "RUNNING FOO",
			exit:    0,
		},
		{
			name:    "PwdPath",
			command: "exec",
			argv:    []string{"--pwd", "/etc", c.env.ImagePath, "pwd"},
			output:  "/etc",
			exit:    0,
		},
		{
			name:    "Arguments",
			command: "run",
			argv:    []string{c.env.ImagePath, "foo"},
			output:  "Running command: foo",
			exit:    127,
		},
		{
			name:    "Permissions",
			command: "exec",
			argv:    []string{c.env.ImagePath, "id", "-un"},
			output:  user.Name,
			exit:    0,
		},
	}
	for _, tt := range stdoutTests {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand(tt.command),
			e2e.WithArgs(tt.argv...),
			e2e.ExpectExit(
				tt.exit,
				e2e.ExpectOutput(e2e.ExactMatch, tt.output),
			),
		)
	}
}

// RunFromURI tests min fuctionality for singularity run/exec URI://
func (c actionTests) RunFromURI(t *testing.T) {
	e2e.PrepRegistry(t, c.env)

	runScript := "testdata/runscript.sh"
	bind := fmt.Sprintf("%s:/.singularity.d/runscript", runScript)

	fi, err := os.Stat(runScript)
	if err != nil {
		t.Fatalf("can't find %s", runScript)
	}
	size := strconv.Itoa(int(fi.Size()))

	tests := []struct {
		name    string
		command string
		argv    []string
		exit    int
	}{
		// Run from supported URI's and check the runscript call works
		{
			name:    "RunFromDockerOK",
			command: "run",
			argv:    []string{"--bind", bind, "docker://busybox:latest", size},
			exit:    0,
		},
		{
			name:    "RunFromLibraryOK",
			command: "run",
			argv:    []string{"--bind", bind, "library://busybox:latest", size},
			exit:    0,
		},
		// TODO(mem): reenable this; disabled while shub is down
		// {
		// 	name:    "RunFromShubOK",
		// 	command: "run",
		// 	argv:    []string{"--bind", bind, "shub://singularityhub/busybox", size},
		// 	exit:    0,
		// },
		{
			name:    "RunFromOrasOK",
			command: "run",
			argv:    []string{"--bind", bind, c.env.OrasTestImage, size},
			exit:    0,
		},
		{
			name:    "RunFromDockerKO",
			command: "run",
			argv:    []string{"--bind", bind, "docker://busybox:latest", "0"},
			exit:    1,
		},
		{
			name:    "RunFromLibraryKO",
			command: "run",
			argv:    []string{"--bind", bind, "library://busybox:latest", "0"},
			exit:    1,
		},
		// TODO(mem): reenable this; disabled while shub is down
		// {
		// 	name:    "RunFromShubKO",
		// 	command: "run",
		// 	argv:    []string{"--bind", bind, "shub://singularityhub/busybox", "0"},
		// 	exit:    1,
		// },
		{
			name:    "RunFromOrasKO",
			command: "run",
			argv:    []string{"--bind", bind, c.env.OrasTestImage, "0"},
			exit:    1,
		},

		// exec from a supported URI's and check the exit code
		{
			name:    "ExecTrueDocker",
			command: "exec",
			argv:    []string{"docker://busybox:latest", "true"},
			exit:    0,
		},
		{
			name:    "ExecTrueLibrary",
			command: "exec",
			argv:    []string{"library://busybox:latest", "true"},
			exit:    0,
		},
		// TODO(mem): reenable this; disabled while shub is down
		// {
		// 	name:    "ExecTrueShub",
		// 	command: "exec",
		// 	argv:    []string{"shub://singularityhub/busybox", "true"},
		// 	exit:    0,
		// },
		{
			name:    "ExecTrueOras",
			command: "exec",
			argv:    []string{c.env.OrasTestImage, "true"},
			exit:    0,
		},
		{
			name:    "ExecFalseDocker",
			command: "exec",
			argv:    []string{"docker://busybox:latest", "false"},
			exit:    1,
		},
		{
			name:    "ExecFalseLibrary",
			command: "exec",
			argv:    []string{"library://busybox:latest", "false"},
			exit:    1,
		},
		// TODO(mem): reenable this; disabled while shub is down
		// {
		// 	name:    "ExecFalseShub",
		// 	command: "exec",
		// 	argv:    []string{"shub://singularityhub/busybox", "false"},
		// 	exit:    1,
		// },
		{
			name:    "ExecFalseOras",
			command: "exec",
			argv:    []string{c.env.OrasTestImage, "false"},
			exit:    1,
		},

		// exec from URI with user namespace enabled
		{
			name:    "ExecTrueDockerUserns",
			command: "exec",
			argv:    []string{"--userns", "docker://busybox:latest", "true"},
			exit:    0,
		},
		{
			name:    "ExecTrueLibraryUserns",
			command: "exec",
			argv:    []string{"--userns", "library://busybox:latest", "true"},
			exit:    0,
		},
		// TODO(mem): reenable this; disabled while shub is down
		// {
		// 	name:    "ExecTrueShubUserns",
		// 	command: "exec",
		// 	argv:    []string{"--userns", "shub://singularityhub/busybox", "true"},
		// 	exit:    0,
		// },
		{
			name:    "ExecTrueOrasUserns",
			command: "exec",
			argv:    []string{"--userns", c.env.OrasTestImage, "true"},
			exit:    0,
		},
		{
			name:    "ExecFalseDockerUserns",
			command: "exec",
			argv:    []string{"--userns", "docker://busybox:latest", "false"},
			exit:    1,
		},
		{
			name:    "ExecFalseLibraryUserns",
			command: "exec",
			argv:    []string{"--userns", "library://busybox:latest", "false"},
			exit:    1,
		},
		// TODO(mem): reenable this; disabled while shub is down
		// {
		// 	name:    "ExecFalseShubUserns",
		// 	command: "exec",
		// 	argv:    []string{"--userns", "shub://singularityhub/busybox", "false"},
		// 	exit:    1,
		// },
		{
			name:    "ExecFalseOrasUserns",
			command: "exec",
			argv:    []string{"--userns", c.env.OrasTestImage, "false"},
			exit:    1,
		},
	}

	for _, tt := range tests {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand(tt.command),
			e2e.WithArgs(tt.argv...),
			e2e.ExpectExit(tt.exit),
		)
	}
}

// PersistentOverlay test the --overlay function
func (c actionTests) PersistentOverlay(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	require.Filesystem(t, "overlay")

	require.Command(t, "mkfs.ext3")
	require.Command(t, "mksquashfs")
	require.Command(t, "dd")

	testdir, err := ioutil.TempDir(c.env.TestDir, "persistent-overlay-")
	if err != nil {
		t.Fatal(err)
	}

	cleanup := func(t *testing.T) {
		if t.Failed() {
			t.Logf("Not removing directory %s for test %s", testdir, t.Name())
			return
		}
		err := os.RemoveAll(testdir)
		if err != nil {
			t.Logf("Error while removing directory %s for test %s: %#v", testdir, t.Name(), err)
		}
	}
	// sandbox overlay implies creation of upper/work directories by
	// Singularity, so we would need privileges to delete the test
	// directory correctly
	defer e2e.Privileged(cleanup)

	squashfsImage := filepath.Join(testdir, "squashfs.simg")
	ext3Img := filepath.Join(testdir, "ext3_fs.img")
	sandboxImage := filepath.Join(testdir, "sandbox")

	// create an overlay directory
	dir, err := ioutil.TempDir(testdir, "overlay-dir-")
	if err != nil {
		t.Fatal(err)
	}

	// create root directory for squashfs image
	squashDir, err := ioutil.TempDir(testdir, "root-squash-dir-")
	if err != nil {
		t.Fatal(err)
	}

	squashMarkerFile := "squash_marker"
	if err := fs.Touch(filepath.Join(squashDir, squashMarkerFile)); err != nil {
		t.Fatal(err)
	}

	// create the squashfs overlay image
	cmd := exec.Command("mksquashfs", squashDir, squashfsImage, "-noappend", "-all-root")
	if res := cmd.Run(t); res.Error != nil {
		t.Fatalf("Unexpected error while running command.\n%s", res)
	}

	// create the overlay ext3 image
	cmd = exec.Command("dd", "if=/dev/zero", "of="+ext3Img, "bs=1M", "count=64", "status=none")
	if res := cmd.Run(t); res.Error != nil {
		t.Fatalf("Unexpected error while running command.\n%s", res)
	}

	cmd = exec.Command("mkfs.ext3", "-q", "-F", ext3Img)
	if res := cmd.Run(t); res.Error != nil {
		t.Fatalf("Unexpected error while running command.\n%s", res)
	}

	// create a sandbox image from test image
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs("--sandbox", sandboxImage, c.env.ImagePath),
		e2e.PostRun(func(t *testing.T) {
			if t.Failed() {
				t.Fatalf("failed to create sandbox %s from test image %s", sandboxImage, c.env.ImagePath)
			}
		}),
		e2e.ExpectExit(0),
	)

	tests := []struct {
		name    string
		argv    []string
		dir     string
		exit    int
		profile e2e.Profile
	}{
		{
			name:    "overlay_create",
			argv:    []string{"--overlay", dir, c.env.ImagePath, "touch", "/dir_overlay"},
			exit:    0,
			profile: e2e.RootProfile,
		},
		{
			name:    "overlay_find",
			argv:    []string{"--overlay", dir, c.env.ImagePath, "test", "-f", "/dir_overlay"},
			exit:    0,
			profile: e2e.RootProfile,
		},
		{
			name:    "overlay_find_with_writable_fail",
			argv:    []string{"--overlay", dir, "--writable", c.env.ImagePath, "true"},
			exit:    255,
			profile: e2e.RootProfile,
		},
		{
			name:    "overlay_find_with_writable_tmpfs",
			argv:    []string{"--overlay", dir + ":ro", "--writable-tmpfs", c.env.ImagePath, "test", "-f", "/dir_overlay"},
			exit:    0,
			profile: e2e.RootProfile,
		},
		{
			name:    "overlay_find_with_writable_tmpfs_fail",
			argv:    []string{"--overlay", dir, "--writable-tmpfs", c.env.ImagePath, "true"},
			exit:    255,
			profile: e2e.RootProfile,
		},
		{
			name:    "overlay_ext3_create",
			argv:    []string{"--overlay", ext3Img, c.env.ImagePath, "touch", "/ext3_overlay"},
			exit:    0,
			profile: e2e.RootProfile,
		},
		{
			name:    "overlay_ext3_find",
			argv:    []string{"--overlay", ext3Img, c.env.ImagePath, "test", "-f", "/ext3_overlay"},
			exit:    0,
			profile: e2e.RootProfile,
		},
		{
			name:    "overlay_multiple_writable_fail",
			argv:    []string{"--overlay", ext3Img, "--overlay", ext3Img, c.env.ImagePath, "true"},
			exit:    255,
			profile: e2e.RootProfile,
		},
		{
			name:    "overlay_squashFS_find",
			argv:    []string{"--overlay", squashfsImage + ":ro", c.env.ImagePath, "test", "-f", fmt.Sprintf("/%s", squashMarkerFile)},
			exit:    0,
			profile: e2e.RootProfile,
		},
		{
			name:    "overlay_squashFS_find_fail_without_ro",
			argv:    []string{"--overlay", squashfsImage, c.env.ImagePath, "true"},
			exit:    255,
			profile: e2e.RootProfile,
		},
		{
			name:    "overlay_multiple_create",
			argv:    []string{"--overlay", ext3Img, "--overlay", squashfsImage + ":ro", c.env.ImagePath, "touch", "/multiple_overlay_fs"},
			exit:    0,
			profile: e2e.RootProfile,
		},
		{
			name:    "overlay_multiple_find_ext3",
			argv:    []string{"--overlay", ext3Img, "--overlay", squashfsImage + ":ro", c.env.ImagePath, "test", "-f", "/multiple_overlay_fs"},
			exit:    0,
			profile: e2e.RootProfile,
		},
		{
			name:    "overlay_multiple_find_squashfs",
			argv:    []string{"--overlay", ext3Img, "--overlay", squashfsImage + ":ro", c.env.ImagePath, "test", "-f", fmt.Sprintf("/%s", squashMarkerFile)},
			exit:    0,
			profile: e2e.RootProfile,
		},
		{
			name:    "overlay_noroot",
			argv:    []string{"--overlay", dir, c.env.ImagePath, "true"},
			exit:    255,
			profile: e2e.UserProfile,
		},
		{
			name:    "overlay_noflag",
			argv:    []string{c.env.ImagePath, "test", "-f", "/foo_overlay"},
			exit:    1,
			profile: e2e.RootProfile,
		},
		{
			// https://github.com/sylabs/singularity/issues/4329
			name:    "SIF_writable_without_overlay_partition_issue_4329",
			argv:    []string{"--writable", c.env.ImagePath, "true"},
			exit:    255,
			profile: e2e.RootProfile,
		},
		{
			// https://github.com/sylabs/singularity/issues/4270
			name:    "overlay_dir_relative_path_issue_4270",
			argv:    []string{"--overlay", filepath.Base(dir), sandboxImage, "test", "-f", "/dir_overlay"},
			dir:     filepath.Dir(dir),
			exit:    0,
			profile: e2e.RootProfile,
		},
	}

	for _, tt := range tests {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(tt.profile),
			e2e.WithDir(tt.dir),
			e2e.WithCommand("exec"),
			e2e.WithArgs(tt.argv...),
			e2e.ExpectExit(tt.exit),
		)
	}
}

func (c actionTests) actionBasicProfiles(t *testing.T) {
	env := c.env

	e2e.EnsureImage(t, env)

	tests := []struct {
		name    string
		command string
		argv    []string
		exit    int
	}{
		{
			name:    "ExecTrue",
			command: "exec",
			argv:    []string{env.ImagePath, "true"},
			exit:    0,
		},
		{
			name:    "ExecPidNsTrue",
			command: "exec",
			argv:    []string{"--pid", env.ImagePath, "true"},
			exit:    0,
		},
		{
			name:    "ExecFalse",
			command: "exec",
			argv:    []string{env.ImagePath, "false"},
			exit:    1,
		},
		{
			name:    "ExecPidNsFalse",
			command: "exec",
			argv:    []string{"--pid", env.ImagePath, "false"},
			exit:    1,
		},
		{
			name:    "RunTrue",
			command: "run",
			argv:    []string{env.ImagePath, "true"},
			exit:    0,
		},
		{
			name:    "RunPidNsTrue",
			command: "run",
			argv:    []string{"--pid", env.ImagePath, "true"},
			exit:    0,
		},
		{
			name:    "RunFalse",
			command: "run",
			argv:    []string{env.ImagePath, "false"},
			exit:    1,
		},
		{
			name:    "RunPidNsFalse",
			command: "run",
			argv:    []string{"--pid", env.ImagePath, "false"},
			exit:    1,
		},
		{
			name:    "RunBindTrue",
			command: "run",
			argv:    []string{"--bind", "/etc/passwd", env.ImagePath, "true"},
			exit:    0,
		},
		{
			name:    "RunBindFalse",
			command: "run",
			argv:    []string{"--bind", "/etc/passwd", env.ImagePath, "false"},
			exit:    1,
		},
	}

	for _, profile := range e2e.Profiles {
		profile := profile

		t.Run(profile.String(), func(t *testing.T) {
			for _, tt := range tests {
				env.RunSingularity(
					t,
					e2e.AsSubtest(tt.name),
					e2e.WithProfile(profile),
					e2e.WithCommand(tt.command),
					e2e.WithArgs(tt.argv...),
					e2e.ExpectExit(tt.exit),
				)
			}
		})
	}
}

func (c actionTests) actionNetwork(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	e2e.Privileged(require.Network)(t)

	tests := []struct {
		name       string
		profile    e2e.Profile
		netType    string
		expectExit int
	}{
		{
			name:       "BridgeNetwork",
			profile:    e2e.RootProfile,
			netType:    "bridge",
			expectExit: 0,
		},
		{
			name:       "PtpNetwork",
			profile:    e2e.RootProfile,
			netType:    "ptp",
			expectExit: 0,
		},
		{
			name:       "UnknownNetwork",
			profile:    e2e.RootProfile,
			netType:    "unknown",
			expectExit: 255,
		},
		{
			name:       "FakerootNetwork",
			profile:    e2e.FakerootProfile,
			netType:    "fakeroot",
			expectExit: 0,
		},
		{
			name:       "NoneNetwork",
			profile:    e2e.UserProfile,
			netType:    "none",
			expectExit: 0,
		},
		{
			name:       "UserBridgeNetwork",
			profile:    e2e.UserProfile,
			netType:    "bridge",
			expectExit: 255,
		},
	}

	for _, tt := range tests {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(tt.profile),
			e2e.WithCommand("exec"),
			e2e.WithArgs("--net", "--network", tt.netType, c.env.ImagePath, "id"),
			e2e.ExpectExit(tt.expectExit),
		)
	}
}

func (c actionTests) actionBinds(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	workspace, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "bind-workspace-", "")
	sandbox, _ := e2e.MakeTempDir(t, workspace, "sandbox-", "")
	defer e2e.Privileged(cleanup)

	contCanaryDir := "/canary"
	hostCanaryDir := filepath.Join(workspace, "canary")

	contCanaryFile := "/canary/file"
	hostCanaryFile := filepath.Join(hostCanaryDir, "file")

	canaryFileBind := hostCanaryFile + ":" + contCanaryFile
	canaryDirBind := hostCanaryDir + ":" + contCanaryDir

	hostHomeDir := filepath.Join(workspace, "home")
	hostWorkDir := filepath.Join(workspace, "workdir")

	createWorkspaceDirs := func(t *testing.T) {
		e2e.Privileged(func(t *testing.T) {
			if err := os.RemoveAll(hostCanaryDir); err != nil && !os.IsNotExist(err) {
				t.Fatalf("failed to delete canary_dir: %s", err)
			}
			if err := os.RemoveAll(hostHomeDir); err != nil && !os.IsNotExist(err) {
				t.Fatalf("failed to delete workspace home: %s", err)
			}
			if err := os.RemoveAll(hostWorkDir); err != nil && !os.IsNotExist(err) {
				t.Fatalf("failed to delete workspace work: %s", err)
			}
		})(t)

		if err := fs.Mkdir(hostCanaryDir, 0777); err != nil {
			t.Fatalf("failed to create canary_dir: %s", err)
		}
		if err := fs.Touch(hostCanaryFile); err != nil {
			t.Fatalf("failed to create canary_file: %s", err)
		}
		if err := os.Chmod(hostCanaryFile, 0777); err != nil {
			t.Fatalf("failed to apply permissions on canary_file: %s", err)
		}
		if err := fs.Mkdir(hostHomeDir, 0777); err != nil {
			t.Fatalf("failed to create workspace home directory: %s", err)
		}
		if err := fs.Mkdir(hostWorkDir, 0777); err != nil {
			t.Fatalf("failed to create workspace work directory: %s", err)
		}
	}

	// convert test image to sandbox
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs("--force", "--sandbox", sandbox, c.env.ImagePath),
		e2e.ExpectExit(0),
	)

	checkHostFn := func(path string, fn func(string) bool) func(*testing.T) {
		return func(t *testing.T) {
			if t.Failed() {
				return
			}
			if !fn(path) {
				t.Errorf("%s not found on host", path)
			}
			if err := os.RemoveAll(path); err != nil {
				t.Errorf("failed to delete %s: %s", path, err)
			}
		}
	}
	checkHostFile := func(path string) func(*testing.T) {
		return checkHostFn(path, fs.IsFile)
	}
	checkHostDir := func(path string) func(*testing.T) {
		return checkHostFn(path, fs.IsDir)
	}

	tests := []struct {
		name    string
		args    []string
		postRun func(*testing.T)
		exit    int
	}{
		{
			name: "NonExistentSource",
			args: []string{
				"--bind", "/non/existent/source/path",
				sandbox,
				"true",
			},
			exit: 255,
		},
		{
			name: "RelativeBindDestination",
			args: []string{
				"--bind", hostCanaryFile + ":relative",
				sandbox,
				"true",
			},
			exit: 255,
		},
		{
			name: "SimpleFile",
			args: []string{
				"--bind", canaryFileBind,
				sandbox,
				"test", "-f", contCanaryFile,
			},
			exit: 0,
		},
		{
			name: "SimpleFilePwd",
			args: []string{
				"--bind", canaryFileBind,
				"--pwd", contCanaryDir,
				sandbox,
				"test", "-f", "file",
			},
			exit: 0,
		},
		{
			name: "SimpleDir",
			args: []string{
				"--bind", canaryDirBind,
				sandbox,
				"test", "-f", contCanaryFile,
			},
			exit: 0,
		},
		{
			name: "SimpleDirPwd",
			args: []string{
				"--bind", canaryDirBind,
				"--pwd", contCanaryDir,
				sandbox,
				"test", "-f", "file",
			},
			exit: 0,
		},
		{
			name: "SimpleFileWritableOK",
			args: []string{
				"--writable",
				"--bind", hostCanaryFile,
				sandbox,
				"test", "-f", hostCanaryFile,
			},
			exit: 0,
		},
		{
			name: "SimpleFileWritableKO",
			args: []string{
				"--writable",
				"--bind", canaryFileBind,
				sandbox,
				"test", "-f", contCanaryFile,
			},
			exit: 255,
		},
		{
			name: "SimpleDirWritableOK",
			args: []string{
				"--writable",
				"--bind", hostCanaryDir,
				sandbox,
				"test", "-f", hostCanaryFile,
			},
			exit: 0,
		},
		{
			name: "SimpleDirWritableKO",
			args: []string{
				"--writable",
				"--bind", canaryDirBind,
				sandbox,
				"test", "-f", contCanaryFile,
			},
			exit: 255,
		},
		{
			name: "HomeContainOverride",
			args: []string{
				"--contain",
				"--bind", hostCanaryDir + ":/home",
				sandbox,
				"test", "-f", "/home/file",
			},
			exit: 0,
		},
		{
			name: "TmpContainOverride",
			args: []string{
				"--contain",
				"--bind", hostCanaryDir + ":/tmp",
				sandbox,
				"test", "-f", "/tmp/file",
			},
			exit: 0,
		},
		{
			name: "VarTmpContainOverride",
			args: []string{
				"--contain",
				"--bind", hostCanaryDir + ":/var/tmp",
				sandbox,
				"test", "-f", "/var/tmp/file",
			},
			exit: 0,
		},
		{
			name: "SymlinkOneLevelFileBind",
			args: []string{
				"--bind", hostCanaryFile + ":/var/etc/symlink1",
				sandbox,
				"test", "-f", "/etc/symlink1",
			},
			exit: 0,
		},
		{
			name: "SymlinkTwoLevelFileBind",
			args: []string{
				"--bind", hostCanaryFile + ":/var/etc/madness/symlink2",
				sandbox,
				"test", "-f", "/madness/symlink2",
			},
			exit: 0,
		},
		{
			name: "SymlinkOneLevelDirBind",
			args: []string{
				"--bind", hostCanaryDir + ":/var/etc",
				sandbox,
				"test", "-f", "/etc/file",
			},
			exit: 0,
		},
		{
			name: "SymlinkTwoLevelDirBind",
			args: []string{
				"--bind", hostCanaryDir + ":/var/etc/madness",
				sandbox,
				"test", "-f", "/madness/file",
			},
			exit: 0,
		},
		{
			name: "SymlinkOneLevelNewDirBind",
			args: []string{
				"--bind", hostCanaryDir + ":/var/etc/new",
				sandbox,
				"test", "-f", "/etc/new/file",
			},
			exit: 0,
		},
		{
			name: "SymlinkTwoLevelNewDirBind",
			args: []string{
				"--bind", hostCanaryDir + ":/var/etc/madness/new",
				sandbox,
				"test", "-f", "/madness/new/file",
			},
			exit: 0,
		},
		{
			name: "NestedBindFile",
			args: []string{
				"--bind", canaryDirBind,
				"--bind", hostCanaryFile + ":" + filepath.Join(contCanaryDir, "file2"),
				sandbox,
				"test", "-f", "/canary/file2",
			},
			postRun: checkHostFile(filepath.Join(hostCanaryDir, "file2")),
			exit:    0,
		},
		{
			name: "NestedBindDir",
			args: []string{
				"--bind", canaryDirBind,
				"--bind", hostCanaryDir + ":" + filepath.Join(contCanaryDir, "dir2"),
				sandbox,
				"test", "-d", "/canary/dir2",
			},
			postRun: checkHostDir(filepath.Join(hostCanaryDir, "dir2")),
			exit:    0,
		},
		{
			name: "MultipleNestedBindDir",
			args: []string{
				"--bind", canaryDirBind,
				"--bind", hostCanaryDir + ":" + filepath.Join(contCanaryDir, "dir2"),
				"--bind", hostCanaryFile + ":" + filepath.Join(filepath.Join(contCanaryDir, "dir2"), "nested"),
				sandbox,
				"test", "-f", "/canary/dir2/nested",
			},
			postRun: checkHostFile(filepath.Join(hostCanaryDir, "nested")),
			exit:    0,
		},
		{
			name: "CustomHomeOneToOne",
			args: []string{
				"--home", hostHomeDir,
				"--bind", hostCanaryDir + ":" + filepath.Join(hostHomeDir, "canary121RO"),
				sandbox,
				"test", "-f", filepath.Join(hostHomeDir, "canary121RO/file"),
			},
			postRun: checkHostDir(filepath.Join(hostHomeDir, "canary121RO")),
			exit:    0,
		},
		{
			name: "CustomHomeBind",
			args: []string{
				"--home", hostHomeDir + ":/home/e2e",
				"--bind", hostCanaryDir + ":/home/e2e/canaryRO",
				sandbox,
				"test", "-f", "/home/e2e/canaryRO/file",
			},
			postRun: checkHostDir(filepath.Join(hostHomeDir, "canaryRO")),
			exit:    0,
		},
		{
			name: "CustomHomeBindWritableOK",
			args: []string{
				"--home", hostHomeDir + ":/home/e2e",
				"--bind", hostCanaryDir + ":/home/e2e/canaryRW",
				"--writable",
				sandbox,
				"test", "-f", "/home/e2e/canaryRW/file",
			},
			postRun: checkHostDir(filepath.Join(hostHomeDir, "canaryRW")),
			exit:    0,
		},
		{
			name: "CustomHomeBindWritableKO",
			args: []string{
				"--home", canaryDirBind,
				"--writable",
				sandbox,
				"true",
			},
			exit: 255,
		},
		{
			name: "WorkdirTmpBind",
			args: []string{
				"--workdir", hostWorkDir,
				"--contain",
				"--bind", hostCanaryDir + ":/tmp/canary/dir",
				sandbox,
				"test", "-f", "/tmp/canary/dir/file",
			},
			postRun: checkHostDir(filepath.Join(hostWorkDir, "tmp", "canary/dir")),
			exit:    0,
		},
		{
			name: "WorkdirTmpBindWritable",
			args: []string{
				"--writable",
				"--workdir", hostWorkDir,
				"--contain",
				"--bind", hostCanaryDir + ":/tmp/canary/dir",
				sandbox,
				"test", "-f", "/tmp/canary/dir/file",
			},
			postRun: checkHostDir(filepath.Join(hostWorkDir, "tmp", "canary/dir")),
			exit:    0,
		},
		{
			name: "WorkdirVarTmpBind",
			args: []string{
				"--workdir", hostWorkDir,
				"--contain",
				"--bind", hostCanaryDir + ":/var/tmp/canary/dir",
				sandbox,
				"test", "-f", "/var/tmp/canary/dir/file",
			},
			postRun: checkHostDir(filepath.Join(hostWorkDir, "var_tmp", "canary/dir")),
			exit:    0,
		},
		{
			name: "WorkdirVarTmpBindWritable",
			args: []string{
				"--writable",
				"--workdir", hostWorkDir,
				"--contain",
				"--bind", hostCanaryDir + ":/var/tmp/canary/dir",
				sandbox,
				"test", "-f", "/var/tmp/canary/dir/file",
			},
			postRun: checkHostDir(filepath.Join(hostWorkDir, "var_tmp", "canary/dir")),
			exit:    0,
		},
		{
			name: "ScratchTmpfsBind",
			args: []string{
				"--scratch", "/scratch",
				"--bind", hostCanaryDir + ":/scratch/dir",
				sandbox,
				"test", "-f", "/scratch/dir/file",
			},
			exit: 0,
		},
		{
			name: "ScratchWorkdirBind",
			args: []string{
				"--workdir", hostWorkDir,
				"--scratch", "/scratch",
				"--bind", hostCanaryDir + ":/scratch/dir",
				sandbox,
				"test", "-f", "/scratch/dir/file",
			},
			postRun: checkHostDir(filepath.Join(hostWorkDir, "scratch/scratch", "dir")),
			exit:    0,
		},
	}

	for _, profile := range e2e.Profiles {
		profile := profile
		createWorkspaceDirs(t)

		t.Run(profile.String(), func(t *testing.T) {
			for _, tt := range tests {
				c.env.RunSingularity(
					t,
					e2e.AsSubtest(tt.name),
					e2e.WithProfile(profile),
					e2e.WithCommand("exec"),
					e2e.WithArgs(tt.args...),
					e2e.PostRun(tt.postRun),
					e2e.ExpectExit(tt.exit),
				)
			}
		})
	}
}

func (c actionTests) exitSignals(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	tests := []struct {
		name string
		args []string
		exit int
	}{
		{
			name: "Exit0",
			args: []string{c.env.ImagePath, "/bin/sh", "-c", "exit 0"},
			exit: 0,
		},
		{
			name: "Exit1",
			args: []string{c.env.ImagePath, "/bin/sh", "-c", "exit 1"},
			exit: 1,
		},
		{
			name: "Exit134",
			args: []string{c.env.ImagePath, "/bin/sh", "-c", "exit 134"},
			exit: 134,
		},
		{
			name: "SignalKill",
			args: []string{c.env.ImagePath, "/bin/sh", "-c", "kill -KILL $$"},
			exit: 137,
		},
		{
			name: "SignalAbort",
			args: []string{c.env.ImagePath, "/bin/sh", "-c", "kill -ABRT $$"},
			exit: 134,
		},
	}

	for _, tt := range tests {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("exec"),
			e2e.WithArgs(tt.args...),
			e2e.ExpectExit(tt.exit),
		)
	}
}

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) func(*testing.T) {
	c := actionTests{
		env: env,
	}

	return testhelper.TestRunner(map[string]func(*testing.T){
		"action URI":            c.RunFromURI,          // action_URI
		"exec":                  c.actionExec,          // singularity exec
		"persistent overlay":    c.PersistentOverlay,   // Persistent Overlay
		"run":                   c.actionRun,           // singularity run
		"shell":                 c.actionShell,         // shell interaction
		"STDPIPE":               c.STDPipe,             // stdin/stdout pipe
		"action basic profiles": c.actionBasicProfiles, // run basic action under different profiles
		"issue 4488":            c.issue4488,           // https://github.com/sylabs/singularity/issues/4488
		"issue 4587":            c.issue4587,           // https://github.com/sylabs/singularity/issues/4587
		"issue 4755":            c.issue4755,           // https://github.com/sylabs/singularity/issues/4755
		"issue 4768":            c.issue4768,           // https://github.com/sylabs/singularity/issues/4768
		"issue 4797":            c.issue4797,           // https://github.com/sylabs/singularity/issues/4797
		"issue 4823":            c.issue4823,           // https://github.com/sylabs/singularity/issues/4823
		"issue 4836":            c.issue4836,           // https://github.com/sylabs/singularity/issues/4836
		"network":               c.actionNetwork,       // test basic networking
		"binds":                 c.actionBinds,         // test various binds
		"exit and signals":      c.exitSignals,         // test exit and signals propagation
	})
}
