// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package types

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	ocitypes "github.com/containers/image/types"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/pkg/util/crypt"
	"golang.org/x/sys/unix"
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
	// ImgCache stores a pointer to the image cache to use.
	ImgCache *cache.Handle
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
	// FixPerms controls if we will ensure owner rwX on container content
	// to preserve <=3.4 behavior.
	// TODO: Deprecate in 3.6, remove in 3.8
	FixPerms bool
	// To warn when the above is needed, we need to know if the target of this
	// bundle will be a sandbox
	SandboxTarget bool
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
		if err := fs.ForceRemoveAll(dir); err != nil {
			errors = append(errors, fmt.Sprintf("could not remove %q: %v", dir, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, " "))
	}
	return nil
}

func canChown(rootfs string) (bool, error) {
	// we always return true when building as user otherwise
	// build process would always fail at this step
	if os.Getuid() != 0 {
		return true, nil
	}

	chownFile := filepath.Join(rootfs, ".chownTest")

	f, err := os.OpenFile(chownFile, os.O_CREATE|os.O_EXCL|unix.O_NOFOLLOW, 0600)
	if err != nil {
		return false, fmt.Errorf("could not create %q: %v", chownFile, err)
	}
	defer f.Close()
	defer os.Remove(chownFile)

	if err := f.Chown(1, 1); os.IsPermission(err) {
		return false, nil
	}

	return true, nil
}

func cleanupDir(path string) {
	if err := os.Remove(path); err != nil {
		sylog.Errorf("Could not cleanup dir %q: %v", path, err)
	}
}

// newBundle creates a minimum bundle with root filesystem in rootfs.
// Any temporary files created during build process will be in tempDir/bundle-temp-*
// directory, that will be cleaned up after successful build.
func newBundle(rootfs, tempDir string, keyInfo *crypt.KeyInfo) (*Bundle, error) {
	rootfsPath := rootfs

	tmpPath, err := ioutil.TempDir(tempDir, "bundle-temp-")
	if err != nil {
		return nil, fmt.Errorf("could not create temp dir in %q: %v", tempDir, err)
	}
	sylog.Debugf("Created temporary directory %q for the bundle", tmpPath)

	if err := os.MkdirAll(rootfsPath, 0755); err != nil {
		cleanupDir(tmpPath)
		return nil, fmt.Errorf("could not create %q: %v", rootfsPath, err)
	}

	// check that chown works with the underlying filesystem containing
	// the temporary sandbox image
	can, err := canChown(rootfsPath)
	if err != nil {
		cleanupDir(tmpPath)
		cleanupDir(rootfsPath)
		return nil, err
	} else if !can {
		defer cleanupDir(rootfsPath)

		rootfsNewPath := filepath.Join(tempDir, filepath.Base(rootfsPath))
		if rootfsNewPath != rootfsPath {
			if err := os.MkdirAll(rootfsNewPath, 0755); err != nil {
				cleanupDir(tmpPath)
				return nil, fmt.Errorf("could not create rootfs dir in %q: %v", rootfsNewPath, err)
			}
			// check that chown works with the underlying filesystem pointed
			// by $TMPDIR and return an error if chown doesn't work
			can, err := canChown(rootfsNewPath)
			if err != nil {
				cleanupDir(tmpPath)
				cleanupDir(rootfsNewPath)
				return nil, err
			} else if !can {
				cleanupDir(tmpPath)
				cleanupDir(rootfsNewPath)
				sylog.Errorf("Could not set files/directories ownership, if %s is on a network filesystem, "+
					"you must set TMPDIR to a local path (eg: TMPDIR=/var/tmp singularity build ...)", rootfsNewPath)
				return nil, fmt.Errorf("ownership change not allowed in %s, aborting", tempDir)
			}
			rootfsPath = rootfsNewPath
		}
	}

	sylog.Debugf("Created directory %q for the bundle", rootfsPath)

	return &Bundle{
		RootfsPath:  rootfsPath,
		TmpDir:      tmpPath,
		JSONObjects: make(map[string][]byte),
		Opts: Options{
			EncryptionKeyInfo: keyInfo,
		},
	}, nil
}
