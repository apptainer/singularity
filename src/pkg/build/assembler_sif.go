// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"encoding/json"
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
	mksquashfs, err := exec.LookPath("mksquashfs")
	if err != nil {
		sylog.Errorf("mksquashfs is not installed on this system")
		return err
	}

	f, err := ioutil.TempFile(b.Path, "squashfs-")
	squashfsPathRoot := f.Name() + ".img"
	f.Close()
	os.Remove(f.Name())
	os.Remove(squashfsPathRoot)

	f, err = ioutil.TempFile(b.Path, "squashfs-")
	squashfsPathSingularityD := f.Name() + ".img"
	f.Close()
	os.Remove(f.Name())
	os.Remove(squashfsPathSingularityD)

	//squashfs for rootfs
	mksquashfsCmd := exec.Command(mksquashfs, b.Rootfs(), squashfsPathRoot, "-noappend")
	mksquashfsCmd.Stdin = os.Stdin
	mksquashfsCmd.Stdout = os.Stdout
	mksquashfsCmd.Stderr = os.Stderr
	err = mksquashfsCmd.Run()
	if err != nil {
		return err
	}

	//squashfs for .singularity.d
	mksquashfsCmd = exec.Command(mksquashfs, b.Path+"/"+b.FSObjects[".singularity.d"], squashfsPathSingularityD, "-noappend")
	mksquashfsCmd.Stdin = os.Stdin
	mksquashfsCmd.Stdout = os.Stdout
	mksquashfsCmd.Stderr = os.Stderr
	err = mksquashfsCmd.Run()
	if err != nil {
		return err
	}

	partitionMap := map[int]string{
		2: "/.singularity.d",
	}

	data, err := json.Marshal(partitionMap)

	f, err = ioutil.TempFile(b.Path, "json-")
	JSONPath := f.Name() + ".json"
	f.Close()
	os.Remove(f.Name())
	os.Remove(JSONPath)

	err = ioutil.WriteFile(JSONPath, data, 0755)

	sifCmd := exec.Command("singularity", "sif", "create", "-P", squashfsPathRoot, "-f", "SQUASHFS", "-p", "SYSTEM", "-c", "LINUX", "-P", squashfsPathSingularityD, "-f", "SQUASHFS", "-p", "DATA", "-c", "SINGULARITY.D", "-L", JSONPath, path)
	sifCmd.Stdin = os.Stdin
	sifCmd.Stdout = os.Stdout
	sifCmd.Stderr = os.Stderr
	err = sifCmd.Run()
	if err != nil {
		return err
	}

	return nil
}
