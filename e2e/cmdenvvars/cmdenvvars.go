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
	"github.com/sylabs/singularity/e2e/internal/testhelper"
)

type ctx struct {
	env e2e.TestEnv
}

func setupTemporaryDir(t *testing.T, testdir, label string) (string, func(*testing.T)) {
	tmpdir, err := ioutil.TempDir(testdir, label+".")
	if err != nil {
		t.Fatalf("failed to create '%s' directory for test %s: %s (%#[3]v)",
			label, t.Name(), err)
	}

	t.Logf("Set up temporary %s directory to %s", label, tmpdir)

	return tmpdir, func(t *testing.T) {
		err := os.RemoveAll(tmpdir)
		if err != nil {
			t.Fatalf("failed to delete temporary %s directory %s: %s", label, tmpdir, err)
		}
	}
}

// setupTemporaryCache creates a temporary cache directory and modifies
// the test environment to use it. The code calling this function is
// responsible for calling the returned function when its done using the
// temporary directory.
func (c *ctx) setupTemporaryCache(t *testing.T) func(*testing.T) {
	cacheDir, cleanup := setupTemporaryDir(t, c.env.TestDir, "cache-dir")

	c.env.ImgCacheDir = cacheDir

	return cleanup
}

// setupTemporaryKeyringDir creates a temporary keyring directory and modifies
// the test environment to use it. The code calling this function is
// responsible for calling the returned function when its done using the
// temporary directory.
func (c *ctx) setupTemporaryKeyringDir(t *testing.T) func(*testing.T) {
	keyringDir, cleanup := setupTemporaryDir(t, c.env.TestDir, "sypgp-dir")

	c.env.KeyringDir = keyringDir

	return cleanup
}

// pullTestImage will pull a known image from the network in order to
// exercise the image cache. It returns the full path to the image.
func (c ctx) pullTestImage(t *testing.T) string {
	// create a temporary directory for the destination image
	tmpdir, err := ioutil.TempDir(c.env.TestDir, "image-cache.")
	if err != nil {
		t.Fatalf("failed to create temporary directory for test %s: %s (%#v)", t.Name(), err, err)
	}

	imgPath := filepath.Join(tmpdir, "testImg.sif")

	cmdArgs := []string{imgPath, "library://alpine:latest"}

	// Pull the specified image to the temporary location
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("pull"),
		e2e.WithArgs(cmdArgs...),
		e2e.ExpectExit(0),
	)

	return imgPath
}

func (c ctx) assertLibraryCacheEntryExists(t *testing.T, imgPath, imgName string) {
	// The cache should exist and have the correct entry
	shasum, err := client.ImageHash(imgPath)
	if err != nil {
		t.Fatalf("Cannot get the shasum for image %s: %s", imgPath, err)
	}

	cacheEntryPath := filepath.Join(c.env.ImgCacheDir, "cache", "library", shasum, imgName)
	if _, err := os.Stat(cacheEntryPath); os.IsNotExist(err) {
		ls(t, c.env.TestDir)
		ls(t, c.env.ImgCacheDir)
		t.Fatalf("Cache entry %s for image %s with name %s does not exists: %s",
			cacheEntryPath, imgPath, imgName, err)
	}
}

// assertCacheDoesNotExist checks that the image cache that is associated to the
// test DOES NOT exists.
func (c ctx) assertCacheDoesNotExist(t *testing.T) {
	cacheRoot := filepath.Join(c.env.ImgCacheDir, "cache")
	if _, err := os.Stat(cacheRoot); !os.IsNotExist(err) {
		// The root of the cache does exists
		t.Fatalf("cache has been incorrectly created (cache root: %s)", cacheRoot)
	}
}

func (c ctx) testSingularityCacheDir(t *testing.T) {
	// Test plan:
	//
	// - create a temporary directory for the cache
	// - pull a known image from the network
	// - assert that there's an entry for this image in the cache
	//
	// If the file is in the temporary cache, it means singularity
	// followed the SINGULARITY_CACHEDIR environment variable (set
	// up deep in the e2e framework) to store the cached image.

	cleanup := c.setupTemporaryCache(t)
	defer cleanup(t)

	imgPath := c.pullTestImage(t)

	// there should be an entry for this image in the library cache
	c.assertLibraryCacheEntryExists(t, imgPath, "alpine_latest.sif")
}

func ls(t *testing.T, dir string) {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			t.Logf("W: skipping path %q due to error: %v\n", path, err)
			return err
		}

		t.Logf("%-20d  %s  %s\n", info.Size(), info.Mode(), path)

		return nil
	})

	if err != nil {
		t.Logf("E: error walking the path %q: %v\n", dir, err)
		return
	}
}

func (c ctx) testSingularityDisableCache(t *testing.T) {
	// Test plan:
	//
	// - create a temporary directory for the cache
	// - disable the cache in the test environment
	// - pull a known image from the network
	// - assert that there is no entry for this image in the cache
	//
	// If the file is not in the temporary cache, it means
	// singularity followed the SINGULARITY_DISABLE_CACHE environment
	// variable (set up deep in the e2e framework) and avoided
	// creating an entry in the library cache. If it fails to do so,
	// we expect the entry to be found in the directory specified by
	// SINGULARITY_CACHEDIR (see testSingularityCacheDir).

	cleanup := c.setupTemporaryCache(t)
	defer cleanup(t)

	// disable the cache; it's safe to do this here because we have
	// a value receiver, not a pointer receiver, so this setting
	// won't propagate to the rest of the tests.
	c.env.DisableCache = true

	c.pullTestImage(t)

	// the cache should not exist
	c.assertCacheDoesNotExist(t)
}

func (c ctx) testSingularityReadOnlyCacheDir(t *testing.T) {
	// Test plan:
	//
	// - create a temporary directory for the cache
	// - make the temporary directory readonly (but accessible,
	//   otherwise we are testing something else)
	// - pull a known image from the network
	// - assert that there is no entry for this image in the cache
	//
	// If the file is not in the temporary cache, it means
	// singularity followed the SINGULARITY_DISABLE_CACHE environment
	// variable (set up deep in the e2e framework) and disabled
	// caching (because the directory is readonly). If it fails to
	// do so (e.g. by "fixing" the access permissions on the
	// directory), we expect the entry to be found in the directory
	// specified by SINGULARITY_CACHEDIR (see
	// testSingularityCacheDir).
	//
	// This use case is common in the context of grid computing
	// where the usage of sandboxes shared between users is a common
	// practice. In that context, the home directory ends up being
	// read-only and no caching is required.
	cleanup := c.setupTemporaryCache(t)
	defer cleanup(t)

	// Change the mode of the image cache to read-only
	err := os.Chmod(c.env.ImgCacheDir, 0555)
	if err != nil {
		t.Fatalf("failed to change the access mode to read-only: %s", err)
	}

	c.pullTestImage(t)

	// Change the mode of the image cache to read-write so that we
	// can delete the cache if it was created. Do this _before_
	// calling c.assertCacheDoesNotExist because that function will
	// fail if it find a cache.
	err = os.Chmod(c.env.ImgCacheDir, 0755)
	if err != nil {
		t.Fatalf("failed to change the access mode to read-only: %s", err)
	}

	// the cache should not exist
	c.assertCacheDoesNotExist(t)
}

func (c ctx) testSingularitySypgpDir(t *testing.T) {
	// Test plan:
	//
	// - create a temporary directory for the keyrings
	// - run 'singularity key list' to create the keyrings
	// - assert that both files were created
	//
	// If the files are in the temporary directory, it means
	// singularity followed the SINGULARITY_SYPGPDIR environment
	// variable (set up deep in the e2e framework) to store the
	// keyrings.

	cleanup := c.setupTemporaryKeyringDir(t)
	defer cleanup(t)

	// run 'key list' to initialize the keyring directory.
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("key"),
		e2e.WithArgs("list"),
		e2e.ExpectExit(0),
	)

	pubKeyringPath := filepath.Join(c.env.KeyringDir, "pgp-public")
	if _, err := os.Stat(pubKeyringPath); os.IsNotExist(err) {
		t.Fatalf("failed to find keyring (expected: %s)", pubKeyringPath)
	}

	privKeyringPath := filepath.Join(c.env.KeyringDir, "pgp-secret")
	if _, err := os.Stat(privKeyringPath); os.IsNotExist(err) {
		t.Fatalf("failed to find keyring (expected: %s)", privKeyringPath)
	}
}

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) func(*testing.T) {
	c := ctx{
		env: env,
	}

	return testhelper.TestRunner(map[string]func(*testing.T){
		"read-only cache directory": c.testSingularityReadOnlyCacheDir,
		"SINGULARITY_CACHEDIR":      c.testSingularityCacheDir,
		"singularity disable cache": c.testSingularityDisableCache,
		"SINGULARITY_SYPGPDIR":      c.testSingularitySypgpDir,
	})
}
