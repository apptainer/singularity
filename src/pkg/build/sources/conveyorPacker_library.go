// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"io/ioutil"
	"os"

	sytypes "github.com/sylabs/singularity/src/pkg/build/types"
	"github.com/sylabs/singularity/src/pkg/client/library"
	"github.com/sylabs/singularity/src/pkg/sylog"
)

// LibraryConveyorPacker only needs to hold a packer to pack the image it pulls
// as well as extra information about the library it's pulling from
type LibraryConveyorPacker struct {
	b *sytypes.Bundle
	LocalPacker
	LibraryURL string
	AuthToken  string
}

// Get downloads container from Singularityhub
func (cp *LibraryConveyorPacker) Get(b *sytypes.Bundle) (err error) {
	sylog.Debugf("Getting container from Library")

	cp.b = b

	// create file for image download
	f, err := ioutil.TempFile(cp.b.Path, "library-img")
	if err != nil {
		return
	}
	defer f.Close()

	cp.b.FSObjects["libraryImg"] = f.Name()

	// get image from library
	if err = client.DownloadImage(cp.b.FSObjects["libraryImg"], b.Recipe.Header["from"], cp.LibraryURL, true, cp.AuthToken); err != nil {
		sylog.Fatalf("failed to Get from %s://%s: %v\n", cp.LibraryURL, cp.b.Recipe.Header["from"], err)
	}

	cp.LocalPacker, err = GetLocalPacker(cp.b.FSObjects["libraryImg"], cp.b)

	return err
}

// CleanUp removes any tmpfs owned by the conveyorPacker on the filesystem
func (cp *LibraryConveyorPacker) CleanUp() {
	os.RemoveAll(cp.b.Path)
}
