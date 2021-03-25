// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package gpu

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sylabs/singularity/pkg/sylog"
)

// NvidiaPaths returns a list of Nvidia libraries/binaries that should be
// mounted into the container in order to use Nvidia GPUs
func NvidiaPaths(configFilePath, userEnvPath string) ([]string, []string, error) {
	if userEnvPath != "" {
		oldPath := os.Getenv("PATH")
		os.Setenv("PATH", userEnvPath)
		defer os.Setenv("PATH", oldPath)
	}

	// Parse nvidia-container-cli for the necessary binaries/libs, fallback to a
	// list of required binaries/libs if the nvidia-container-cli is unavailable
	nvidiaFiles, err := nvidiaContainerCli("list", "--binaries", "--libraries")
	if err != nil {
		sylog.Verbosef("nvidiaContainerCli returned: %v", err)
		sylog.Verbosef("Falling back to nvliblist.conf")

		nvidiaFiles, err = gpuliblist(configFilePath)
		if err != nil {
			return nil, nil, fmt.Errorf("could not read %s: %v", filepath.Base(configFilePath), err)
		}
	}

	return paths(nvidiaFiles)
}

// NvidiaIpcsPath returns list of nvidia ipcs driver.
func NvidiaIpcsPath(envPath string) []string {
	const persistencedSocket = "/var/run/nvidia-persistenced/socket"

	if envPath != "" {
		oldPath := os.Getenv("PATH")
		os.Setenv("PATH", envPath)
		defer os.Setenv("PATH", oldPath)
	}

	var nvidiaFiles []string
	nvidiaFiles, err := nvidiaContainerCli("list", "--ipcs")
	if err != nil {
		sylog.Verbosef("nvidiaContainerCli returned: %v", err)
		sylog.Verbosef("Falling back to default path %s", persistencedSocket)

		// nvidia-container-cli may not be installed, check
		// default path
		_, err := os.Stat(persistencedSocket)
		if os.IsNotExist(err) {
			sylog.Verbosef("persistenced socket %s not found", persistencedSocket)
		} else {
			nvidiaFiles = append(nvidiaFiles, persistencedSocket)
		}
	}

	return nvidiaFiles
}

// nvidiaContainerCli runs `nvidia-container-cli list` and returns list of
// libraries, ipcs and binaries for proper NVIDIA work. This may return duplicates!
func nvidiaContainerCli(args ...string) ([]string, error) {
	nvidiaCLIPath, err := exec.LookPath("nvidia-container-cli")
	if err != nil {
		return nil, fmt.Errorf("could not find nvidia-container-cli: %v", err)
	}

	var out bytes.Buffer
	cmd := exec.Command(nvidiaCLIPath, args...)
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("could not execute nvidia-container-cli list: %v", err)
	}

	var libs []string
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.Contains(line, ".so") {
			// Handle the library reported by nvidia-container-cli
			libs = append(libs, line)
			// Look for and add any symlinks for this library
			soPath := strings.SplitAfter(line, ".so")[0]
			soPaths, err := soLinks(soPath)
			if err != nil {
				sylog.Errorf("while finding links for %s: %v", soPath, err)
			}
			libs = append(libs, soPaths...)
		} else {
			// this is a binary -> need full path
			libs = append(libs, line)
		}
	}
	return libs, nil
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
