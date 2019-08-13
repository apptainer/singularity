// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cmdenvvars

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/sylabs/scs-library-client/client"
	"github.com/sylabs/singularity/e2e/internal/e2e"
)

type ctx struct {
	env e2e.TestEnv
}

func (c *ctx) testSingularityCacheDir(t *testing.T) {
	// The intent of the test is simple:
	// - create 2 temporary directories, one where the image will be pulled and one where the
	//   image cache should be created,
	// - pull an image,
	// - check whether we have the correct entry in the cache, within the directory we created.
	// If the file is in our cache, it means the e2e framework correctly set the SINGULARITY_CACHE_DIR
	// while executing the pull command.
	cacheDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("failed to create temporary directory: %s", err)
	}
	defer func() {
		err := os.RemoveAll(cacheDir)
		if err != nil {
			t.Fatalf("failed to delete temporary directory %s: %s", cacheDir, err)
		}
	}()

	c.env.TestDir, err = ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("failed to create temporary directory: %s", err)
	}
	defer func() {
		err := os.RemoveAll(c.env.TestDir)
		if err != nil {
			t.Fatalf("failed to delete temporary directory %s: %s", c.env.TestDir, err)
		}
	}()

	c.env.ImgCacheDir = cacheDir
	imgName := "testImg.sif"
	imgPath := filepath.Join(c.env.TestDir, imgName)
	cmdArgs := []string{imgPath, "library://alpine:latest"}

	// Build the image. We make sure to use RunSingularity since the goal here is to check
	// whether it does the correct thing or not.
	c.env.RunSingularity(
		t,
		e2e.WithPrivileges(false),
		e2e.WithCommand("pull"),
		e2e.WithArgs(cmdArgs...),
		e2e.ExpectExit(0),
	)

	shasum, err := client.ImageHash(imgPath)
	if err != nil {
		t.Fatalf("failed to get sha256sum for %s", imgPath)
	}
	cacheEntryPath := filepath.Join(cacheDir, "cache", "library", shasum, "alpine_latest.sif")
	if _, err := os.Stat(cacheEntryPath); os.IsNotExist(err) {
		t.Fatalf("cache entry is missing (expected: %s)", cacheEntryPath)
	}
}

func (c *ctx) testSingularitySypgpDir(t *testing.T) {
	// Create a temporary directory to be used for a keyring
	keyringDir, err := ioutil.TempDir("", "e2e-sypgp-env-")
	if err != nil {
		t.Fatalf("failed to create temporary directory: %s", err)
	}
	defer func() {
		err := os.RemoveAll(keyringDir)
		if err != nil {
			t.Fatalf("failed to delete temporary directory %s: %s", keyringDir, err)
		}
	}()

	// Run 'key list' to initialize the keyring.
	cmdArgs := []string{"list"}
	c.env.KeyringDir = keyringDir
	c.env.RunSingularity(
		t,
		e2e.WithPrivileges(false),
		e2e.WithCommand("key"),
		e2e.WithArgs(cmdArgs...),
		e2e.ExpectExit(0),
	)

	pubKeyringPath := filepath.Join(keyringDir, "pgp-public")
	if _, err := os.Stat(pubKeyringPath); os.IsNotExist(err) {
		t.Fatalf("failed to find keyring (expected: %s)", pubKeyringPath)
	}

	privKeyringPath := filepath.Join(keyringDir, "pgp-secret")
	if _, err := os.Stat(privKeyringPath); os.IsNotExist(err) {
		t.Fatalf("failed to find keyring (expected: %s)", privKeyringPath)
	}

}

// RunE2ETests is the bootstrap to run all instance tests.
func RunE2ETests(env e2e.TestEnv) func(*testing.T) {
	c := &ctx{
		env: env,
	}

	return func(t *testing.T) {
		t.Run("testSingularityCacheDir", c.testSingularityCacheDir)
		t.Run("testSingularitySypgpDir", c.testSingularitySypgpDir)
	}
}
