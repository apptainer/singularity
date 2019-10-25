// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package gpu

import (
	"fmt"
	"path/filepath"
)

// NvidiaDevices return list of all non-GPU nvidia devices present on host. If withGPU
// is true all GPUs are included in the resulting list as well.
func NvidiaDevices(withGPU bool) ([]string, error) {
	nvidiaGlob := "/dev/nvidia*"
	if !withGPU {
		nvidiaGlob = "/dev/nvidia[^0-9]*"
	}
	devs, err := filepath.Glob(nvidiaGlob)
	if err != nil {
		return nil, fmt.Errorf("could not list nvidia devices: %v", err)
	}
	return devs, nil
}

// RocmDevices return list of all non-GPU rocm devices present on host. If withGPU
// is true all GPUs are included in the resulting list as well.
func RocmDevices(withGPU bool) ([]string, error) {
	rocmGlob := "/dev/dri/card*"
	if !withGPU {
		rocmGlob = "/dev/dri/card[^0-9]*"
	}
	devs, err := filepath.Glob(rocmGlob)
	if err != nil {
		return nil, fmt.Errorf("could not list rocm devices: %v", err)
	}
	return devs, nil
}
