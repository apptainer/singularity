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
	"github.com/sylabs/singularity/internal/pkg/test/tool/exec"
	"github.com/sylabs/singularity/internal/pkg/test/tool/require"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

func actionBasic(ctx *e2e.TestContext) {
	t, env, profile := ctx.Get()

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
	}

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
}

func actionRun(ctx *e2e.TestContext) {
	t, env, profile := ctx.Get()

	e2e.EnsureImage(t, env)

	tests := []struct {
		name string
		argv []string
		exit int
	}{
		{
			name: "NoCommand",
			argv: []string{env.ImagePath},
			exit: 0,
		},
		{
			name: "True",
			argv: []string{env.ImagePath, "true"},
			exit: 0,
		},
		{
			name: "False",
			argv: []string{env.ImagePath, "false"},
			exit: 1,
		},
		{
			name: "ScifTestAppGood",
			argv: []string{"--app", "testapp", env.ImagePath},
			exit: 0,
		},
		{
			name: "ScifTestAppBad",
			argv: []string{"--app", "fakeapp", env.ImagePath},
			exit: 1,
		},
	}

	for _, tt := range tests {
		env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(profile),
			e2e.WithCommand("run"),
			e2e.WithArgs(tt.argv...),
			e2e.ExpectExit(tt.exit),
		)
	}
}

func actionExec(ctx *e2e.TestContext) {
	t, env, profile := ctx.Get()

	e2e.EnsureImage(t, env)

	u := profile.User(t)

	testdata, err := fs.MakeTmpDir(env.TestDir, "testdata", 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testdata) // clean up

	testdataTmp := filepath.Join(testdata, "tmp")
	if err := os.Mkdir(testdataTmp, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a temp testfile
	tmpfile, err := fs.MakeTmpFile(testdataTmp, "testSingularityExec.", 0644)
	if err != nil {
		t.Fatal(err)
	}

	tmpfilePath := filepath.Join("/tmp", filepath.Base(tmpfile.Name()))
	vartmpfilePath := filepath.Join("/var/tmp", filepath.Base(tmpfile.Name()))
	homePath := filepath.Join("/home", filepath.Base(tmpfile.Name()))

	tests := []struct {
		name             string
		argv             []string
		exit             int
		excludedProfiles []e2e.SingularityProfile
	}{
		{
			name: "NoCommand",
			argv: []string{env.ImagePath},
			exit: 1,
		},
		{
			name: "True",
			argv: []string{env.ImagePath, "true"},
			exit: 0,
		},
		{
			name: "TrueAbsPAth",
			argv: []string{env.ImagePath, "/bin/true"},
			exit: 0,
		},
		{
			name: "False",
			argv: []string{env.ImagePath, "false"},
			exit: 1,
		},
		{
			name: "FalseAbsPath",
			argv: []string{env.ImagePath, "/bin/false"},
			exit: 1,
		},
		// Scif apps tests
		{
			name: "ScifTestAppGood",
			argv: []string{"--app", "testapp", env.ImagePath, "testapp.sh"},
			exit: 0,
		},
		{
			name: "ScifTestAppBad",
			argv: []string{"--app", "fakeapp", env.ImagePath, "testapp.sh"},
			exit: 1,
		},
		{
			name: "ScifTestfolderOrg",
			argv: []string{env.ImagePath, "test", "-d", "/scif"},
			exit: 0,
		},
		{
			name: "ScifTestfolderOrg",
			argv: []string{env.ImagePath, "test", "-d", "/scif/apps"},
			exit: 0,
		},
		{
			name: "ScifTestfolderOrg",
			argv: []string{env.ImagePath, "test", "-d", "/scif/data"},
			exit: 0,
		},
		{
			name: "ScifTestfolderOrg",
			argv: []string{env.ImagePath, "test", "-d", "/scif/apps/foo"},
			exit: 0,
		},
		{
			name: "ScifTestfolderOrg",
			argv: []string{env.ImagePath, "test", "-d", "/scif/apps/bar"},
			exit: 0,
		},
		// blocked by issue [scif-apps] Files created at install step fall into an unexpected path #2404
		{
			name: "ScifTestfolderOrg",
			argv: []string{env.ImagePath, "test", "-f", "/scif/apps/foo/filefoo.exec"},
			exit: 0,
		},
		{
			name: "ScifTestfolderOrg",
			argv: []string{env.ImagePath, "test", "-f", "/scif/apps/bar/filebar.exec"},
			exit: 0,
		},
		{
			name: "ScifTestfolderOrg",
			argv: []string{env.ImagePath, "test", "-d", "/scif/data/foo/output"},
			exit: 0,
		},
		{
			name: "ScifTestfolderOrg",
			argv: []string{env.ImagePath, "test", "-d", "/scif/data/foo/input"},
			exit: 0,
		},
		{
			name: "ContainOnly",
			argv: []string{"--contain", env.ImagePath, "test", "-f", tmpfilePath},
			exit: 1,
		},
		{
			name: "WorkdirOnly",
			argv: []string{"--workdir", testdata, env.ImagePath, "test", "-f", tmpfilePath},
			exit: 1,
		},
		{
			name: "WorkdirContain",
			argv: []string{"--workdir", testdata, "--contain", env.ImagePath, "test", "-f", tmpfilePath},
			exit: 0,
		},
		{
			name: "PwdGood",
			argv: []string{"--pwd", "/etc", env.ImagePath, "true"},
			exit: 0,
		},
		{
			name: "Home",
			argv: []string{"--home", testdata, env.ImagePath, "test", "-f", tmpfile.Name()},
			exit: 0,
		},
		{
			name: "HomePath",
			argv: []string{"--home", testdataTmp + ":/home", env.ImagePath, "test", "-f", homePath},
			exit: 0,
		},
		{
			name: "HomeTmp",
			argv: []string{"--home", "/tmp", env.ImagePath, "true"},
			exit: 0,
		},
		{
			name: "HomeTmpExplicit",
			argv: []string{"--home", "/tmp:/home", env.ImagePath, "true"},
			exit: 0,
		},
		{
			name: "UserBindVarTmp",
			argv: []string{"--bind", testdataTmp + ":/var/tmp", env.ImagePath, "test", "-f", vartmpfilePath},
			exit: 0,
		},
		{
			name: "UserBindTmp",
			argv: []string{"--bind", testdataTmp + ":/tmp", env.ImagePath, "test", "-f", tmpfilePath},
			exit: 0,
		},
		{
			// as /root directory is always present in images, this test
			// could not run with RootProfile, RootUserNamespaceProfile and
			// FakerootProfile
			name: "NoHome",
			argv: []string{"--no-home", env.ImagePath, "ls", "-ld", u.Dir},
			exit: 2,
			excludedProfiles: []e2e.SingularityProfile{
				e2e.RootProfile,
				e2e.RootUserNamespaceProfile,
				e2e.FakerootProfile,
			},
		},
	}

	for _, tt := range tests {
		env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(profile),
			e2e.WithDir("/"),
			e2e.WithCommand("exec"),
			e2e.WithArgs(tt.argv...),
			e2e.PreRun(func(t *testing.T) {
				if tt.excludedProfiles != nil && profile.In(tt.excludedProfiles...) {
					t.Skipf("test skipped for %q", profile)
				}
			}),
			e2e.ExpectExit(tt.exit),
		)
	}
}

func actionShell(ctx *e2e.TestContext) {
	t, env, profile := ctx.Get()

	e2e.EnsureImage(t, env)

	u := profile.User(t)

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
			argv: []string{env.ImagePath},
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
			argv: []string{env.ImagePath},
			consoleOps: []e2e.SingularityConsoleOp{
				e2e.ConsoleSendLine("hostname"),
				e2e.ConsoleExpect(hostname),
				e2e.ConsoleSendLine("exit"),
			},
			exit: 0,
		},
		{
			name: "ShellHome",
			argv: []string{env.ImagePath},
			consoleOps: []e2e.SingularityConsoleOp{
				e2e.ConsoleSendLine("cd && pwd"),
				e2e.ConsoleExpect(u.Dir),
				e2e.ConsoleSendLine("exit"),
			},
		},
		{
			name: "ShellHomeContain",
			argv: []string{"--contain", env.ImagePath},
			consoleOps: []e2e.SingularityConsoleOp{
				e2e.ConsoleSendLine("cd && pwd"),
				e2e.ConsoleExpect(u.Dir),
				e2e.ConsoleSendLine("exit"),
			},
		},
		{
			name: "ShellBadCommand",
			argv: []string{env.ImagePath},
			consoleOps: []e2e.SingularityConsoleOp{
				e2e.ConsoleSendLine("_a_fake_command"),
				e2e.ConsoleSendLine("exit"),
			},
			exit: 127,
		},
	}

	for _, tt := range tests {
		env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(profile),
			e2e.WithCommand("shell"),
			e2e.WithArgs(tt.argv...),
			e2e.ConsoleRun(tt.consoleOps...),
			e2e.ExpectExit(tt.exit),
		)
	}
}

func actionStdPipe(ctx *e2e.TestContext) {
	t, env, profile := ctx.Get()

	e2e.EnsureImage(t, env)

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
			argv:    []string{env.ImagePath, "grep", "hi"},
			input:   "hi",
			exit:    0,
		},
		{
			name:    "FalseSTDIN",
			command: "exec",
			argv:    []string{env.ImagePath, "grep", "hi"},
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
		env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(profile),
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

	u := profile.User(t)

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
			argv:    []string{"--app", "foo", env.ImagePath},
			output:  "RUNNING FOO",
			exit:    0,
		},
		{
			name:    "PwdPath",
			command: "exec",
			argv:    []string{"--pwd", "/etc", env.ImagePath, "pwd"},
			output:  "/etc",
			exit:    0,
		},
		{
			name:    "Arguments",
			command: "run",
			argv:    []string{env.ImagePath, "foo"},
			output:  "Running command: foo",
			exit:    127,
		},
		{
			name:    "Permissions",
			command: "exec",
			argv:    []string{env.ImagePath, "id", "-un"},
			output:  u.Name,
			exit:    0,
		},
	}
	for _, tt := range stdoutTests {
		env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(profile),
			e2e.WithCommand(tt.command),
			e2e.WithArgs(tt.argv...),
			e2e.ExpectExit(
				tt.exit,
				e2e.ExpectOutput(e2e.ExactMatch, tt.output),
			),
		)
	}
}

func actionRunFromURI(ctx *e2e.TestContext) {
	t, env, profile := ctx.Get()

	e2e.PrepRegistry(t, env)

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
			argv:    []string{"--bind", bind, env.OrasTestImage, size},
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
			argv:    []string{"--bind", bind, env.OrasTestImage, "0"},
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
			argv:    []string{env.OrasTestImage, "true"},
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
			argv:    []string{env.OrasTestImage, "false"},
			exit:    1,
		},
	}

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
}

// PersistentOverlay test the --overlay function
func actionPersistentOverlay(ctx *e2e.TestContext) {
	t, env, profile := ctx.Get()

	if !profile.In(e2e.RootProfile) {
		t.Skipf("could not run those tests with %q", profile)
	}

	require.Command(t, "mkfs.ext3")
	require.Command(t, "dd")
	require.Command(t, "mksquashfs")

	e2e.EnsureImage(t, env)

	imgDir, err := ioutil.TempDir(env.TestDir, "img_overlay")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(imgDir)

	squashfsImage := filepath.Join(imgDir, "squashfs.simg")
	ext3Image := filepath.Join(imgDir, "ext3_fs.img")

	dir, err := ioutil.TempDir(env.TestDir, "overlay_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Create dirfs for squashfs
	squashDir, err := ioutil.TempDir(env.TestDir, "overlay_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(squashDir)

	content := []byte("temporary file's content")
	tmpfile, err := e2e.WriteTempFile(squashDir, "bogus", string(content))
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("mksquashfs", squashDir, squashfsImage, "-noappend", "-all-root")
	if res := cmd.Run(t); res.Error != nil {
		t.Fatalf("Unexpected error while running command.\n%s", res)
	}

	//  Create the overlay ext3 fs
	cmd = exec.Command("dd", "if=/dev/zero", "of="+ext3Image, "bs=1M", "count=768", "status=none")
	if res := cmd.Run(t); res.Error != nil {
		t.Fatalf("Unexpected error while running command.\n%s", res)
	}

	cmd = exec.Command("mkfs.ext3", "-q", "-F", ext3Image)
	if res := cmd.Run(t); res.Error != nil {
		t.Fatalf("Unexpected error while running command.\n%s", res)
	}

	tests := []struct {
		name    string
		argv    []string
		exit    int
		profile e2e.SingularityProfile
	}{
		{
			name:    "overlay_create",
			argv:    []string{"--overlay", dir, env.ImagePath, "touch", "/dir_overlay"},
			exit:    0,
			profile: e2e.RootProfile,
		},
		{
			name:    "overlay_find",
			argv:    []string{"--overlay", dir, env.ImagePath, "test", "-f", "/dir_overlay"},
			exit:    0,
			profile: e2e.RootProfile,
		},
		{
			name:    "overlay_ext3_create",
			argv:    []string{"--overlay", ext3Image, env.ImagePath, "touch", "/ext3_overlay"},
			exit:    0,
			profile: e2e.RootProfile,
		},
		{
			name:    "overlay_ext3_find",
			argv:    []string{"--overlay", ext3Image, env.ImagePath, "test", "-f", "/ext3_overlay"},
			exit:    0,
			profile: e2e.RootProfile,
		},
		{
			name:    "overlay_squashFS_find",
			argv:    []string{"--overlay", squashfsImage, env.ImagePath, "test", "-f", fmt.Sprintf("/%s", filepath.Base(tmpfile))},
			exit:    0,
			profile: e2e.RootProfile,
		},
		{
			name:    "overlay_multiple_create",
			argv:    []string{"--overlay", ext3Image, "--overlay", squashfsImage, env.ImagePath, "touch", "/multiple_overlay_fs"},
			exit:    0,
			profile: e2e.RootProfile,
		},
		{
			name:    "overlay_multiple_find_ext3",
			argv:    []string{"--overlay", ext3Image, "--overlay", squashfsImage, env.ImagePath, "test", "-f", "/multiple_overlay_fs"},
			exit:    0,
			profile: e2e.RootProfile,
		},
		{
			name:    "overlay_multiple_find_squashfs",
			argv:    []string{"--overlay", ext3Image, "--overlay", squashfsImage, env.ImagePath, "test", "-f", fmt.Sprintf("/%s", filepath.Base(tmpfile))},
			exit:    0,
			profile: e2e.RootProfile,
		},
		{
			name:    "overlay_noroot",
			argv:    []string{"--overlay", dir, env.ImagePath, "test", "-f", "/foo_overlay"},
			exit:    255,
			profile: e2e.UserProfile,
		},
		{
			name:    "overlay_noflag",
			argv:    []string{env.ImagePath, "test", "-f", "/foo_overlay"},
			exit:    1,
			profile: e2e.RootProfile,
		},
	}

	for _, tt := range tests {
		env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(tt.profile),
			e2e.WithCommand("exec"),
			e2e.WithArgs(tt.argv...),
			e2e.ExpectExit(tt.exit),
		)
	}
}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(env e2e.TestEnv) func(*testing.T) {
	return func(t *testing.T) {
		tests := map[string]func(*e2e.TestContext){
			"Basic":             actionBasic,             // basic tests
			"Run":               actionRun,               // singularity run
			"Exec":              actionExec,              // singularity exec
			"StdPipe":           actionStdPipe,           // stdin/stdout pipe
			"RunFromURI":        actionRunFromURI,        // run/exec from URI
			"Shell":             actionShell,             // shell interaction
			"PersistentOverlay": actionPersistentOverlay, // exec with overlay
		}

		for _, profile := range e2e.Profiles {
			t.Run(profile.Name(), func(t *testing.T) {
				profile.Require(t)

				for name, fn := range tests {
					t.Run(name, func(t *testing.T) {
						ctx := e2e.NewTestContext(t, env, profile)
						fn(ctx)
					})
				}
			})
		}
	}
}
