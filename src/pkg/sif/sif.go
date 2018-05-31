// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sif

import (
	"io"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/golang/glog"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/singularityware/singularity/src/pkg/image"
)

// SIF describes a SIF image.
type SIF struct {
	path string
}

// FromSandbox converts the sandbox, s, to a SIF file.
func FromSandbox(sandbox *image.Sandbox, imagePath string) (*SIF, error) {
	mksquashfs, err := exec.LookPath("mksquashfs")
	if err != nil {
		glog.Error("mksquashfs is not installed on this system")
		return nil, err
	}

	f, err := ioutil.TempFile("", "squashfs-")
	squashfsPath := f.Name() + ".img"
	f.Close()
	os.Remove(squashfsPath)

	mksquashfsCmd := exec.Command(mksquashfs, sandbox.Rootfs(), squashfsPath, "-noappend")
	mksquashfsCmd.Stdin = os.Stdin
	mksquashfsCmd.Stdout = os.Stdout
	mksquashfsCmd.Stderr = os.Stderr
	err = mksquashfsCmd.Run()
	if err != nil {
		return nil, err
	}

	sifCmd := exec.Command("singularity", "sif", "create", "-P", squashfsPath, "-f", "SQUASHFS", "-p", "SYSTEM", "-c", "LINUX", imagePath)
	sifCmd.Stdin = os.Stdin
	sifCmd.Stdout = os.Stdout
	sifCmd.Stderr = os.Stderr
	err = sifCmd.Run()
	if err != nil {
		return nil, err
	}

	return &SIF{path: imagePath}, nil

}

// FromPath returns a SIF object of the file located at path.
func FromPath(path string) *SIF {
	return &SIF{}
}

// FromReadSeeker returns a SIF object from the supplied ReadSeeker.
func FromReadSeeker(f io.ReadSeeker) *SIF {
	return &SIF{}
}

// Root returns the root specification of the SIF.
func (i *SIF) Root() *specs.Root {
	return &specs.Root{}
}

// Rootfs returns the root FS of the SIF.
func (i *SIF) Rootfs() string {
	return i.path
}

// isSIF checks the "magic" of the given file and
// determines if the file is of the SIF type
func isSIF(f io.ReadSeeker) bool {
	return false
}
