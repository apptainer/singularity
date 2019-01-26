// Copyright (c) 2017-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cachecli

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

// CleanLibraryCache : clean only library cache, will return a error if one occurs
func CleanLibraryCache() error {
	sylog.Debugf("Removing: %v", cache.Library())

	err := os.RemoveAll(cache.Library())

	return err
}

// CleanOciCache : clean only oci cache, will return a error if one occurs
func CleanOciCache() error {
	sylog.Debugf("Removing: %v", cache.OciTemp())

	err := os.RemoveAll(cache.OciTemp())

	return err
}

// CleanBlobCache : clean only blob cache, will return a error if one occurs
func CleanBlobCache() error {
	sylog.Debugf("Removing: %v", cache.OciBlob())

	err := os.RemoveAll(cache.OciBlob())

	return err

}

func cleanLibraryCache(cacheName string) bool {
	foundMatch := false
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
			if c.Name() == cacheName {
				sylog.Debugf("Removing: %v", join(cache.Library(), "/", f.Name(), "/", c.Name()))
				err = os.RemoveAll(join(cache.Library(), "/", f.Name(), "/", c.Name()))
				sylog.Verbosef("Removed: %v", join(cache.Library(), "/", f.Name(), "/", c.Name()))
				if err != nil {
					sylog.Fatalf("Failed while removing cache file: %v: %v", join(cache.Library(), "/", f.Name(), "/", c.Name()), err)
					os.Exit(255)
				}
				foundMatch = true
			}
		}
	}

	return foundMatch
}

func cleanOciCache(cacheName string) bool {
	foundMatch := false
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
			if b.Name() == cacheName {
				sylog.Debugf("Removing: %v", join(cache.OciTemp(), "/", f.Name(), "/", b.Name()))
				err = os.RemoveAll(join(cache.OciTemp(), "/", f.Name(), "/", b.Name()))
				sylog.Verbosef("Removed: %v", join(cache.OciTemp(), "/", f.Name(), "/", b.Name()))
				if err != nil {
					sylog.Fatalf("Failed while removing: %v: %v", join(cache.OciTemp(), "/", f.Name(), "/", b.Name()), err)
					os.Exit(255)
				}
				foundMatch = true
			}
		}
	}

	return foundMatch
}

// CleanCacheName : clean a cache with a specific name (cacheName). if libraryCache == true; search only the
// library. if ociCache == true; search the oci cache. returns false if no match found.
func CleanCacheName(cacheName string, libraryCache, ociCache bool) bool {
	foundMatch := false
	if libraryCache == true {
		foundMatch = cleanLibraryCache(cacheName)
	}
	if ociCache == true {
		foundMatch = cleanOciCache(cacheName)
	}
	if libraryCache != true && ociCache != true {
		foundMatch = cleanLibraryCache(cacheName)
		foundMatch = cleanOciCache(cacheName)
	}
	return foundMatch
}

var err error

// CleanSingularityCache : the main function that drives all these other functions, if allClean == true; clean
// all cache. if typeNameClean contains somthing; only clean that type. if cacheName contains somthing; clean only
// cache with that name.
func CleanSingularityCache(allClean bool, typeNameClean, cacheName string) error {
	libraryClean := false
	ociClean := false
	blobClean := false

	// split the string for each `,` then loop throught it and find what flags are there.
	// then see whats true/false later. heres the benefit of doing it like this; if the user
	// specified `library` twice, it will still only be printed once.
	if len(typeNameClean) >= 1 {
		for _, nameType := range strings.Split(typeNameClean, ",") {
			if nameType == "library" {
				libraryClean = true
			} else if nameType == "oci" {
				ociClean = true
			} else if nameType == "blob" || nameType == "blobs" {
				blobClean = true
			} else if nameType == "all" {
				allClean = true
			} else {
				sylog.Fatalf("Not a valid type: %v", typeNameClean)
				os.Exit(2)
			}
		}
	} else {
		libraryClean = true
		ociClean = true
		blobClean = true
	}

	if len(cacheName) >= 1 && allClean != true {
		foundMatch := CleanCacheName(cacheName, libraryClean, ociClean)
		if foundMatch != true {
			sylog.Warningf("No cache found with givin name: %v", cacheName)
			os.Exit(0)
		}
		return nil
	} else if len(cacheName) >= 1 && allClean == true || len(typeNameClean) >= 1 && allClean == true {
		sylog.Fatalf("These flags are not compatible with each other")
		os.Exit(2)
	}

	if allClean == true {
		err = cache.Clean()
	}
	if libraryClean == true {
		err = CleanLibraryCache()
	}
	if ociClean == true {
		err = CleanOciCache()
	}
	if blobClean == true {
		err = CleanBlobCache()
	}

	return err
}
