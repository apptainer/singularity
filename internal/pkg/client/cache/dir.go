// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Package cache provides support for automatic caching of any image supported by containers/image
package cache

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strconv"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/pkg/syfs"
)

var ErrBadChecksum = errors.New("hash does not match")

const (
	// DirEnv specifies the environment variable which can set the directory
	// for image downloads to be cached in
	DirEnv = "SINGULARITY_CACHEDIR"

	// DisableCacheEnv specifies whether the image should be used
	DisableEnv = "SINGULARITY_DISABLE_CACHE"

	// CacheDir specifies the name of the directory relative to the
	// singularity data directory where images are cached in by
	// default.
	// Uses "~/.singularity/cache" which will not clash with any 2.x cache
	// directory.
	CacheDir = "cache"
)

// Config describes the requested configuration requested when a new handle is created,
// as defined by the user through command flags and environment variables.
type Config struct {
	// BaseDir specifies the location where the user wants the cache to be created.
	BaseDir string

	// Disable specifies whether the user request the cache to be disabled by default.
	Disable bool
}

// Handle is an structure representing a cache
type Handle struct {
	// basedir is the base directory of the image cache. By default, it is set
	// to $HOME/.singularity. Users can also set the SINGULARITY_CACHEDIR
	// environment variable to set BaseDir. baseDir is also used when the code
	// sets the SINGULARITY_CACHEDIR for a child process that needs to use the
	// image cache. baseDir is not meant be modified once an handle is
	// initialized; its value can be accessed using GetBasedir()
	baseDir string

	// rootDir is the cache root directory, within BaseDir. This is the
	// directory Singularity actually manages, i.e., that can safely be
	// deleted as opposed to the base directory that is potentially managed
	// (passed in) by the user.
	// It was in the past a global variable that was set when calling cache
	// functions and updated based on the value of the environment variable.
	// This approach created problem in multi-threaded cases (e.g., unit
	// tests) where each thread was concurrently updating the environment
	// variable creating inconsistencies.
	rootDir string

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

	// Oras provides the location of the ORAS cache
	Oras string

	// disabled specifies if the test is disabled
	disabled bool
}

// NewHandle initializes a new cache within a given directory. It does not set
// the environment variable to specify the location of the cache, the caller
// being in charge of doing so. This also allows us to have a thread-safe
// function, changing the value of the environment variable potentially
// impacting other threads (e.g., while running unit tests). If baseDir is an
// empty string, the image cache will be located to the default location, i.e.,
// $HOME/.singularity.
func NewHandle(cfg Config) (*Handle, error) {
	newCache := new(Handle)

	// Check whether the cache is disabled by the user.

	// strconv.ParseBool("") raises an error so we cannot directly use strconv.ParseBool(os.Getenv(DisableEnv))
	envCacheDisabled := os.Getenv(DisableEnv)
	if envCacheDisabled == "" {
		envCacheDisabled = "0"
	}
	var err error
	// We check if the environment variable to disable the cache is set
	newCache.disabled, err = strconv.ParseBool(envCacheDisabled)
	if err != nil {
		return nil, fmt.Errorf("failed to parse environment variable %s: %s", DisableEnv, err)
	}
	// If the cache is not already disabled, we check if the configuration that was passed in
	// request the cache to be disabled
	if !newCache.disabled && cfg.Disable {
		newCache.disabled = true
	}
	// If the cache is disabled, we stop here. Basically we return a valid handle that is not fully initialized
	// since it would create the directories required by an enabled cache.
	if newCache.disabled {
		return newCache, nil
	}

	// cfg is what is requested so we should not change any value that it contains
	baseDir := cfg.BaseDir
	if baseDir == "" {
		baseDir = getCacheBasedir()
	}

	ep, err := fs.FirstExistingParent(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get first existing parent of cache directory: %v", err)
	}

	// We check if we can write to the basedir or its first existing parent,
	// if not we disable the caching mechanism
	if !fs.IsWritable(ep) {
		newCache.disabled = true
		return newCache, nil
	}

	// create basedir plus any required parent dir if cache is enabled and it does not exist
	if err := initCacheDir(baseDir); err != nil {
		return nil, fmt.Errorf("failed initializing cache directory: %s", err)
	}

	/* Initialize the root directory of the cache */
	rootDir := getCacheRoot(baseDir)
	// We make sure that the rootDir is actually a valid value
	// FIXME: not really necessary anymore
	user, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("failed to get user information: %s", err)
	}
	if rootDir == "" || rootDir == user.HomeDir {
		return nil, fmt.Errorf("invalid root directory")
	}

	if err = initCacheDir(rootDir); err != nil {
		return nil, fmt.Errorf("failed initializing caching directory: %s", err)
	}

	newCache.baseDir = baseDir
	newCache.rootDir = rootDir
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
	newCache.Oras, err = getOrasCachePath(newCache)
	if err != nil {
		return nil, fmt.Errorf("failed getting the path to the ORAS cache")
	}

	return newCache, nil
}

// getCacheBaseDir figures out where the cache base directory is.
//
// Singularity makes the following assumptions:
// - the default location for caches is specified by RootDefault
// - a user can specify the environment variable specified by DirEnv to
//   change the location
// - a user can change the location of a cache at any time
// - but in the context of a Singularity command, the cache location
//   cannot change once the command starts executing
func getCacheBasedir() string {
	// If the user defined the special environment variable, we use its value
	// as base directory.
	basedir := os.Getenv(DirEnv)
	if basedir != "" {
		return basedir
	}

	// If the environment variable is not set, we use the default cache.
	sylog.Debugf("environment variable %s not set, using default image cache", DirEnv)
	basedir = syfs.ConfigDir()

	return basedir
}

// getCacheRoot figures out what the root directory is.
// Note: basedir and root are different and described earlier.
// The DirEnv environment variable can be used to set the basedir (or it
// is set to a default location) and the root is the actual cache directory
// within basedir, which we can safely delete (it is only supposed to contain
// files and directories that we create).
func getCacheRoot(basedir string) string {
	return path.Join(basedir, CacheDir)
}

// GetBasedir returns the image cache's base directory.
func (c *Handle) GetBasedir() string {
	return c.baseDir
}

// IsDisabled returns true if the cache is disabled
func (c *Handle) IsDisabled() bool {
	return c.disabled
}

// updateCacheSubdir update/create a sub-cache (directory) within the cache,
// for example, the 'shub' cache.
func updateCacheSubdir(c *Handle, subdir string) (string, error) {
	// This function may act on an cache object that is not fully initialized
	// so it is not a method on a Handle but rather an independent
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
		sylog.Fatalf("Unable to get abs filepath: %v", err)
	}

	if err := initCacheDir(absdir); err != nil {
		sylog.Fatalf("Unable to initialize caching directory: %v", err)
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

// cleanAllCaches is an utility function that wipes all files in the
// cache directory, will return a error if one occurs
func (c *Handle) cleanAllCaches() {
	if c.disabled {
		return
	}

	cacheDirs := map[string]string{
		"library": c.Library,
		"oci":     c.OciTemp,
		"blob":    c.OciBlob,
		"shub":    c.Shub,
		"oras":    c.Oras,
		"net":     c.Net,
	}

	for name, dir := range cacheDirs {
		if err := os.RemoveAll(dir); err != nil {
			sylog.Verbosef("unable to clean %s cache, directory %s: %v", name, dir, err)
		}
	}
}
