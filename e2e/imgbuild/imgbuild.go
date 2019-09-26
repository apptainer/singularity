// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package imgbuild

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/e2e/internal/testhelper"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

var testFileContent = "Test file content\n"

type imgBuildTests struct {
	env e2e.TestEnv
}

func (c imgBuildTests) tempDir(t *testing.T, namespace string) (string, func()) {
	dn, err := fs.MakeTmpDir(c.env.TestDir, namespace+".", 0755)
	if err != nil {
		t.Errorf("failed to create temporary directory: %#v", err)
	}

	cleanup := func() {
		if t.Failed() {
			t.Logf("Test %s failed, not removing %s", t.Name(), dn)

			return
		}

		if err := os.RemoveAll(dn); err != nil {
			t.Logf("Failed to remove %s for test %s: %#v", dn, t.Name(), err)
		}
	}

	return dn, cleanup
}

func (c imgBuildTests) buildFrom(t *testing.T) {
	e2e.PrepRegistry(t, c.env)

	// use a trailing slash in tests for sandbox intentionally to make sure
	// `singularity build -s /tmp/sand/ docker://alpine` works,
	// see https://github.com/sylabs/singularity/issues/4407
	tt := []struct {
		name       string
		dependency string
		buildSpec  string
	}{
		{
			name:      "BusyBox",
			buildSpec: "../examples/busybox/Singularity",
		},
		{
			name:       "Debootstrap",
			dependency: "debootstrap",
			buildSpec:  "../examples/debian/Singularity",
		},
		{
			name:      "DockerURI",
			buildSpec: "docker://busybox",
		},
		{
			name:      "DockerDefFile",
			buildSpec: "../examples/docker/Singularity",
		},
		// TODO(mem): reenable this; disabled while shub is down
		// {
		// 	name:       "ShubURI",
		// 	buildSpec:  "shub://GodloveD/busybox",
		// },
		// TODO(mem): reenable this; disabled while shub is down
		// {
		// 	name:       "ShubDefFile",
		// 	buildSpec:  "../examples/shub/Singularity",
		// },
		{
			name:      "LibraryDefFile",
			buildSpec: "../examples/library/Singularity",
		},
		{
			name:      "OrasURI",
			buildSpec: c.env.OrasTestImage,
		},
		{
			name:       "Yum",
			dependency: "yum",
			buildSpec:  "../examples/centos/Singularity",
		},
		{
			name:       "Zypper",
			dependency: "zypper",
			buildSpec:  "../examples/opensuse/Singularity",
		},
	}

	for _, tc := range tt {
		dn, cleanup := c.tempDir(t, "build-from")
		defer cleanup()

		imagePath := path.Join(dn, "sandbox")

		// Pass --sandbox because sandboxes take less time to
		// build by skipping the SIF creation step.
		args := []string{"--force", "--sandbox", imagePath, tc.buildSpec}

		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tc.name),
			e2e.WithProfile(e2e.RootProfile),
			e2e.WithCommand("build"),
			e2e.WithArgs(args...),
			e2e.PreRun(func(t *testing.T) {
				if tc.dependency == "" {
					return
				}

				if _, err := exec.LookPath(tc.dependency); err != nil {
					t.Skipf("%v not found in path", tc.dependency)
				}
			}),
			e2e.PostRun(func(t *testing.T) {
				if t.Failed() {
					return
				}

				defer os.RemoveAll(imagePath)
				c.env.ImageVerify(t, imagePath)
			}),
			e2e.ExpectExit(0),
		)
	}
}

func (c imgBuildTests) nonRootBuild(t *testing.T) {
	tt := []struct {
		name      string
		buildSpec string
		args      []string
	}{
		{
			name:      "local sif",
			buildSpec: "testdata/busybox.sif",
		},
		{
			name:      "local sif to sandbox",
			buildSpec: "testdata/busybox.sif",
			args:      []string{"--sandbox"},
		},
		{
			name:      "library sif",
			buildSpec: "library://sylabs/tests/busybox:1.0.0",
		},
		{
			name:      "library sif sandbox",
			buildSpec: "library://sylabs/tests/busybox:1.0.0",
			args:      []string{"--sandbox"},
		},
		{
			name:      "library sif sha",
			buildSpec: "library://sylabs/tests/busybox:sha256.8b5478b0f2962eba3982be245986eb0ea54f5164d90a65c078af5b83147009ba",
		},
		// TODO: uncomment when shub is working
		//{
		//		name:      "shub busybox",
		//		buildSpec: "shub://GodloveD/busybox",
		//},
		{
			name:      "docker busybox",
			buildSpec: "docker://busybox:latest",
		},
	}

	for _, tc := range tt {
		dn, cleanup := c.tempDir(t, "non-root-build")
		defer cleanup()

		imagePath := path.Join(dn, "container")

		args := append(tc.args, imagePath, tc.buildSpec)

		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tc.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("build"),
			e2e.WithArgs(args...),
			e2e.PostRun(func(t *testing.T) {
				c.env.ImageVerify(t, imagePath)
			}),
			e2e.ExpectExit(0),
		)
	}
}

func (c imgBuildTests) buildLocalImage(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	tmpdir, cleanup := c.tempDir(t, "build-local-image")

	defer cleanup()

	liDefFile := e2e.PrepareDefFile(e2e.DefFileDetails{
		Bootstrap: "localimage",
		From:      c.env.ImagePath,
	})
	defer os.Remove(liDefFile)

	labels := make(map[string]string)
	labels["FOO"] = "bar"
	liLabelDefFile := e2e.PrepareDefFile(e2e.DefFileDetails{
		Bootstrap: "localimage",
		From:      c.env.ImagePath,
		Labels:    labels,
	})
	defer os.Remove(liLabelDefFile)

	sandboxImage := path.Join(tmpdir, "test-sandbox")

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs("--sandbox", sandboxImage, c.env.ImagePath),
		e2e.PostRun(func(t *testing.T) {
			c.env.ImageVerify(t, sandboxImage)
		}),
		e2e.ExpectExit(0),
	)

	localSandboxDefFile := e2e.PrepareDefFile(e2e.DefFileDetails{
		Bootstrap: "localimage",
		From:      sandboxImage,
		Labels:    labels,
	})
	defer os.Remove(localSandboxDefFile)

	tt := []struct {
		name      string
		buildSpec string
	}{
		{"SIFToSIF", c.env.ImagePath},
		{"SandboxToSIF", sandboxImage},
		{"LocalImage", liDefFile},
		{"LocalImageLabel", liLabelDefFile},
		{"LocalImageSandbox", localSandboxDefFile},
	}

	for i, tc := range tt {
		imagePath := filepath.Join(tmpdir, fmt.Sprintf("image-%d", i))
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tc.name),
			e2e.WithProfile(e2e.RootProfile),
			e2e.WithCommand("build"),
			e2e.WithArgs(imagePath, tc.buildSpec),
			e2e.PostRun(func(t *testing.T) {
				c.env.ImageVerify(t, imagePath)
			}),
			e2e.ExpectExit(0),
		)
	}
}

func (c imgBuildTests) badPath(t *testing.T) {
	dn, cleanup := c.tempDir(t, "bad-path")
	defer cleanup()

	imagePath := path.Join(dn, "container")

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs(imagePath, "/some/dumb/path"),
		e2e.ExpectExit(255),
	)
}

func (c imgBuildTests) buildMultiStageDefinition(t *testing.T) {
	tmpfile, err := e2e.WriteTempFile(c.env.TestDir, "testFile-", testFileContent)
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpfile) // clean up

	tests := []struct {
		name    string
		dfd     []e2e.DefFileDetails
		correct e2e.DefFileDetails // a bit hacky, but this allows us to check final image for correct artifacts
	}{
		// Simple copy from stage one to final stage
		{
			name: "FileCopySimple",
			dfd: []e2e.DefFileDetails{
				{
					Bootstrap: "docker",
					From:      "alpine:latest",
					Stage:     "one",
					Files: []e2e.FilePair{
						{
							Src: tmpfile,
							Dst: "StageOne2.txt",
						},
						{
							Src: tmpfile,
							Dst: "StageOne.txt",
						},
					},
				},
				{
					Bootstrap: "docker",
					From:      "alpine:latest",
					FilesFrom: []e2e.FileSection{
						{
							Stage: "one",
							Files: []e2e.FilePair{
								{
									Src: "StageOne2.txt",
									Dst: "StageOneCopy2.txt",
								},
								{
									Src: "StageOne.txt",
									Dst: "StageOneCopy.txt",
								},
							},
						},
					},
				},
			},
			correct: e2e.DefFileDetails{
				Files: []e2e.FilePair{
					{
						Src: tmpfile,
						Dst: "StageOneCopy2.txt",
					},
					{
						Src: tmpfile,
						Dst: "StageOneCopy.txt",
					},
				},
			},
		},
		// Complex copy of files from stage one and two to stage three, then final copy from three to final stage
		{
			name: "FileCopyComplex",
			dfd: []e2e.DefFileDetails{
				{
					Bootstrap: "docker",
					From:      "alpine:latest",
					Stage:     "one",
					Files: []e2e.FilePair{
						{
							Src: tmpfile,
							Dst: "StageOne2.txt",
						},
						{
							Src: tmpfile,
							Dst: "StageOne.txt",
						},
					},
				},
				{
					Bootstrap: "docker",
					From:      "alpine:latest",
					Stage:     "two",
					Files: []e2e.FilePair{
						{
							Src: tmpfile,
							Dst: "StageTwo2.txt",
						},
						{
							Src: tmpfile,
							Dst: "StageTwo.txt",
						},
					},
				},
				{
					Bootstrap: "docker",
					From:      "alpine:latest",
					Stage:     "three",
					FilesFrom: []e2e.FileSection{
						{
							Stage: "one",
							Files: []e2e.FilePair{
								{
									Src: "StageOne2.txt",
									Dst: "StageOneCopy2.txt",
								},
								{
									Src: "StageOne.txt",
									Dst: "StageOneCopy.txt",
								},
							},
						},
						{
							Stage: "two",
							Files: []e2e.FilePair{
								{
									Src: "StageTwo2.txt",
									Dst: "StageTwoCopy2.txt",
								},
								{
									Src: "StageTwo.txt",
									Dst: "StageTwoCopy.txt",
								},
							},
						},
					},
				},
				{
					Bootstrap: "docker",
					From:      "alpine:latest",
					FilesFrom: []e2e.FileSection{
						{
							Stage: "three",
							Files: []e2e.FilePair{
								{
									Src: "StageOneCopy2.txt",
									Dst: "StageOneCopyFinal2.txt",
								},
								{
									Src: "StageOneCopy.txt",
									Dst: "StageOneCopyFinal.txt",
								},
								{
									Src: "StageTwoCopy2.txt",
									Dst: "StageTwoCopyFinal2.txt",
								},
								{
									Src: "StageTwoCopy.txt",
									Dst: "StageTwoCopyFinal.txt",
								},
							},
						},
					},
				},
			},
			correct: e2e.DefFileDetails{
				Files: []e2e.FilePair{
					{
						Src: tmpfile,
						Dst: "StageOneCopyFinal2.txt",
					},
					{
						Src: tmpfile,
						Dst: "StageOneCopyFinal.txt",
					},
					{
						Src: tmpfile,
						Dst: "StageTwoCopyFinal2.txt",
					},
					{
						Src: tmpfile,
						Dst: "StageTwoCopyFinal.txt",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		dn, cleanup := c.tempDir(t, "multi-stage-definition")
		defer cleanup()

		imagePath := path.Join(dn, "container")

		defFile := e2e.PrepareMultiStageDefFile(tt.dfd)

		// sandboxes take less time to build
		args := []string{"--sandbox", imagePath, defFile}

		c.env.RunSingularity(
			t,
			e2e.WithProfile(e2e.RootProfile),
			e2e.WithCommand("build"),
			e2e.WithArgs(args...),
			e2e.PostRun(func(t *testing.T) {
				defer os.Remove(defFile)

				e2e.DefinitionImageVerify(t, c.env.CmdPath, imagePath, tt.correct)
			}),
			e2e.ExpectExit(0),
		)
	}
}

func (c imgBuildTests) buildDefinition(t *testing.T) {
	tmpfile, err := e2e.WriteTempFile(c.env.TestDir, "testFile-", testFileContent)
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpfile) // clean up

	tt := map[string]e2e.DefFileDetails{
		"Empty": {
			Bootstrap: "docker",
			From:      "alpine:latest",
		},
		"Help": {
			Bootstrap: "docker",
			From:      "alpine:latest",
			Help: []string{
				"help info line 1",
				"help info line 2",
				"help info line 3",
			},
		},
		"Files": {
			Bootstrap: "docker",
			From:      "alpine:latest",
			Files: []e2e.FilePair{
				{
					Src: tmpfile,
					Dst: "NewName2.txt",
				},
				{
					Src: tmpfile,
					Dst: "NewName.txt",
				},
			},
		},
		"Test": {
			Bootstrap: "docker",
			From:      "alpine:latest",
			Test: []string{
				"echo testscript line 1",
				"echo testscript line 2",
				"echo testscript line 3",
			},
		},
		"Startscript": {
			Bootstrap: "docker",
			From:      "alpine:latest",
			StartScript: []string{
				"echo startscript line 1",
				"echo startscript line 2",
				"echo startscript line 3",
			},
		},
		"Runscript": {
			Bootstrap: "docker",
			From:      "alpine:latest",
			RunScript: []string{
				"echo runscript line 1",
				"echo runscript line 2",
				"echo runscript line 3",
			},
		},
		"Env": {
			Bootstrap: "docker",
			From:      "alpine:latest",
			Env: []string{
				"testvar1=one",
				"testvar2=two",
				"testvar3=three",
			},
		},
		"Labels": {
			Bootstrap: "docker",
			From:      "alpine:latest",
			Labels: map[string]string{
				"customLabel1": "one",
				"customLabel2": "two",
				"customLabel3": "three",
			},
		},
		"Pre": {
			Bootstrap: "docker",
			From:      "alpine:latest",
			Pre: []string{
				filepath.Join(c.env.TestDir, "PreFile1"),
			},
		},
		"Setup": {
			Bootstrap: "docker",
			From:      "alpine:latest",
			Setup: []string{
				filepath.Join(c.env.TestDir, "SetupFile1"),
			},
		},
		"Post": {
			Bootstrap: "docker",
			From:      "alpine:latest",
			Post: []string{
				"PostFile1",
			},
		},
		"AppHelp": {
			Bootstrap: "docker",
			From:      "alpine:latest",
			Apps: []e2e.AppDetail{
				{
					Name: "foo",
					Help: []string{
						"foo help info line 1",
						"foo help info line 2",
						"foo help info line 3",
					},
				},
				{
					Name: "bar",
					Help: []string{
						"bar help info line 1",
						"bar help info line 2",
						"bar help info line 3",
					},
				},
			},
		},
		"AppEnv": {
			Bootstrap: "docker",
			From:      "alpine:latest",
			Apps: []e2e.AppDetail{
				{
					Name: "foo",
					Env: []string{
						"testvar1=fooOne",
						"testvar2=fooTwo",
						"testvar3=fooThree",
					},
				},
				{
					Name: "bar",
					Env: []string{
						"testvar1=barOne",
						"testvar2=barTwo",
						"testvar3=barThree",
					},
				},
			},
		},
		"AppLabels": {
			Bootstrap: "docker",
			From:      "alpine:latest",
			Apps: []e2e.AppDetail{
				{
					Name: "foo",
					Labels: map[string]string{
						"customLabel1": "fooOne",
						"customLabel2": "fooTwo",
						"customLabel3": "fooThree",
					},
				},
				{
					Name: "bar",
					Labels: map[string]string{
						"customLabel1": "barOne",
						"customLabel2": "barTwo",
						"customLabel3": "barThree",
					},
				},
			},
		},
		"AppFiles": {
			Bootstrap: "docker",
			From:      "alpine:latest",
			Apps: []e2e.AppDetail{
				{
					Name: "foo",
					Files: []e2e.FilePair{
						{
							Src: tmpfile,
							Dst: "FooFile2.txt",
						},
						{
							Src: tmpfile,
							Dst: "FooFile.txt",
						},
					},
				},
				{
					Name: "bar",
					Files: []e2e.FilePair{
						{
							Src: tmpfile,
							Dst: "BarFile2.txt",
						},
						{
							Src: tmpfile,
							Dst: "BarFile.txt",
						},
					},
				},
			},
		},
		"AppInstall": {
			Bootstrap: "docker",
			From:      "alpine:latest",
			Apps: []e2e.AppDetail{
				{
					Name: "foo",
					Install: []string{
						"FooInstallFile1",
					},
				},
				{
					Name: "bar",
					Install: []string{
						"BarInstallFile1",
					},
				},
			},
		},
		"AppRun": {
			Bootstrap: "docker",
			From:      "alpine:latest",
			Apps: []e2e.AppDetail{
				{
					Name: "foo",
					Run: []string{
						"echo foo runscript line 1",
						"echo foo runscript line 2",
						"echo foo runscript line 3",
					},
				},
				{
					Name: "bar",
					Run: []string{
						"echo bar runscript line 1",
						"echo bar runscript line 2",
						"echo bar runscript line 3",
					},
				},
			},
		},
		"AppTest": {
			Bootstrap: "docker",
			From:      "alpine:latest",
			Apps: []e2e.AppDetail{
				{
					Name: "foo",
					Test: []string{
						"echo foo testscript line 1",
						"echo foo testscript line 2",
						"echo foo testscript line 3",
					},
				},
				{
					Name: "bar",
					Test: []string{
						"echo bar testscript line 1",
						"echo bar testscript line 2",
						"echo bar testscript line 3",
					},
				},
			},
		},
	}

	for name, dfd := range tt {
		dn, cleanup := c.tempDir(t, "build-definition")
		defer cleanup()

		imagePath := path.Join(dn, "container")

		defFile := e2e.PrepareDefFile(dfd)

		c.env.RunSingularity(
			t,
			e2e.AsSubtest(name),
			e2e.WithProfile(e2e.RootProfile),
			e2e.WithCommand("build"),
			e2e.WithArgs("--sandbox", imagePath, defFile),
			e2e.PostRun(func(t *testing.T) {
				defer os.Remove(defFile)

				e2e.DefinitionImageVerify(t, c.env.CmdPath, imagePath, dfd)
			}),
			e2e.ExpectExit(0),
		)
	}
}

func (c *imgBuildTests) ensureImageIsEncrypted(t *testing.T, imgPath string) {
	sifID := "2" // Which SIF descriptor slots contains encryption information
	cmdArgs := []string{"info", sifID, imgPath}
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("sif"),
		e2e.WithArgs(cmdArgs...),
		e2e.ExpectExit(
			0,
			e2e.ExpectOutput(e2e.ContainMatch, "Fstype:    Encrypted squashfs"),
		),
	)
}

func (c imgBuildTests) buildEncryptPemFile(t *testing.T) {
	// Expected results for a successful command execution
	expectedExitCode := 0
	expectedStderr := ""

	// We create a temporary directory to store the image, making sure tests
	// will not pollute each other
	dn, cleanup := c.tempDir(t, "pem-encryption")
	defer cleanup()

	// Generate the PEM file
	pemFile, _ := e2e.GeneratePemFiles(t, c.env.TestDir)

	// If the version of cryptsetup is not compatible with Singularity encryption,
	// the build commands are expected to fail
	err := e2e.CheckCryptsetupVersion()
	if err != nil {
		expectedExitCode = 255
		// todo: fix the problem with catching stderr, until then we do not do a real check
		// expectedStderr = "FATAL:   While performing build: unable to encrypt filesystem at
		// /tmp/sbuild-718337349/squashfs-770818633: available cryptsetup is not supported"
		expectedStderr = ""
	}

	// First with the command line argument
	imgPath1 := filepath.Join(dn, "encrypted_cmdline_option.sif")
	cmdArgs := []string{"--encrypt", "--pem-path", pemFile, imgPath1, "library://alpine:latest"}
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs(cmdArgs...),
		e2e.ExpectExit(
			expectedExitCode,
			e2e.ExpectError(e2e.ContainMatch, expectedStderr),
		),
	)
	// If the command was supposed to succeed, we check the image
	if expectedExitCode == 0 {
		c.ensureImageIsEncrypted(t, imgPath1)
	}

	// Second with the environment variable
	pemEnvVar := fmt.Sprintf("%s=%s", "SINGULARITY_ENCRYPTION_PEM_PATH", pemFile)
	imgPath2 := filepath.Join(dn, "encrypted_env_var.sif")
	cmdArgs = []string{"--encrypt", imgPath2, "library://alpine:latest"}
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs(cmdArgs...),
		e2e.WithEnv(append(os.Environ(), pemEnvVar)),
		e2e.ExpectExit(
			expectedExitCode,
			e2e.ExpectError(e2e.ContainMatch, expectedStderr),
		),
	)
	// If the command was supposed to succeed, we check the image
	if expectedExitCode == 0 {
		c.ensureImageIsEncrypted(t, imgPath2)
	}
}

// buildEncryptPassphrase is exercising the build command for encrypted containers
// while using a passphrase. Note that it covers both the normal case and when the
// version of cryptsetup available is not compliant.
func (c imgBuildTests) buildEncryptPassphrase(t *testing.T) {
	// Expected results for a successful command execution
	expectedExitCode := 0
	expectedStderr := ""

	// We create a temporary directory to store the image, making sure tests
	// will not pollute each other
	dn, cleanup := c.tempDir(t, "passphrase-encryption")
	defer cleanup()

	// If the version of cryptsetup is not compatible with Singularity encryption,
	// the build commands are expected to fail
	err := e2e.CheckCryptsetupVersion()
	if err != nil {
		expectedExitCode = 255
		expectedStderr = ": available cryptsetup is not supported"
	}

	// First with the command line argument, only using --passphrase
	passphraseInput := []e2e.SingularityConsoleOp{
		e2e.ConsoleSendLine(e2e.Passphrase),
	}
	cmdlineTestImgPath := filepath.Join(dn, "encrypted_cmdline_option.sif")
	// The image is deleted during cleanup of the temporary directory
	cmdArgs := []string{"--passphrase", cmdlineTestImgPath, "library://alpine:latest"}
	c.env.RunSingularity(
		t,
		e2e.AsSubtest("passphrase flag"),
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs(cmdArgs...),
		e2e.ConsoleRun(passphraseInput...),
		e2e.ExpectExit(
			expectedExitCode,
			e2e.ExpectError(e2e.ContainMatch, expectedStderr),
		),
	)
	// If the command was supposed to succeed, we check the image
	if expectedExitCode == 0 {
		c.ensureImageIsEncrypted(t, cmdlineTestImgPath)
	}

	// With the command line argument, using --encrypt and --passphrase
	cmdlineTest2ImgPath := filepath.Join(dn, "encrypted_cmdline2_option.sif")
	cmdArgs = []string{"--encrypt", "--passphrase", cmdlineTest2ImgPath, "library://alpine:latest"}
	c.env.RunSingularity(
		t,
		e2e.AsSubtest("encrypt and passphrase flags"),
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs(cmdArgs...),
		e2e.ConsoleRun(passphraseInput...),
		e2e.ExpectExit(
			expectedExitCode,
			e2e.ExpectError(e2e.ContainMatch, expectedStderr),
		),
	)
	// If the command was supposed to succeed, we check the image
	if expectedExitCode == 0 {
		c.ensureImageIsEncrypted(t, cmdlineTest2ImgPath)
	}

	// With the environment variable
	passphraseEnvVar := fmt.Sprintf("%s=%s", "SINGULARITY_ENCRYPTION_PASSPHRASE", e2e.Passphrase)
	envvarImgPath := filepath.Join(dn, "encrypted_env_var.sif")
	cmdArgs = []string{"--encrypt", envvarImgPath, "library://alpine:latest"}
	c.env.RunSingularity(
		t,
		e2e.AsSubtest("passphrase env var"),
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs(cmdArgs...),
		e2e.WithEnv(append(os.Environ(), passphraseEnvVar)),
		e2e.ExpectExit(
			expectedExitCode,
			e2e.ExpectError(e2e.ContainMatch, expectedStderr),
		),
	)
	// If the command was supposed to succeed, we check the image
	if expectedExitCode == 0 {
		c.ensureImageIsEncrypted(t, envvarImgPath)
	}

	// Finally a test that must fail: try to specify the passphrase on the command line
	dummyImgPath := filepath.Join(dn, "dummy_encrypted_env_var.sif")
	cmdArgs = []string{"--encrypt", "--passphrase", e2e.Passphrase, dummyImgPath, "library://alpine:latest"}
	c.env.RunSingularity(
		t,
		e2e.AsSubtest("passphrase on cmdline"),
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs(cmdArgs...),
		e2e.WithEnv(append(os.Environ(), passphraseEnvVar)),
		e2e.ExpectExit(
			1,
			e2e.ExpectError(e2e.RegexMatch, `^Error for command \"build\": accepts 2 arg\(s\), received 3`),
		),
	)
}

func (c imgBuildTests) buildUpdateSandbox(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	const badSandbox = "/bad/sandbox/path"

	testDir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "build-sandbox-", "")
	defer e2e.Privileged(cleanup)

	tests := []struct {
		name     string
		args     []string
		exitCode int
	}{
		{
			name:     "Sandbox",
			args:     []string{"--force", "--sandbox", testDir, c.env.ImagePath},
			exitCode: 0,
		},
		{
			name:     "UpdateWithoutSandboxFlag",
			args:     []string{"--update", testDir, c.env.ImagePath},
			exitCode: 255,
		},
		{
			name:     "UpdateWithBadSandboxpPath",
			args:     []string{"--update", "--sandbox", badSandbox, c.env.ImagePath},
			exitCode: 255,
		},
		{
			name:     "UpdateWithFileAsSandbox",
			args:     []string{"--update", "--sandbox", c.env.ImagePath, c.env.ImagePath},
			exitCode: 255,
		},
		{
			name:     "UpdateSandbox",
			args:     []string{"--update", "--sandbox", testDir, c.env.ImagePath},
			exitCode: 0,
		},
	}

	for _, tt := range tests {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.RootProfile),
			e2e.WithCommand("build"),
			e2e.WithArgs(tt.args...),
			e2e.ExpectExit(tt.exitCode),
		)
	}
}

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) func(*testing.T) {
	c := imgBuildTests{
		env: env,
	}

	return testhelper.TestRunner(map[string]func(*testing.T){
		"bad path":                        c.badPath, // try to build from a non existen path
		"build encrypt with PEM file":     c.buildEncryptPemFile,
		"build encrypted with passphrase": c.buildEncryptPassphrase,    // build encrypted images
		"definition":                      c.buildDefinition,           // builds from definition template
		"from local image":                c.buildLocalImage,           // build and image from an existing image
		"from":                            c.buildFrom,                 // builds from definition file and URI
		"multistage":                      c.buildMultiStageDefinition, // multistage build from definition templates
		"non-root build":                  c.nonRootBuild,              // build sifs from non-root
		"build and update sandbox":        c.buildUpdateSandbox,        // build/update sandbox
	})
}
