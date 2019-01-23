// Copyright (c) 2017-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cacheCli

import (
	"fmt"
	"strings"
	"io/ioutil"
	"os"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/client/cache"

)

func join(strs ...string) string {
	var sb strings.Builder
	for _, str := range strs {
		sb.WriteString(str)
	}
	return sb.String()
}

func find_size(size int64) string {
	var size_f float64
	if size <= 10000 {
		size_f = float64(size) / 1000
		return join(fmt.Sprintf("%.2f", size_f), " Kb")
	} else if size <= 1000000000 {
		size_f = float64(size) / 1000000
		return join(fmt.Sprintf("%.2f", size_f), " Mb")
	} else if size >= 1000000000 {
		size_f = float64(size) / 1000000000
		return join(fmt.Sprintf("%.2f", size_f), " Gb")
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
			fmt.Printf("%-22s %-22s %-16s %s\n", c.Name(), fileInfo.ModTime().Format("2006-01-02 15:04:05"), find_size(fileInfo.Size()), "Library")
		}
	}
	return
}

func listOciCache() {
	// loop thrught oci-tmp cache
	blobs, err := ioutil.ReadDir(cache.OciTemp())
	if err != nil {
		sylog.Fatalf("Failed while opening oci-tmp folder: %v", err)
		os.Exit(255)
	}
	for _, f := range blobs {
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
			fmt.Printf("%-22s %-22s %-16s %s\n", b.Name(), fileInfo.ModTime().Format("2006-01-02 15:04:05"), find_size(fileInfo.Size()), "Oci Tmp")
		}
	}
	return
}

func ListSingularityCache(libraryList, ociList bool) error {
	sylog.Debugf("Starting list...")

	fmt.Printf("%-22s %-22s %-16s %s\n", "NAME", "DATE CREATED", "SIZE", "TYPE")

	if libraryList == true {
		listLibraryCache()
	}
	if ociList == true {
		listOciCache()
	}
	if libraryList != true && ociList != true {
		listLibraryCache()
		listOciCache()
	}

	return nil
}
