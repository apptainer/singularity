// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

func cleanLibraryCacheName(cacheName string) (bool, error) {
	foundMatch := false
	cacheHdl, err := cache.NewHandle()
	if cacheHdl == nil || err != nil {
		return false, fmt.Errorf("unable to create new cache handle")
	}
	libraryCacheFiles, err := ioutil.ReadDir(cacheHdl.Library)
	if err != nil {
		return false, fmt.Errorf("unable to opening library cache folder: %v", err)
	}
	for _, f := range libraryCacheFiles {
		cont, err := ioutil.ReadDir(filepath.Join(cacheHdl.Library, f.Name()))
		if err != nil {
			return false, fmt.Errorf("unable to look in library cache folder: %v", err)
		}
		for _, c := range cont {
			if c.Name() == cacheName {
				sylog.Debugf("Removing: %v", filepath.Join(cacheHdl.Library, f.Name(), c.Name()))
				err = os.RemoveAll(filepath.Join(cacheHdl.Library, f.Name(), c.Name()))
				if err != nil {
					return false, fmt.Errorf("unable to remove library cache: %v", err)
				}
				foundMatch = true
			}
		}
	}
	return foundMatch, nil
}

func cleanOciCacheName(cacheName string) (bool, error) {
	foundMatch := false

	c, err := cache.NewHandle()
	if c == nil || err != nil {
		return false, fmt.Errorf("unable to create new cache handle")
	}

	blobs, err := ioutil.ReadDir(c.OciTemp)
	if err != nil {
		return false, fmt.Errorf("unable to opening oci-tmp cache folder: %v", err)
	}
	for _, f := range blobs {
		blob, err := ioutil.ReadDir(filepath.Join(c.OciTemp, f.Name()))
		if err != nil {
			return false, fmt.Errorf("unable to look in oci-tmp cache folder: %v", err)
		}
		for _, b := range blob {
			if b.Name() == cacheName {
				sylog.Debugf("Removing: %v", filepath.Join(c.OciTemp, f.Name(), b.Name()))
				err = os.RemoveAll(filepath.Join(c.OciTemp, f.Name(), b.Name()))
				if err != nil {
					return false, fmt.Errorf("unable to remove oci-tmp cache: %v", err)
				}
				foundMatch = true
			}
		}
	}
	return foundMatch, nil
}

// CleanCacheName : will clean a container with the same name as cacheName (in the cache directory).
// if libraryCache is true; only search thrught library cache. if ociCache is true; only search the
// oci-tmp cache. if both are false; search all cache, and if both are true; again, search all cache.
func CleanCacheName(cacheName string, libraryCache, ociCache bool) (bool, error) {
	if libraryCache == ociCache {
		matchLibrary, err := cleanLibraryCacheName(cacheName)
		if err != nil {
			return false, err
		}
		matchOci, err := cleanOciCacheName(cacheName)
		if err != nil {
			return false, err
		}
		if matchLibrary || matchOci {
			return true, nil
		}
		return false, nil
	}

	match := false
	if libraryCache {
		match, err := cleanLibraryCacheName(cacheName)
		if err != nil {
			return false, err
		}
		return match, nil
	} else if ociCache {
		match, err := cleanOciCacheName(cacheName)
		if err != nil {
			return false, err
		}
		return match, nil
	}
	return match, nil
}

// CleanSingularityCache : the main function that drives all these other functions, if allClean is true; clean
// all cache. if typeNameClean contains somthing; only clean that type. if cacheName contains somthing; clean only
// cache with that name.
func CleanSingularityCache(cleanAll bool, cacheCleanTypes []string, cacheName string) error {
	libraryClean := false
	ociClean := false
	blobClean := false

	c, err := cache.NewHandle()
	if c == nil || err != nil {
		return fmt.Errorf("failed to create new cache handle")
	}

	for _, t := range cacheCleanTypes {
		switch t {
		case "library":
			libraryClean = true
		case "oci":
			ociClean = true
		case "blob", "blobs":
			blobClean = true
		case "all":
			cleanAll = true
		default:
			// The caller checks the returned error and exit when appropriate
			return fmt.Errorf("not a valid type: %s", t)
		}
	}

	if len(cacheName) >= 1 && !cleanAll {
		foundMatch, err := CleanCacheName(cacheName, libraryClean, ociClean)
		if err != nil {
			return err
		}
		if !foundMatch {
			sylog.Warningf("No cache found with given name: %s", cacheName)
		}
		return nil
	}

	if cleanAll {
		if err := c.Clean("all"); err != nil {
			return err
		}
	}

	if libraryClean {
		if err := c.Clean("library"); err != nil {
			return err
		}
	}
	if ociClean {
		if err := c.Clean("oci"); err != nil {
			return err
		}
	}
	if blobClean {
		if err := c.Clean("blob"); err != nil {
			return err
		}
	}
	return nil
}
