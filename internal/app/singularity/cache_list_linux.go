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

func listLibraryCache() error {
	// loop through library cache
	libraryCacheFiles, err := ioutil.ReadDir(cache.Library())
	if err != nil {
		return fmt.Errorf("unable to opening cache folder: %v", err)
	}
	for _, f := range libraryCacheFiles {
		cont, err := ioutil.ReadDir(filepath.Join(cache.Library(), f.Name()))
		if err != nil {
			return fmt.Errorf("unable to looking in cache: %v", err)
		}
		for _, c := range cont {
			fileInfo, err := os.Stat(filepath.Join(cache.Library(), f.Name(), c.Name()))
			if err != nil {
				return fmt.Errorf("unable to get stat: %v", err)
			}
			printFileSize, err := findSize(fileInfo.Size())
			if err != nil {
				// no need to describe the error, since it is already
				sylog.Warningf("%v", err)
			}
			fmt.Printf("%-22s %-22s %-16s %s\n", c.Name(), fileInfo.ModTime().Format("2006-01-02 15:04:05"), printFileSize, "library")
		}
	}
	return nil
}

func listOciCache() error {
	// loop through oci-tmp cache
	ociTmp, err := ioutil.ReadDir(cache.OciTemp())
	if err != nil {
		return fmt.Errorf("while opening oci-tmp folder: %v", err)
	}
	for _, f := range ociTmp {
		blob, err := ioutil.ReadDir(filepath.Join(cache.OciTemp(), f.Name()))
		if err != nil {
			return fmt.Errorf("unable to looking in cache: %v", err)
		}
		for _, b := range blob {
			fileInfo, err := os.Stat(filepath.Join(cache.OciTemp(), f.Name(), b.Name()))
			if err != nil {
				return fmt.Errorf("unable to get stat: %v", err)
			}
			printFileSize, err := findSize(fileInfo.Size())
			if err != nil {
				// no need to describe the error, since it is already
				sylog.Warningf("%v", err)
			}
			fmt.Printf("%-22s %-22s %-16s %s\n", b.Name(), fileInfo.ModTime().Format("2006-01-02 15:04:05"), printFileSize, "oci")
		}
	}
	return nil
}

func listBlobCache(printList bool) error {
	// loop through ociBlob cache
	count := 0
	var totalSize int64

	_, err := os.Stat(filepath.Join(cache.OciBlob(), "/blobs"))
	if os.IsNotExist(err) {
		return nil
	}
	blobs, err := ioutil.ReadDir(filepath.Join(cache.OciBlob(), "/blobs/"))
	if err != nil {
		return fmt.Errorf("unable to opening oci folder: %v", err)
	}
	for _, f := range blobs {
		blob, err := ioutil.ReadDir(filepath.Join(cache.OciBlob(), "/blobs/", f.Name()))
		if err != nil {
			return fmt.Errorf("unable to looking in cache: %v", err)
		}
		for _, b := range blob {
			fileInfo, err := os.Stat(filepath.Join(cache.OciBlob(), "/blobs/", f.Name(), b.Name()))
			if err != nil {
				return fmt.Errorf("unable to get stat: %v", err)
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
	if printList != true && count >= 1 {
		printFileSize, err := findSize(totalSize)
		if err != nil {
			// no need to describe the error, since it is already
			sylog.Warningf("%v", err)
		}
		fmt.Printf("\nThere are %d oci blob file(s) using %v of space. Use: '-T=blob' to list\n", count, printFileSize)
	}
	return nil
}

// ListSingularityCache : list local singularity cache, typeNameList : is a string of what cache
// to list (seprate each type with a comma; like this: library,oci,blob) allList : force list all cache.
func ListSingularityCache(cacheListTypes []string, listAll bool) error {
	libraryList := false
	ociList := false
	blobList := false
	listBlobSum := false

	for _, t := range cacheListTypes {
		switch t {
		case "library":
			libraryList = true
		case "oci":
			ociList = true
		case "blob", "blobs":
			blobList = true
		case "blobSum":
			listBlobSum = true
		case "all":
			listAll = true
		default:
			sylog.Fatalf("Not a valid type: %v", t)
			os.Exit(2)
		}
	}

	fmt.Printf("%-22s %-22s %-16s %s\n", "NAME", "DATE CREATED", "SIZE", "TYPE")

	if libraryList || listAll {
		if err := listLibraryCache(); err != nil {
			return err
		}
	}
	if ociList || listAll {
		if err := listOciCache(); err != nil {
			return err
		}
	}
	if blobList || listAll {
		if err := listBlobCache(true); err != nil {
			return err
		}
		// dont list blob summary after listing all blobs
		listBlobSum = false
	}
	if listBlobSum {
		if err := listBlobCache(false); err != nil {
			return err
		}
	}
	return nil
}
