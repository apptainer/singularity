// Copyright (c) 2017-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cacheCli

import (
	"os"

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

var err error

func CleanSingularityCache(allClean, libraryClean, ociClean bool) error {

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


