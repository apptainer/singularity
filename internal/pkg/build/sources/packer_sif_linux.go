// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"syscall"

	"github.com/sylabs/sif/pkg/sif"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
	"github.com/sylabs/singularity/pkg/image"
	"github.com/sylabs/singularity/pkg/image/unpacker"
	"github.com/sylabs/singularity/pkg/util/loop"
)

// ErrNoMetaData ... FIXME
// TODO: fix this!
var ErrNoMetaData = errors.New("no metadata found for system partition")

// Pack puts relevant objects in a Bundle!
func (p *SIFPacker) Pack() (*types.Bundle, error) {

	err := p.unpackSIF(p.b, p.srcfile)
	if err != nil {
		sylog.Errorf("unpackSIF Failed: %s", err)
		return nil, err
	}

	return p.b, nil
}

// getMetaData will return a dataType
func getMetaData(fimg *sif.FileImage, dataType sif.Datatype) (sigs []*sif.Descriptor, descr []*sif.Descriptor, err error) {
	descr = make([]*sif.Descriptor, 1)

	descr[0], _, err = fimg.GetPartPrimSys()
	if err != nil {
		return nil, nil, fmt.Errorf("no primary partition found")
	}

	// GetFromDescrID
	sigs, _, err = fimg.GetLinkedDescrsByType(uint32(0), dataType)
	if err != nil {
		return nil, nil, ErrNoMetaData
	}

	return
}

// First pass just assumes a single system partition, later passes will handle more complex sif files
// unpackSIF parses through the sif file and places each component in the sandbox
func (p *SIFPacker) unpackSIF(b *types.Bundle, srcfile string) (err error) {
	img, err := image.Init(srcfile, false)
	if err != nil {
		return fmt.Errorf("could not open image %s: %s", srcfile, err)
	}
	defer img.File.Close()

	if !img.HasRootFs() {
		return fmt.Errorf("no root filesystem found in %s", srcfile)
	}

	// Open the SIF
	fimg, err := sif.LoadContainer(srcfile, true)
	if err != nil {
		sylog.Fatalf("failed to load SIF container file: %s", err)
	}
	defer fimg.UnloadContainer()

	sifData, _, err := getMetaData(&fimg, sif.DataLabels)
	if err == ErrNoMetaData {
		sylog.Warningf("No metadata partition found")
	} else if err != nil {
		sylog.Fatalf("Unable to get label metadata: %s", err)
	} else if err == nil {
		b.Recipe.ImageData.Labels = make(map[string]string, 1)

		for _, v := range sifData {
			metaData := v.GetData(&fimg)
			err := json.Unmarshal(metaData, &b.Recipe.ImageData.Labels)
			if err != nil {
				sylog.Fatalf("Unable to get json: %s", err)
			}
		}
	}

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
			return fmt.Errorf("while copying partition data to bundle: %v", err)
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
