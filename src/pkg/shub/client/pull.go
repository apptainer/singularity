// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	sytypes "github.com/singularityware/singularity/src/pkg/build/types"
	"github.com/singularityware/singularity/src/pkg/sylog"
)

// Timeout for an image pull in seconds
const pullTimeout = 1800

// DownloadImage will retrieve an image from the Container Singularityhub,
// saving it into the specified file
func (s *ShubClient) DownloadImage(shubRef string, Force bool) (err error) {
	sylog.Debugf("Downloading container from Shub")

	//use custom parser to make sure we have a valid shub URI
	s.ShubURI, err = ShubParseReference(shubRef)
	if err != nil {
		sylog.Fatalf("Invalid shub URI: %v", err)
		return
	}

	//create empty bundle to build into
	s.Bundle, err = sytypes.NewBundle("shub")
	if err != nil {
		return
	}

	// Get the image manifest
	if err = s.getManifest(); err != nil {
		sylog.Fatalf("Failed to get manifest from Shub: %v", err)
		return
	}

	// retrieve the image
	if err = s.fetchImage(s.Bundle); err != nil {
		sylog.Fatalf("Failed to get image from Shub: %v", err)
		return
	}

	return err
}
