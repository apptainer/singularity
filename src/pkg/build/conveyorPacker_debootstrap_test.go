// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"fmt"
	"os"
	"runtime"
	"testing"
)

func TestARCH(t *testing.T) {
	fmt.Println(runtime.GOARCH)

	t.Fatal("")
}

func TestDebootstrapConveyor(t *testing.T) {

	//test must be run as root
	if os.Getuid() != 0 {
		return
	}

	testDef := Definition{}

	testDef.Header = map[string]string{
		"Bootstrap": "debootstrap",
		"OSVersion": "bionic",
		"MirrorURL": "http://us.archive.ubuntu.com/ubuntu/",
		"Include":   "apt python ",
	}

	dc := DebootstrapConveyor{}

	err := dc.Get(testDef)
	if err != nil {
		t.Fatalf("Debootstrap Get failed: %v", err)
	}

	t.Fatal("")
}

func TestDebootstrapPacker(t *testing.T) {

	//test must be run as root
	if os.Getuid() != 0 {
		return
	}

	testDef := Definition{}

	testDef.Header = map[string]string{
		"Bootstrap": "debootstrap",
		"OSVersion": "bionic",
		"MirrorURL": "http://us.archive.ubuntu.com/ubuntu/",
		"Include":   "apt python ",
	}

	dcp := DebootstrapConveyorPacker{}

	err := dcp.Get(testDef)
	if err != nil {
		t.Fatalf("Debootstrap Get failed: %v", err)
	}

	_, err = dcp.Pack()
	if err != nil {
		t.Fatalf("Debootstrap Pack failed: %v", err)
	}

	t.Fatal("")
}
