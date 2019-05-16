// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Package syfs provides functions to access singularity's file system
// layout.
package syfs

import (
	"os"
	"os/user"
	"path/filepath"
	"sync"

	"github.com/sylabs/singularity/internal/pkg/sylog"
)

const singularityDir = ".singularity"

// cache contains the information for the current user
var cache struct {
	sync.Once
	configDir string // singularity user configuration directory
}

// ConfigDir returns the directory where the singularity user
// configuration and data is located.
func ConfigDir() string {
	cache.Do(func() {
		cache.configDir = configDir()
		sylog.Debugf("Using singularity directory %q", cache.configDir)
	})

	return cache.configDir
}

func configDir() string {
	user, err := user.Current()

	if err != nil {
		sylog.Warningf("Could not lookup the current user's information: %s", err)

		cwd, err := os.Getwd()
		if err != nil {
			sylog.Warningf("Could not get current working directory: %s", err)
			return singularityDir
		}

		return filepath.Join(cwd, singularityDir)
	}

	return filepath.Join(user.HomeDir, singularityDir)
}

// ConfigDirForUsername returns the directory where the singularity
// configuration and data for the specified username is located.
func ConfigDirForUsername(username string) (string, error) {
	u, err := user.Lookup(username)

	if err != nil {
		return "", err
	}

	if cu, err := user.Current(); err == nil && u.Username == cu.Username {
		return ConfigDir(), nil
	}

	return filepath.Join(u.HomeDir, singularityDir), nil
}
