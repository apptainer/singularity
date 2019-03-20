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

// findSize takes a size in bytes and converts it to a human-readable string representation
// expressing kB, MB or GB (whatever is smaller, but still larger than one).
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

// listLibraryCache will loop through and list all library cache (~/.singularity/cache/library).
// will return the amount of library containers, the total space thoughts containers are using,
// and an error if one occurs.
func listLibraryCache(listFiles bool) (int, int64, error) {
	var totalSize int64
	count := 0

	libraryCacheFiles, err := ioutil.ReadDir(cache.Library())
	if err != nil {
		return 0, 0, fmt.Errorf("unable to open library cache directory: %v", err)
	}
	for _, f := range libraryCacheFiles {
		checkStat, err := os.Stat(filepath.Join(cache.Library(), f.Name()))
		if err != nil {
			return 0, 0, fmt.Errorf("unable to open stat on: %v: %v", filepath.Join(cache.Library(), f.Name()), err)
		}
		if checkStat.Mode().IsDir() {
			cont, err := ioutil.ReadDir(filepath.Join(cache.Library(), f.Name()))
			if err != nil {
				return 0, 0, fmt.Errorf("unable to look in library cache: %v", err)
			}
			for _, c := range cont {
				fileInfo, err := os.Stat(filepath.Join(cache.Library(), f.Name(), c.Name()))
				if err != nil {
					return 0, 0, fmt.Errorf("unable to get stat for library cache: %v", err)
				}
				if listFiles {
					fmt.Printf("%-22s %-22s %-16s %s\n", c.Name(), fileInfo.ModTime().Format("2006-01-02 15:04:05"), findSize(fileInfo.Size()), "library")
				}
				count++
				totalSize += fileInfo.Size()
			}
		} else {
			// stray file in ~/.singularity/cache/library
			sylog.Debugf("stray file in cache dir: %v", filepath.Join(cache.Library(), f.Name()))
		}
	}
	return count, totalSize, nil
}

// listOciCache lists all the entries in the OCI cache
func listOciCache(listFiles bool) (int, int64, error) {
	var totalSize int64
	count := 0

	ociTmp, err := ioutil.ReadDir(cache.OciTemp())
	if err != nil {
		return 0, 0, fmt.Errorf("unable to open oci-tmp directory: %v", err)
	}
	for _, f := range ociTmp {
		checkStat, err := os.Stat(filepath.Join(cache.OciTemp(), f.Name()))
		if err != nil {
			return 0, 0, fmt.Errorf("unable to open stat on: %v: %v", filepath.Join(cache.OciTemp(), f.Name()), err)
		}
		if checkStat.Mode().IsDir() {
			blob, err := ioutil.ReadDir(filepath.Join(cache.OciTemp(), f.Name()))
			if err != nil {
				return 0, 0, fmt.Errorf("unable to look in oci-tmp cache: %v", err)
			}
			for _, b := range blob {
				fileInfo, err := os.Stat(filepath.Join(cache.OciTemp(), f.Name(), b.Name()))
				if err != nil {
					return 0, 0, fmt.Errorf("unable to get stat for oci-tmp cache: %v", err)
				}
				if listFiles {
					fmt.Printf("%-22s %-22s %-16s %s\n", b.Name(), fileInfo.ModTime().Format("2006-01-02 15:04:05"), findSize(fileInfo.Size()), "oci")
				}
				count++
				totalSize += fileInfo.Size()
			}
		} else {
			// stray file in ~/.singularity/cache/library
			sylog.Debugf("stray file in cache dir: %v", filepath.Join(cache.Library(), f.Name()))
		}
	}
	return count, totalSize, nil
}

func listBlobCache(printList bool) (int, int64, error) {
	// loop through ociBlob cache
	count := 0
	var totalSize int64

	_, err := os.Stat(filepath.Join(cache.OciBlob(), "/blobs"))
	if os.IsNotExist(err) {
		return 0, 0, nil
	}
	blobs, err := ioutil.ReadDir(filepath.Join(cache.OciBlob(), "/blobs/"))
	if err != nil {
		return 0, 0, fmt.Errorf("unable to open oci-blob directory: %v", err)
	}
	for _, f := range blobs {
		checkStat, err := os.Stat(filepath.Join(cache.OciBlob(), "blobs", f.Name()))
		if err != nil {
			return 0, 0, fmt.Errorf("unable to open stat on: %v: %v", filepath.Join(cache.OciBlob(), "blobs", f.Name()), err)
		}
		if checkStat.Mode().IsDir() {
			blob, err := ioutil.ReadDir(filepath.Join(cache.OciBlob(), "/blobs/", f.Name()))
			if err != nil {
				return 0, 0, fmt.Errorf("unable to look in oci-blob cache: %v", err)
			}
			for _, b := range blob {
				fileInfo, err := os.Stat(filepath.Join(cache.OciBlob(), "/blobs/", f.Name(), b.Name()))
				if err != nil {
					return 0, 0, fmt.Errorf("unable to get stat for oci-blob cache: %v", err)
				}
				if printList {
					fmt.Printf("%-22.20s %-22s %-16s %s\n", b.Name(), fileInfo.ModTime().Format("2006-01-02 15:04:05"), findSize(fileInfo.Size()), "blob")
				}
				count++
				totalSize += fileInfo.Size()
			}
		} else {
			// stray file in ~/.singularity/cache/library
			sylog.Debugf("stray file in cache dir: %v", filepath.Join(cache.Library(), f.Name()))
		}
	}
	return count, totalSize, nil
}

// ListSingularityCache will list local singularity cache, typeNameList is a []string of what cache
// to list (seprate each type with a comma; like: library,oci,blob) allList force list all cache.
func ListSingularityCache(cacheListTypes []string, listAll, cacheListSummary bool) error {
	libraryList := false
	ociList := false
	blobList := false
	blobSum := false

	for _, t := range cacheListTypes {
		switch t {
		case "library":
			libraryList = true
		case "oci":
			ociList = true
		case "blob", "blobs":
			blobList = true
		case "blobSum":
			blobSum = true
		case "all":
			listAll = true
		case "":
		default:
			sylog.Fatalf("Not a valid type: %v", t)
			os.Exit(2)
		}
	}

	if listAll {
		libraryList = true
		ociList = true
		blobList = true
	}

	var containerCount int
	var containerSpace int64
	var blobCount int
	var blobSpace int64

	// this next part is very messy, but it ensures that the '--summary' flag will be
	// compatible with '--type=', and '--all' flag.

	if !cacheListSummary {
		fmt.Printf("%-22s %-22s %-16s %s\n", "NAME", "DATE CREATED", "SIZE", "TYPE")
	}

	if listAll {
		libraryCount, librarySize, err := listLibraryCache(true)
		if err != nil {
			return err
		}
		containerCount += libraryCount
		containerSpace += librarySize
	} else if libraryList {
		libraryCount, librarySize, err := listLibraryCache(!cacheListSummary)
		if err != nil {
			return err
		}
		containerCount += libraryCount
		containerSpace += librarySize
	}

	if listAll {
		ociCount, ociSize, err := listOciCache(true)
		if err != nil {
			return err
		}
		containerCount += ociCount
		containerSpace += ociSize
	} else if ociList {
		ociCount, ociSize, err := listOciCache(!cacheListSummary)
		if err != nil {
			return err
		}
		containerCount += ociCount
		containerSpace += ociSize
	}

	if listAll {
		blobsCount, blobsSize, err := listBlobCache(true)
		if err != nil {
			return err
		}
		blobCount = blobsCount
		blobSpace = blobsSize
	} else if blobSum {
		blobsCount, blobsSize, err := listBlobCache(false)
		if err != nil {
			return err
		}
		blobCount = blobsCount
		blobSpace = blobsSize
	} else if blobList {
		blobsCount, blobsSize, err := listBlobCache(!cacheListSummary)
		if err != nil {
			return err
		}
		blobCount = blobsCount
		blobSpace = blobsSize
	}

	if !listAll || cacheListSummary {
		fmt.Printf("\nThere %d containers using: %v, %d oci blob file(s) using %v of space.\n", containerCount, findSize(containerSpace), blobCount, findSize(blobSpace))
		fmt.Printf("Total space used: %v\n", findSize(containerSpace+blobSpace))
	}

	return nil
}
