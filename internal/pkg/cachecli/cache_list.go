// Copyright (c) 2017-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cachecli

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

func join(strs ...string) string {
	var sb strings.Builder
	for _, str := range strs {
		sb.WriteString(str)
	}
	return sb.String()
}

func findSize(size int64) string {
	var sizeF float64
	if size <= 10000 {
		sizeF = float64(size) / 1000
		return join(fmt.Sprintf("%.2f", sizeF), " Kb")
	} else if size <= 1000000000 {
		sizeF = float64(size) / 1000000
		return join(fmt.Sprintf("%.2f", sizeF), " Mb")
	} else if size >= 1000000000 {
		sizeF = float64(size) / 1000000000
		return join(fmt.Sprintf("%.2f", sizeF), " Gb")
	}
	return "ERROR: failed to detect file size."
}

func listLibraryCache() {
	// loop thrught library cache
	libraryCacheFiles, err := ioutil.ReadDir(cache.Library())
	if err != nil {
		sylog.Fatalf("Failed while opening cache folder: %v", err)
		os.Exit(255)
	}
	for _, f := range libraryCacheFiles {
		cont, err := ioutil.ReadDir(join(cache.Library(), "/", f.Name()))
		if err != nil {
			sylog.Fatalf("Failed while looking in cache: %v", err)
			os.Exit(255)
		}
		for _, c := range cont {
			fileInfo, err := os.Stat(join(cache.Library(), "/", f.Name(), "/", c.Name()))
			if err != nil {
				sylog.Fatalf("Unable to get stat: %v", err)
				os.Exit(255)
			}
			fmt.Printf("%-22s %-22s %-16s %s\n", c.Name(), fileInfo.ModTime().Format("2006-01-02 15:04:05"), findSize(fileInfo.Size()), "library")
		}
	}
	return
}

func listOciCache() {
	// loop thrught oci-tmp cache
	ociTmp, err := ioutil.ReadDir(cache.OciTemp())
	if err != nil {
		sylog.Fatalf("Failed while opening oci-tmp folder: %v", err)
		os.Exit(255)
	}
	for _, f := range ociTmp {
		blob, err := ioutil.ReadDir(join(cache.OciTemp(), "/", f.Name()))
		if err != nil {
			sylog.Fatalf("Failed while looking in cache: %v", err)
			os.Exit(255)
		}
		for _, b := range blob {
			fileInfo, err := os.Stat(join(cache.OciTemp(), "/", f.Name(), "/", b.Name()))
			if err != nil {
				sylog.Fatalf("Unable to get stat: %v", err)
				os.Exit(255)
			}
			fmt.Printf("%-22s %-22s %-16s %s\n", b.Name(), fileInfo.ModTime().Format("2006-01-02 15:04:05"), findSize(fileInfo.Size()), "oci")
		}
	}
	return
}

func listBlobCache(printList bool) {
	// loop thrught ociBlob cache
	count := 0
	var totalSize int64

	_, err = os.Stat(join(cache.OciBlob(), "/blobs"))
	if os.IsNotExist(err) {
		return
	}
	blobs, err := ioutil.ReadDir(join(cache.OciBlob(), "/blobs/"))
	if err != nil {
		sylog.Fatalf("Failed while opening oci folder: %v", err)
		os.Exit(255)
	}
	for _, f := range blobs {
		blob, err := ioutil.ReadDir(join(cache.OciBlob(), "/blobs/", f.Name()))
		if err != nil {
			sylog.Fatalf("Failed while looking in cache: %v", err)
			os.Exit(255)
		}
		for _, b := range blob {
			fileInfo, err := os.Stat(join(cache.OciBlob(), "/blobs/", f.Name(), "/", b.Name()))
			if err != nil {
				sylog.Fatalf("Unable to get stat: %v", err)
				os.Exit(255)
			}
			if printList == true {
				fmt.Printf("%-22.20s %-22s %-16s %s\n", b.Name(), fileInfo.ModTime().Format("2006-01-02 15:04:05"), findSize(fileInfo.Size()), "blob")
			}
			count++
			totalSize += fileInfo.Size()
		}
	}
	if printList != true && count >= 1 {
		fmt.Printf("\nThere are: %d blob file(s) using: %v of space, use: -t=blob to list\n", count, findSize(totalSize))
	}
	return
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
			if nameType == "library" {
				libraryList = true
			} else if nameType == "oci" {
				ociList = true
			} else if nameType == "blob" || nameType == "blobs" {
				blobList = true
			} else if nameType == "all" {
				allList = true
			} else {
				sylog.Fatalf("Not a valid type: %v", typeNameList)
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
		listLibraryCache()
		listOciCache()
		listBlobCache(true)
		return nil
	}

	if libraryList == true {
		listLibraryCache()
	}
	if ociList == true {
		listOciCache()
	}
	if blobList == true {
		listBlobCache(true)
	}
	if listBlobSum == true {
		listBlobCache(false)
	}
	return nil
}
