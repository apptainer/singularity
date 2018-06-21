// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/singularityware/singularity/src/pkg/sylog"
)

// SIFAssembler doesnt store anything
type SIFAssembler struct {
}

// Assemble creates a SIF image from a Bundle
func (a *SIFAssembler) Assemble(b *Bundle, path string) (err error) {

	defer os.RemoveAll(b.path)

	mksquashfs, err := exec.LookPath("mksquashfs")
	if err != nil {
		sylog.Errorf("mksquashfs is not installed on this system")
		return err
	}

	f, err := ioutil.TempFile(b.path, "squashfs-")
	squashfsPath := f.Name() + ".img"
	f.Close()
	os.Remove(f.Name())
	os.Remove(squashfsPath)

	mksquashfsCmd := exec.Command(mksquashfs, b.Rootfs(), squashfsPath, "-noappend")
	mksquashfsCmd.Stdin = os.Stdin
	mksquashfsCmd.Stdout = os.Stdout
	mksquashfsCmd.Stderr = os.Stderr
	err = mksquashfsCmd.Run()
	if err != nil {
		return err
	}

	sifCmd := exec.Command("singularity", "sif", "create", "-P", squashfsPath, "-f", "SQUASHFS", "-p", "SYSTEM", "-c", "LINUX", path)
	sifCmd.Stdin = os.Stdin
	sifCmd.Stdout = os.Stdout
	sifCmd.Stderr = os.Stderr
	err = sifCmd.Run()
	if err != nil {
		return err
	}

	return nil
}
