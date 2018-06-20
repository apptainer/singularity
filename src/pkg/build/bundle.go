// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"
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
}

// NewBundle creates a Bundle environment
// TODO: choose appropriate location for TempDir, currently using /tmp
func NewBundle() (b *Bundle, err error) {
	b = &Bundle{}

	dir, err := ioutil.TempDir("", "sbuild-"+strconv.FormatInt(time.Now().Unix(), 10)+"-")
	if err != nil {
		return nil, err
	}

	b.Path = dir

	b.FSObjects = map[string]string{
		"rootfs":         "fs",
		".singularity.d": ".singularity.d",
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
