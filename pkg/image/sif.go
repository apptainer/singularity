/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package image

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/golang/glog"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

type SIF struct {
	path string
}

// SIFFromSandbox converts the sandbox, s, to a SIF file
func SIFFromSandbox(sandbox *Sandbox, imagePath string) (*SIF, error) {
	mksquashfs, err := exec.LookPath("mksquashfs")
	if err != nil {
		glog.Error("mksquashfs is not installed on this system")
		return nil, err
	}

	f, err := ioutil.TempFile("", "squashfs-")
	squashfsPath := f.Name() + ".img"
	f.Close()
	os.Remove(squashfsPath)

	mksquashfsCmd := exec.Command(mksquashfs, sandbox.Rootfs(), squashfsPath, "-noappend", "-all-root")
	mksfsout, err := mksquashfsCmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	fmt.Println(string(mksfsout))

	sif, err := exec.LookPath("sif")
	if err != nil {
		glog.Error("sif is not installed on this system")
		return nil, err
	}

	sifCmd := exec.Command(sif, "create", "-P", squashfsPath, "-f", "SQUASHFS", "-p", "SYSTEM", "-c", "LINUX", imagePath)
	sifout, err := sifCmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	fmt.Println(string(sifout))

	return &SIF{path: imagePath}, nil

}

// SIFFromPath returns a SIF object of the file located at path
func SIFFromPath(path string) *SIF {
	return &SIF{}
}

func SIFFromReadSeeker(f io.ReadSeeker) *SIF {
	return &SIF{}
}

func (i *SIF) Root() *specs.Root {
	return &specs.Root{}
}

func (i *SIF) Rootfs() string {
	return i.path
}

// isSIF checks the "magic" of the given file and
// determines if the file is of the SIF type
func isSIF(f io.ReadSeeker) bool {
	return false
}
