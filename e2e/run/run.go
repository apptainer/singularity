// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package run

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/e2e/internal/testhelper"
	"github.com/sylabs/singularity/internal/pkg/cache"
)

type ctx struct {
	env e2e.TestEnv
}

// testRun555Cache tests the specific case where the cache directory is
// 0555 for access rights, and we try to run a singularity run command
// using that directory as cache. This reflects a problem that is important
// for the grid use case.
func (c ctx) testRun555Cache(t *testing.T) {
	tempDir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "", "")
	defer cleanup(t)
	cacheDir := filepath.Join(tempDir, "image-cache")
	err := os.Mkdir(cacheDir, 0555)
	if err != nil {
		t.Fatalf("failed to create a temporary image cache: %s", err)
	}
	// Directory is deleted when tempDir is deleted

	cmdArgs := []string{"library://lolcow"}
	// We explicitly pass the environment to the command, not through c.env.ImgCacheDir
	// because c.env is shared between all the tests, something we do not want here.
	cacheDirEnv := fmt.Sprintf("%s=%s", cache.DirEnv, cacheDir)
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("run"),
		e2e.WithArgs(cmdArgs...),
		e2e.WithEnv(append(os.Environ(), cacheDirEnv)),
		e2e.ExpectExit(0),
	)
}

func (c ctx) testRunPEMEncrypted(t *testing.T) {
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

	imgPath := filepath.Join(tempDir, "encrypted_cmdline_pem-path.sif")
	cmdArgs := []string{"--encrypt", "--pem-path", pemPubFile, imgPath, "library://alpine:3.11.5"}
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

func (c ctx) testRunPassphraseEncrypted(t *testing.T) {
	// If the version of cryptsetup is not compatible with Singularity encryption,
	// the build commands are expected to fail
	err := e2e.CheckCryptsetupVersion()
	if err != nil {
		t.Skip("cryptsetup is not compatible, skipping test")
	}

	passphraseEnvVar := fmt.Sprintf("%s=%s", "SINGULARITY_ENCRYPTION_PASSPHRASE", e2e.Passphrase)

	// We create a temporary directory to store the image, making sure tests
	// will not pollute each other
	tempDir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "", "")
	defer cleanup(t)

	imgPath := filepath.Join(tempDir, "encrypted_cmdline_passphrase.sif")
	cmdArgs := []string{"--encrypt", imgPath, "library://alpine:3.11.5"}
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs(cmdArgs...),
		e2e.WithEnv(append(os.Environ(), passphraseEnvVar)),
		e2e.ExpectExit(0),
	)

	passphraseInput := []e2e.SingularityConsoleOp{
		e2e.ConsoleSendLine(e2e.Passphrase),
	}

	// Interactive command
	cmdArgs = []string{"--passphrase", imgPath, "/bin/true"}
	c.env.RunSingularity(
		t,
		e2e.AsSubtest("interactive passphrase"),
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("exec"),
		e2e.WithArgs(cmdArgs...),
		e2e.ConsoleRun(passphraseInput...),
		e2e.ExpectExit(0),
	)

	// Using the environment variable to specify the passphrase
	cmdArgs = []string{imgPath}
	c.env.RunSingularity(
		t,
		e2e.AsSubtest("env var passphrase"),
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("run"),
		e2e.WithArgs(cmdArgs...),
		e2e.WithEnv(append(os.Environ(), passphraseEnvVar)),
		e2e.ExpectExit(0),
	)

	// Ensure decryption works with an IPC namespace
	cmdArgs = []string{"--ipc", imgPath}
	c.env.RunSingularity(
		t,
		e2e.AsSubtest("env var passphrase with ipc namespace"),
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("run"),
		e2e.WithArgs(cmdArgs...),
		e2e.WithEnv(append(os.Environ(), passphraseEnvVar)),
		e2e.ExpectExit(0),
	)

	// Ensure decryption works with containall (IPC and PID namespaces)
	cmdArgs = []string{"--containall", imgPath}
	c.env.RunSingularity(
		t,
		e2e.AsSubtest("env var passphrase with containall"),
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("run"),
		e2e.WithArgs(cmdArgs...),
		e2e.WithEnv(append(os.Environ(), passphraseEnvVar)),
		e2e.ExpectExit(0),
	)

	// Specifying the passphrase on the command line should always fail
	cmdArgs = []string{"--passphrase", e2e.Passphrase, imgPath}
	c.env.RunSingularity(
		t,
		e2e.AsSubtest("passphrase on cmdline"),
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("run"),
		e2e.WithArgs(cmdArgs...),
		e2e.WithEnv(append(os.Environ(), passphraseEnvVar)),
		e2e.ExpectExit(255),
	)
}

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) testhelper.Tests {
	c := ctx{
		env: env,
	}

	return testhelper.Tests{
		"0555 cache":           c.testRun555Cache,
		"passphrase encrypted": c.testRunPassphraseEncrypted,
		"PEM encrypted":        c.testRunPEMEncrypted,
	}
}
