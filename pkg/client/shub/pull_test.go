// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"os"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

const (
	shubURI     = "shub://ikaneshiro/singularityhub:latest"
	shubImgPath = "/tmp/shub-test_img.simg"
)

// TestDownloadImage tests if we can pull an image from Singularity Hub
func TestDownloadImage(t *testing.T) {

	if testing.Short() {
		t.SkipNow()
	}

	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	err := DownloadImage(shubImgPath, shubURI, false, false)
	if err != nil {
		t.Fatalf("failed to Get from %s: %v\n", shubURI, err)
	}

	//clean up
	err = os.Remove(shubImgPath)
	if err != nil {
		t.Fatalf("failed to clean up test environment: %v", err)
	}
}
