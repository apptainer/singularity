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

func cleanLibraryCache(imgCache *cache.Handle) error {
	if imgCache == nil {
		return fmt.Errorf("invalid image cache handle")
	}

	return cleanCacheDir("library", imgCache.Library)
}

func cleanOciCache(imgCache *cache.Handle) error {
	if imgCache == nil {
		return fmt.Errorf("invalid image cache handle")
	}
	return cleanCacheDir("oci-tmp", imgCache.OciTemp)
}

func cleanBlobCache(imgCache *cache.Handle) error {
	if imgCache == nil {
		return fmt.Errorf("invalid image cache handle")
	}
	return cleanCacheDir("oci-blob", imgCache.OciBlob)
}

func cleanShubCache(imgCache *cache.Handle) error {
	if imgCache == nil {
		return fmt.Errorf("invalid image cache handle")
	}
	return cleanCacheDir("shub", imgCache.Shub)
}

func cleanNetCache(imgCache *cache.Handle) error {
	if imgCache == nil {
		return fmt.Errorf("invalid image cache handle")
	}
	return cleanCacheDir("net", imgCache.Net)
}

func cleanOrasCache(imgCache *cache.Handle) error {
	if imgCache == nil {
		return fmt.Errorf("invalid image cache handle")
	}
	return cleanCacheDir("oras", imgCache.Oras)
}

// cleanCache cleans the given type of cache cacheType. It will return a
// error if one occurs.
func cleanCache(imgCache *cache.Handle, cacheType string) error {
	switch cacheType {
	case "library":
		return cleanLibraryCache(imgCache)
	case "oci":
		return cleanOciCache(imgCache)
	case "shub":
		return cleanShubCache(imgCache)
	case "blob", "blobs":
		return cleanBlobCache(imgCache)
	case "net":
		return cleanNetCache(imgCache)
	case "oras":
		return cleanOrasCache(imgCache)
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
func CleanSingularityCache(imgCache *cache.Handle, cleanAll bool, cacheCleanTypes []string, cacheName []string) error {
	imgCache, err := cache.NewHandle("")
	if err != nil {
		return fmt.Errorf("failed to create a new image cache handle: %s", err)
	}

	cacheDirs := map[string]string{
		"library": imgCache.Library,
		"oci":     imgCache.OciTemp,
		"shub":    imgCache.Shub,
		"blob":    imgCache.OciBlob,
		"net":     imgCache.Net,
		"oras":    imgCache.Oras,
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
		case "net":
			cacheTypes = append(cacheTypes, t)
		case "oras":
			cacheTypes = append(cacheTypes, t)
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
		cacheTypes = []string{"library", "oci", "shub", "blob", "net", "oras"}
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
			if err := cleanCache(imgCache, cacheType); err != nil {
				return err
			}
		}
	}

	return nil
}
