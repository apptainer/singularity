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
func cleanCacheDir(name, dir string) error {
	sylog.Debugf("Removing: %v", dir)

	err := os.RemoveAll(dir)
	if err != nil {
		// wrap the error in a user-friendly message
		err = fmt.Errorf("unable to clean %s cache: %v", name, err)
	}

	return err
}

func cleanLibraryCache() error {
	return cleanCacheDir("library", cache.Library())
}

func cleanOciCache() error {
	return cleanCacheDir("oci-tmp", cache.OciTemp())
}

func cleanBlobCache() error {
	return cleanCacheDir("oci-blob", cache.OciBlob())
}

func cleanShubCache() error {
	return cleanCacheDir("shub", cache.Shub())
}

// cleanCache cleans the given type of cache cacheType. It will return a
// error if one occurs.
func cleanCache(cacheType string) error {
	switch cacheType {
	case "library":
		return cleanLibraryCache()
	case "oci":
		return cleanOciCache()
	case "shub":
		return cleanShubCache()
	case "blob", "blobs":
		return cleanBlobCache()
	default:
		// The caller checks the returned error and will exit as required
		return fmt.Errorf("not a valid type: %s", cacheType)
	}
}

func removeCacheEntry(name, cacheType, cacheDir string) (bool, error) {
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
			if err := os.Remove(path); err != nil {
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

// CleanSingularityCache is the main function that drives all these other functions, if cleanAll is true; clean
// all cache. if cacheCleanTypes contains somthing; only clean that type. if cacheName contains somthing; clean only
// cache with that name.
func CleanSingularityCache(cleanAll bool, cacheCleanTypes []string, cacheName []string) error {
	cacheDirs := map[string]string{
		"library": cache.Library(),
		"oci":     cache.OciTemp(),
		"shub":    cache.Shub(),
		"blob":    cache.OciBlob(),
	}
	cacheTypes := []string{}

	for _, t := range cacheCleanTypes {
		switch t {
		case "library":
			cacheTypes = append(cacheTypes, t)
		case "oci":
			cacheTypes = append(cacheTypes, t)
		case "shub":
			cacheTypes = append(cacheTypes, t)
		case "blob", "blobs":
			cacheTypes = append(cacheTypes, "blob")
		case "all":
			// cacheTypes contains "all", fall back to
			// cleaning all entries, but continue validating
			// entries just to be on the safe side
			cleanAll = true
		default:
			// The caller checks the returned error and exit when appropriate
			return fmt.Errorf("not a valid type: %s", t)
		}
	}

	if cleanAll {
		// cleanAll overrides all the specified names
		cacheTypes = []string{"library", "oci", "shub", "blob"}
	}

	if len(cacheName) > 0 {
		// a name was specified, only clean matching entries
		for _, name := range cacheName {
			matches := 0
			for _, cacheType := range cacheTypes {
				cacheDir := cacheDirs[cacheType]
				sylog.Debugf("Removing cache type %q with name %q from directory %q ...", cacheType, name, cacheDir)
				foundMatch, err := removeCacheEntry(name, cacheType, cacheDir)
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
		for _, cacheType := range cacheTypes {
			sylog.Debugf("Cleaning %s cache...", cacheType)
			if err := cleanCache(cacheType); err != nil {
				return err
			}
		}
	}

	return nil
}
