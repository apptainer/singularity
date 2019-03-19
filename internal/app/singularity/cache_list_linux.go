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

// findSize will take a int64 (like: '4781321234') and convert it to human readable output
// in Kb, Mb, and Gb. output will be a string.
// TODO: I'm sure theres a default 'human readable' function
func findSize(size int64) (string, error) {
	var sizeF float64
	if size <= 1000000 {
		sizeF = float64(size) / 1000
		return strings.Join([]string{fmt.Sprintf("%.2f", sizeF), " Kb"}, ""), nil
	} else if size <= 1000000000 {
		sizeF = float64(size) / 1000000
		return strings.Join([]string{fmt.Sprintf("%.2f", sizeF), " Mb"}, ""), nil
	} else if size >= 1000000000 {
		sizeF = float64(size) / 1000000000
		return strings.Join([]string{fmt.Sprintf("%.2f", sizeF), " Gb"}, ""), nil
	}
	return "", fmt.Errorf("failed to detect file size")
}

// listLibraryCache will loop throught and list all library cache (~/.singularity/cache/library).
// will return the amount of library containers, the total space thoughts containers are using,
// and an error if one occures.
func listLibraryCache(listFiles bool) (int, int64, error) {
	var totalSize int64
	count := 0

	libraryCacheFiles, err := ioutil.ReadDir(cache.Library())
	if err != nil {
		return 0, 0, fmt.Errorf("unable to open library cache folder: %v", err)
	}
	for _, f := range libraryCacheFiles {
		cont, err := ioutil.ReadDir(filepath.Join(cache.Library(), f.Name()))
		if err != nil {
			return 0, 0, fmt.Errorf("unable to look in library cache: %v", err)
		}
		for _, c := range cont {
			fileInfo, err := os.Stat(filepath.Join(cache.Library(), f.Name(), c.Name()))
			if err != nil {
				return 0, 0, fmt.Errorf("unable to get stat for library cache: %v", err)
			}
			printFileSize, err := findSize(fileInfo.Size())
			if err != nil {
				// no need to describe the error, since it is already
				sylog.Warningf("%v", err)
			}
			if !listFiles {
				fmt.Printf("%-22s %-22s %-16s %s\n", c.Name(), fileInfo.ModTime().Format("2006-01-02 15:04:05"), printFileSize, "library")
			}
			count++
			totalSize += fileInfo.Size()
		}
	}
	return count, totalSize, nil
}

// listOciCache will list all you oci-tmp cache.
func listOciCache(listFiles bool) (int, int64, error) {
	var totalSize int64
	count := 0

	ociTmp, err := ioutil.ReadDir(cache.OciTemp())
	if err != nil {
		return 0, 0, fmt.Errorf("unable to open oci-tmp folder: %v", err)
	}
	for _, f := range ociTmp {
		blob, err := ioutil.ReadDir(filepath.Join(cache.OciTemp(), f.Name()))
		if err != nil {
			return 0, 0, fmt.Errorf("unable to look in oci-tmp cache: %v", err)
		}
		for _, b := range blob {
			fileInfo, err := os.Stat(filepath.Join(cache.OciTemp(), f.Name(), b.Name()))
			if err != nil {
				return 0, 0, fmt.Errorf("unable to get stat for oci-tmp cache: %v", err)
			}
			printFileSize, err := findSize(fileInfo.Size())
			if err != nil {
				// no need to describe the error, since it is already
				sylog.Warningf("%v", err)
			}
			if !listFiles {
				fmt.Printf("%-22s %-22s %-16s %s\n", b.Name(), fileInfo.ModTime().Format("2006-01-02 15:04:05"), printFileSize, "oci")
			}
			count++
			totalSize += fileInfo.Size()
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
		return 0, 0, fmt.Errorf("unable to open oci-blob folder: %v", err)
	}
	for _, f := range blobs {
		blob, err := ioutil.ReadDir(filepath.Join(cache.OciBlob(), "/blobs/", f.Name()))
		if err != nil {
			return 0, 0, fmt.Errorf("unable to look in oci-blob cache: %v", err)
		}
		for _, b := range blob {
			fileInfo, err := os.Stat(filepath.Join(cache.OciBlob(), "/blobs/", f.Name(), b.Name()))
			if err != nil {
				return 0, 0, fmt.Errorf("unable to get stat for oci-blob cache: %v", err)
			}
			if printList == true {
				printFileSize, err := findSize(fileInfo.Size())
				if err != nil {
					// no need to describe the error, since it is already
					sylog.Warningf("%v", err)
				}
				fmt.Printf("%-22.20s %-22s %-16s %s\n", b.Name(), fileInfo.ModTime().Format("2006-01-02 15:04:05"), printFileSize, "blob")
			}
			count++
			totalSize += fileInfo.Size()
		}
	}
	return count, totalSize, nil
}

// ListSingularityCache : list local singularity cache, typeNameList : is a string of what cache
// to list (seprate each type with a comma; like this: library,oci,blob) allList : force list all cache.
func ListSingularityCache(cacheListTypes []string, listAll, cacheListSummery bool) error {
	libraryList := false
	ociList := false
	blobList := false
	//listBlobSum := false

	for _, t := range cacheListTypes {
		switch t {
		case "library":
			libraryList = true
		case "oci":
			ociList = true
		case "blob", "blobs":
			blobList = true
		case "all":
			listAll = true
		default:
			sylog.Fatalf("Not a valid type: %v", t)
			os.Exit(2)
		}
	}

	var containerCount int
	var containerSpace int64
	var blobCount int
	var blobSpace int64
	var totalSpace int64

	if !cacheListSummery {
		fmt.Printf("%-22s %-22s %-16s %s\n", "NAME", "DATE CREATED", "SIZE", "TYPE")
	}

	if libraryList || listAll {
		libraryCount, librarySize, err := listLibraryCache(cacheListSummery)
		if err != nil {
			return err
		}
		containerCount += libraryCount
		containerSpace += librarySize
	}
	if ociList || listAll {
		ociCount, ociSize, err := listOciCache(cacheListSummery)
		if err != nil {
			return err
		}
		containerCount += ociCount
		containerSpace += ociSize
	}
	if blobList || listAll {
		blobsCount, blobsSize, err := listBlobCache(true)
		if err != nil {
			return err
		}
		blobCount = blobsCount
		blobSpace = blobsSize
	} else {
		blobsCount, blobsSize, err := listBlobCache(false)
		if err != nil {
			return err
		}
		blobCount = blobsCount
		blobSpace = blobsSize
	}

	if !listAll || cacheListSummery {
		totalSpace = containerSpace + blobSpace
		realTotalSpace, err := findSize(totalSpace)
		if err != nil {
			return err
		}
		realContainerSpace, _ := findSize(containerSpace)
		realBlobSpace, _ := findSize(blobSpace)
		fmt.Printf("\nThere %v containers using: %v, %v oci blob file(s) using %v of space.\n", containerCount, realContainerSpace, blobCount, realBlobSpace)
		fmt.Printf("Total space used: %v\n", realTotalSpace)
	}

	return nil
}
