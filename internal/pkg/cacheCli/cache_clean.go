// Copyright (c) 2017-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cacheCli

import (
	"os"
	"io/ioutil"
	"strings"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
)

func CleanLibraryCache() error {
	sylog.Debugf("Removing: %v", cache.Library())

	err := os.RemoveAll(cache.Library())

	return err
}

func CleanOciCache() error {
	sylog.Debugf("Removing: %v", cache.OciTemp())

	err := os.RemoveAll(cache.OciTemp())

	return err
}

func cleanLibraryCache(cacheName string) error {
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
				if err != nil {
					return err
				}
			}
		}
	}

	return err
}

func cleanOciCache(cacheName string) error {
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
				if err != nil {
					return err
				}
			}
		}
	}

	return err
}

func CleanCacheName(cacheName string, libraryCache, ociCache bool) error {
	if libraryCache == true {
		err = cleanLibraryCache(cacheName)
		if err != nil {
			sylog.Fatalf("%v", err)
			return err
		}
	}
	if ociCache == true {
		err = cleanOciCache(cacheName)
		if err != nil {
			sylog.Fatalf("%v", err)
			return err
		}
	}
	if libraryCache != true && ociCache != true {
		err = cleanLibraryCache(cacheName)
		if err != nil {
			sylog.Fatalf("%v", err)
		}
		err = cleanOciCache(cacheName)
		if err != nil {
			sylog.Fatalf("%v", err)
			return err
		}
	}

	return err
}

var err error

//func CleanSingularityCache(allClean, libraryClean, ociClean bool, cacheName string) error {
func CleanSingularityCache(allClean bool, typeNameClean, cacheName string) error {
	libraryClean := false
	ociClean := false

	if len(typeNameClean) >= 1 {
		for _, nameType := range strings.Split(typeNameClean, ",") {
			if nameType == "library" {
				libraryClean = true
			} else if nameType == "oci" {
				ociClean = true
			} else {
				sylog.Fatalf("Not a valid type: %v", typeNameClean)
				os.Exit(2)
			}

		}
	} else {
		libraryClean = true
		ociClean = true
	}

	if len(typeNameClean) >= 1 {
		if typeNameClean == "library" {
			libraryClean = true
		} else if typeNameClean == "oci" {
			ociClean = true
		} else {
			sylog.Fatalf("Not a valit type: %v", typeNameClean)
			os.Exit(2)
		}
	}

	if len(cacheName) >= 1 && allClean != true {
		err = CleanCacheName(cacheName, libraryClean, ociClean)
		return err
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
	if libraryClean != true && ociClean != true {
		err = cache.Clean()
	}

	sylog.Debugf("DONE!")

	return err
}


