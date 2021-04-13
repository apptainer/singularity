// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package gpu

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hpcng/singularity/pkg/sylog"
)

// NvidiaPaths returns a list of Nvidia libraries/binaries that should be
// mounted into the container in order to use Nvidia GPUs
func NvidiaPaths(configFilePath string) ([]string, []string, error) {
	nvidiaFiles, err := gpuliblist(configFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("could not read %s: %v", filepath.Base(configFilePath), err)
	}

	return paths(nvidiaFiles)
}

// NvidiaIpcsPath returns a list of nvidia driver ipcs.
// Currently this is only the persistenced socket (if found).
func NvidiaIpcsPath() ([]string, error) {
	const persistencedSocket = "/var/run/nvidia-persistenced/socket"
	var nvidiaFiles []string
	_, err := os.Stat(persistencedSocket)
	// If it doesn't exist that's okay - probably persistenced isn't running.
	if os.IsNotExist(err) {
		sylog.Verbosef("persistenced socket %s not found", persistencedSocket)
		return nil, nil
	}
	// If we can't stat it, we probably can't bind mount it.
	if err != nil {
		return nil, fmt.Errorf("could not stat %s: %v", persistencedSocket, err)
	}

	nvidiaFiles = append(nvidiaFiles, persistencedSocket)
	return nvidiaFiles, nil
}

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
