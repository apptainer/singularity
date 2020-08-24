// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"path"
	"path/filepath"
	"testing"

	"github.com/sylabs/scs-library-client/client"
	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/e2e/internal/testhelper"
	"github.com/sylabs/singularity/internal/pkg/cache"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

type cacheTests struct {
	env e2e.TestEnv
}

const (
	imgName = "alpine_latest.sif"
	imgURL  = "library://alpine:latest"
)

func prepTest(t *testing.T, testEnv e2e.TestEnv, testName string, cacheParentDir string, imagePath string) {
	// If the test imageFile is already present check it's not also in the cache
	// at the start of our test - we expect to pull it again and then see it
	// appear in the cache.
	if fs.IsFile(imagePath) {
		ensureNotCached(t, testName, imagePath, cacheParentDir)
	}

	testEnv.ImgCacheDir = cacheParentDir
	testEnv.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("pull"),
		e2e.WithArgs([]string{"--force", imagePath, imgURL}...),
		e2e.ExpectExit(0),
	)

	ensureCached(t, testName, imagePath, cacheParentDir)
}

func (c cacheTests) testNoninteractiveCacheCmds(t *testing.T) {
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
			name:               "clean force days beyond age",
			options:            []string{"clean", "--force", "--days", "30"},
			expectedOutput:     "",
			needImage:          true,
			expectedEmptyCache: false,
			exit:               0,
		},
		{
			name:               "clean force days within age",
			options:            []string{"clean", "--force", "--days", "0"},
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
		_, err := cache.New(cache.Config{ParentDir: cacheDir})
		if err != nil {
			t.Fatalf("Could not create image cache handle: %v", err)
		}

		if tt.needImage {
			prepTest(t, c.env, tt.name, cacheDir, imagePath)
		}

		c.env.ImgCacheDir = cacheDir
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("cache"),
			e2e.WithArgs(tt.options...),
			e2e.ExpectExit(tt.exit),
		)

		if tt.needImage && tt.expectedEmptyCache {
			ensureNotCached(t, tt.name, imagePath, cacheDir)
		}
	}
}

func (c cacheTests) testInteractiveCacheCmds(t *testing.T) {
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
			name:               "clean normal force",
			options:            []string{"clean", "--force"},
			expectedEmptyCache: true,
			exit:               0,
		},
		{
			name:               "clean dry-run confirmed",
			options:            []string{"clean", "--dry-run"},
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
		{
			name:               "clean days beyond age",
			options:            []string{"clean", "--days", "30"},
			expect:             "Do you want to continue? [N/y]",
			send:               "y",
			expectedEmptyCache: false,
			exit:               0,
		},
		{
			name:               "clean days within age",
			options:            []string{"clean", "--days", "0"},
			expect:             "Do you want to continue? [N/y]",
			send:               "y",
			expectedEmptyCache: true,
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
		_, err := cache.New(cache.Config{ParentDir: cacheDir})
		if err != nil {
			t.Fatalf("Could not create image cache handle: %v", err)
		}

		c.env.ImgCacheDir = cacheDir
		prepTest(t, c.env, tc.name, cacheDir, imagePath)

		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tc.name),
			e2e.WithProfile(e2e.UserProfile),
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
			ensureNotCached(t, tc.name, imagePath, cacheDir)
		} else {
			ensureCached(t, tc.name, imagePath, cacheDir)
		}
	}
}

// ensureNotCached checks the entry related to an image is not in the cache
func ensureNotCached(t *testing.T, testName string, imagePath string, cacheParentDir string) {
	shasum, err := client.ImageHash(imagePath)
	if err != nil {
		t.Fatalf("couldn't compute hash of image %s: %v", imagePath, err)
	}

	// Where the cached image should be
	cacheImagePath := path.Join(cacheParentDir, "cache", "library", shasum)

	// The image file shouldn't be present
	if e2e.PathExists(t, cacheImagePath) {
		t.Fatalf("%s failed: %s is still in the cache (%s)", testName, imgName, cacheImagePath)
	}
}

// ensureCached checks the entry related to an image is really in the cache
func ensureCached(t *testing.T, testName string, imagePath string, cacheParentDir string) {
	shasum, err := client.ImageHash(imagePath)
	if err != nil {
		t.Fatalf("couldn't compute hash of image %s: %v", imagePath, err)
	}

	// Where the cached image should be
	cacheImagePath := path.Join(cacheParentDir, "cache", "library", shasum)

	// The image file shouldn't be present
	if !e2e.PathExists(t, cacheImagePath) {
		t.Fatalf("%s failed: %s is not in the cache (%s)", testName, imgName, cacheImagePath)
	}
}

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) testhelper.Tests {
	c := cacheTests{
		env: env,
	}

	np := testhelper.NoParallel

	return testhelper.Tests{
		"interactive commands":     np(c.testInteractiveCacheCmds),
		"non-interactive commands": np(c.testNoninteractiveCacheCmds),
		"issue5097":                np(c.issue5097),
		"issue5350":                np(c.issue5350),
	}
}
