// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"errors"
	"fmt"

	"github.com/sylabs/singularity/internal/pkg/cache"
	"github.com/sylabs/singularity/pkg/sylog"
)

var (
	errInvalidCacheHandle = errors.New("invalid cache handle")
)

// cleanCache cleans the given type of cache cacheType. It will return a
// error if one occurs.
func cleanCache(imgCache *cache.Handle, cacheType string, dryRun bool, days int) error {
	if imgCache == nil {
		return fmt.Errorf("invalid image cache handle")
	}
	return imgCache.CleanCache(cacheType, dryRun, days)
}

// CleanSingularityCache is the main function that drives all these
// other functions. If force is true, remove the entries, otherwise only
// provide a summary of what would have been done. If cacheCleanTypes
// contains something, only clean that type. The special value "all" is
// interpreted as "all types of entries". If cacheName contains
// something, clean only cache entries matching that name.
func CleanSingularityCache(imgCache *cache.Handle, dryRun bool, cacheCleanTypes []string, days int) error {
	if imgCache == nil {
		return errInvalidCacheHandle
	}

	// Default is all caches
	cachesToClean := append(cache.OciCacheTypes, cache.FileCacheTypes...)

	// If specified caches, and we don't have 'all' specified then clean the specified
	// ones only.
	if len(cacheCleanTypes) > 0 && !stringInSlice("all", cacheCleanTypes) {
		cachesToClean = cacheCleanTypes
	}

	for _, cacheType := range cachesToClean {
		sylog.Debugf("Cleaning %s cache...", cacheType)
		if err := cleanCache(imgCache, cacheType, dryRun, days); err != nil {
			return err
		}
	}

	return nil
}
