// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

// cleanCacheDir cleans the cache named name in the directory dir.
func cleanCacheDir(name, dir string, op func(string) error) error {
	sylog.Debugf("Removing: %v", dir)

	err := op(dir)
	if err != nil {
		// wrap the error in a user-friendly message
		err = fmt.Errorf("unable to clean %s cache: %v", name, err)
	}

	return err
}

func cleanLibraryCache(imgCache *cache.Handle, op func(string) error) error {
	return cleanCacheDir("library", imgCache.Library, op)
}

func cleanOciCache(imgCache *cache.Handle, op func(string) error) error {
	return cleanCacheDir("oci-tmp", imgCache.OciTemp, op)
}

func cleanBlobCache(imgCache *cache.Handle, op func(string) error) error {
	return cleanCacheDir("oci-blob", imgCache.OciBlob, op)
}

func cleanShubCache(imgCache *cache.Handle, op func(string) error) error {
	return cleanCacheDir("shub", imgCache.Shub, op)
}

func cleanNetCache(imgCache *cache.Handle, op func(string) error) error {
	return cleanCacheDir("net", imgCache.Net, op)
}

func cleanOrasCache(imgCache *cache.Handle, op func(string) error) error {
	return cleanCacheDir("oras", imgCache.Oras, op)
}

// cleanCache cleans the given type of cache cacheType. It will return a
// error if one occurs.
func cleanCache(imgCache *cache.Handle, cacheType string, op func(string) error) error {
	if imgCache == nil {
		return fmt.Errorf("invalid image cache handle")
	}

	switch cacheType {
	case "library":
		return cleanLibraryCache(imgCache, op)
	case "oci":
		return cleanOciCache(imgCache, op)
	case "shub":
		return cleanShubCache(imgCache, op)
	case "blob", "blobs":
		return cleanBlobCache(imgCache, op)
	case "net":
		return cleanNetCache(imgCache, op)
	case "oras":
		return cleanOrasCache(imgCache, op)
	default:
		// The caller checks the returned error and will exit as required
		return fmt.Errorf("not a valid type: %s", cacheType)
	}
}

func removeCacheEntry(name, cacheType, cacheDir string, op func(string) error) (bool, error) {
	foundMatch := false
	done := fmt.Errorf("done")
	err := filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			sylog.Debugf("Error while walking directory %s for cache %s at %s while looking for entry %s: %+v",
				cacheDir,
				cacheType,
				path,
				name,
				err)
			return err
		}
		if !info.IsDir() && info.Name() == name {
			sylog.Debugf("Removing entry %s from cache %s at %s", name, cacheType, path)
			if err := op(path); err != nil {
				return fmt.Errorf("unable to remove entry %s from cache %s at path %s: %v", name, cacheType, path, err)
			}
			foundMatch = true
			return done
		}
		return nil
	})
	if err == done {
		err = nil
	}
	return foundMatch, err
}

// CleanSingularityCache is the main function that drives all these
// other functions. If force is true, remove the entries, otherwise only
// provide a summary of what would have been done. If cacheCleanTypes
// contains something, only clean that type. The special value "all" is
// interpreted as "all types of entries". If cacheName contains
// something, clean only cache entries matching that name.
func CleanSingularityCache(imgCache *cache.Handle, force bool, cacheCleanTypes []string, cacheName []string) error {
	if imgCache == nil {
		return errInvalidCacheHandle
	}

	cacheTypes, err := normalizeCacheList(cacheCleanTypes)
	if err != nil {
		return err
	}

	op := func(path string) error {
		fmt.Printf("Would remove %s\n", path)
		return nil
	}

	if len(cacheName) > 0 {
		if force {
			op = os.Remove
		}

		// a name was specified, only clean matching entries
		for _, name := range cacheName {
			matches := 0
			for _, cacheType := range cacheTypes {
				cacheDir, _ := cacheTypeToDir(imgCache, cacheType)
				sylog.Debugf("Removing cache type %q with name %q from directory %q ...", cacheType, name, cacheDir)
				foundMatch, err := removeCacheEntry(name, cacheType, cacheDir, op)
				if err != nil {
					return err
				}
				if foundMatch {
					matches++
				}
			}

			if matches == 0 {
				sylog.Warningf("No cache found with given name: %s", name)
			}
		}
	} else {
		// no name specified, clean everything in the specified
		// cache types
		if force {
			op = os.RemoveAll
		}

		for _, cacheType := range cacheTypes {
			sylog.Debugf("Cleaning %s cache...", cacheType)
			if err := cleanCache(imgCache, cacheType, op); err != nil {
				return err
			}
		}
	}

	return nil
}
