// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package run

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
)

type ctx struct {
	env e2e.TestEnv
}

// testRun555Cache tests the specific case where the cache directory is
// 0555 for access rights, and we try to run a singularity run command
// using that directory as cache. This reflects a problem that is important
// for the grid use case.
func (c *ctx) testRun555Cache(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "e2e-run-555-")
	if err != nil {
		t.Fatalf("failed to create temporary directory: %s", err)
	}
	defer func() {
		err := os.RemoveAll(tempDir)
		if err != nil {
			t.Fatalf("failed to delete temporary directory %s: %s", tempDir, err)
		}
	}()
	cacheDir := filepath.Join(tempDir, "image-cache")
	err = os.Mkdir(cacheDir, 0555)
	if err != nil {
		t.Fatalf("failed to create a temporary image cache: %s", err)
	}
	// Directory is deleted when tempDir is deleted

	cmdArgs := []string{"library://godlovedc/funny/lolcow"}
	c.env.ImgCacheDir = cacheDir
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("run"),
		e2e.WithArgs(cmdArgs...),
		e2e.ExpectExit(0),
	)
}

func (c *ctx) testRunPEMEncrypted(t *testing.T) {
	// If the version of cryptsetup is not compatible with Singularity encryption,
	// the build commands are expected to fail
	err := e2e.CheckCryptsetupVersion()
	if err != nil {
		t.Skip("cryptsetup is not compatible, skipping test")
	}

	// It is too complicated right now to deal with a PEM file, the Sylabs infrastructure
	// does not let us attach one to a image in the library, so we generate one.
	pemPubFile, pemPrivFile := e2e.GeneratePemFiles(t, c.env.TestDir)

	// We create a temporary directory to store the image, making sure tests
	// will not pollute each other
	tempDir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "", "")
	defer cleanup(t)

	imgPath := filepath.Join(tempDir, "encrypted_cmdline_option.sif")
	cmdArgs := []string{"--encrypt", "--pem-path", pemPubFile, imgPath, "library://alpine:latest"}
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs(cmdArgs...),
		e2e.ExpectExit(0),
	)

	// Using command line
	cmdArgs = []string{"--pem-path", pemPrivFile, imgPath}
	c.env.RunSingularity(
		t,
		e2e.AsSubtest("pem file cmdline"),
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("run"),
		e2e.WithArgs(cmdArgs...),
		e2e.ExpectExit(0),
	)

	// Using environment variable
	cmdArgs = []string{imgPath}
	pemEnvVar := fmt.Sprintf("%s=%s", "SINGULARITY_ENCRYPTION_PEM_PATH", pemPrivFile)
	c.env.RunSingularity(
		t,
		e2e.AsSubtest("pem file cmdline"),
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("run"),
		e2e.WithArgs(cmdArgs...),
		e2e.WithEnv(append(os.Environ(), pemEnvVar)),
		e2e.ExpectExit(0),
	)
}

func (c *ctx) testRunPassphraseEncrypted(t *testing.T) {
	// Expected results for a successful command execution
	expectedExitCode := 0
	expectedStderr := ""

	passphraseEncryptedURL := "library://vallee/default/passphrase_encrypted_alpine"
	passphraseEncryptedFingerprint := "sha256.e784d01b94e4b5a42d9e9b54bc2c0400630604bb896de1e65d8c77e25ca5b5e7"
	passphraseInput := []e2e.SingularityConsoleOp{
		e2e.ConsoleSendLine(e2e.Passphrase),
	}

	// If the version of cryptsetup is not compatible with Singularity encryption,
	// the build commands are expected to fail
	err := e2e.CheckCryptsetupVersion()
	if err != nil {
		expectedExitCode = 255
		// todo: fix the problen with catching stderr, until then we do not do a real check
		//expectedStderr = "FATAL:   While performing build: unable to encrypt filesystem at /tmp/sbuild-718337349/squashfs-770818633: available cryptsetup is not supported"
		expectedStderr = ""
	}

	// Interactive command
	cmdArgs := []string{"--passphrase", passphraseEncryptedURL + ":" + passphraseEncryptedFingerprint}
	c.env.RunSingularity(
		t,
		e2e.AsSubtest("interactive passphrase"),
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("run"),
		e2e.WithArgs(cmdArgs...),
		e2e.ConsoleRun(passphraseInput...),
		e2e.ExpectExit(
			expectedExitCode,
			e2e.ExpectError(e2e.ContainMatch, expectedStderr),
		),
	)

	// Using the environment variable to specify the passphrase
	passphraseEnvVar := fmt.Sprintf("%s=%s", "SINGULARITY_ENCRYPTION_PASSPHRASE", e2e.Passphrase)
	cmdArgs = []string{passphraseEncryptedURL + ":" + passphraseEncryptedFingerprint}
	c.env.RunSingularity(
		t,
		e2e.AsSubtest("env var passphrase"),
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("run"),
		e2e.WithArgs(cmdArgs...),
		e2e.WithEnv(append(os.Environ(), passphraseEnvVar)),
		e2e.ExpectExit(
			expectedExitCode,
			e2e.ExpectError(e2e.ContainMatch, expectedStderr),
		),
	)

	// Specifying the passphrase on the command line should always fail
	cmdArgs = []string{"--passphrase", e2e.Passphrase, passphraseEncryptedURL + ":" + passphraseEncryptedFingerprint}
	c.env.RunSingularity(
		t,
		e2e.AsSubtest("passphrase on cmdline"),
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("run"),
		e2e.WithArgs(cmdArgs...),
		e2e.WithEnv(append(os.Environ(), passphraseEnvVar)),
		e2e.ExpectExit(
			255,
			e2e.ExpectError(e2e.ContainMatch, expectedStderr),
		),
	)
}

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) func(*testing.T) {
	c := &ctx{
		env: env,
	}

	return func(t *testing.T) {
		t.Run("run555cache", c.testRun555Cache)
		t.Run("runPassphraseEncrypted", c.testRunPassphraseEncrypted)
		t.Run("runPEMEncrypted", c.testRunPEMEncrypted)
	}
}
