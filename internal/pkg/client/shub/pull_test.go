// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package shub

import (
	"context"
	"os"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

const (
	shubImageURI = "shub://ikaneshiro/singularityhub:latest"
	shubImgPath  = "/tmp/shub-test_img.simg"
)

// TestDownloadImage tests if we can pull an image from Singularity Hub
func TestDownloadImage(t *testing.T) {
	// TODO(mem): reenable this; disabled while shub is down
	t.Skip("Skipping tests that access singularity hub")

	if testing.Short() {
		t.SkipNow()
	}

	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	shubURI, err := ParseReference(shubImageURI)
	if err != nil {
		t.Fatalf("failed to parse shub uri: %v", err)
	}

	// Get the image manifest
	manifest, err := GetManifest(shubURI, false)
	if err != nil {
		t.Fatalf("failed to get manifest from shub: %s", err)
	}

	err = DownloadImage(context.Background(), manifest, shubImgPath, shubImageURI, false, false)
	if err != nil {
		t.Fatalf("failed to Get from %s: %v\n", shubURI, err)
	}

	// clean up
	err = os.Remove(shubImgPath)
	if err != nil {
		t.Fatalf("failed to clean up test environment: %v", err)
	}
}
