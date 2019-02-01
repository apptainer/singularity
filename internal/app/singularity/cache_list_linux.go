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
	// loop thrught library cache
	libraryCacheFiles, err := ioutil.ReadDir(cache.Library())
	if err != nil {
		return fmt.Errorf("Unable to opening cache folder: %v", err)
	}
	for _, f := range libraryCacheFiles {
		cont, err := ioutil.ReadDir(filepath.Join(cache.Library(), f.Name()))
		if err != nil {
			return fmt.Errorf("Unable to looking in cache: %v", err)
		}
		for _, c := range cont {
			fileInfo, err := os.Stat(filepath.Join(cache.Library(), f.Name(), c.Name()))
			if err != nil {
				return fmt.Errorf("Unable to get stat: %v", err)
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
	// loop thrught oci-tmp cache
	ociTmp, err := ioutil.ReadDir(cache.OciTemp())
	if err != nil {
		return fmt.Errorf("while opening oci-tmp folder: %v", err)
	}
	for _, f := range ociTmp {
		blob, err := ioutil.ReadDir(filepath.Join(cache.OciTemp(), f.Name()))
		if err != nil {
			return fmt.Errorf("Unable to looking in cache: %v", err)
		}
		for _, b := range blob {
			fileInfo, err := os.Stat(filepath.Join(cache.OciTemp(), f.Name(), b.Name()))
			if err != nil {
				return fmt.Errorf("Unable to get stat: %v", err)
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
	// loop thrught ociBlob cache
	count := 0
	var totalSize int64

	_, err = os.Stat(filepath.Join(cache.OciBlob(), "/blobs"))
	if os.IsNotExist(err) {
		return nil
	}
	blobs, err := ioutil.ReadDir(filepath.Join(cache.OciBlob(), "/blobs/"))
	if err != nil {
		return fmt.Errorf("Unable to opening oci folder: %v", err)
	}
	for _, f := range blobs {
		blob, err := ioutil.ReadDir(filepath.Join(cache.OciBlob(), "/blobs/", f.Name()))
		if err != nil {
			return fmt.Errorf("Unable to looking in cache: %v", err)
		}
		for _, b := range blob {
			fileInfo, err := os.Stat(filepath.Join(cache.OciBlob(), "/blobs/", f.Name(), b.Name()))
			if err != nil {
				return fmt.Errorf("Unable to get stat: %v", err)
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
func ListSingularityCache(typeNameList string, allList bool) error {
	libraryList := false
	ociList := false
	blobList := false
	listBlobSum := false

	// split the string for each `,` then loop throught it and find what flags are there.
	// then see whats true/false later. heres the benefit of doing it like this; if the user
	// specified `library` twice, it will still only be printed once.
	if len(typeNameList) >= 1 {
		for _, nameType := range strings.Split(typeNameList, ",") {
			switch nameType {
			case "library":
				libraryList = true
			case "oci":
				ociList = true
			case "blob", "blobs":
				blobList = true
			case "all":
				allList = true
			default:
				sylog.Fatalf("Not a valid type: %v", nameType)
				os.Exit(2)
			}
		}
	} else {
		libraryList = true
		ociList = true
		listBlobSum = true
	}

	fmt.Printf("%-22s %-22s %-16s %s\n", "NAME", "DATE CREATED", "SIZE", "TYPE")

	if allList == true {
		err = listLibraryCache()
		if err != nil {
			return err
		}
		err = listOciCache()
		if err != nil {
			return err
		}
		err = listBlobCache(true)
		if err != nil {
			return err
		}
		return nil
	}
	if libraryList == true {
		err = listLibraryCache()
		if err != nil {
			return err
		}
	}
	if ociList == true {
		err = listOciCache()
		if err != nil {
			return err
		}
	}
	if blobList == true {
		err = listBlobCache(true)
		if err != nil {
			return err
		}
	}
	if listBlobSum == true {
		err = listBlobCache(false)
		if err != nil {
			return err
		}
	}
	return nil
}
