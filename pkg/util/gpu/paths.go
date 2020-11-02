// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package gpu

import (
	"bufio"
	"debug/elf"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sylabs/singularity/pkg/sylog"
)

const systemLdconfig = "/sbin/ldconfig"

// gpuliblist returns libraries/binaries listed in a gpu lib list config file, typically
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

// soLinks returns a list of versioned symlinks resolving to a specified library file
func soLinks(libPath string) (paths []string, err error) {
	bareLibPath := strings.SplitAfter(libPath, ".so")[0]
	libCandidates := []string{}
	libGlobPaths, _ := filepath.Glob(fmt.Sprintf("%s*", bareLibPath))
	if len(libGlobPaths) == 0 {
		// should have at least found current lib
		return paths, fmt.Errorf("library not found: %s", libPath)
	}
	// check all files with a similar name (up to .so extension) and
	// work out which are symlinks rather than regular files
	for _, lPath := range libGlobPaths {
		if fi, err := os.Lstat(lPath); err == nil {
			if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
				libCandidates = append(libCandidates, lPath)
			}
		} else {
			sylog.Warningf("error extracting file info for %s: %v", lPath, err)
		}
	}
	// resolve symlinks and check if they eventually point to driver
	for _, lPath := range libCandidates {
		if resolvedLib, err := filepath.EvalSymlinks(lPath); err == nil {
			if resolvedLib == libPath {
				// symlinkCandidate resolves (eventually) to required lib
				sylog.Debugf("Idenfified %s as a symlink for %s", lPath, libPath)
				paths = append(paths, lPath)
			}
		} else {
			// error resolving symlink?
			sylog.Warningf("unable to resolve symlink for %s: %v", lPath, err)
		}
	}
	return paths, nil
}

// paths takes a list of library/binary files (absolute paths, or bare filenames) and processes them into lists of
// resolved library and binary paths to be bound into the container.
func paths(gpuFileList []string) ([]string, []string, error) {
	machine, err := elfMachine()
	if err != nil {
		return nil, nil, fmt.Errorf("could not retrieve ELF machine ID: %v", err)
	}
	ldCache, err := ldCache()
	if err != nil {
		return nil, nil, fmt.Errorf("could not retrieve ld cache: %v", err)
	}

	// Track processed binaries/libraries to eliminate duplicates
	bins := make(map[string]struct{})
	libs := make(map[string]struct{})

	var libraries []string
	var binaries []string
	for _, file := range gpuFileList {
		// if the file contains an ".so", treat it as a library
		if strings.Contains(file, ".so") {
			// If we have an absolute path, add it 'as-is', plus any symlinks that resolve to it
			if filepath.IsAbs(file) {
				elib, err := elf.Open(file)
				if err != nil {
					sylog.Debugf("ignoring library %s: %s", file, err)
					continue
				}

				if elib.Machine == machine {
					libraries = append(libraries, file)
					links, err := soLinks(file)
					if err != nil {
						sylog.Warningf("ignoring symlinks to %s: %v", file, err)
					} else {
						libraries = append(libraries, links...)
					}
				}
				if err := elib.Close(); err != nil {
					sylog.Warningf("Could not close ELIB: %v", err)
				}
			} else {
				for libPath, libName := range ldCache {
					if !strings.HasPrefix(libName, file) {
						continue
					}
					if _, ok := libs[libName]; !ok {
						elib, err := elf.Open(libPath)
						if err != nil {
							sylog.Debugf("ignoring library %s: %s", libName, err)
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
			}
		} else {
			// treat the file as a binary file - find on PATH and add it to the bind list
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

// ldcache retrieves a map of absolute path of a library to it's bare name using the system ld cache via `ldconfig -p`
func ldCache() (map[string]string, error) {
	// walk through the ldconfig output and add entries which contain the filenames
	// returned by nvidia-container-cli OR the nvliblist.conf file contents
	ldConfig, err := exec.LookPath("ldconfig")
	if err != nil {
		return nil, fmt.Errorf("could not lookup ldconfig: %v", err)
	}
	out, err := exec.Command(ldConfig, "-p").Output()
	// #5002 If we failed and our ldconfig is not in standard POSIX location, try that
	if err != nil && ldConfig != systemLdconfig {
		sylog.Warningf("%s failed - trying %s", ldConfig, systemLdconfig)
		out, err = exec.Command(systemLdconfig, "-p").Output()
	}
	if err != nil {
		return nil, fmt.Errorf("could not execute ldconfig: %v", err)
	}

	// sample ldconfig -p output:
	// libnvidia-ml.so.1 (libc6,x86-64) => /usr/lib64/nvidia/libnvidia-ml.so.1
	r, err := regexp.Compile(`(?m)^(.*)\s*\(.*\)\s*=>\s*(.*)$`)
	if err != nil {
		return nil, fmt.Errorf("could not compile ldconfig regexp: %v", err)
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
	return ldCache, nil
}

// elfMachine returns the ELF Machine ID for this system, w.r.t the currently running process
func elfMachine() (machine elf.Machine, err error) {
	// get elf machine to match correct libraries during ldconfig lookup
	self, err := elf.Open("/proc/self/exe")
	if err != nil {
		return 0, fmt.Errorf("could not open /proc/self/exe: %v", err)
	}
	if err := self.Close(); err != nil {
		sylog.Warningf("Could not close ELF: %v", err)
	}
	return self.Machine, nil
}
