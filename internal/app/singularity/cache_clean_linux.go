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

// CleanLibraryCache : clean only library cache, will return a error if one occurs
func CleanLibraryCache() error {
	sylog.Debugf("Removing: %v", cache.Library())

	err := os.RemoveAll(cache.Library())
	if err != nil {
		return fmt.Errorf("Unable to clean cache: %v", err)
	}

	return nil
}

// CleanOciCache : clean only oci cache, will return a error if one occurs
func CleanOciCache() error {
	sylog.Debugf("Removing: %v", cache.OciTemp())

	err := os.RemoveAll(cache.OciTemp())
	if err != nil {
		return fmt.Errorf("Unable to clean cache: %v", err)
	}

	return nil
}

// CleanBlobCache : clean only blob cache, will return a error if one occurs
func CleanBlobCache() error {
	sylog.Debugf("Removing: %v", cache.OciBlob())

	err := os.RemoveAll(cache.OciBlob())
	if err != nil {
		return fmt.Errorf("Unable to clean cache: %v", err)
	}

	return nil

}

func cleanLibraryCache(cacheName string) (bool, error) {
	foundMatch := false
	libraryCacheFiles, err := ioutil.ReadDir(cache.Library())
	if err != nil {
		return false, fmt.Errorf("Unable to opening cache folder: %v", err)
	}
	for _, f := range libraryCacheFiles {
		cont, err := ioutil.ReadDir(filepath.Join(cache.Library(), f.Name()))
		if err != nil {
			return false, fmt.Errorf("Unable to look in cache folder: %v", err)
		}
		for _, c := range cont {
			if c.Name() == cacheName {
				sylog.Debugf("Removing: %v", filepath.Join(cache.Library(), f.Name(), c.Name()))
				err = os.RemoveAll(filepath.Join(cache.Library(), f.Name()))
				if err != nil {
					return false, fmt.Errorf("Unable to remove cache: %v", err)
				}
				foundMatch = true
			}
		}
	}

	return foundMatch, nil
}

func cleanOciCache(cacheName string) (bool, error) {
	foundMatch := false
	blobs, err := ioutil.ReadDir(cache.OciTemp())
	if err != nil {
		return false, fmt.Errorf("Unable to opening cache folder: %v", err)
	}
	for _, f := range blobs {
		blob, err := ioutil.ReadDir(filepath.Join(cache.OciTemp(), f.Name()))
		if err != nil {
			return false, fmt.Errorf("Unable to look in cache folder: %v", err)
		}
		for _, b := range blob {
			if b.Name() == cacheName {
				sylog.Debugf("Removing: %v", filepath.Join(cache.OciTemp(), f.Name(), b.Name()))
				err = os.RemoveAll(filepath.Join(cache.OciTemp(), f.Name()))
				if err != nil {
					return false, fmt.Errorf("Unable to remove cache: %v", err)
				}
				foundMatch = true
			}
		}
	}

	return foundMatch, nil
}

// CleanCacheName : will clean a container with the same name as cacheName (in the cache directory).
// if libraryCache == true; only search thrught library cache. if ociCache == true; only search the
// oci-tmp cache. if both are false; search all cache, and if both are true; again, search all cache.
func CleanCacheName(cacheName string, libraryCache, ociCache bool) (bool, error) {
	if libraryCache == ociCache {
		matchLibrary, err := cleanLibraryCache(cacheName)
		if err != nil {
			return false, err
		}
		matchOci, err := cleanOciCache(cacheName)
		if err != nil {
			return false, err
		}
		if matchLibrary == true || matchOci == true {
			return true, nil
		}
		return false, nil
	}

	match := false
	if libraryCache == true {
		match, err = cleanLibraryCache(cacheName)
		if err != nil {
			return false, err
		}
		return match, nil
	} else if ociCache == true {
		match, err = cleanOciCache(cacheName)
		if err != nil {
			return false, err
		}
		return match, nil
	}
	return false, nil
}

var err error

// CleanSingularityCache : the main function that drives all these other functions, if allClean == true; clean
// all cache. if typeNameClean contains somthing; only clean that type. if cacheName contains somthing; clean only
// cache with that name.
func CleanSingularityCache(cleanAll bool, cacheCleanTypes []string, cacheName string) error {
	libraryClean := false
	ociClean := false
	blobClean := false

	// split the string for each `,` then loop throught it and find what flags are there.
	// then see whats true/false later. heres the benefit of doing it like this; if the user
	// specified `library` twice, it will still only be printed once.
	if len(cacheCleanTypes) >= 1 {
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
				sylog.Fatalf("Not a valid type: %v", t)
				os.Exit(2)
			}
		}
	}

	if len(cacheName) >= 1 && cleanAll != true {
		foundMatch, err := CleanCacheName(cacheName, libraryClean, ociClean)
		if err != nil {
			return err
		}
		if foundMatch != true {
			sylog.Warningf("No cache found with givin name: %v", cacheName)
			os.Exit(0)
		}
		return nil
	}

	if cleanAll == true {
		err = cache.Clean()
		if err != nil {
			return err
		}
	}
	if libraryClean == true {
		err = CleanLibraryCache()
		if err != nil {
			return err
		}
	}
	if ociClean == true {
		err = CleanOciCache()
		if err != nil {
			return err
		}
	}
	if blobClean == true {
		err = CleanBlobCache()
		if err != nil {
			return err
		}
	}

	return nil
}
