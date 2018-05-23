// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package image

import (
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// Image describes the interface that an image type must implement.
type Image interface {
	RuntimeImage
	BuildtimeImage
	// Crypto related functions
	//Sign()   bool
	//Verify() bool
}

// RuntimeImage describes the interface that a runtime image type must implement.
type RuntimeImage interface {
	Root() *specs.Root
}

// BuildtimeImage describes the interface that a build-time image type must implement.
type BuildtimeImage interface {
	Rootfs() string
}

/*
func GetImage(r *specs.Root) Image {
	rtype := CheckType(r.Path)
	switch rtype {
	case "sif":
		return NewSIF()
	case "squashfs":
		return NewSquashFS()
	case "sandbox":
		return NewSandbox()
	default:
		return NewSandbox()
		//Do-Some-Error-Something
	}

}

func CheckType(path string) string {
	if isSIF(path) {
		return "sif"
	} else if isSquashFS(path) {
		return "squashfs"
	} else if isSandbox(path) {
		return "sandbox"
	} else {
		return "default"
	}
}
*/
