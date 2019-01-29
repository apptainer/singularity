// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"syscall"

	"github.com/sylabs/sif/pkg/sif"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
	"github.com/sylabs/singularity/pkg/util/loop"
)

// Pack puts relevant objects in a Bundle!
func (p *SIFPacker) Pack() (*types.Bundle, error) {

	err := p.unpackSIF(p.b, p.srcfile)
	if err != nil {
		sylog.Errorf("unpackSIF Failed: %s", err)
		return nil, err
	}

	return p.b, nil
}

// First pass just assumes a single system partition, later passes will handle more complex sif files
// unpackSIF parses through the sif file and places each component in the sandbox
func (p *SIFPacker) unpackSIF(b *types.Bundle, rootfs string) (err error) {

	// load the container
	fimg, err := sif.LoadContainer(rootfs, true)
	if err != nil {
		sylog.Errorf("error loading sif file %s: %s\n", rootfs, err)
		return err
	}
	defer fimg.UnloadContainer()

	// Get the default system partition image as rootfs part
	part, _, err := fimg.GetPartPrimSys()
	if err != nil {
		return err
	}

	// record the fs type
	mountType := ""
	fstype, err := part.GetFsType()
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
		Offset:    uint64(part.Fileoff),
		SizeLimit: uint64(part.Filelen),
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
	loopdev.MaxLoopDevices = 256
	loopdev.Info = info

	if err := loopdev.AttachFromPath(src, os.O_RDONLY, &number); err != nil {
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
		sylog.Errorf("Mount Failed: %s", err)
		return err
	}
	defer syscall.Unmount(tmpmnt, 0)

	//copy filesystem into dest
	sylog.Debugf("Copying filesystem from %s to %s\n", tmpmnt, dest)
	var stderr bytes.Buffer
	cmd := exec.Command("cp", "-r", tmpmnt+`/.`, dest)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cp Failed: %v: %v", err, stderr)
	}

	return nil
}
