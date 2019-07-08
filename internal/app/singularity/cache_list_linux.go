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
	"strings"

	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

// findSize takes a size in bytes and converts it to a human-readable string representation
// expressing kB, MB, GB or TB (whatever is smaller, but still larger than one).
func findSize(size int64) string {

	var factor float64
	var unit string
	switch {
	case size < 1e6:
		factor = 1e3
		unit = "kB"
	case size < 1e9:
		factor = 1e6
		unit = "MB"
	case size < 1e12:
		factor = 1e9
		unit = "GB"
	default:
		factor = 1e12
		unit = "TB"
	}
	return fmt.Sprintf("%.2f %s", float64(size)/factor, unit)
}

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

	cacheDirs, err := ioutil.ReadDir(cachePath)
	if err != nil {
		return 0, 0, fmt.Errorf("unable to open cache %s at directory %s: %v", name, cachePath, err)
	}

	var (
		totalSize int64
		count     int
	)

	for _, dir := range cacheDirs {
		checkStat, err := os.Stat(filepath.Join(cachePath, dir.Name()))
		if err != nil {
			return 0, 0, fmt.Errorf("unable to open stat on: %v: %v", filepath.Join(cachePath, dir.Name()), err)
		}

		if !checkStat.Mode().IsDir() {
			// stray file in ~/.singularity/cache
			sylog.Debugf("stray file in cache dir: %v", filepath.Join(cachePath, dir.Name()))
			continue
		}

		cacheEntries, err := ioutil.ReadDir(filepath.Join(cachePath, dir.Name()))
		if err != nil {
			return 0, 0, fmt.Errorf("unable to look in: %s: %v", cachePath, err)
		}

		for _, entry := range cacheEntries {
			fileInfo, err := os.Stat(filepath.Join(cachePath, dir.Name(), entry.Name()))
			if err != nil {
				return 0, 0, fmt.Errorf("unable to get stat for: %s: %v", cachePath, err)
			}

			if printList {
				fmt.Printf("%-24.22s %-22s %-16s %s\n",
					entry.Name(),
					fileInfo.ModTime().Format("2006-01-02 15:04:05"),
					findSize(fileInfo.Size()),
					name)
			}
			totalSize += fileInfo.Size()
		}

		count += len(cacheEntries)
	}

	return count, totalSize, nil
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

	cacheTypes, err := normalizeCacheList(cacheListTypes)
	if err != nil {
		return err
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

	for _, cacheType := range cacheTypes {
		if cacheType == "blob" {
			// the type blob is special: 1. there's a
			// separate counter for it; 2. the cache entries
			// are actually one level deeper
			cacheDir, _ := cacheTypeToDir(imgCache, cacheType)
			cacheDir = filepath.Join(cacheDir, "blobs")
			blobsCount, blobsSize, err := listTypeCache(cacheListVerbose, cacheType, cacheDir)
			if err != nil {
				fmt.Print(err)
				return err
			}
			blobCount = blobsCount
			blobSpace = blobsSize
			totalSpace += blobsSize
			blobsShown = true
		} else {
			cacheDir, _ := cacheTypeToDir(imgCache, cacheType)
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
	}

	if cacheListVerbose {
		fmt.Print("\n")
	}

	out := new(strings.Builder)
	out.WriteString("There are")
	if containersShown {
		fmt.Fprintf(out, " %d container file(s) using %s", containerCount, findSize(containerSpace))
	}
	if containersShown && blobsShown {
		fmt.Fprintf(out, " and")
	}
	if blobsShown {
		fmt.Fprintf(out, " %d oci blob file(s) using %s", blobCount, findSize(blobSpace))
	}
	out.WriteString(" of space\n")

	fmt.Print(out.String())
	fmt.Printf("Total space used: %s\n", findSize(totalSpace))

	return nil
}
