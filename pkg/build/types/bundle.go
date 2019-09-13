// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package types

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	ocitypes "github.com/containers/image/types"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/util/crypt"
)

const OCIConfigJSON = "oci-config"

// Bundle is the temporary environment used during the image building process.
type Bundle struct {
	JSONObjects map[string][]byte `json:"jsonObjects"`
	Recipe      Definition        `json:"rawDeffile"`
	Opts        Options           `json:"opts"`

	RootfsPath string `json:"rootfsPath"` // where actual fs to chroot will appear
	TmpDir     string `json:"tmpPath"`    // where temp files required during build will appear
}

// Options defines build time behavior to be executed on the bundle.
type Options struct {
	// Sections are the parts of the definition to run during the build.
	Sections []string `json:"sections"`
	// TmpDir specifies a non-standard temporary location to perform a build.
	TmpDir string
	// LibraryURL contains URL to library where base images can be pulled.
	LibraryURL string `json:"libraryURL"`
	// LibraryAuthToken contains authentication token to access specified library.
	LibraryAuthToken string `json:"libraryAuthToken"`
	// contains docker credentials if specified.
	DockerAuthConfig *ocitypes.DockerAuthConfig
	// EncryptionKeyInfo specifies the key used for filesystem
	// encryption if applicable.
	// A nil value indicates encryption should not occur.
	EncryptionKeyInfo *crypt.KeyInfo
	// NoTest indicates if build should skip running the test script.
	NoTest bool `json:"noTest"`
	// Force automatically deletes an existing container at build destination while performing build.
	Force bool `json:"force"`
	// Update detects and builds using an existing sandbox container at build destination.
	Update bool `json:"update"`
	// NoHTTPS instructs builder not to use secure connection.
	NoHTTPS bool `json:"noHTTPS"`
	// NoCleanUp allows a user to prevent a bundle from being cleaned up after a failed build.
	// useful for debugging.
	NoCleanUp bool `json:"noCleanUp"`
	// NoCache when true, will not use any cache, or make cache.
	NoCache bool
	// ImgCache stores a pointer to the image cache to use.
	ImgCache *cache.Handle
}

// NewEncryptedBundle creates an Encrypted Bundle environment.
func NewEncryptedBundle(rootfs, tempDir string, keyInfo *crypt.KeyInfo) (b *Bundle, err error) {
	return newBundle(rootfs, tempDir, keyInfo)
}

// NewBundle creates a Bundle environment.
func NewBundle(rootfs, tempDir string) (b *Bundle, err error) {
	return newBundle(rootfs, tempDir, nil)
}

// RunSection iterates through the sections specified in a bundle
// and returns true if the given string, s, is a section of the
// definition that should be executed during the build process.
func (b *Bundle) RunSection(s string) bool {
	for _, section := range b.Opts.Sections {
		if section == "none" {
			return false
		}
		if section == "all" || section == s {
			return true
		}
	}
	return false
}

// Remove cleans up any bundle files.
func (b *Bundle) Remove() error {
	var errors []string
	for _, dir := range []string{b.TmpDir, b.RootfsPath} {
		if err := os.RemoveAll(dir); err != nil {
			errors = append(errors, fmt.Sprintf("could not remove %q: %v", dir, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, " "))
	}
	return nil
}

// newBundle creates a minimum bundle with root filesystem in rootfs.
// Any temporary files created during build process will be in tempDir/bundle-temp-*
// directory, that will be cleaned up after successful build.
func newBundle(rootfs, tempDir string, keyInfo *crypt.KeyInfo) (*Bundle, error) {
	tmpPath, err := ioutil.TempDir(tempDir, "bundle-temp-")
	if err != nil {
		return nil, fmt.Errorf("could not create temp dir in %q: %v", tempDir, err)
	}
	sylog.Debugf("Created temporary directory %q for the bundle", tmpPath)

	if err := os.MkdirAll(rootfs, 0755); err != nil {
		if err := os.Remove(tmpPath); err != nil {
			sylog.Errorf("Could not cleanup temp dir %q: %v", tmpPath, err)
		}
		return nil, fmt.Errorf("could not create %q: %v", rootfs, err)
	}
	sylog.Debugf("Created directory %q for the bundle", rootfs)

	return &Bundle{
		RootfsPath:  rootfs,
		TmpDir:      tmpPath,
		JSONObjects: make(map[string][]byte),
		Opts: Options{
			EncryptionKeyInfo: keyInfo,
		},
	}, nil
}
