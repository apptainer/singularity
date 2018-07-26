// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"syscall"

	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/loop"
	"github.com/sylabs/sif/pkg/sif"
)

// SIFPacker holds the locations of where to pack from and to
type SIFPacker struct {
	srcfile string
	tmpfs   string
}

// Pack puts relevant objects in a Bundle!
func (p *SIFPacker) Pack() (b *Bundle, err error) {

	b, err = NewBundle(p.tmpfs)
	if err != nil {
		return
	}

	err = p.unpackSIF(b, p.srcfile)
	if err != nil {
		sylog.Errorf("unpackSIF Failed", err.Error())
		return nil, err
	}

	return b, nil
}

// First pass just assumes a single system partition, later passes will handle more complex sif files
// unpackSIF parses throught the sif file and places each component in the sandbox
func (p *SIFPacker) unpackSIF(b *Bundle, rootfs string) (err error) {

	// load the container
	fimg, err := sif.LoadContainer(rootfs, false)
	if err != nil {
		sylog.Errorf("error loading sif file %s: %s\n", rootfs, err)
		return err
	}
	defer fimg.UnloadContainer()

	// Get the default system partition image as rootfs
	rootfsPart, _, err := fimg.GetPartFromGroup(sif.DescrDefaultGroup)
	if err != nil {
		return err
	}

	// Check that this is a system partition
	parttype, err := rootfsPart.GetPartType()
	if err != nil {
		return err
	}
	if parttype != sif.PartSystem {
		return fmt.Errorf("Default partition is not system")
	}

	// record the fs type
	mountType := ""
	fstype, err := rootfsPart.GetFsType()
	if err != nil {
		return err
	}
	if fstype == sif.FsSquash {
		mountType = "squashfs"
	} else if fstype == sif.FsExt3 {
		mountType = "ext3"
	} else {
		return fmt.Errorf("unknown file system type: %v", fstype)
	}

	info := &loop.Info64{
		Offset:    uint64(rootfsPart.Fileoff),
		SizeLimit: uint64(rootfsPart.Filelen),
		Flags:     loop.FlagsAutoClear,
	}

	//copy partition contents to bundle rootfs
	err = unpackImagePartion(fimg.Fp.Name(), b.Rootfs(), mountType, info)
	if err != nil {
		return fmt.Errorf("While copying partition data to bundle: %v", err)
	}

	return nil
}

// unpackImagePart temporarily mounts an image parition using a loop device and then copies its contents to the destination directory
func unpackImagePartion(src, dest, mountType string, info *loop.Info64) (err error) {

	var number int
	number = 0
	loopdev := new(loop.Device)

	if err := loopdev.Attach(src, os.O_RDONLY, &number); err != nil {
		return err
	}

	if err := loopdev.SetStatus(info); err != nil {
		return err
	}

	tmpmnt, err := ioutil.TempDir("", "tmpmnt-")
	if err != nil {
		return fmt.Errorf("Failed to make tmp mount point: %v", err)
	}
	defer os.RemoveAll(tmpmnt)

	path := fmt.Sprintf("/dev/loop%d", number)
	sylog.Debugf("Mounting loop device %s to %s\n", path, tmpmnt)
	err = syscall.Mount(path, tmpmnt, mountType, syscall.MS_NOSUID|syscall.MS_RDONLY|syscall.MS_NODEV, "errors=remount-ro")
	if err != nil {
		sylog.Errorf("Mount Failed", err.Error())
		return err
	}
	defer syscall.Unmount(tmpmnt, 0)

	//copy filesystem into dest
	sylog.Debugf("Copying filesystem from %s to %s\n", tmpmnt, dest)
	cmd := exec.Command("cp", "-r", tmpmnt+`/.`, dest)
	err = cmd.Run()
	if err != nil {
		sylog.Errorf("cp Failed", err.Error())
		return err
	}

	return nil
}
