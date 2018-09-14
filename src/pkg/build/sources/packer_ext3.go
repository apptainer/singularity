// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"syscall"

	"github.com/singularityware/singularity/src/pkg/build/types"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/loop"
	args "github.com/singularityware/singularity/src/runtime/engines/singularity/rpc"
)

// Ext3Packer holds the locations of where to back from and to, aswell as image offset info
type Ext3Packer struct {
	srcfile string
	b       *types.Bundle
	info    *loop.Info64
}

// Pack puts relevant objects in a Bundle!
func (p *Ext3Packer) Pack() (*types.Bundle, error) {
	rootfs := p.srcfile

	err := p.unpackExt3(p.b, p.info, rootfs)
	if err != nil {
		sylog.Errorf("unpackExt3 Failed: %s", err)
		return nil, err
	}

	return p.b, nil
}

// unpackExt3 mounts the ext3 image using a loop device and then copies its contents to the bundle
func (p *Ext3Packer) unpackExt3(b *types.Bundle, info *loop.Info64, rootfs string) (err error) {
	tmpmnt, err := ioutil.TempDir(p.b.Path, "mnt")

	var number int
	info.Flags = loop.FlagsAutoClear
	arguments := &args.LoopArgs{
		Image: rootfs,
		Mode:  os.O_RDONLY,
		Info:  *info,
	}
	err = getLoopDevice(arguments)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/dev/loop%d", number)
	sylog.Debugf("Mounting loop device %s to %s\n", path, tmpmnt)
	err = syscall.Mount(path, tmpmnt, "ext3", syscall.MS_NOSUID|syscall.MS_RDONLY|syscall.MS_NODEV, "errors=remount-ro")
	if err != nil {
		sylog.Errorf("Mount Failed: %s", err)
		return err
	}
	defer syscall.Unmount(tmpmnt, 0)

	//copy filesystem into bundle rootfs
	sylog.Debugf("Copying filesystem from %s to %s in Bundle\n", tmpmnt, b.Rootfs())
	cmd := exec.Command("cp", "-r", tmpmnt+`/.`, b.Rootfs())
	err = cmd.Run()
	if err != nil {
		sylog.Errorf("cp Failed: %s", err)
		return err
	}

	return err
}

// getLoopDevice attaches a loop device with the specified arguments
func getLoopDevice(arguments *args.LoopArgs) error {
	var reply int
	reply = 1
	loopdev := new(loop.Device)

	if err := loopdev.AttachFromPath(arguments.Image, arguments.Mode, &reply); err != nil {
		return err
	}

	if err := loopdev.SetStatus(&arguments.Info); err != nil {
		return err
	}
	return nil
}
