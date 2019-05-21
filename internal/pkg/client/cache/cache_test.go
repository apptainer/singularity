// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"os"
	"path/filepath"
	"testing"

	client "github.com/sylabs/singularity/pkg/client/library"
)

const (
	invalidSHASum = ""
	validName     = "myImageName"
	invalidName   = ""
)

// createFakeImage allocates the minimum resources required to simulate a
// valid image in the context of cache testing. It returns the hash of the
// cache entry.
func createFakeCachedImage(t *testing.T, base string) string {
	// For now we just assign a dummy hash, the directory will be renamed
	// later based on the actual hash. By doing so, we can easily extend this
	// code later.
	tempHash := "0"

	err := os.MkdirAll(filepath.Join(base, tempHash), 0755)
	if err != nil {
		t.Fatalf("cannot create directory %s: %s", filepath.Join(base, tempHash), err)
	}
	validImage := filepath.Join(base, tempHash, validName)
	_, err = os.Create(validImage) // no need to explicitly delete the file, it will be when cleaning the cache
	if err != nil {
		t.Fatalf("cannot create file %s: %s", validImage, err)
	}

	hash, err := client.ImageHash(validImage)
	if err != nil {
		t.Fatalf("cannot get hash of image: %s", err)
	}

	os.Rename(filepath.Join(base, tempHash), filepath.Join(base, hash))

	return hash
}
