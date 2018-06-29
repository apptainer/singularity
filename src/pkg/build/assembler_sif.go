// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"container/list"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/satori/go.uuid"
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
		Arch:       sif.HdrArchAMD64,
		ID:         uuid.NewV4(),
		Inputlist:  list.New(),
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

	// extra data needed for the creation of a partition descriptor
	pinfo := sif.Partition{
		Fstype:   sif.FsSquash,
		Parttype: sif.PartSystem,
	}

	// serialize the partition data for integration with the base descriptor input
	if err := binary.Write(&parinput.Extra, binary.LittleEndian, pinfo); err != nil {
		return fmt.Errorf("while serializing pinfo: %s", err)
	}

	// add this descriptor input element to the list
	cinfo.Inputlist.PushBack(parinput)

	// test container creation with two partition input descriptors
	if err := sif.CreateContainer(cinfo); err != nil {
		return fmt.Errorf("while creating container: %s", err)
	}

	return nil
}

// Assemble creates a SIF image from a Bundle
func (a *SIFAssembler) Assemble(b *Bundle, path string) (err error) {
	mksquashfs, err := exec.LookPath("mksquashfs")
	if err != nil {
		sylog.Errorf("mksquashfs is not installed on this system")
		return
	}

	f, err := ioutil.TempFile("", "squashfs-")
	squashfsPath := f.Name() + ".img"
	f.Close()
	os.Remove(f.Name())
	os.Remove(squashfsPath)

	mksquashfsCmd := exec.Command(mksquashfs, b.Rootfs(), squashfsPath, "-noappend")
	mksquashfsCmd.Stdin = os.Stdin
	mksquashfsCmd.Stdout = os.Stdout
	mksquashfsCmd.Stderr = os.Stderr
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
