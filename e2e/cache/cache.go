// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/sylabs/scs-library-client/client"
	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
)

type cacheTests struct {
	env e2e.TestEnv
}

const (
	imgName = "alpine_latest.sif"
	imgURL  = "library://alpine:latest"
)

func prepTest(t *testing.T, testEnv e2e.TestEnv, testName string, h *cache.Handle, imagePath string) {
	ensureCacheEmpty(t, testName, imagePath, h)

	testEnv.ImgCacheDir = h.GetBasedir()
	testEnv.RunSingularity(
		t,
		e2e.WithCommand("pull"),
		e2e.WithArgs([]string{"--force", imagePath, imgURL}...),
		e2e.ExpectExit(0),
	)

	ensureCacheNotEmpty(t, testName, imagePath, h)
}

func (c *cacheTests) testNoninteractiveCacheCmds(t *testing.T) {
	tests := []struct {
		name               string
		options            []string
		needImage          bool
		expectedEmptyCache bool
		expectedOutput     string
		exit               int
	}{
		{
			name:               "clean force",
			options:            []string{"clean", "--force"},
			expectedOutput:     "",
			needImage:          true,
			expectedEmptyCache: true,
			exit:               0,
		},
		{
			name:           "clean help",
			options:        []string{"clean", "--help"},
			expectedOutput: "Clean your local Singularity cache",
			needImage:      false,
			exit:           0,
		},
		{
			name:           "list help",
			options:        []string{"list", "--help"},
			expectedOutput: "List your local Singularity cache",
			needImage:      false,
			exit:           0,
		},
		{
			name:               "list type",
			options:            []string{"list", "--type", "library"},
			needImage:          true,
			expectedOutput:     "There are 1 container file",
			expectedEmptyCache: false,
			exit:               0,
		},
		{
			name:               "list verbose",
			needImage:          true,
			options:            []string{"list", "--verbose"},
			expectedOutput:     "NAME",
			expectedEmptyCache: false,
			exit:               0,
		},
	}

	// A directory where we store the image and used by separate commands
	tempDir, imgStoreCleanup := e2e.MakeTempDir(t, "", "", "image store")
	defer imgStoreCleanup(t)
	imagePath := filepath.Join(tempDir, imgName)

	for _, tt := range tests {
		// Each test get its own clean cache directory
		cacheDir, cleanup := e2e.MakeCacheDir(t, "")
		defer cleanup(t)
		h, err := cache.NewHandle(cache.Config{BaseDir: cacheDir})
		if err != nil {
			t.Fatalf("Could not create image cache handle: %v", err)
		}

		if tt.needImage {
			prepTest(t, c.env, tt.name, h, imagePath)
		}

		c.env.ImgCacheDir = cacheDir
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithCommand("cache"),
			e2e.WithArgs(tt.options...),
			e2e.ExpectExit(tt.exit),
		)

		if tt.needImage && tt.expectedEmptyCache {
			ensureCacheEmpty(t, tt.name, imagePath, h)
		}
	}
}

func (c *cacheTests) testInteractiveCacheCmds(t *testing.T) {
	tt := []struct {
		name               string
		options            []string
		expect             string
		send               string
		exit               int
		expectedEmptyCache bool // Is the cache supposed to be empty after the command is executed
	}{
		{
			name:               "clean normal confirmed",
			options:            []string{"clean"},
			expect:             "Do you want to continue? [N/y]",
			send:               "y",
			expectedEmptyCache: true,
			exit:               0,
		},
		{
			name:               "clean normal not confirmed",
			options:            []string{"clean"},
			expect:             "Do you want to continue? [N/y]",
			send:               "n",
			expectedEmptyCache: false,
			exit:               0,
		},
		{
			name:               "clean dry-run confirmed",
			options:            []string{"clean", "--dry-run"},
			expect:             "Do you want to continue? [N/y]",
			send:               "y",
			expectedEmptyCache: false,
			exit:               0,
		},
		{
			name:               "clean dry-run not confirmed",
			options:            []string{"clean", "--dry-run"},
			expect:             "Do you want to continue? [N/y]",
			send:               "n",
			expectedEmptyCache: false,
			exit:               0,
		},
		{
			name:               "clean name confirmed",
			options:            []string{"clean", "--name", imgName},
			expect:             "Do you want to continue? [N/y]",
			send:               "y",
			expectedEmptyCache: true,
			exit:               0,
		},
		{
			name:               "clean name not confirmed",
			options:            []string{"clean", "--name", imgName},
			expect:             "Do you want to continue? [N/y]",
			send:               "n",
			expectedEmptyCache: false,
			exit:               0,
		},
		{
			name:               "clean type confirmed",
			options:            []string{"clean", "--type", "library"},
			expect:             "Do you want to continue? [N/y]",
			send:               "y",
			expectedEmptyCache: true,
			exit:               0,
		},
		{
			name:               "clean type not confirmed",
			options:            []string{"clean", "--type", "library"},
			expect:             "Do you want to continue? [N/y]",
			send:               "n",
			expectedEmptyCache: false,
			exit:               0,
		},
	}

	// A directory where we store the image and used by separate commands
	tempDir, imgStoreCleanup := e2e.MakeTempDir(t, "", "", "image store")
	defer imgStoreCleanup(t)
	imagePath := filepath.Join(tempDir, imgName)

	for _, tc := range tt {
		// Each test get its own clean cache directory
		cacheDir, cleanup := e2e.MakeCacheDir(t, "")
		defer cleanup(t)
		h, err := cache.NewHandle(cache.Config{cacheDir})
		if err != nil {
			t.Fatalf("Could not create image cache handle: %v", err)
		}

		prepTest(t, c.env, tc.name, h, imagePath)

		c.env.ImgCacheDir = cacheDir
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tc.name),
			e2e.WithCommand("cache"),
			e2e.WithArgs(tc.options...),
			e2e.ConsoleRun(
				e2e.ConsoleExpect(tc.expect),
				e2e.ConsoleSendLine(tc.send),
			),
			e2e.ExpectExit(tc.exit),
		)

		// Check the content of the cache
		if tc.expectedEmptyCache {
			ensureCacheEmpty(t, tc.name, imagePath, h)
		} else {
			ensureCacheNotEmpty(t, tc.name, imagePath, h)
		}
	}
}

// ensureDirEmpty checks if a directory is empty. If it is not empty, the test fails;
// if the directory does not exist, we return an error to give us a chance to test
// for the expected parent directory (cache clean commands delete different directories
// based on the user's input).
func ensureDirEmpty(t *testing.T, testName string, dir string) error {
	fi, err := ioutil.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return err
		}
		t.Fatalf("Could not read dir %q: %v", dir, err)
	}

	if len(fi) != 0 {
		t.Fatalf("Dir %q is not empty", dir)
	}

	return nil
}

// ensureCacheEmpty checks if the entry related to an image is in the cache or not.
// Cache commands do not necessarily delete the same files/directories based on the options used.
// The best option is to check whether there is an entry in the cache, i.e.,
// <cache_root>/library/<shasum>/<imagename>
func ensureCacheEmpty(t *testing.T, testName string, imagePath string, h *cache.Handle) {
	shasum, err := client.ImageHash(imagePath)
	if err != nil {
		if os.IsNotExist(err) {
			// We may not have the image yet and in that case, we check if the library cache directory is empty
			err := ensureDirEmpty(t, testName, h.Library)
			if err != nil {
				// The library directory of the cache is not there, checking the root (we want to make sure that the cache is still coherent)
				ensureDirEmpty(t, testName, filepath.Join(h.GetBasedir(), "root"))
			}
		} else {
			t.Fatalf("failed to compute shasum for %s: %s", imgName, err)
		}
	}

	path := h.LibraryImage(shasum, imgName)
	if e2e.PathExists(t, path) {
		t.Fatalf("%s failed: %s is still in the cache (%s)", testName, imgName, path)
	}
}

func ensureCacheNotEmpty(t *testing.T, testName string, imagePath string, h *cache.Handle) {
	// Cache commands do not necessarily delete the same files/directories based on the options used.
	// The best option is to check whether there is an entry in the cache, i.e., <cache_root>/library/<shasum>/imagename
	shasum, err := client.ImageHash(imagePath)
	if err != nil {
		t.Fatalf("failed to compute shasum for %s: %s", imagePath, err)
	}

	exists, err := h.LibraryImageExists(shasum, imgName)
	if err != nil {
		t.Fatalf("failed to check if image exists: %s", err)
	}
	if !exists {
		path := h.LibraryImage(shasum, imgName)
		t.Fatalf("failed to pull image; %s does not exist (image is: %s, cache entry is: %s)", imgName, imagePath, path)
	}
}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(env e2e.TestEnv) func(*testing.T) {
	return func(t *testing.T) {
		c := &cacheTests{
			env: env,
		}
		t.Run("cacheInteractiveCmds", c.testInteractiveCacheCmds)
		t.Run("cacheNoninteractiveCmds", c.testNoninteractiveCacheCmds)
	}
}
