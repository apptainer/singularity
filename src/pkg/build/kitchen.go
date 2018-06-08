// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"os"
	"path/filepath"
	"io/ioutil"
	"time"
	"strconv"
)

// Kitchen is the temporary build environment used during the image
// building process. A Kitchen is the programmatic representation of
// the directory structure which will constitute this environment.
// /tmp/...:
//     fs/ - A chroot filesystem
//     .singularity.d/ - Container metadata (from 2.x image format)
//     config.json (optional) - Contain information for OCI image bundle
//     etc... - The Kitchen dir can theoretically contain arbitrary directories,
//              files, etc... which can be interpreted by the Chef
type Kitchen struct {
	// FSObjects is a map of the filesystem objects contained in the Kitchen. An object
	// will be built as one section of a SIF file.
	//
	// Known FSObjects labels:
	//   * rootfs -> root file system
	//   * .singularity.d -> .singularity.d directory (includes image exec scripts)
	//   * data -> directory containing data files
	FSObjects   map[string]string
	JSONObjects map[string][]byte
	Recipe      Definition
	path        string
}

// NewKitchen creates a Kitchen environment
// TODO: choose appropriate location for TempDir, currently using /tmp
func NewKitchen() (k *Kitchen, err error) {
	k = &Kitchen{}

	dir, err := ioutil.TempDir("", "sbuild-"+strconv.FormatInt(time.Now().Unix(),10)+"-")
	if err != nil {
		return nil, err
	}

	k.path = dir

	k.FSObjects = map[string]string{
		"rootfs": "fs",
	}

	if err = os.MkdirAll(filepath.Join(k.path, k.FSObjects["rootfs"]), 0755); err != nil {
		return
	}

	return k, nil
}

// Rootfs give the path to the root filesystem in the kitchen
func (k *Kitchen) Rootfs() (string) {
	return filepath.Join(k.path, k.FSObjects["rootfs"])
}
