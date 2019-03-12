// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package types

import (
	"io/ioutil"
	"os"
	"path/filepath"

	ocitypes "github.com/containers/image/types"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

// Bundle is the temporary build environment used during the image
// building process. A Bundle is the programmatic representation of
// the directory structure which will constitute this environmenb.
// /tmp/...:
//     fs/ - A chroot filesystem
//     .singularity.d/ - Container metadata (from 2.x image format)
//     config.json (optional) - Contain information for OCI image bundle
//     etc... - The Bundle dir can theoretically contain arbitrary directories,
//              files, etc... which can be interpreted by the Chef
type Bundle struct {
	// FSObjects is a map of the filesystem objects contained in the Bundle. An object
	// will be built as one section of a SIF file.
	//
	// Known FSObjects labels:
	//   * rootfs -> root file system
	//   * .singularity.d -> .singularity.d directory (includes image exec scripts)
	//   * data -> directory containing data files
	FSObjects   map[string]string `json:"fsObjects"`
	JSONObjects map[string][]byte `json:"jsonObjects"`
	Recipe      Definition        `json:"rawDeffile"`
	BindPath    []string          `json:"bindPath"`
	Path        string            `json:"bundlePath"`
	Opts        Options           `json:"opts"`
}

// Options ...
type Options struct {
	// TmpDir specifies a non-standard temporary location to perform a build
	TmpDir string
	// sections are the parts of the definition to run during the build
	Sections []string `json:"sections"`
	// noTest indicates if build should skip running the test script
	NoTest bool `json:"noTest"`
	// force automatically deletes an existing container at build destination while performing build
	Force bool `json:"force"`
	// update detects and builds using an existing sandbox container at build destination
	Update bool `json:"update"`
	// noHTTPS
	NoHTTPS bool `json:"noHTTPS"`
	// contains docker credentials if specified
	DockerAuthConfig *ocitypes.DockerAuthConfig
	// NoCleanUp allows a user to prevent a bundle from being cleaned up after a failed build
	// useful for debugging
	NoCleanUp bool `json:"noCleanUp"`
}

// NewBundle creates a Bundle environment
func NewBundle(bundleDir, bundlePrefix string) (b *Bundle, err error) {
	b = &Bundle{}
	b.JSONObjects = make(map[string][]byte)

	if bundlePrefix == "" {
		bundlePrefix = "sbuild-"
	}

	b.Path, err = ioutil.TempDir(bundleDir, bundlePrefix+"-")
	if err != nil {
		return nil, err
	}
	sylog.Debugf("Created temporary directory for bundle %v\n", b.Path)

	b.FSObjects = map[string]string{
		"rootfs": "fs",
	}

	for _, fso := range b.FSObjects {
		if err = os.MkdirAll(filepath.Join(b.Path, fso), 0755); err != nil {
			return
		}
	}

	return b, nil
}

// Rootfs give the path to the root filesystem in the Bundle
func (b *Bundle) Rootfs() string {
	return filepath.Join(b.Path, b.FSObjects["rootfs"])
}

// RunSection iterates through the sections specified in a bundle
// and returns true if the given string, s, is a section of the
// definition that should be executed during the build process
func (b Bundle) RunSection(s string) bool {
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
