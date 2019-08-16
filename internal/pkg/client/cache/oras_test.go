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

	"github.com/sylabs/singularity/internal/pkg/oras"
	"github.com/sylabs/singularity/internal/pkg/test"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

func TestOrasImage(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	chechIfCacheDisabled(t)

	// Create a clean empty image cache
	imageCacheDir, err := ioutil.TempDir("", "image-cache-")
	if err != nil {
		t.Fatalf("failed to create a temporary image cache")
	}
	defer os.RemoveAll(imageCacheDir)

	// Create a file in the image cache that would be an invalid image
	const invalidImageName = "invalidImage"
	basedir := filepath.Join(imageCacheDir, CacheDir, OrasDir)
	err = os.MkdirAll(basedir, 0755) // Clean up is implicit when destroying the cache
	if err != nil {
		t.Fatalf("failed to create %s: %s", basedir, err)
	}
	invalidImagePath := filepath.Join(basedir, invalidImageName)
	err = fs.Touch(invalidImagePath)
	if err != nil {
		t.Fatalf("failed to create temporary file: %s", err)
	}

	// Create a file in the image cache that would be reported as a valid image
	const validImageName = "validImage"
	basedir = filepath.Join(imageCacheDir, CacheDir, OrasDir, "0") // For now we assume the sum is zero and will update later
	err = os.MkdirAll(basedir, 0755)                               // Clean up in implicit when destroying the cache
	if err != nil {
		t.Fatalf("failed to create %s: %s", basedir, err)
	}
	validImagePath := filepath.Join(basedir, validImageName)
	err = fs.Touch(validImagePath)
	if err != nil {
		t.Fatalf("failed to create temporary file: %s", err)
	}
	hash, err := oras.ImageHash(validImagePath)
	if err != nil {
		t.Fatalf("failed to get hash for image %s: %s", validImagePath, err)
	}
	newBasedir := filepath.Join(imageCacheDir, CacheDir, OrasDir, hash)
	err = os.Rename(basedir, newBasedir)
	if err != nil {
		t.Fatalf("failed to rename %s to %s: %s", basedir, newBasedir, err)
	}
	validImagePath = filepath.Join(newBasedir, validImageName)

	c, err := NewHandle(Config{BaseDir: imageCacheDir})
	if err != nil {
		t.Fatalf("failed to create an image cache handle: %s", err)
	}

	tests := []struct {
		name           string
		sum            string
		imgName        string
		shallPass      bool
		expectedResult string
	}{
		{
			name:           "Image not existing",
			sum:            "",
			imgName:        "",
			shallPass:      false,
			expectedResult: filepath.Join(imageCacheDir, CacheDir, OrasDir, ""),
		},
		{
			name:           "Invalid image",
			sum:            "",
			imgName:        invalidImageName,
			shallPass:      false,
			expectedResult: invalidImagePath,
		},
		{
			name:           "Valid image",
			sum:            hash,
			imgName:        validImageName,
			shallPass:      true,
			expectedResult: validImagePath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.OrasImage(tt.sum, tt.imgName)
			if result != tt.expectedResult {
				t.Fatalf("test %s returned %s instead of %s", tt.name, result, tt.expectedResult)
			}

			exists, err := c.OrasImageExists(tt.sum, tt.imgName)
			if err != nil && tt.shallPass {
				t.Fatalf("failed to check if image exists: %s", err)
			}
			if !exists && tt.shallPass {
				t.Fatalf("test %s reported image as not existing", tt.name)
			}
			if exists && !tt.shallPass {
				t.Fatalf("test %s reported image as existing", tt.name)
			}
		})
	}
}
