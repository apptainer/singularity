// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package gpu

import (
	"bufio"
	"bytes"
	"debug/elf"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sylabs/singularity/internal/pkg/sylog"
)

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
			// this is a library -> add library name
			lib := filepath.Base(line)
			libs = append(libs, lib)

			// also add library without .xxx.xx suffix
			bareLib := strings.SplitAfter(lib, ".so")[0]
			libs = append(libs, bareLib)
		} else {
			// this is a binary -> need full path
			libs = append(libs, line)
		}
	}
	return libs, nil
}

// gpuliblist returns libraries listed in a gpu lib list config file, typically
// located in buildcfg.SINGULARITY_CONFDIR
func gpuliblist(configFilePath string) ([]string, error) {
	file, err := os.Open(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("could not open %s: %v", configFilePath, err)
	}
	defer file.Close()

	var libs []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && line[0] != '#' {
			libs = append(libs, line)
		}
	}
	return libs, nil
}

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

// RocmPaths returns a list of rocm libraries/binaries that should be
// mounted into the container in order to use AMD GPUs
func RocmPaths(configFilePath, userEnvPath string) ([]string, []string, error) {
	if userEnvPath != "" {
		oldPath := os.Getenv("PATH")
		os.Setenv("PATH", userEnvPath)
		defer os.Setenv("PATH", oldPath)
	}

	rocmFiles, err := gpuliblist(configFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("could not read %s: %v", filepath.Base(configFilePath), err)
	}

	return paths(rocmFiles)
}

// paths handles generic library parsing functionality once the platform
// specific libs/binaries have been identified
func paths(gpuFileList []string) ([]string, []string, error) {
	// walk through the ldconfig output and add entries which contain the filenames
	// returned by nvidia-container-cli OR the nvliblist.conf file contents
	ldConfig, err := exec.LookPath("ldconfig")
	if ee, ok := err.(*exec.Error); ok && ee.Err == exec.ErrNotFound {
		sylog.Debugf("Could not find ldconfig in PATH")
		ldConfig = "ldconfig"
	}
	if err != nil {
		return nil, nil, fmt.Errorf("could not lookup ldconfig: %v", err)
	}
	out, err := exec.Command(ldConfig, "-p").Output()
	if err != nil {
		return nil, nil, fmt.Errorf("could not execute ldconfig: %v", err)
	}

	// sample ldconfig -p output:
	// libnvidia-ml.so.1 (libc6,x86-64) => /usr/lib64/nvidia/libnvidia-ml.so.1
	r, err := regexp.Compile(`(?m)^(.*)\s*\(.*\)\s*=>\s*(.*)$`)
	if err != nil {
		return nil, nil, fmt.Errorf("could not compile ldconfig regexp: %v", err)
	}

	// get elf machine to match correct libraries during ldconfig lookup
	self, err := elf.Open("/proc/self/exe")
	if err != nil {
		return nil, nil, fmt.Errorf("could not open /proc/self/exe: %v", err)
	}

	machine := self.Machine
	if err := self.Close(); err != nil {
		sylog.Warningf("Could not close ELF: %v", err)
	}

	// store library name with associated path
	ldCache := make(map[string]string)
	for _, match := range r.FindAllSubmatch(out, -1) {
		if match != nil {
			// libName is the "libnvidia-ml.so.1" (from the above example)
			// libPath is the "/usr/lib64/nvidia/libnvidia-ml.so.1" (from the above example)
			libName := strings.TrimSpace(string(match[1]))
			libPath := strings.TrimSpace(string(match[2]))
			ldCache[libPath] = libName
		}
	}

	// trach binaries/libraries to eliminate duplicates
	bins := make(map[string]struct{})
	libs := make(map[string]struct{})

	var libraries []string
	var binaries []string
	for _, file := range gpuFileList {
		// if the file contains a ".so", treat it as a library
		if strings.Contains(file, ".so") {
			for libPath, libName := range ldCache {
				if !strings.HasPrefix(libName, file) {
					continue
				}
				if _, ok := libs[libName]; !ok {
					elib, err := elf.Open(libPath)
					if err != nil {
						sylog.Debugf("ignore library %s: %s", libName, err)
						continue
					}

					if elib.Machine == machine {
						libs[libName] = struct{}{}
						libraries = append(libraries, libPath)
					}

					if err := elib.Close(); err != nil {
						sylog.Warningf("Could not close ELIB: %v", err)
					}
				}
			}
		} else {
			// treat the file as a binary file - add it to the bind list
			// no need to check the ldconfig output
			binary, err := exec.LookPath(file)
			if err != nil {
				continue
			}
			if _, ok := bins[binary]; !ok {
				bins[binary] = struct{}{}
				binaries = append(binaries, binary)
			}
		}
	}

	return libraries, binaries, nil
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
