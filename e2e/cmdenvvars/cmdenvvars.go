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

func setupTempDirs(t *testing.T, readOnly bool) (string, string, func(t *testing.T)) {
	cacheDir, err := ioutil.TempDir("", "e2e-imgcache-")
	if err != nil {
		t.Fatalf("failed to create temporary directory: %s", err)
	}

	testDir, err := ioutil.TempDir("", "")
	if err != nil {
		os.RemoveAll(cacheDir) // Something went wrong before we can setup a cleanup function so we do our best to manually cleanup
		t.Fatalf("failed to create temporary directory: %s", err)
	}

	return testDir, cacheDir, func(t *testing.T) {
		err := os.RemoveAll(cacheDir)
		if err != nil {
			t.Fatalf("failed to delete temporary directory %s: %s", cacheDir, err)
		}

		err = os.RemoveAll(testDir)
		if err != nil {
			t.Fatalf("failed to delete temporary directory %s: %s", testDir, err)
		}
	}
}

func (c *ctx) testSingularityImgCache(t *testing.T, disableCache bool) string {
	if disableCache {
		c.env.DisableCache = true
	}

	imgName := "testImg.sif"
	imgPath := filepath.Join(c.env.TestDir, imgName)
	cmdArgs := []string{imgPath, "library://alpine:latest"}

	// Build the image. We make sure to use RunSingularity since the goal here is to check
	// whether it does the correct thing or not.
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("pull"),
		e2e.WithArgs(cmdArgs...),
		e2e.ExpectExit(0),
	)

	return imgPath
}

// cacheExists checks that the image cache that is associated to the test exists
// and is valid (i.e., include the correct entry)
func (c *ctx) cacheIsNotExist(t *testing.T, imgPath string) {
	cacheRoot := filepath.Join(c.env.ImgCacheDir, "cache")
	if _, err := os.Stat(cacheRoot); !os.IsNotExist(err) {
		// The root of the cache does exists
		t.Fatalf("cache has been incorrectly created (cache root: %s)", cacheRoot)
	}
}

func (c *ctx) testSingularityCacheDir(t *testing.T) {
	// The intent of the test is simple:
	// - create 2 temporary directories, one where the image will be pulled and one where the
	//   image cache should be created,
	// - pull an image,
	// - check whether we have the correct entry in the cache, within the directory we created.
	// If the file is in our cache, it means the e2e framework correctly set the SINGULARITY_CACHE_DIR
	// while executing the pull command.

	testDir, cacheDir, cleanup := setupTempDirs(t, false)
	c.env.TestDir = testDir
	defer cleanup(t)

	c.env.ImgCacheDir = cacheDir
	imgPath := c.testSingularityImgCache(t, false)

	// The cache should exist and have the correct entry
	shasum, err := client.ImageHash(imgPath)
	if err != nil {
		t.Fatalf("Cannot get the shasum for image %s: %s", imgPath, err)
	}
	cacheEntryPath := filepath.Join(c.env.ImgCacheDir, "cache", "library", shasum, "alpine_latest.sif")
	if _, err := os.Stat(cacheEntryPath); os.IsNotExist(err) {
		t.Fatalf("Cache entry %s does not exists: %s", cacheEntryPath, err)
	}
}

func (c *ctx) testSingularityDisableCache(t *testing.T) {
	testDir, cacheDir, cleanup := setupTempDirs(t, false)
	c.env.TestDir = testDir
	defer cleanup(t)

	c.env.ImgCacheDir = cacheDir
	imgPath := c.testSingularityImgCache(t, true)

	// the cache should not exist
	c.cacheIsNotExist(t, imgPath)
}

// This test checks if the cache is correctly and implicitly disabled
// when its target location is read-only.
//
// This use case is common in the context of Grid computing where the
// usage of sandboxes shared between users is a common practice. In that
// context, the home directory ends up being read-only and no caching
// is required.
func (c *ctx) testSingularityReadOnlyCacheDir(t *testing.T) {
	testDir, cacheDir, cleanup := setupTempDirs(t, true)
	c.env.TestDir = testDir
	defer cleanup(t)

	// Change the mode of the image cache to read-only
	err := os.Chmod(cacheDir, 0444)
	if err != nil {
		t.Fatalf("failed to change the access mode to read-only: %s", err)
	}

	c.env.ImgCacheDir = cacheDir
	imgPath := c.testSingularityImgCache(t, false)

	// Change the mode of the image cache back so we can actually check everything
	err = os.Chmod(cacheDir, 0755)
	if err != nil {
		t.Fatalf("failed to change the access mode to read-only: %s", err)
	}

	// the cache should not exist
	c.cacheIsNotExist(t, imgPath)
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
		e2e.WithProfile(e2e.UserProfile),
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

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) func(*testing.T) {
	c := &ctx{
		env: env,
	}

	return func(t *testing.T) {
		t.Run("testSingularityCacheDir", c.testSingularityCacheDir)
		t.Run("testSingularityDisableDir", c.testSingularityDisableCache)
		t.Run("testSingularitySypgpDir", c.testSingularitySypgpDir)
		t.Run("testReadOnlyCacheDir", c.testSingularityReadOnlyCacheDir)
	}
}
