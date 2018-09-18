// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package assemblers

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"

	"github.com/satori/go.uuid"
	"github.com/singularityware/singularity/src/pkg/build/types"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/sylabs/sif/pkg/sif"
)

// SIFAssembler doesnt store anything
type SIFAssembler struct {
}

func createSIFSinglePart(path string, squashfile string) (err error) {
	// general info for the new SIF file creation
	cinfo := sif.CreateInfo{
		Pathname:   path,
		Launchstr:  sif.HdrLaunch,
		Sifversion: sif.HdrVersion,
		ID:         uuid.NewV4(),
	}

	// data we need to create a system partition descriptor
	parinput := sif.DescriptorInput{
		Datatype: sif.DataPartition,
		Groupid:  sif.DescrDefaultGroup,
		Link:     sif.DescrUnusedLink,
		Fname:    squashfile,
	}
	// open up the data object file for this descriptor
	if parinput.Fp, err = os.Open(parinput.Fname); err != nil {
		return fmt.Errorf("while opening partition file: %s", err)
	}
	defer parinput.Fp.Close()
	fi, err := parinput.Fp.Stat()
	if err != nil {
		return fmt.Errorf("while calling start on partition file: %s", err)
	}
	parinput.Size = fi.Size()

	err = parinput.SetPartExtra(sif.FsSquash, sif.PartPrimSys, sif.GetSIFArch(runtime.GOARCH))
	if err != nil {
		return
	}

	// add this descriptor input element to the list
	cinfo.InputDescr = append(cinfo.InputDescr, parinput)

	// test container creation with two partition input descriptors
	if _, err := sif.CreateContainer(cinfo); err != nil {
		return fmt.Errorf("while creating container: %s", err)
	}

	return nil
}

// Assemble creates a SIF image from a Bundle
func (a *SIFAssembler) Assemble(b *types.Bundle, path string) (err error) {
	defer os.RemoveAll(b.Path)

	sylog.Infof("Creating SIF file...")

	// insert help
	err = insertHelpScript(b)
	if err != nil {
		return fmt.Errorf("While inserting help script: %v", err)
	}

	// insert labels
	err = insertLabelsJSON(b)
	if err != nil {
		return fmt.Errorf("While inserting labels JSON: %v", err)
	}

	// insert definition
	err = insertDefinition(b)
	if err != nil {
		return fmt.Errorf("While inserting definition: %v", err)
	}

	mksquashfs, err := exec.LookPath("mksquashfs")
	if err != nil {
		sylog.Errorf("mksquashfs is not installed on this system")
		return
	}

	f, err := ioutil.TempFile(b.Path, "squashfs-")
	squashfsPath := f.Name() + ".img"
	f.Close()
	os.Remove(f.Name())
	os.Remove(squashfsPath)

	mksquashfsCmd := exec.Command(mksquashfs, b.Rootfs(), squashfsPath, "-noappend")
	err = mksquashfsCmd.Run()
	defer os.Remove(squashfsPath)
	if err != nil {
		return
	}

	err = createSIFSinglePart(path, squashfsPath)
	if err != nil {
		return
	}

	return
}
