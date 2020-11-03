// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/sylabs/singularity/internal/pkg/cache"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

// listTypeCache will list a cache type with given name (cacheType). The options are 'library', and 'oci'.
// Will return: the number of containers for that type (int), the total space the container type is using (int64),
// and an error if one occurs.
func listTypeCache(printList bool, name, cachePath string) (int, int64, error) {
	_, err := os.Stat(cachePath)
	if os.IsNotExist(err) {
		return 0, 0, nil
	} else if err != nil {
		return 0, 0, fmt.Errorf("unable to open cache %s at directory %s: %v", name, cachePath, err)
	}

	cacheEntries, err := ioutil.ReadDir(cachePath)
	if err != nil {
		return 0, 0, fmt.Errorf("unable to open cache %s at directory %s: %v", name, cachePath, err)
	}

	var (
		totalSize int64
	)

	for _, entry := range cacheEntries {

		if printList {
			fmt.Printf("%-24.22s %-22s %-16s %s\n",
				entry.Name(),
				entry.ModTime().Format("2006-01-02 15:04:05"),
				fs.FindSize(entry.Size()),
				name)
		}
		totalSize += entry.Size()
	}

	return len(cacheEntries), totalSize, nil
}

// ListSingularityCache will list the local singularity cache for the
// types specified by cacheListTypes. If cacheListTypes contains the
// value "all", all the cache entries are considered. If cacheListVerbose is
// true, the entries will be shown in the output, otherwise only a
// summary is provided.
func ListSingularityCache(imgCache *cache.Handle, cacheListTypes []string, cacheListVerbose bool) error {
	if imgCache == nil {
		return errInvalidCacheHandle
	}

	var (
		containerCount, blobCount             int
		containerSpace, blobSpace, totalSpace int64
	)

	if cacheListVerbose {
		fmt.Printf("%-24s %-22s %-16s %s\n", "NAME", "DATE CREATED", "SIZE", "TYPE")
	}

	containersShown := false
	blobsShown := false

	// If types requested includes "all" then we don't want to filter anything
	if stringInSlice("all", cacheListTypes) {
		cacheListTypes = []string{}
	}

	for _, cacheType := range cache.OciCacheTypes {
		// the type blob is special: 1. there's a
		// separate counter for it; 2. the cache entries
		// are actually one level deeper
		if len(cacheListTypes) > 0 && !stringInSlice(cacheType, cacheListTypes) {
			continue
		}
		cacheDir, err := imgCache.GetOciCacheDir(cacheType)
		if err != nil {
			return err
		}
		cacheDir = filepath.Join(cacheDir, "blobs", "sha256")
		blobsCount, blobsSize, err := listTypeCache(cacheListVerbose, cacheType, cacheDir)
		if err != nil {
			fmt.Print(err)
			return err
		}
		blobCount = blobsCount
		blobSpace = blobsSize
		totalSpace += blobsSize
		blobsShown = true
	}
	for _, cacheType := range cache.FileCacheTypes {
		if len(cacheListTypes) > 0 && !stringInSlice(cacheType, cacheListTypes) {
			continue
		}
		cacheDir, err := imgCache.GetFileCacheDir(cacheType)
		if err != nil {
			return err
		}
		count, size, err := listTypeCache(cacheListVerbose, cacheType, cacheDir)
		if err != nil {
			fmt.Print(err)
			return err
		}
		containerCount += count
		containerSpace += size
		totalSpace += size
		containersShown = true
	}

	if cacheListVerbose {
		fmt.Print("\n")
	}

	out := new(strings.Builder)
	out.WriteString("There are")
	if containersShown {
		fmt.Fprintf(out, " %d container file(s) using %s", containerCount, fs.FindSize(containerSpace))
	}
	if containersShown && blobsShown {
		fmt.Fprintf(out, " and")
	}
	if blobsShown {
		fmt.Fprintf(out, " %d oci blob file(s) using %s", blobCount, fs.FindSize(blobSpace))
	}
	out.WriteString(" of space\n")

	fmt.Print(out.String())
	fmt.Printf("Total space used: %s\n", fs.FindSize(totalSpace))

	return nil
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
