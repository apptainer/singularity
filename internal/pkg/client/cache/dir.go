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

	// StateInitialized represents the state of a give cache after successful initialization
	StateInitialized = "initialized"

	// StateInvalid represents the state of an invalid cache
	StateInvalid = "invalid"
)

// SingularityCache is an opaque structure representing a cache
type SingularityCache struct {
	// Basedir for the cache. This directory is never entirely deleted since it
	// can be specified by the user via the DirEnv environment variable.
	BaseDir string

	// Root directory of the cache, within the basedir. This is the directory
	// Singularity actually manages, i.e., that can safely be deleted as
	// opposed to the base directory that is potentially managed (passed in)
	// by the user
	Root string

	// State of the cache. We enable manual change of the state mainly for
	// testing
	State string

	// Default specifies if the handle points at the default image cache or
	// not. This enables quick lookup.
	Default bool

	// PreviousDirEnv stores the value of the DirEnv environment variable
	// before it is implicitly modified. This is used to restore the
	// environment configuration in some specific contexts.
	PreviousDirEnv string

	// Library provides the location of the Library cache
	Library string

	// OciTemp provides the location of the OciTemp cache
	OciTemp string

	// OciBlob provides the location of the OciBlob cache
	OciBlob string

	// Net provides the location of the Net cache
	Net string

	// Shub provides the location of the Shub cache
	Shub string
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

	return Init(basedir)
}

// Init initializes a new cache within a given directory
func Init(baseDir string) (*SingularityCache, error) {
	rootDir := getCacheRoot(baseDir)
	if err := initCacheDir(rootDir); err != nil {
		return nil, fmt.Errorf("failed initializing caching directory: %s", err)
	}

	newCache := new(SingularityCache)
	savedDirEnv := os.Getenv(DirEnv)
	os.Setenv(DirEnv, baseDir)
	newCache.PreviousDirEnv = savedDirEnv

	newCache.BaseDir = baseDir
	isDefaultCache, err := isDefaultBasedir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to check if this is the default cache: %s", err)
	}
	newCache.Root = rootDir
	newCache.State = StateInitialized
	newCache.Default = isDefaultCache
	newCache.Library, err = getLibraryCachePath(newCache)
	if err != nil {
		return nil, fmt.Errorf("failed getting the path to the Library cache: %s", err)
	}
	newCache.OciTemp, err = getOciTempCachePath(newCache)
	if err != nil {
		return nil, fmt.Errorf("failed getting the path to the OCI temp cache")
	}
	newCache.OciBlob, err = getOciBlobCachePath(newCache)
	if err != nil {
		return nil, fmt.Errorf("failed getting the path to the OCI blob cache")
	}
	newCache.Net, err = getNetCachePath(newCache)
	if err != nil {
		return nil, fmt.Errorf("failed getting the path to the Net cache")
	}
	newCache.Shub, err = getShubCachePath(newCache)
	if err != nil {
		return nil, fmt.Errorf("failed getting the path to the Shub cache")
	}

	// Sanity check to ensure that everything is fine before returning the
	// handle. We do not know a way at the moment to reach the error case.
	if !newCache.IsValid() {
		return nil, fmt.Errorf("unable to correctly initialize new cache")
	}

	return newCache, nil
}

// Destroy a specific Singularity cache
func (c *SingularityCache) Destroy() error {
	sylog.Debugf("Removing: %v", c.Root)
	if !c.IsValid() {
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
	if c.State != StateInitialized {
		return false
	}

	if c.BaseDir == "" {
		return false
	}

	// Since Clean/Destroy delete everything in the cache directory,
	// we make sure that when the user set the environment variable
	// to specify where the cache should be, it cannot be in critical
	// directory such as $HOME
	usr, err := user.Current()
	if err != nil {
		return false
	}

	// The root cannot be empty of the Home directory. If the root was
	// to be the home directory, destroying the cache would delete
	// user's data
	if c.Root == "" || c.Root == usr.HomeDir {
		return false
	}

	// Basic check of the sub-cache validity
	if c.Library == "" || c.Net == "" || c.OciTemp == "" || c.OciBlob == "" || c.Shub == "" {
		return false
	}

	return true
}

// Clean wipes all files in the cache directory, will return a error if one occurs
// Since renamed Destroy() but kept for backward compatibility
func (c *SingularityCache) Clean() error {
	return c.Destroy()
}

func isDefaultBasedir(basedir string) (bool, error) {
	usr, err := user.Current()
	if err != nil {
		return false, fmt.Errorf("failed to get user: %s", err)
	}

	if basedir == path.Join(usr.HomeDir, BasedirDefault) {
		return true, nil
	}

	return false, nil
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

// getCacheRoot figures out what the root directory is
func getCacheRoot(basedir string) string {
	return path.Join(basedir, RootDefault)
}

// updateCacheSubdir update/create a sub-cache (directory) within the cache,
// for example, the 'shub' cache.
func updateCacheSubdir(c *SingularityCache, subdir string) (string, error) {
	// This function may act on an cache object that is not fully initialized
	// so it is not a method on a SingularityCache but rather an independent
	// function.
	// Because the cache object may not be initialized, we do NOT check its validity
	if c == nil {
		return "", fmt.Errorf("invalid cache handle")
	}

	// If the subdir is empty, it will lead to new collision since it would
	// succeed but point at the cache's root
	if subdir == "" {
		return "", fmt.Errorf("invalid parameter")
	}

	absdir, err := filepath.Abs(filepath.Join(c.Root, subdir))
	if err != nil {
		return "", fmt.Errorf("unable to get abs filepath: %v", err)
	}

	if err := initCacheDir(absdir); err != nil {
		return "", fmt.Errorf("unable to initialize caching directory: %v", err)
	}

	sylog.Debugf("Caching directory set to %s", absdir)

	return absdir, nil
}

// initCacheDir initializes a sub-cache within a cache, e.g., the shub sub-cache.
func initCacheDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		sylog.Debugf("Creating cache directory: %s", dir)
		if err := fs.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("couldn't create cache directory %v: %v", dir, err)
		}
	} else if err != nil {
		return fmt.Errorf("unable to stat %s: %s", dir, err)
	}

	if !fs.IsDir(dir) {
		return fmt.Errorf("%s is not a directory", dir)
	}

	return nil
}
