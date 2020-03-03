// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Package cache provides support for caching SIF, OCI, SHUB images and any OCI layers used to build them
package cache

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/pkg/syfs"
)

var (
	ErrBadChecksum = errors.New("hash does not match")
	ErrInvalidCacheType   = errors.New("invalid cache type")
)

const (
	// DirEnv specifies the environment variable which can set the directory
	// for image downloads to be cached in
	DirEnv = "SINGULARITY_CACHEDIR"
	// DisableCacheEnv specifies whether the image should be used
	DisableEnv = "SINGULARITY_DISABLE_CACHE"
	// SubDirName specifies the name of the directory relative to the
	// ParentDir specified when the cache is created.
	// By default the cache will be placed at "~/.singularity/cache" which
	// will not clash with any 2.x cache directory.
	SubDirName = "cache"

	// The Libary cache holds SIF images pulled from the library
	LibraryCacheType = "library"
	// The OCITemp cache holds SIF images created from OCI sources
	OciTempCacheType = "oci-tmp"
	// The OCIBlob cache holds OCI blobs (layers) pulled from OCI sources
	OciBlobCacheType = "blob"
	// The Shub cache holds images pulled from Singularity Hub
	ShubCacheType = "shub"
	// The Oras cache holds SIF images pulled from Oras sources
	OrasCacheType = "oras"
	// The Net cache holds images pulled from http(s) internet sources
	NetCacheType = "net"
)

var (
	FileCacheTypes = []string {
		LibraryCacheType,
		OciTempCacheType,
		ShubCacheType,
		OrasCacheType,
		NetCacheType,
	}
	OciCacheTypes = [] string {
		OciBlobCacheType,
	}
)


// Config describes the requested configuration requested when a new handle is created,
// as defined by the user through command flags and environment variables.
type Config struct {
	// ParentDir specifies the location where the user wants the cache to be created.
	ParentDir string
	// Disable specifies whether the user request the cache to be disabled by default.
	Disable bool
}

// Handle is an structure representing the image cache, it's location and subdirectories
type Handle struct {
	// parentDir is the parent of the cache root. This is the directory that is supplied
	// when initializing the cache
	parentDir string
	// rootDir is the cache root directory, and is inside parentDir. This is the
	// directory Singularity actually manages, i.e., that can safely be
	// deleted as opposed to the parent directory that is potentially managed
	// (passed in) by the user.
	rootDir string
	// If the cache is disabled
	disabled bool
}


func (h *Handle) GetOciCacheDir(cacheType string) string {
	return h.GetCacheTypeDir(cacheType)
}

// LibraryImage creates a directory inside cache with the name of the hash of the image.
func (h *Handle) GetEntry(cacheType string, hash string) (e *Entry, err error) {
	if h.disabled {
		return nil, nil
	}

	e = &Entry{}

	cacheDir := h.GetCacheTypeDir(cacheType)
	e.Path = filepath.Join(cacheDir, hash)

	_, err = os.Stat(e.Path)
	// If there is no existing dir return an entry with a TmpPath for the caller
	// to use and then Finalize
	if err != nil && os.IsNotExist(err) {
		e.Exists = false
		f, err := fs.MakeTmpFile(cacheDir, "tmp_", 0700)
		if err != nil {
			return nil, err
		}
		err = f.Close()
		if err != nil {
			return nil, err
		}
		e.TmpPath = f.Name()
		return e, nil
	}
	// Other error in Stat
	if err != nil{
		return nil, err
	}

	// It exists. Caller can use the Path directly
	e.Exists = true
	return e, nil
}

func (h *Handle) CleanCache(cacheType string, dryRun bool) error {
	dir := h.GetCacheTypeDir(cacheType)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("unable to clean %s cache, directory %s: %v", cacheType, dir, err)
	}
	return nil
}

// New initializes a cache within the directory specified in Config.ParentDir
func New(cfg Config) (h *Handle, err error) {
	h = new(Handle)

	// Check whether the cache is disabled by the user.
	// strconv.ParseBool("") raises an error so we cannot directly use strconv.ParseBool(os.Getenv(DisableEnv))
	envCacheDisabled := os.Getenv(DisableEnv)
	if envCacheDisabled == "" {
		envCacheDisabled = "0"
	}

	// We check if the environment variable to disable the cache is set
	cacheDisabled, err := strconv.ParseBool(envCacheDisabled)
	if err != nil {
		return nil, fmt.Errorf("failed to parse environment variable %s: %s", DisableEnv, err)
	}
	// If the cache is not already disabled, we check if the configuration that was passed in
	// request the cache to be disabled
	if ! cacheDisabled && cfg.Disable {
		h.disabled = true
	}
	// If the cache is disabled, we stop here. Basically we return a valid handle that is not fully initialized
	// since it would create the directories required by an enabled cache.
	if h.disabled {
		return h, nil
	}

	// cfg is what is requested so we should not change any value that it contains
	parentDir := cfg.ParentDir
	if parentDir == "" {
		parentDir = getCacheParentDir()
	}
	h.parentDir = parentDir

	ep, err := fs.FirstExistingParent(parentDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get first existing parent of cache directory: %v", err)
	}

	// We check if we can write to the basedir or its first existing parent,
	// if not we disable the caching mechanism
	if !fs.IsWritable(ep) {
		h.disabled = true
		return h, nil
	}

	// Initialize the root directory of the cache
	rootDir := path.Join(parentDir, SubDirName)
	h.rootDir = rootDir
	if err = initCacheDir(rootDir); err != nil {
		return nil, fmt.Errorf("failed initializing caching directory: %s", err)
	}
	// Initialize the subdirectories of the cache
	for _, ct := range FileCacheTypes {
		dir := h.GetCacheTypeDir(ct)
		if err = initCacheDir(dir); err != nil {
			return nil, fmt.Errorf("failed initializing caching directory: %s", err)
		}
	}


	return h, nil
}

// getCacheParentDir figures out where the parent directory of the cache is.
//
// Singularity makes the following assumptions:
// - the default location for caches is specified by RootDefault
// - a user can specify the environment variable specified by DirEnv to
//   change the location
// - a user can change the location of a cache at any time
// - but in the context of a Singularity command, the cache location
//   cannot change once the command starts executing
func getCacheParentDir() string {
	// If the user defined the special environment variable, we use its value
	// as base directory.
	parentDir := os.Getenv(DirEnv)
	if parentDir != "" {
		return parentDir
	}

	// If the environment variable is not set, we use the default cache.
	sylog.Debugf("environment variable %s not set, using default image cache", DirEnv)
	parentDir = syfs.ConfigDir()

	return parentDir
}

// IsDisabled returns true if the cache is disabled
func (h *Handle) IsDisabled() bool {
	return h.disabled
}

// updateCacheSubdir update/create a sub-cache (directory) within the cache,
// for example, the 'shub' cache.
func updateCacheSubdir(h *Handle, subdir string) (string, error) {
	// This function may act on an cache object that is not fully initialized
	// so it is not a method on a Handle but rather an independent
	// function.
	// Because the cache object may not be initialized, we do NOT check its validity
	if h == nil {
		return "", fmt.Errorf("invalid cache handle")
	}

	// If the subdir is empty, it will lead to new collision since it would
	// succeed but point at the cache's root
	if subdir == "" {
		return "", fmt.Errorf("invalid parameter")
	}

	absdir, err := filepath.Abs(filepath.Join(h.rootDir, subdir))
	if err != nil {
		sylog.Fatalf("Unable to get abs filepath: %v", err)
	}

	if err := initCacheDir(absdir); err != nil {
		sylog.Fatalf("Unable to initialize caching directory: %v", err)
	}

	sylog.Debugf("Caching directory set to %s", absdir)
	return absdir, nil
}

// Return the directory for a specific CacheType
func (h *Handle) GetCacheTypeDir(cacheType string) string {
	return path.Join(h.rootDir, cacheType)
}

// cleanAllCaches is an utility function that wipes all files in the
// cache directory, will return a error if one occurs
func (h *Handle) cleanAllCaches() {
	if h.disabled {
		return
	}

	for _, ct := range FileCacheTypes {
		dir := h.GetCacheTypeDir(ct)
		if err := os.RemoveAll(dir); err != nil {
			sylog.Verbosef("unable to clean %s cache, directory %s: %v", ct, dir, err)
		}
	}
}

func initCacheDir(dir string) error {
	if fi, err := os.Stat(dir); os.IsNotExist(err) {
		sylog.Debugf("Creating cache directory: %s", dir)
		if err := fs.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("couldn't create cache directory %v: %v", dir, err)
		}
	} else if err != nil {
		return fmt.Errorf("unable to stat %s: %s", dir, err)
	} else if fi.Mode().Perm() != 0700 {
		// enforce permission on cache directory to prevent
		// potential information leak
		if err := os.Chmod(dir, 0700); err != nil {
			return fmt.Errorf("couldn't enforce permission 0700 on %s: %s", dir, err)
		}
	}

	return nil
}


