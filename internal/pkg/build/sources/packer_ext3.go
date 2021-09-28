// Copyright (c) 2019-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"github.com/hpcng/singularity/pkg/build/types"
	"github.com/hpcng/singularity/pkg/image"
	"github.com/hpcng/singularity/pkg/sylog"
	"github.com/hpcng/singularity/pkg/util/loop"

	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"syscall"
)

// Ext3Packer holds the locations of where to back from and to, aswell as image offset info
type Ext3Packer struct {
	srcfile string
	b       *types.Bundle
	img     *image.Image
}

// Pack puts relevant objects in a Bundle!
func (p *Ext3Packer) Pack(context.Context) (*types.Bundle, error) {
	err := unpackExt3(p.b, p.img)
	if err != nil {
		sylog.Errorf("while unpacking ext3 image: %v", err)
		return nil, err
	}

	return p.b, nil
}

// unpackExt3 mounts the ext3 image using a loop device and then copies its contents to the bundle
func unpackExt3(b *types.Bundle, img *image.Image) error {
	info := &loop.Info64{
		Offset:    img.Partitions[0].Offset,
		SizeLimit: img.Partitions[0].Size,
		Flags:     loop.FlagsAutoClear,
	}

	var number int
	loopdev := &loop.Device{
		MaxLoopDevices: loop.GetMaxLoopDevices(),
		Info:           info,
	}

	if err := loopdev.AttachFromFile(img.File, os.O_RDONLY, &number); err != nil {
		return fmt.Errorf("while attaching image to loop device: %v", err)
	}

	tmpmnt, err := ioutil.TempDir(b.TmpDir, "mnt")
	if err != nil {
		return fmt.Errorf("while making tmp mount point: %v", err)
	}

	path := fmt.Sprintf("/dev/loop%d", number)
	sylog.Debugf("Mounting loop device %s to %s\n", path, tmpmnt)
	err = syscall.Mount(path, tmpmnt, "ext3", syscall.MS_NOSUID|syscall.MS_RDONLY|syscall.MS_NODEV, "errors=remount-ro")
	if err != nil {
		return fmt.Errorf("while mounting image: %v", err)
	}
	defer syscall.Unmount(tmpmnt, 0)

	// copy filesystem into bundle rootfs
	sylog.Debugf("Copying filesystem from %s to %s in Bundle\n", tmpmnt, b.RootfsPath)
	var stderr bytes.Buffer
	cmd := exec.Command("cp", "-a", tmpmnt+`/.`, b.RootfsPath)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("while copying files: %v: %v", err, stderr.String())
	}

	return nil
}
