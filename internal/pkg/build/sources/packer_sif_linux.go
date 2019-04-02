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

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
	"github.com/sylabs/singularity/pkg/image"
	"github.com/sylabs/singularity/pkg/image/unpacker"
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
func (p *SIFPacker) unpackSIF(b *types.Bundle, srcfile string) (err error) {
	img, err := image.Init(srcfile, false)
	if err != nil {
		return fmt.Errorf("could not open image %s: %s", srcfile, err)
	}
	defer img.File.Close()

	switch img.Partitions[0].Type {
	case image.SQUASHFS:
		// create a reader for rootfs partition
		reader, err := image.NewPartitionReader(img, "", 0)
		if err != nil {
			return fmt.Errorf("could not extract root filesystem: %s", err)
		}

		s := unpacker.NewSquashfs()

		// extract root filesystem
		if err := s.ExtractAll(reader, b.Rootfs()); err != nil {
			return fmt.Errorf("root filesystem extraction failed: %s", err)
		}
	case image.EXT3:
		info := &loop.Info64{
			Offset:    uint64(img.Partitions[0].Offset),
			SizeLimit: uint64(img.Partitions[0].Size),
			Flags:     loop.FlagsAutoClear,
		}

		// extract ext3 partition by mounting
		sylog.Debugf("Ext3 partition detected, mounting to extract.")
		err = unpackImagePartition(img.File, b.Rootfs(), "ext3", info)
		if err != nil {
			return fmt.Errorf("While copying partition data to bundle: %v", err)
		}
	default:
		return fmt.Errorf("unrecognized partition format")
	}

	return nil
}

// unpackImagePartition temporarily mounts an image parition using a loop device and then copies its contents to the destination directory
func unpackImagePartition(src *os.File, dest, mountType string, info *loop.Info64) (err error) {
	number := 0
	loopdev := new(loop.Device)
	loopdev.MaxLoopDevices = 256
	loopdev.Info = info

	if err := loopdev.AttachFromFile(src, os.O_RDONLY, &number); err != nil {
		return err
	}

	tmpmnt, err := ioutil.TempDir("", "tmpmnt-")
	if err != nil {
		return fmt.Errorf("failed to make tmp mount point: %v", err)
	}
	defer os.RemoveAll(tmpmnt)

	path := fmt.Sprintf("/dev/loop%d", number)
	sylog.Debugf("Mounting loop device %s to %s\n", path, tmpmnt)
	err = syscall.Mount(path, tmpmnt, mountType, syscall.MS_NOSUID|syscall.MS_RDONLY|syscall.MS_NODEV, "errors=remount-ro")
	if err != nil {
		sylog.Errorf("mount Failed: %s", err)
		return err
	}
	defer syscall.Unmount(tmpmnt, 0)

	// copy filesystem into dest
	sylog.Debugf("Copying filesystem from %s to %s\n", tmpmnt, dest)
	var stderr bytes.Buffer
	cmd := exec.Command("cp", "-r", tmpmnt+`/.`, dest)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cp failed: %v: %v", err, stderr.String())
	}

	return nil
}
