// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"testing"

	"github.com/singularityware/singularity/src/pkg/test"
)

const (
	shubURI = "//ikaneshiro/singularityhub:latest"
)

// TestDownloadImage tests if we can pull an image from Singularity Hub
func TestDownloadImage(t *testing.T) {

	if testing.Short() {
		t.SkipNow()
	}

	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	sc := &ShubClient{}

	err := sc.DownloadImage(shubURI, false)

	//clean up tmpfs since assembler isn't called
	defer sc.CleanUp()
	if err != nil {
		t.Fatalf("failed to Get from %s: %v\n", shubURI, err)
	}
}
