// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oci

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	oci "github.com/containers/image/oci/layout"
	"github.com/containers/image/types"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/test"
	buildTypes "github.com/sylabs/singularity/pkg/build/types"
)

const (
	invalidOCIURI = "library://valid_uri_but_not_real_image"
	invalidHash   = "1n2o3t4e5x6i7s9t0i1n2g3h4a5s6h"
)

// getInvalidRef enerate an invalid reference.
// cacheRootPath is the path to the root directory of the dummy OCI cache as
// returned by createDummyOCICache() or getTestCacheInfo()
func getInvalidRef(cacheRootPath string) string {
	return cacheRootPath + ":" + invalidHash
}

// getValidOCIURI generates a valid reference based on the dummy cache used for
// testing.
// ref is a reference to an image in the dummy OCI cache, as returned by
// getTestCacheInfo(). This abstracts the syntax of an OCI URI
func getValidOCIURI(ref string) string {
	return filepath.Join("oci://", ref)
}

// createIndexFile creates the index.json file of the dummy OCI cache.
// dir is a root directory of the dummy OCI cache, as returned by
// createDummyOCICache() or getTestCacheInfo()
func createIndexFile(t *testing.T, dir string, sum string) {
	// A set of structures that represents the structure of the index.json file.
	// It is meant to ease the creation of the file (populate the structure and
	// generate the file from it).
	type myOciPlatform struct {
		Architecture string `json:"architecture"`
		OS           string `json:"os"`
	}

	type myOciAnnotations struct {
		Name string `json:"org.opencontainers.image.ref.name"`
	}

	type myOciManifest struct {
		MediaType   string           `json:"mediaType"`
		Digest      string           `json:"digest"`
		Size        int              `json:"size"`
		Annotations myOciAnnotations `json:"annotations"`
		Platform    myOciPlatform    `json:"platform"`
	}

	type myJSON struct {
		SchemaVersion int             `json:"schemaVersion"`
		Manifests     []myOciManifest `json:"manifests"`
	}

	// Generate the file
	path := filepath.Join(dir, "index.json")
	var main myJSON
	var manifest myOciManifest
	var annotations myOciAnnotations
	var platform myOciPlatform

	platform.Architecture = "amd64"
	platform.OS = "linux"

	annotations.Name = sum

	manifest.MediaType = "application/vnd.oci.image.manifest.v1+json"
	manifest.Digest = "sha256:" + sum
	manifest.Size = 1
	manifest.Annotations = annotations
	manifest.Platform = platform

	main.SchemaVersion = 2
	main.Manifests = append(main.Manifests, manifest)
	data, jsonErr := json.Marshal(main)
	if jsonErr != nil {
		t.Fatalf("cannot unmarshal JSON: %s\n", jsonErr)
	}
	err := ioutil.WriteFile(path, data, 0664)
	if err != nil {
		t.Fatalf("cannot create index file: %s\n", err)
	}
}

// createDummyOCUCache creates a dummy OCI cache that can be used for some of our
// tests. It creates a temporary directory, the entire structure of folders
// within the cache, generates the index.json file, as well as a dummy blob.
// The function returns the path of the root directory of the dummy OCI cache
// and a SHA256 hash associated to the unique entry in the OCI cache.
func createDummyOCICache(t *testing.T) (string, string) {
	// Temporary directory that will serve as root of the OCI cache
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("cannot create temporary directory: %s\n", err)
	}

	// Create the directory structure.
	blobPath := filepath.Join(dir, "blobs")
	shaPath := filepath.Join(blobPath, "sha256")
	sum := sha256.New()
	sumFilename := hex.EncodeToString(sum.Sum(nil))
	path := filepath.Join(shaPath, sumFilename)
	err = os.MkdirAll(shaPath, 0755)
	if err != nil {
		os.RemoveAll(dir)
		t.Fatalf("cannot create cache directory: %s\n", err)
	}

	// Create the SHA256 file
	f, err := os.Create(path)
	if err != nil {
		os.RemoveAll(dir)
		t.Fatalf("cannot create file: %s\n", err)
	}
	defer f.Close()

	// Create the index.json file
	createIndexFile(t, dir, sumFilename)

	return dir, sumFilename
}

// getTestCacheInfo is a wrapper around createDummyOCICache that returns all the
// required information related to the dummy OCI cache for the execution of our
// tests.
// The function returns the root directory of the dummy OCI cache; the SHA256
// hash associated to the unique entry in the cache; and the reference of the
// dummy entry in the cache.
func getTestCacheInfo(t *testing.T) (string, string, string) {
	cacheRootDir, shasum := createDummyOCICache(t)
	dir := filepath.Join("//", cacheRootDir) // the first part of the ref is a path
	ref := dir + ":" + shasum                // then it is the reference to a blob

	return dir, shasum, ref
}

func TestParseURI(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	cacheDir, _, ref := getTestCacheInfo(t)
	defer os.RemoveAll(cacheDir)

	tests := []struct {
		name       string
		uri        string
		shouldPass bool
	}{
		{
			name:       "invalid URI",
			uri:        "",
			shouldPass: false,
		},
		{
			name:       "valid URI; invalid transport",
			uri:        invalidOCIURI,
			shouldPass: false,
		},
		{
			name:       "valid URI; valid transport",
			uri:        getValidOCIURI(ref),
			shouldPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseURI(tt.uri)
			if tt.shouldPass == false && err == nil {
				t.Fatal("invalid test passed")
			}
			if tt.shouldPass == true {
				if err != nil {
					t.Fatalf("error occurred during the execution a valid case: %s\n", err)
				}
			}
		})
	}
}

func createValidSysCtx() *types.SystemContext {
	opts := buildTypes.Options{
		NoHTTPS: true,
		DockerAuthConfig: &types.DockerAuthConfig{
			Username: "dummyuser",
			Password: "dummypassword",
		},
	}
	validSysCtx := &types.SystemContext{
		OCIInsecureSkipTLSVerify:    opts.NoHTTPS,
		DockerInsecureSkipTLSVerify: opts.NoHTTPS,
		OSChoice:                    "linux",
	}

	return validSysCtx
}

func createValidImageRef(t *testing.T, ref string) types.ImageReference {
	srcRef, err := oci.ParseReference(ref)
	if err != nil {
		t.Fatalf("cannot parser reference: %s\n", err)
	}
	return srcRef
}

func createInvalidImageRef(t *testing.T, invalidRef string) types.ImageReference {
	srcRef, err := oci.ParseReference(invalidRef)
	if err != nil {
		t.Fatalf("cannot parser reference: %s\n", err)
	}
	return srcRef
}

func TestConvertReference(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	cacheDir, _, ref := getTestCacheInfo(t)
	defer os.RemoveAll(cacheDir)
	imgCache, err := cache.NewHandle(cache.Config{BaseDir: cacheDir})
	if err != nil {
		t.Fatalf("failed to create an image cache handle")
	}

	tests := []struct {
		name       string
		ctx        *types.SystemContext
		ref        types.ImageReference
		shouldPass bool
	}{
		{
			name:       "valid image ref; undefined context",
			ref:        createValidImageRef(t, ref),
			ctx:        nil,
			shouldPass: true,
		},
		{
			name:       "valid image ref; valid context",
			ref:        createValidImageRef(t, ref),
			ctx:        createValidSysCtx(),
			shouldPass: true,
		},
		{
			name:       "invalid image ref; undefined context",
			ref:        createInvalidImageRef(t, getInvalidRef(cacheDir)),
			ctx:        nil,
			shouldPass: false,
		},
		{
			name:       "invalid image ref; valid context",
			ref:        createInvalidImageRef(t, getInvalidRef(cacheDir)),
			ctx:        createValidSysCtx(),
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ConvertReference(imgCache, tt.ref, tt.ctx)
			if tt.shouldPass == true && err != nil {
				t.Fatalf("test expected to succeeded but failed: %s\n", err)
			}
			if tt.shouldPass == false && err == nil {
				t.Fatal("test expected to fail but succeeded")
			}
		})
	}
}

// TestImageNameAndImageSHA tests both ImageName() and ImageSHA()
func TestImageNameAndImageSHA(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	// We create a dummy OCI cache to run all our tests
	cacheDir, _, _ := getTestCacheInfo(t)
	defer os.RemoveAll(cacheDir)
	imgCache, err := cache.NewHandle(cache.Config{BaseDir: cacheDir})
	if imgCache == nil || err != nil {
		t.Fatal("failed to create an image cache handle")
	}

	validSysCtx := createValidSysCtx()
	tests := []struct {
		name       string
		uri        string
		ctx        *types.SystemContext
		shouldPass bool
	}{
		{
			name:       "empty URI; undefined context",
			uri:        "",
			ctx:        nil,
			shouldPass: false,
		},
		{
			name:       "invalid URI; undefined context",
			uri:        invalidOCIURI,
			ctx:        nil,
			shouldPass: false,
		},
		{
			name:       "valid URI, undefined context",
			uri:        getValidOCIURI(cacheDir),
			ctx:        nil,
			shouldPass: true,
		},
		{
			name:       "empty URI; valid context",
			uri:        "",
			ctx:        validSysCtx,
			shouldPass: false,
		},
		{
			name:       "invalid URI; valid context",
			uri:        invalidOCIURI,
			ctx:        validSysCtx,
			shouldPass: false,
		},
		{
			name:       "valid URI; valid context",
			uri:        getValidOCIURI(cacheDir),
			ctx:        validSysCtx,
			shouldPass: true,
		},
	}

	for _, tt := range tests {
		testName := "ParseImageName - " + tt.name
		t.Run(testName, func(t *testing.T) {
			_, err := ParseImageName(imgCache, tt.uri, tt.ctx)
			if tt.shouldPass == true && err != nil {
				t.Fatalf("test expected to succeeded but failed: %s\n", err)
			}
			if tt.shouldPass == false && err == nil {
				t.Fatal("test expected to fail but succeeded")
			}
		})

		testName = "ImageSHA - " + tt.name
		t.Run(testName, func(t *testing.T) {
			_, err := ImageSHA(tt.uri, tt.ctx)
			if tt.shouldPass == true && err != nil {
				t.Fatal("test expected to succeeded but failed")
			}
			if tt.shouldPass == false && err == nil {
				t.Fatal("test expected to fail but succeeded")
			}
		})
	}
}

func TestNewImageSource(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	// Because of the nature of the context.Context type, there is really
	// not any invalid case.
	var validCtx context.Context
	validSys := createValidSysCtx()

	tests := []struct {
		name      string
		ctx       context.Context
		sys       *types.SystemContext
		shallPass bool
	}{
		{
			name:      "valid ctx, undefained sys",
			ctx:       validCtx,
			sys:       nil,
			shallPass: false,
		},
		{
			name:      "valid ctx, valid sys",
			ctx:       validCtx,
			sys:       validSys,
			shallPass: false,
			// In theory this case should succeed but it would require more
			// than the minimalistic image ref we handle here and we do not
			// have the testing helper function to do this at the moment.
			// In a nutshell, a manifest is missing for NewImageSource() to
			// succeed and creating a dummy manifest is non-trivial.
			// It will be fully tested by the E2E testing framework.
			// We keep this test because its code path is not the same than
			// other cases.
		},
	}

	// We create a minimalistic image reference that is valid enough for testing
	cacheDir, _, ref := getTestCacheInfo(t)
	defer os.RemoveAll(cacheDir)
	imgCache, err := cache.NewHandle(cache.Config{BaseDir: cacheDir})
	if err != nil {
		t.Fatalf("failed to create an image cache handle: %s", err)
	}

	imgRef := createValidImageRef(t, ref)
	validImgRef, err := ConvertReference(imgCache, imgRef, nil)
	if err != nil {
		t.Fatalf("failed to convert image reference: %s", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validImgRef.NewImageSource(tt.ctx, tt.sys)
			if tt.shallPass == true && err != nil {
				t.Fatalf("test %s failed while expected to succeed: %s", tt.name, err)
			}
			if tt.shallPass == false && err == nil {
				t.Fatalf("test %s succeeded while expected to fail", tt.name)
			}
		})
	}
}
