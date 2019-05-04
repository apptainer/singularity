// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Package cache provides support for automatic caching of any image supported by containers/image
package cache

import (
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

const (
	// DirEnv specifies the environment variable which can set the directory
	// for image downloads to be cached in
	DirEnv = "SINGULARITY_CACHEDIR"

	// BasedirDefault specifies the directory inside of ${HOME} that images are
	// cached in by default.
	// Uses "~/.singularity/cache" which will not clash with any 2.x cache
	// directory.
	BasedirDefault = ".singularity"

	// RootDefault is the default directory created in the base directory that will actually
	// host the cache
	RootDefault = "cache"
)

// SingularityCache is an opaque structure representing a cache
type SingularityCache struct {
	// Basedir for the cache. This directory is never entirely deleted since it
	// can be specified by the user via the DirEnv environment variable.
	BaseDir string

	// Root directory of the cache, within the basedir. This is the directory
	// Singularity actually manages
	Root string

	// State of the directory. We enable manual change of the state mainly for testing
	State string
}

// Create a new Singularity cache
func Create() (*SingularityCache, error) {
	// Singularity makes the following assumptions:
	// - the default location for caches is specified by RootDefault
	// - a user can specify the environment variable specified by DirEnv to change the location
	// - a user can change the location of a cache at any time
	// - but in the context of a Singularity command, the cache location cannot change once the command starts executing
	basedir, err := getCacheBasedir()
	if err != nil {
		return nil, fmt.Errorf("failed to get root of the cache: %s", err)
	}

	return Init(basedir), nil
}

// Init initializes a new cache within a given directory
func Init(baseDir string) (*SingularityCache, error) {
	rootDir, err := getCacheRoot(baseDir)
	if err != nil {
		return nil, fmt.Errorf("unable to get the root directory: %s", err)
	}

	if err := initCacheDir(rootDir); err != nil {
		return nil, fmt.Errorf("unable to initialize caching directory: %s", err)
	}

	newCache := new(SingularityCache)
	if newCache == nil {
		return nil, fmt.Errorf("failed to allocate new object")
	}
	newCache.Root = rootDir
	newCache.State = "initialized"

	return newCache, nil
}

// Destroy a specific Singularity cache
func (c *SingularityCache) Destroy() error {
	sylog.Debugf("Removing: %v", c.Root)
	if c.IsValid() == false {
		return fmt.Errorf("invalid cache")
	}

	err := os.RemoveAll(c.Root)
	if err != nil {
		return fmt.Errorf("failed to delete the cache: %s", err)
	}

	return nil
}

// IsValid checks whether a given Singularity cache is valid or not
func (c *SingularityCache) IsValid() bool {
	// Since Clean/Destroy delete everything in the cache directory,
	// we make sure that when the user set the environment variable
	// to specify where the cache should be, it cannot be in critical
	// directory such as $HOME
	usr, err := user.Current()
	if c.Root == "" || c.Root == usr.HomeDir {
		return false
	}

	if c.State != "initialized" {
		return false
	}

	return true
}

// Clean : wipes all files in the cache directory, will return a error if one occurs
// Since renamed Destroy() but kept for backward compatibility
func (c *SingularityCache) Clean() error {
	return c.Destroy()
}

// Figure out where the cache directory is.
func getCacheBasedir() (string, error) {
	// If the user defined the special environment variable, we use its value
	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("couldn't determine user home directory: %s", err)
	}

	// Assuming the user set the environment variable
	basedir := os.Getenv(DirEnv)
	if basedir == "" {
		basedir = path.Join(usr.HomeDir, BasedirDefault)
	}

	return basedir, nil
}

// Figure out what the root directory is
func getCacheRoot(basedir string) (string, error) {
	root := path.Join(basedir, RootDefault)

	return root, nil
}

func (c *SingularityCache) updateCacheSubdir(subdir string) (string, error) {
	if c.IsValid() == false {
		return "", fmt.Errorf("invalid cache")
	}

	absdir, err := filepath.Abs(filepath.Join(c.Root, subdir))
	if err != nil {
		return "", fmt.Errorf("Unable to get abs filepath: %v", err)
	}

	if err := initCacheDir(absdir); err != nil {
		return "", fmt.Errorf("Unable to initialize caching directory: %v", err)
	}

	sylog.Debugf("Caching directory set to %s", absdir)

	return absdir, nil
}

func initCacheDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		sylog.Debugf("Creating cache directory: %s", dir)
		if err := fs.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("couldn't create cache directory %v: %v", dir, err)
		}
	} else if err != nil {
		return fmt.Errorf("unable to stat %s: %s", dir, err)
	}

	return nil
}
