// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package imgbuild

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/util/bin"
	"github.com/sylabs/singularity/pkg/util/crypt"
)

var testFileContent = "Test file content\n"

type imgBuildTests struct {
	env e2e.TestEnv
}

const (
	passphrase = "e2e-passphrase"
)

func (c *imgBuildTests) buildFrom(t *testing.T) {
	e2e.PrepRegistry(t, c.env)

	tests := []struct {
		name       string
		dependency string
		buildSpec  string
		sandbox    bool
	}{
		{"BusyBox", "", "../examples/busybox/Singularity", false},
		{"Debootstrap", "debootstrap", "../examples/debian/Singularity", true},
		{"DockerURI", "", "docker://busybox", true},
		{"DockerDefFile", "", "../examples/docker/Singularity", true},
		// TODO(mem): reenable this; disabled while shub is down
		// {"ShubURI", "", "shub://GodloveD/busybox", true},
		// TODO(mem): reenable this; disabled while shub is down
		// {"ShubDefFile", "", "../examples/shub/Singularity", true},
		{"LibraryDefFile", "", "../examples/library/Singularity", true},
		{"OrasURI", "", c.env.OrasTestImage, true},
		{"Yum", "yum", "../examples/centos/Singularity", true},
		{"Zypper", "zypper", "../examples/opensuse/Singularity", true},
	}

	for _, tt := range tests {
		imagePath := path.Join(c.env.TestDir, "container")

		// conditionally build a sandbox
		args := []string{}
		if tt.sandbox {
			args = []string{"--sandbox"}
		}
		args = append(args, imagePath, tt.buildSpec)

		c.env.RunSingularity(
			t,
			e2e.WithPrivileges(true),
			e2e.WithCommand("build"),
			e2e.WithArgs(args...),
			e2e.PreRun(func(t *testing.T) {
				if tt.dependency != "" {
					if _, err := exec.LookPath(tt.dependency); err != nil {
						t.Skipf("%v not found in path", tt.dependency)
					}
				}
			}),
			e2e.PostRun(func(t *testing.T) {
				defer os.RemoveAll(imagePath)

				c.env.ImageVerify(t, imagePath)
			}),
			e2e.ExpectExit(0),
		)
	}
}

func (c *imgBuildTests) nonRootBuild(t *testing.T) {
	tests := []struct {
		name      string
		buildSpec string
		sandbox   bool
	}{
		{
			name:      "local sif",
			buildSpec: "testdata/busybox.sif",
			sandbox:   false,
		},
		{
			name:      "local sif to sandbox",
			buildSpec: "testdata/busybox.sif",
			sandbox:   true,
		},
		{
			name:      "library sif",
			buildSpec: "library://sylabs/tests/busybox:1.0.0",
			sandbox:   false,
		},
		{
			name:      "library sif sandbox",
			buildSpec: "library://sylabs/tests/busybox:1.0.0",
			sandbox:   true,
		},
		{
			name:      "library sif sha",
			buildSpec: "library://sylabs/tests/busybox:sha256.8b5478b0f2962eba3982be245986eb0ea54f5164d90a65c078af5b83147009ba",
			sandbox:   false,
		},
		// TODO: uncomment when shub is working
		//{
		//		name:      "shub busybox",
		//		buildSpec: "shub://GodloveD/busybox",
		//		sandbox:   false,
		//},
		{
			name:      "docker busybox",
			buildSpec: "docker://busybox:latest",
			sandbox:   false,
		},
	}

	for _, tt := range tests {
		imagePath := path.Join(c.env.TestDir, "container")

		// conditionally build a sandbox
		args := []string{}
		if tt.sandbox {
			args = []string{"--sandbox"}
		}
		args = append(args, imagePath, tt.buildSpec)

		c.env.RunSingularity(
			t,
			e2e.WithPrivileges(false),
			e2e.WithCommand("build"),
			e2e.WithArgs(args...),
			e2e.PostRun(func(t *testing.T) {
				defer os.RemoveAll(imagePath)

				c.env.ImageVerify(t, imagePath)
			}),
			e2e.ExpectExit(0),
		)
	}
}

func (c *imgBuildTests) buildLocalImage(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	tmpdir, err := ioutil.TempDir(c.env.TestDir, "build-local-image.")
	if err != nil {
		t.Errorf("Cannot create temporary directory: %+v", err)
	}

	defer os.RemoveAll(tmpdir)

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
		e2e.WithPrivileges(true),
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

	tests := []struct {
		name      string
		buildSpec string
	}{
		{"SIFToSIF", c.env.ImagePath},
		{"SandboxToSIF", sandboxImage},
		{"LocalImage", liDefFile},
		{"LocalImageLabel", liLabelDefFile},
		{"LocalImageSandbox", localSandboxDefFile},
	}

	for i, tt := range tests {
		imagePath := filepath.Join(tmpdir, fmt.Sprintf("image-%d", i))
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithPrivileges(true),
			e2e.WithCommand("build"),
			e2e.WithArgs(imagePath, tt.buildSpec),
			e2e.PostRun(func(t *testing.T) {
				c.env.ImageVerify(t, imagePath)
			}),
			e2e.ExpectExit(0),
		)
	}
}

func (c *imgBuildTests) badPath(t *testing.T) {
	imagePath := path.Join(c.env.TestDir, "container")
	c.env.RunSingularity(
		t,
		e2e.WithPrivileges(true),
		e2e.WithCommand("build"),
		e2e.WithArgs(imagePath, "/some/dumb/path"),
		e2e.ExpectExit(255),
	)
}

func (c *imgBuildTests) buildMultiStageDefinition(t *testing.T) {
	tmpfile, err := e2e.WriteTempFile(c.env.TestDir, "testFile-", testFileContent)
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpfile) // clean up

	tests := []struct {
		name    string
		force   bool
		sandbox bool
		dfd     []e2e.DefFileDetails
		correct e2e.DefFileDetails // a bit hacky, but this allows us to check final image for correct artifacts
	}{
		// Simple copy from stage one to final stage
		{"FileCopySimple", false, true, []e2e.DefFileDetails{
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
						}}},
			}},
			e2e.DefFileDetails{
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
		{"FileCopyComplex", false, true,
			[]e2e.DefFileDetails{
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
							}},
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
						}},
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
							}}},
				},
			},
			e2e.DefFileDetails{
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
		imagePath := path.Join(c.env.TestDir, "container")
		defFile := e2e.PrepareMultiStageDefFile(tt.dfd)

		args := []string{}
		if tt.force {
			args = append([]string{"--force"}, args...)
		}
		if tt.sandbox {
			args = append([]string{"--sandbox"}, args...)
		}
		args = append(args, imagePath, defFile)

		c.env.RunSingularity(
			t,
			e2e.WithPrivileges(true),
			e2e.WithCommand("build"),
			e2e.WithArgs(args...),
			e2e.PostRun(func(t *testing.T) {
				defer os.Remove(defFile)
				defer os.RemoveAll(imagePath)

				e2e.DefinitionImageVerify(t, c.env.CmdPath, imagePath, tt.correct)
			}),
			e2e.ExpectExit(0),
		)
	}
}

func (c *imgBuildTests) buildDefinition(t *testing.T) {
	tmpfile, err := e2e.WriteTempFile(c.env.TestDir, "testFile-", testFileContent)
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpfile) // clean up

	tests := []struct {
		name    string
		force   bool
		sandbox bool
		dfd     e2e.DefFileDetails
	}{
		{"Empty", false, true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "alpine:latest",
		}},
		{"Help", false, true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Help: []string{
				"help info line 1",
				"help info line 2",
				"help info line 3",
			},
		}},
		{"Files", false, true, e2e.DefFileDetails{
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
		}},
		{"Test", false, true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Test: []string{
				"echo testscript line 1",
				"echo testscript line 2",
				"echo testscript line 3",
			},
		}},
		{"Startscript", false, true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "alpine:latest",
			StartScript: []string{
				"echo startscript line 1",
				"echo startscript line 2",
				"echo startscript line 3",
			},
		}},
		{"Runscript", false, true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "alpine:latest",
			RunScript: []string{
				"echo runscript line 1",
				"echo runscript line 2",
				"echo runscript line 3",
			},
		}},
		{"Env", false, true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Env: []string{
				"testvar1=one",
				"testvar2=two",
				"testvar3=three",
			},
		}},
		{"Labels", false, true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Labels: map[string]string{
				"customLabel1": "one",
				"customLabel2": "two",
				"customLabel3": "three",
			},
		}},
		{"Pre", false, true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Pre: []string{
				filepath.Join(c.env.TestDir, "PreFile1"),
			},
		}},
		{"Setup", false, true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Setup: []string{
				filepath.Join(c.env.TestDir, "SetupFile1"),
			},
		}},
		{"Post", false, true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Post: []string{
				"PostFile1",
			},
		}},
		{"AppHelp", false, true, e2e.DefFileDetails{
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
		}},
		{"AppEnv", false, true, e2e.DefFileDetails{
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
		}},
		{"AppLabels", false, true, e2e.DefFileDetails{
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
		}},
		{"AppFiles", false, true, e2e.DefFileDetails{
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
		}},
		{"AppInstall", false, true, e2e.DefFileDetails{
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
		}},
		{"AppRun", false, true, e2e.DefFileDetails{
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
		}},
		{"AppTest", false, true, e2e.DefFileDetails{
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
		}},
	}

	for _, tt := range tests {
		imagePath := path.Join(c.env.TestDir, "container")
		defFile := e2e.PrepareDefFile(tt.dfd)

		args := []string{}
		if tt.force {
			args = append([]string{"--force"}, args...)
		}
		if tt.sandbox {
			args = append([]string{"--sandbox"}, args...)
		}
		args = append(args, imagePath, defFile)

		c.env.RunSingularity(
			t,
			e2e.WithPrivileges(true),
			e2e.WithCommand("build"),
			e2e.WithArgs(args...),
			e2e.PostRun(func(t *testing.T) {
				defer os.Remove(defFile)
				defer os.RemoveAll(imagePath)

				e2e.DefinitionImageVerify(t, c.env.CmdPath, imagePath, tt.dfd)
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
		e2e.WithCommand("sif"),
		e2e.WithArgs(cmdArgs...),
		e2e.ExpectExit(
			0,
			e2e.ExpectOutput(e2e.ContainMatch, "Fstype:    Encrypted squashfs"),
		),
	)
}

func checkCryptsetupVersion() error {
	cryptsetup, err := bin.Cryptsetup()
	if err != nil {
		return err
	}

	cmd := exec.Command(cryptsetup, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run cryptsetup --version: %s", err)
	}

	if !strings.Contains(string(out), "cryptsetup 2.") {
		return fmt.Errorf("incompatible cryptsetup version")
	}

	return nil
}

func (c *imgBuildTests) generatePemFile(t *testing.T) string {
	// Temporary file to save the PEM file
	tempPemFile, err := ioutil.TempFile(c.env.TestDir, "pem-")
	if err != nil {
		t.Fatalf("failed to create temporary file: %s", err)
	}
	tempPemFile.Close()
	// File is deleted when the e2e framework deletes TestDir

	rsaKey, err := crypt.GenerateRSAKey(2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %s", err)
	}

	err = crypt.SavePublicPEM(tempPemFile.Name(), rsaKey)
	if err != nil {
		t.Fatalf("failed to generate PEM file: %s", err)
	}

	return tempPemFile.Name()
}

func (c *imgBuildTests) buildEncryptPemFile(t *testing.T) {
	// Expected results for a successful command execution
	expectedExitCode := 0
	expectedStderr := ""

	// We create a temporary directory to store the image, making sure tests
	// will not pollute each other
	tempDir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "", "")
	defer cleanup(t)

	// Generate the PEM file
	pemFile := c.generatePemFile(t)

	// If the version of cryptsetup is not compatible with Singularity encryption,
	// the build commands are expected to fail
	err := checkCryptsetupVersion()
	if err != nil {
		expectedExitCode = 255
		// todo: fix the problen with catching stderr, until then we do not do a real check
		//expectedStderr = "FATAL:   While performing build: unable to encrypt filesystem at /tmp/sbuild-718337349/squashfs-770818633: available cryptsetup is not supported"
		expectedStderr = ""
	}

	// First with the command line argument
	imgPath1 := filepath.Join(tempDir, "encrypted_cmdline_option.sif")
	cmdArgs := []string{"--encrypt", "--pem-path", pemFile, imgPath1, "library://alpine:latest"}
	c.env.RunSingularity(
		t,
		e2e.WithCommand("build"),
		e2e.WithPrivileges(true),
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
	passphraseEnvVar := fmt.Sprintf("%s=%s", "SINGULARITY_ENCRYPTION_PEM_PATH", pemFile)
	imgPath2 := filepath.Join(tempDir, "encrypted_env_var.sif")
	cmdArgs = []string{"--encrypt", imgPath2, "library://alpine:latest"}
	c.env.RunSingularity(
		t,
		e2e.WithCommand("build"),
		e2e.WithArgs(cmdArgs...),
		e2e.WithPrivileges(true),
		e2e.WithEnv(append(os.Environ(), passphraseEnvVar)),
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
func (c *imgBuildTests) buildEncryptPassphrase(t *testing.T) {
	// Expected results for a successful command execution
	expectedExitCode := 0
	expectedStderr := ""

	// We create a temporary directory to store the image, making sure tests
	// will not pollute each other
	tempDir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "", "")
	defer cleanup(t)

	// If the version of cryptsetup is not compatible with Singularity encryption,
	// the build commands are expected to fail
	err := checkCryptsetupVersion()
	if err != nil {
		expectedExitCode = 255
		// todo: fix the problen with catching stderr, until then we do not do a real check
		//expectedStderr = "FATAL:   While performing build: unable to encrypt filesystem at /tmp/sbuild-718337349/squashfs-770818633: available cryptsetup is not supported"
		expectedStderr = ""
	}

	// First with the command line argument
	passphraseInput := []e2e.SingularityConsoleOp{
		e2e.ConsoleSendLine(passphrase),
	}
	imgPath1 := filepath.Join(tempDir, "encrypted_cmdline_option.sif")
	cmdArgs := []string{"--encrypt", "--passphrase", imgPath1, "library://alpine:latest"}
	c.env.RunSingularity(
		t,
		e2e.WithCommand("build"),
		e2e.WithPrivileges(true),
		e2e.WithArgs(cmdArgs...),
		e2e.ConsoleRun(passphraseInput...),
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
	passphraseEnvVar := fmt.Sprintf("%s=%s", "SINGULARITY_ENCRYPTION_PASSPHRASE", passphrase)
	imgPath2 := filepath.Join(tempDir, "encrypted_env_var.sif")
	cmdArgs = []string{"--encrypt", imgPath2, "library://alpine:latest"}
	c.env.RunSingularity(
		t,
		e2e.WithCommand("build"),
		e2e.WithArgs(cmdArgs...),
		e2e.WithPrivileges(true),
		e2e.WithEnv(append(os.Environ(), passphraseEnvVar)),
		e2e.ExpectExit(
			expectedExitCode,
			e2e.ExpectError(e2e.ContainMatch, expectedStderr),
		),
	)
	// If the command was supposed to succeed, we check the image
	if expectedExitCode == 0 {
		c.ensureImageIsEncrypted(t, imgPath2)
	}

	// Finally a test that must fail: try to specify the passphrase on the command line
	imgPath3 := filepath.Join(tempDir, "dummy_encrypted_env_var.sif")
	cmdArgs = []string{"--encrypt", "--passphrase", passphrase, imgPath3, "library://alpine:latest"}
	c.env.RunSingularity(
		t,
		e2e.WithCommand("build"),
		e2e.WithArgs(cmdArgs...),
		e2e.WithPrivileges(true),
		e2e.WithEnv(append(os.Environ(), passphraseEnvVar)),
		e2e.ExpectExit(
			1,
			e2e.ExpectError(e2e.RegexMatch, `^Error for command \"build\": accepts 2 arg\(s\), received 3`),
		),
	)

}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(env e2e.TestEnv) func(*testing.T) {
	c := &imgBuildTests{
		env: env,
	}

	return func(t *testing.T) {
		// builds from definition file and URI
		t.Run("From", c.buildFrom)
		// build and image from an existing image
		t.Run("FromLocalImage", c.buildLocalImage)
		// build sifs from non-root
		t.Run("NonRootBuild", c.nonRootBuild)
		// try to build from a non existen path
		t.Run("badPath", c.badPath)
		// builds from definition template
		t.Run("Definition", c.buildDefinition)
		// multistage build from definition templates
		t.Run("MultiStage", c.buildMultiStageDefinition)
		// build encrypted images
		t.Run("buildEncryptPassphrase", c.buildEncryptPassphrase)
		t.Run("buildEncryptPemFile", c.buildEncryptPemFile)
	}
}
