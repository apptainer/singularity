// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"fmt"
	"testing"

	"github.com/singularityware/singularity/src/pkg/test"
)

// TestShubPacker checks if we can create a Bundle from the pulled image
func TestShubParser(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	validShubURIs := []string{
		`//username/container`,
		`//username/container:tag`,
		`//username/container@00000000000000000000000000000000`,
		`//registry/username/container`,
		`//registry/with/levels/username/container`,
		`//registry/user-name/container-with-dash`,
		`//registry/username/container.with.period`,
		`//username/container:tag-with-dash`,
		`//username/container:tag_wtih_underscore`,
		`//username/container:tag.with.period`,
	}

	invalidShubURIs := []string{
		`//username/`,
		`//username/container:`,
		`//username/container@`,
		`//username/container@0000000000000000000000000000000`,
		`//username/container@000000000000000000000000000000000`,
		`//username/container@abcdefghijklmnopqrstuvwxyz123456`,
		`//registry/user.name/container`,
		`//username.with.period/container:tag`,
		`//-username/container:`,
		`//username-/container:`,
		`//-registry/username/container:`,
		`//registry-/username/container:`,
	}

	for _, uri := range validShubURIs {
		fmt.Println("Starting parsing of: ", uri)
		_, err := ShubParseReference(uri)
		if err != nil {
			t.Fatalf("failed to parse valid URI: %v %v", uri, err)
		}
	}

	for _, uri := range invalidShubURIs {
		fmt.Println("Starting parsing of: ", uri)
		_, err := ShubParseReference(uri)
		if err == nil {
			t.Fatalf("failed to catch invalid URI: %v %v", uri, err)
		}
	}
}
