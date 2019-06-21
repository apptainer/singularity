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
	"github.com/sylabs/singularity/pkg/syfs"
)

const (
	// DirEnv specifies the environment variable which can set the directory
	// for image downloads to be cached in
	DirEnv = "SINGULARITY_CACHEDIR"

	// CacheDir specifies the name of the directory relative to the
	// singularity data directory where images are cached in by
	// default.
	// Uses "~/.singularity/cache" which will not clash with any 2.x cache
	// directory.
	CacheDir = "cache"
)

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
}

// NewHandle initializes a new cache within a given directory. It does not set
// the environment variable to specify the location of the cache, the caller
// being in charge of doing so. This also allows us to have a thread-safe
// function, changing the value of the environment variable potentially
// impacting other threads (e.g., while running unit tests). If baseDir is an
// empty string, the image cache will be located to the default location, i.e.,
// $HOME/.singularity.
func NewHandle(baseDir string) (*Handle, error) {
	if baseDir == "" {
		baseDir = getCacheBasedir()
	}

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

	newCache := new(Handle)
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
	sylog.Infof("environment variable %s not set, using default image cache", DirEnv)
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

// Root is the root location where all of singularity caching happens. Library, Shub,
// and oci image formats supported by containers/image repository will be cached inside
//
// Defaults to ${HOME}/.singularity/cache
func (c *Handle) Root() string {
	updateCacheRoot(c)

	return c.rootDir
}

func updateCacheRoot(c *Handle) {
	if d := os.Getenv(DirEnv); d != "" && d != syfs.ConfigDir() {
		c.rootDir = d
	} else {
		c.rootDir = path.Join(syfs.ConfigDir(), CacheDir)
	}

	if err := initCacheDir(c.rootDir); err != nil {
		sylog.Fatalf("Unable to initialize caching directory: %v", err)
	}
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
	cacheDirs := map[string]string{
		"library": c.Library,
		"oci":     c.OciTemp,
		"blob":    c.OciBlob,
		"shub":    c.Shub,
		"oras":    c.Oras,
	}

	for name, dir := range cacheDirs {
		if err := os.RemoveAll(dir); err != nil {
			sylog.Verbosef("unable to clean %s cache, directory %s: %v", name, dir, err)
		}
	}
}
