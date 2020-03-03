// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

var (
	errInvalidCacheHandle = errors.New("invalid cache handle")
)


// cleanCache cleans the given type of cache cacheType. It will return a
// error if one occurs.
func cleanCache(imgCache *cache.Handle, cacheType string, dryRun bool) error {
	if imgCache == nil {
		return fmt.Errorf("invalid image cache handle")
	}
	return imgCache.CleanCache(cacheType, dryRun)
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
func CleanSingularityCache(imgCache *cache.Handle, dryRun bool, cacheCleanTypes []string, cacheName []string) error {
	if imgCache == nil {
		return errInvalidCacheHandle
	}

	/*
	if len(cacheName) > 0 {


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
		return nil
	}

    */

	for _, cacheType := range cache.FileCacheTypes {
		sylog.Debugf("Cleaning %s cache...", cacheType)
		if err := cleanCache(imgCache, cacheType, dryRun); err != nil {
			return err
		}
	}

	return nil
}
