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

// CleanCache will clean a type of cache (cacheType string). will return a error if one occurs.
func CleanCache(cacheType []string) error {
	cacheToRemove := ""
	for _, c := range cacheType {
		switch c {
		case "library":
			cacheToRemove = cache.Library()
		case "oci":
			cacheToRemove = cache.OciTemp()
		case "shub":
			cacheToRemove = cache.Shub()
		case "blob", "blobs":
			cacheToRemove = cache.OciBlob()
		case "all":
			cacheToRemove = cache.Root()
		default:
			return fmt.Errorf("not a valid type: %s", c)
		}
		err := os.RemoveAll(cacheToRemove)
		if err != nil {
			return fmt.Errorf("unable to remove: %s: %v", cacheToRemove, err)
		}
	}
	return nil
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

func cleanShubCacheName(cacheName string) (bool, error) {
	foundMatch := false
	blobs, err := ioutil.ReadDir(cache.Shub())
	if err != nil {
		return false, fmt.Errorf("unable to open shub cache folder: %v", err)
	}
	for _, f := range blobs {
		blob, err := ioutil.ReadDir(filepath.Join(cache.Shub(), f.Name()))
		if err != nil {
			return false, fmt.Errorf("unable to look in shub cache folder: %v", err)
		}
		for _, b := range blob {
			if b.Name() == cacheName {
				sylog.Debugf("Removing: %v", filepath.Join(cache.Shub(), f.Name(), b.Name()))
				err = os.RemoveAll(filepath.Join(cache.Shub(), f.Name(), b.Name()))
				if err != nil {
					return false, fmt.Errorf("unable to remove shub cache: %v", err)
				}
				foundMatch = true
			}
		}
	}
	return foundMatch, nil
}

// CleanCacheName will clean a container with the same name as cacheName (in the cache directory).
// if libraryCache is true; only search thrught library cache. if ociCache is true; only search the
// oci-tmp cache. if both are false; search all cache, and if both are true; again, search all cache.
func CleanCacheName(cacheName string, libraryCache, ociCache, shubCache bool) (bool, error) {
	if libraryCache == ociCache && libraryCache == shubCache {
		matchLibrary, err := cleanLibraryCacheName(cacheName)
		if err != nil {
			return false, err
		}
		matchOci, err := cleanOciCacheName(cacheName)
		if err != nil {
			return false, err
		}
		matchShub, err := cleanShubCacheName(cacheName)
		if err != nil {
			return false, err
		}
		if matchLibrary || matchOci || matchShub {
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

// CleanSingularityCache is the main function that drives all these other functions, if allClean is true; clean
// all cache. if typeNameClean contains somthing; only clean that type. if cacheName contains somthing; clean only
// cache with that name.
func CleanSingularityCache(cleanAll bool, cacheCleanTypes []string, cacheName []string) error {
	libraryClean := false
	ociClean := false
	shubClean := false
	blobClean := false

	for _, t := range cacheCleanTypes {
		switch t {
		case "library":
			libraryClean = true
		case "oci":
			ociClean = true
		case "shub":
			shubClean = true
		case "blob", "blobs":
			blobClean = true
		case "all":
			cleanAll = true
		default:
			// The caller checks the returned error and exit when appropriate
			return fmt.Errorf("not a valid type: %s", t)
		}
	}

	if len(cacheName) >= 2 && !cleanAll {
		for _, n := range cacheName {
			foundMatch, err := CleanCacheName(n, libraryClean, ociClean, shubClean)
			if err != nil {
				return err
			}
			if !foundMatch {
				sylog.Warningf("No cache found with given name: %s", n)
			}
		}
		return nil
	}

	if cleanAll {
		if err := CleanCache([]string{"all"}); err != nil {
			return err
		}
	}

	if libraryClean {
		if err := CleanCache([]string{"library"}); err != nil {
			return err
		}
	}
	if ociClean {
		if err := CleanCache([]string{"oci"}); err != nil {
			return err
		}
	}
	if shubClean {
		if err := CleanCache([]string{"shub"}); err != nil {
			return err
		}
	}
	if blobClean {
		if err := CleanCache([]string{"blob"}); err != nil {
			return err
		}
	}
	return nil
}
