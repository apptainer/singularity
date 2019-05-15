// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Package cache provides support for automatic caching of any container
// image.
//
// A user can choose the location of the cache by setting the
// SINGULARITY_CACHEDIR environment variable. This enables users to:
// - set the location of the Singularity image cache based on volumes with a
//   lot of free space (image caches can be huge in term of disk space),
// - users may want to share their cache with others,
// - users may want to have different caches for different projects.
// For similar reasons, it is beneficial to keep the cache location separate
// from the rest of user-specific Singularity files. In other words,
// $HOME/.singularity usually contains cache, instances and sypgp and it is
// beneficial to manipulate the cache directory separately.
//
// Conceptually, Singularity has a cache *base directory*. By default, the
// cache base directory is located in $HOME/.singularity. If a user sets the
// SINGULARITY_CACHEDIR environment variable, the cache base directory is then
// set to its value.
// Within the cache base directory, a 'cache' directory is created that is not
// user-configurable, i.e., the user cannot change the name of that directory.
// That directory is also named the *root* of the cache. These choices allows
// us to never have to delete the cache base directory which can contains user
// data when the user sets the SINGULARITY_CACHEDIR environment variable. When
// deleting the cache, only the root directory of the cache is deleted, which
// is supposed to only be managed by Singularity.
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

	// BasedirDefault specifies the default value of the cache base directory. The
	// path of the user's home directory is prepended at runtime. Ultimately, the
	// default cache base directory is ~/.singularity/cache".
	// Using "~/.singularity/cache" also does not clash with any 2.x cache
	// directory.
	BasedirDefault = ".singularity"

	// RootDefault is the default cache root directory created within the base
	// directory. This value is not supposed to be set by the user.
	rootDefault = "cache"
)

// SingularityCache is an opaque structure representing a cache
type SingularityCache struct {
	// BaseDir is the cache base directory. This directory is never entirely
	// deleted since it can be specified by the user via the DirEnv environment
	// variable.
	BaseDir string

	// rootDir is the cache root directory, within basedir. This is the directory
	// Singularity actually manages, i.e., that can safely be deleted as
	// opposed to the base directory that is potentially managed (passed in)
	// by the user.
	rootDir string

	// ValidState specifies if the cache is in a valid state or not. This is
	// mainly used for testing, where a unit test can switch a cache's state
	// from valid to invalid in order to reach error cases.
	ValidState bool

	// Default specifies if the handle points at the default image cache or
	// not. This enables quick lookup. This is for instance used in the
	// context of unit tests execution to make sure we do not delete the
	// image cache in '$HOME/.singularity/cache' (developers may not
	// appreciate that some unit tests always delete their default image
	// cache).
	Default bool

	// PreviousDirEnv stores the value of the DirEnv environment variable
	// before it is implicitly modified. This is used to restore the
	// environment configuration in some specific contexts, and is therefore
	// not meant to be exposed to users.
	previousDirEnv string

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

// NewHandle creates a new Singularity cache handle that can then be used to interact with a cache. The location of the cache is driven by the DirEnv directory. If it points to a location where a cache is not already present, a new cache will implicitly be created; otherwise, the new cache handle will point at the existing cache.
func NewHandle() (*SingularityCache, error) {
	// Singularity makes the following assumptions:
	// - the default location for caches is specified by RootDefault
	// - a user can specify the environment variable specified by DirEnv to change the location
	// - a user can change the location of a cache at any time
	// - but in the context of a Singularity command, the cache location cannot change once the command starts executing
	basedir, err := getCacheBasedir()
	if err != nil {
		return nil, fmt.Errorf("failed to get root of the cache: %s", err)
	}

	return hdlInit(basedir)
}

// Init initializes a new cache within a given directory
func hdlInit(baseDir string) (*SingularityCache, error) {
	rootDir := getCacheRoot(baseDir)
	if err := initCacheDir(rootDir); err != nil {
		return nil, fmt.Errorf("failed initializing caching directory: %s", err)
	}

	newCache := new(SingularityCache)
	savedDirEnv := os.Getenv(DirEnv)
	os.Setenv(DirEnv, baseDir)
	newCache.previousDirEnv = savedDirEnv

	newCache.BaseDir = baseDir
	isDefaultCache, err := isDefaultBasedir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to check if this is the default cache: %s", err)
	}
	newCache.rootDir = rootDir
	newCache.ValidState = true
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

// Destroy a specific cache managed by a handler
func (c *SingularityCache) Destroy() error {
	sylog.Debugf("Removing: %v", c.rootDir)
	if !c.IsValid() {
		return fmt.Errorf("invalid cache")
	}

	err := os.RemoveAll(c.rootDir)
	if err != nil {
		return fmt.Errorf("failed to delete the cache: %s", err)
	}

	return nil
}

// IsValid checks whether a given Singularity cache is valid or not
func (c *SingularityCache) IsValid() bool {
	if !c.ValidState {
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

	// The root cannot be empty or the Home directory. If the root was
	// to be the home directory, destroying the cache would delete
	// user's data. Note: the root is *not* exposed to the user, only
	// the base directory is exposed and can be set by the user.
	if c.rootDir == "" || c.rootDir == usr.HomeDir {
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
	// If the user defined the special environment variable, we use its value as base directory.
	basedir := os.Getenv(DirEnv)
	if basedir != "" {
		return basedir, nil
	}

	// If the environment variable is not set, we use the default cache.
	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("couldn't determine user home directory: %s", err)
	}
	basedir = path.Join(usr.HomeDir, BasedirDefault)

	return basedir, nil
}

// getCacheRoot figures out what the root directory is.
func getCacheRoot(basedir string) string {
	// Note: basedir and root are different and described earlier.
	// The DirEnv environment variable can be used to set the basedir (or it
	// is set to a default location) and the root is the actual cache directory
	// within basedir, which we can safely delete (it is only supposed to contain
	// files and directories that we create).
	return path.Join(basedir, rootDefault)
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

	absdir, err := filepath.Abs(filepath.Join(c.rootDir, subdir))
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
	fInfo, err := os.Stat(dir)
	switch {
	case os.IsNotExist(err):
		// The directory does not exist, we create it
		sylog.Debugf("Creating cache directory: %s", dir)
		if err := fs.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("couldn't create cache directory %v: %v", dir, err)
		}
		return nil
	case err != nil:
		// A actual error occurred
		return fmt.Errorf("unable to stat %s: %s", dir, err)
	case !fInfo.IsDir():
		// This is actually not a directory
		return fmt.Errorf("%s is not a directory", dir)
	default:
		// The directory exists
		return nil
	}
}
