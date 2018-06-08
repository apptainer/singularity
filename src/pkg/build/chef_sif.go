// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/golang/glog"
)


type SIFChef struct {

}

func (c *SIFChef) Cook(k *Kitchen, path string) (err error) {
    mksquashfs, err := exec.LookPath("mksquashfs")
	if err != nil {
		glog.Error("mksquashfs is not installed on this system")
		return err
	}

	f, err := ioutil.TempFile("", "squashfs-")
	squashfsPath := f.Name() + ".img"
	f.Close()
	os.Remove(f.Name())
	os.Remove(squashfsPath)

	mksquashfsCmd := exec.Command(mksquashfs, k.Rootfs(), squashfsPath, "-noappend")
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
