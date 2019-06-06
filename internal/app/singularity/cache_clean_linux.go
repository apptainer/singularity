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

func cleanLibraryCache() error {
	sylog.Debugf("Removing: %v", cache.Library())

	err := os.RemoveAll(cache.Library())
	if err != nil {
		return fmt.Errorf("unable to clean library cache: %v", err)
	}

	return nil
}

func cleanOciCache() error {
	sylog.Debugf("Removing: %v", cache.OciTemp())

	err := os.RemoveAll(cache.OciTemp())
	if err != nil {
		return fmt.Errorf("unable to clean oci-tmp cache: %v", err)
	}

	return nil
}

func cleanBlobCache() error {
	sylog.Debugf("Removing: %v", cache.OciBlob())

	err := os.RemoveAll(cache.OciBlob())
	if err != nil {
		return fmt.Errorf("unable to clean oci-blob cache: %v", err)
	}

	return nil

}

// CleanCache : clean a type of cache (cacheType string). will return a error if one occurs.
func CleanCache(cacheType string) error {
	switch cacheType {
	case "library":
		err := cleanLibraryCache()
		return err
	case "oci":
		err := cleanOciCache()
		return err
	case "blob", "blobs":
		err := cleanBlobCache()
		return err
	default:
		// The caller checks the returned error and will exit as required
		return fmt.Errorf("not a valid type: %s", cacheType)
	}
}

func cleanLibraryCacheName(cacheName string) (bool, error) {
	foundMatch := false
	libraryCacheFiles, err := ioutil.ReadDir(cache.Library())
	if err != nil {
		return false, fmt.Errorf("unable to opening library cache folder: %v", err)
	}
	for _, f := range libraryCacheFiles {
		cont, err := ioutil.ReadDir(filepath.Join(cache.Library(), f.Name()))
		if err != nil {
			return false, fmt.Errorf("unable to look in library cache folder: %v", err)
		}
		for _, c := range cont {
			if c.Name() == cacheName {
				sylog.Debugf("Removing: %v", filepath.Join(cache.Library(), f.Name(), c.Name()))
				err = os.RemoveAll(filepath.Join(cache.Library(), f.Name(), c.Name()))
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
	blobs, err := ioutil.ReadDir(cache.OciTemp())
	if err != nil {
		return false, fmt.Errorf("unable to opening oci-tmp cache folder: %v", err)
	}
	for _, f := range blobs {
		blob, err := ioutil.ReadDir(filepath.Join(cache.OciTemp(), f.Name()))
		if err != nil {
			return false, fmt.Errorf("unable to look in oci-tmp cache folder: %v", err)
		}
		for _, b := range blob {
			if b.Name() == cacheName {
				sylog.Debugf("Removing: %v", filepath.Join(cache.OciTemp(), f.Name(), b.Name()))
				err = os.RemoveAll(filepath.Join(cache.OciTemp(), f.Name(), b.Name()))
				if err != nil {
					return false, fmt.Errorf("unable to remove oci-tmp cache: %v", err)
				}
				foundMatch = true
			}
		}
	}
	return foundMatch, nil
}

// cleanCacheEntry locates cache entries matching cacheName and removes
// them from the specified cache types.
func cleanCacheEntry(cacheName string, cacheTypes []string) (bool, error) {
	matches := 0

	for _, cache := range cacheTypes {
		var cleaner func(string) (bool, error)

		switch cache {
		case "library":
			cleaner = cleanLibraryCacheName

		case "oci":
			cleaner = cleanOciCacheName

		default:
			continue
		}

		if found, err := cleaner(cacheName); err != nil {
			return false, err
		} else if found {
			matches++
		}
	}

	return matches > 0, nil
}

// CleanSingularityCache : the main function that drives all these other functions, if allClean is true; clean
// all cache. if typeNameClean contains somthing; only clean that type. if cacheName contains somthing; clean only
// cache with that name.
func CleanSingularityCache(cacheCleanTypes []string, cacheName string) error {
	caches := []string{}

	for _, t := range cacheCleanTypes {
		switch t {
		case "blobs":
			t = "blob"
			fallthrough

		case "library", "oci", "blob":
			caches = append(caches, t)

		case "all":
			caches = []string{"library", "oci", "blob"}

		default:
			// The caller checks the returned error and exit when appropriate
			return fmt.Errorf("not a valid type: %s", t)
		}
	}

	if len(cacheName) > 0 {
		foundMatch, err := cleanCacheEntry(cacheName, caches)
		if err != nil {
			return err
		}
		if !foundMatch {
			sylog.Warningf("No cache found with given name: %s", cacheName)
		}
	} else {
		for _, cache := range caches {
			if err := CleanCache(cache); err != nil {
				return err
			}
		}
	}

	return nil
}
