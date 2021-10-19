// Copyright (c) 2018-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package gpu

import (
	"fmt"
	"path/filepath"
)

// RocmPaths returns a list of rocm libraries/binaries that should be
// mounted into the container in order to use AMD GPUs
func RocmPaths(configFilePath string) ([]string, []string, error) {
	rocmFiles, err := gpuliblist(configFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("could not read %s: %v", filepath.Base(configFilePath), err)
	}

	return paths(rocmFiles)
}

// RocmDevices return list of all non-GPU rocm devices present on host. If withGPU
// is true all GPUs are included in the resulting list as well.
func RocmDevices(withGPU bool) ([]string, error) {
	// Must bind in all GPU DRI devices
	rocmGlob := "/dev/dri/card*"
	if !withGPU {
		rocmGlob = "/dev/dri/card[^0-9]*"
	}
	devs, err := filepath.Glob(rocmGlob)
	if err != nil {
		return nil, fmt.Errorf("could not list rocm devices: %v", err)
	}
	// /dev/kfd is also required
	devs = append(devs, "/dev/kfd")
	return devs, nil
}
