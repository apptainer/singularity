// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"fmt"
	"os"
	"testing"
)

const (
	shubURI = "shub://ikaneshiro/singularityhub:latest"
)

// TestShubConveyor tests if we can pull an image from singularity hub
func TestShubConveyor(t *testing.T) {
	def, err := NewDefinitionFromURI(shubURI)
	if err != nil {
		t.Fatalf("unable to parse URI %s: %v\n", shubURI, err)
	}

	sc := &ShubConveyor{}

	if err := sc.Get(def); err != nil {
		//clean up tmpfs since assembler isnt called
		os.RemoveAll(sc.tmpfs)
		t.Fatalf("failed to Get from %s: %v\n", shubURI, err)
	}
	//clean up tmpfs since assembler isnt called
	os.RemoveAll(sc.tmpfs)
}

// TestShubPacker checks if we can create a Bundle from the pulled image
func TestShubPacker(t *testing.T) {
	def, err := NewDefinitionFromURI(shubURI)
	if err != nil {
		t.Fatalf("unable to parse URI %s: %v\n", shubURI, err)
	}

	scp := &ShubConveyorPacker{}

	if err := scp.Get(def); err != nil {
		//clean up tmpfs since assembler isnt called
		os.RemoveAll(scp.tmpfs)
		t.Fatalf("failed to Get from %s: %v\n", shubURI, err)
	}

	//clean up tmpfs since assembler isnt called
	defer os.RemoveAll(scp.tmpfs)

	_, err = scp.Pack()
	if err != nil {
		t.Fatalf("failed to Pack from %s: %v\n", shubURI, err)
	}
}

// TestShubPacker checks if we can create a Bundle from the pulled image
func TestShubParser(t *testing.T) {
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
		_, err := shubParseReference(uri)
		if err != nil {
			t.Fatalf("failed to parse valid URI: %v %v", uri, err)
		}
	}

	for _, uri := range invalidShubURIs {
		fmt.Println("Starting parsing of: ", uri)
		_, err := shubParseReference(uri)
		if err == nil {
			t.Fatalf("failed to catch invalid URI: %v %v", uri, err)
		}
	}
}
