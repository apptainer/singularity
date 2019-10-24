// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sign

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
)

type ctx struct {
	env             e2e.TestEnv
	imgCache        string
	keyringDir      string
	passphraseInput []e2e.SingularityConsoleOp
}

const imgURL = "library://sylabs/tests/unsigned:1.0.0"
const imgName = "testImage.sif"

func (c ctx) singularitySignHelpOption(t *testing.T) {
	c.env.KeyringDir = c.keyringDir
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("sign"),
		e2e.WithArgs("--help"),
		e2e.ExpectExit(
			0,
			e2e.ExpectOutput(e2e.ContainMatch, "Attach a cryptographic signature to an image"),
		),
	)
}

func (c *ctx) prepareImage(t *testing.T) (string, func(*testing.T)) {
	// Get a refresh unsigned image
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("failed to create temporary directory: %s", err)
	}
	imgPath := filepath.Join(tempDir, imgName)
	e2e.PullImage(t, c.env, imgURL, imgPath)

	return filepath.Join(tempDir, "testImage.sif"), func(t *testing.T) {
		err := os.RemoveAll(tempDir)
		if err != nil {
			t.Fatalf("failed to delete temporary directory: %s", err)
		}
	}
}

func (c ctx) singularitySignIDOption(t *testing.T) {
	imgPath, cleanup := c.prepareImage(t)
	defer cleanup(t)

	tests := []struct {
		name       string
		args       []string
		expectOp   e2e.SingularityCmdResultOp
		expectExit int
	}{
		{
			name:       "sign deffile",
			args:       []string{"--sif-id", "1", imgPath},
			expectOp:   e2e.ExpectOutput(e2e.ContainMatch, "Signature created and applied to "+imgPath),
			expectExit: 0,
		},
		{
			name:       "sign non-exsistent ID",
			args:       []string{"--sif-id", "5", imgPath},
			expectOp:   e2e.ExpectError(e2e.ContainMatch, "no descriptor found for id 5"),
			expectExit: 255,
		},
	}

	c.env.KeyringDir = c.keyringDir
	c.env.ImgCacheDir = c.imgCache

	for _, tt := range tests {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("sign"),
			e2e.WithArgs(tt.args...),
			e2e.ConsoleRun(c.passphraseInput...),
			e2e.ExpectExit(tt.expectExit, tt.expectOp),
		)
	}
}

func (c ctx) singularitySignAllOption(t *testing.T) {
	imgPath, cleanup := c.prepareImage(t)
	defer cleanup(t)

	tests := []struct {
		name       string
		args       []string
		expectOp   e2e.SingularityCmdResultOp
		expectExit int
	}{
		{
			name:       "sign default",
			args:       []string{imgPath},
			expectOp:   e2e.ExpectOutput(e2e.ContainMatch, "Signature created and applied to "+imgPath),
			expectExit: 0,
		},
		{
			name:       "sign all",
			args:       []string{"--all", imgPath},
			expectOp:   e2e.ExpectOutput(e2e.ContainMatch, "Signature created and applied to "+imgPath),
			expectExit: 0,
		},
	}

	c.env.KeyringDir = c.keyringDir
	c.env.ImgCacheDir = c.imgCache

	for _, tt := range tests {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("sign"),
			e2e.WithArgs(tt.args...),
			e2e.ConsoleRun(c.passphraseInput...),
			e2e.ExpectExit(tt.expectExit, tt.expectOp),
		)
	}
}

func (c ctx) singularitySignGroupIDOption(t *testing.T) {
	imgPath, cleanup := c.prepareImage(t)
	defer cleanup(t)

	tests := []struct {
		name       string
		args       []string
		expectOp   e2e.SingularityCmdResultOp
		expectExit int
	}{
		{
			name:       "groupID 0",
			args:       []string{"--groupid", "1", imgPath},
			expectOp:   e2e.ExpectOutput(e2e.ContainMatch, "Signature created and applied to "+imgPath),
			expectExit: 0,
		},
		{
			name:       "groupID 5",
			args:       []string{"--groupid", "5", imgPath},
			expectOp:   e2e.ExpectOutput(e2e.ContainMatch, "no descriptors found for groupid 5"),
			expectExit: 255,
		},
	}

	c.env.KeyringDir = c.keyringDir
	c.env.ImgCacheDir = c.imgCache

	for _, tt := range tests {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("sign"),
			e2e.WithArgs(tt.args...),
			e2e.ConsoleRun(c.passphraseInput...),
			e2e.ExpectExit(tt.expectExit, tt.expectOp),
		)
	}
}

func (c ctx) singularitySignKeyidxOption(t *testing.T) {
	imgPath, cleanup := c.prepareImage(t)
	defer cleanup(t)

	cmdArgs := []string{"--keyidx", "0", imgPath}
	c.env.KeyringDir = c.keyringDir
	c.env.ImgCacheDir = c.imgCache
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("sign"),
		e2e.WithArgs(cmdArgs...),
		e2e.ConsoleRun(c.passphraseInput...),
		e2e.ExpectExit(
			0,
			e2e.ExpectOutput(e2e.ContainMatch, "Signature created and applied to "+imgPath),
		),
	)
}

func (c *ctx) generateKeypair(t *testing.T) {
	keyGenInput := []e2e.SingularityConsoleOp{
		e2e.ConsoleSendLine("e2e sign test key"),
		e2e.ConsoleSendLine("jdoe@sylabs.io"),
		e2e.ConsoleSendLine("sign e2e test"),
		e2e.ConsoleSendLine("passphrase"),
		e2e.ConsoleSendLine("passphrase"),
		e2e.ConsoleSendLine("n"),
	}

	c.env.KeyringDir = c.keyringDir
	c.env.RunSingularity(
		t,
		e2e.ConsoleRun(keyGenInput...),
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("key"),
		e2e.WithArgs("newpair"),
		e2e.ExpectExit(0),
	)
}

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) func(*testing.T) {
	c := ctx{
		env: env,
	}

	return func(t *testing.T) {
		var err error
		// To speed up the tests, we use a common image cache (we pull the same image several times)
		c.imgCache, err = ioutil.TempDir("", "e2e-sign-imgcache-")
		if err != nil {
			t.Fatalf("failed to create temporary directory: %s", err)
		}
		defer func() {
			err := os.RemoveAll(c.imgCache)
			if err != nil {
				t.Fatalf("failed to delete temporary cache: %s", err)
			}
		}()

		// We need one single key pair in a single keyring for all the tests
		c.keyringDir, err = ioutil.TempDir("", "e2e-sign-keyring-")
		if err != nil {
			t.Fatalf("failed to create temporary directory: %s", err)
		}
		defer func() {
			err := os.RemoveAll(c.keyringDir)
			if err != nil {
				t.Fatalf("failed to delete temporary directory: %s", err)
			}
		}()
		c.generateKeypair(t)

		c.passphraseInput = []e2e.SingularityConsoleOp{
			e2e.ConsoleSendLine("passphrase"),
		}
		t.Run("singularitySignAllOption", c.singularitySignAllOption)
		t.Run("singularitySignHelpOption", c.singularitySignHelpOption)
		t.Run("singularitySignIDOption", c.singularitySignIDOption)
		t.Run("singularitySignGroupIDOption", c.singularitySignGroupIDOption)
		t.Run("singularitySignKeyidxOption", c.singularitySignKeyidxOption)
	}
}
