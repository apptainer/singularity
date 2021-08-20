// Copyright (c) 2019-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Package bin provides access to external binaries
package bin

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/hpcng/singularity/internal/pkg/buildcfg"
	"github.com/hpcng/singularity/internal/pkg/util/env"
	"github.com/hpcng/singularity/pkg/sylog"
	"github.com/hpcng/singularity/pkg/util/singularityconf"
	"github.com/pkg/errors"
)

// FindBin returns the path to the named binary, or an error if it is not found.
func FindBin(name string) (path string, err error) {
	switch name {
	// Basic system executables that we assume are always on PATH
	case "true", "mkfs.ext3", "cp", "rm", "dd":
		return findOnPath(name)
	// Bootstrap related executables that we assume are on PATH
	case "mount", "mknod", "debootstrap", "pacstrap", "dnf", "yum", "rpm", "curl", "uname", "zypper", "SUSEConnect", "rpmkeys":
		return findOnPath(name)
	// Configurable executables that are found at build time, can be overridden
	// in singularity.conf. If config value is "" will look on PATH.
	case "ldconfig", "unsquashfs", "mksquashfs", "cryptsetup", "nvidia-container-cli", "go":
		return findFromConfig(name)
	// distro provided setUID executables that are used in the fakeroot flow to setup subuid/subgid mappings
	case "newuidmap", "newgidmap":
		return findOnPath(name)
	}
	return "", fmt.Errorf("unknown executable name %q", name)
}

// findOnPath performs a simple search on PATH for the named executable, returning its full path.
// env.DefaultPath` is appended to PATH to ensure standard locations are searched. This
// is necessary as some distributions don't include sbin on user PATH etc.
func findOnPath(name string) (path string, err error) {
	oldPath := os.Getenv("PATH")
	defer os.Setenv("PATH", oldPath)
	os.Setenv("PATH", oldPath+":"+env.DefaultPath)

	path, err = exec.LookPath(name)
	if err != nil {
		sylog.Debugf("Found %q at %q", name, path)
	}
	return path, err
}

// findFromConfig retrieves the path to an executable from singularity.conf,
// or searches PATH if not set there.
func findFromConfig(name string) (path string, err error) {
	cfg := singularityconf.GetCurrentConfig()
	if cfg == nil {
		cfg, err = singularityconf.Parse(buildcfg.SINGULARITY_CONF_FILE)
		if err != nil {
			return "", errors.Wrap(err, "unable to parse singularity configuration file")
		}
	}

	switch name {
	case "cryptsetup":
		path = cfg.CryptsetupPath
	case "go":
		path = cfg.GoPath
	case "ldconfig":
		path = cfg.LdconfigPath
	case "mksquashfs":
		path = cfg.MksquashfsPath
	case "nvidia-container-cli":
		path = cfg.NvidiaContainerCliPath
	case "unsquashfs":
		path = cfg.UnsquashfsPath
	default:
		return "", fmt.Errorf("unknown executable name %q", name)
	}

	if path == "" {
		return findOnPath(name)
	}

	sylog.Debugf("Using %q at %q (from singularity.conf)", name, path)

	// Use lookPath with the absolute path to confirm it is accessible & executable
	return exec.LookPath(path)
}
