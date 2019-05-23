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
// cache base directory is $HOME/.singularity. If a user sets the
// SINGULARITY_CACHEDIR environment variable, the cache base directory is then
// its value.
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

	client "github.com/sylabs/scs-library-client/client"
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

// SingularityCache is an opaque structure representing a cache
type SingularityCache struct {
	// rootDir is the cache root directory, within basedir. This is the
	// directory Singularity actually manages, i.e., that can safely be
	// deleted as opposed to the base directory that is potentially managed
	// (passed in) by the user.
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
}

// NewHandle creates a new Singularity cache handle that can then be used to
// interact with a cache. The location of the cache is driven by the DirEnv
// directory. If it points to a location where a cache is not already present,
// a new cache will implicitly be created; otherwise, the new cache handle
// will point at the existing cache.
func NewHandle() (*SingularityCache, error) {
	// Singularity makes the following assumptions:
	// - the default location for caches is specified by RootDefault
	// - a user can specify the environment variable specified by DirEnv to
	//   change the location
	// - a user can change the location of a cache at any time
	// - but in the context of a Singularity command, the cache location
	//   cannot change once the command starts executing
	basedir, err := getCacheBasedir()
	if err != nil {
		return nil, fmt.Errorf("failed to get root of the cache: %s", err)
	}

	return hdlInit(basedir)
}

// Init initializes a new cache within a given directory
func hdlInit(baseDir string) (*SingularityCache, error) {
	rootDir := getCacheRoot(baseDir)
	fmt.Println("Setting cache root to: ", rootDir)
	if err := initCacheDir(rootDir); err != nil {
		return nil, fmt.Errorf("failed initializing caching directory: %s", err)
	}

	newCache := new(SingularityCache)
	os.Setenv(DirEnv, baseDir)

	newCache.rootDir = rootDir
	var err error
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
	if !newCache.isValid() {
		return nil, fmt.Errorf("unable to correctly initialize new cache")
	}

	return newCache, nil
}

func (c *SingularityCache) destroy() error {
	sylog.Debugf("Removing: %v", c.rootDir)
	err := os.RemoveAll(c.rootDir)
	if err != nil {
		return fmt.Errorf("failed to delete the cache: %s", err)
	}
	return nil
}

// Clean wipes all files in the cache directory, will return a error if one
// occurs.
func (c *SingularityCache) Clean(cacheType string) error {
	if !c.isValid() {
		return fmt.Errorf("invalid cache")
	}

	switch cacheType {
	case "library":
		return c.cleanLibraryCache()
	case "oci":
		return c.cleanOciCache()
	case "blob", "blobs":
		return c.cleanBlobCache()
	case "net":
		return c.CleanNetCache()
	case "shub":
		return c.CleanShubCache()
	case "all":
		fallthrough
	default:
		return c.destroy()
	}
}

// isValid checks whether a given Singularity cache is valid or not
func (c *SingularityCache) isValid() bool {
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

// getCacheBaseDir figures out where the cache base directory is.
func getCacheBasedir() (string, error) {
	// If the user defined the special environment variable, we use its value
	// as base directory.
	basedir := os.Getenv(DirEnv)
	if basedir != "" {
		return basedir, nil
	}

	// If the environment variable is not set, we use the default cache.
	basedir = syfs.ConfigDir()

	return basedir, nil
}

// getCacheRoot figures out what the root directory is.
func getCacheRoot(basedir string) string {
	// Note: basedir and root are different and described earlier.
	// The DirEnv environment variable can be used to set the basedir (or it
	// is set to a default location) and the root is the actual cache directory
	// within basedir, which we can safely delete (it is only supposed to contain
	// files and directories that we create).
	return path.Join(basedir, CacheDir)
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

// initCacheDir initializes a sub-cache within a cache, e.g., the shub
// sub-cache.
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

// checkImageHash checks whether the sha256 hash of an image in the cache
// matches the sum passed in.
func checkImageHash(path, sum string) bool {
	cacheSum, err := client.ImageHash(path)
	if err != nil {
		return false
	}

	if sum != cacheSum {
		return false
	}

	return true
}
