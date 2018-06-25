// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// Copyright (c) 2018, Vanessa Sochat. All rights reserved.
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
	args "github.com/singularityware/singularity/src/runtime/engines/singularity/rpc"
)

// Ext3Packer only needs to hold the conveyor to have the needed data to pack
type Ext3Packer struct {
	srcfile string
	tmpfs   string
	info    *loop.Info64
}

// Pack puts relevant objects in a Bundle!
func (p *Ext3Packer) Pack() (b *Bundle, err error) {
	rootfs := p.srcfile

	b, err = NewBundle(p.tmpfs)
	if err != nil {
		return
	}

	err = p.unpackExt3(b, p.info, rootfs)
	if err != nil {
		sylog.Errorf("unpackExt3 Failed", err.Error())
		return nil, err
	}

	return b, nil
}

// unpackExt3 mounts the ext3 image using a loop device and then copies its contents to the bundle
func (p *Ext3Packer) unpackExt3(b *Bundle, info *loop.Info64, rootfs string) (err error) {
	tmpmnt, err := ioutil.TempDir(p.tmpfs, "mnt")

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
		sylog.Errorf("Mount Failed", err.Error())
		return err
	}
	defer syscall.Unmount(tmpmnt, 0)

	//copy filesystem into bundle rootfs
	sylog.Debugf("Copying filesystem from %s to %s in Bundle\n", tmpmnt, b.Rootfs())
	cmd := exec.Command("cp", "-r", tmpmnt+`/.`, b.Rootfs())
	err = cmd.Run()
	if err != nil {
		sylog.Errorf("cp Failed", err.Error())
		return err
	}

	return err
}

// getLoopDevice attaches a loop device with the specified arguments
func getLoopDevice(arguments *args.LoopArgs) error {
	var reply int
	reply = 1
	loopdev := new(loop.Device)

	if err := loopdev.Attach(arguments.Image, arguments.Mode, &reply); err != nil {
		return err
	}

	if err := loopdev.SetStatus(&arguments.Info); err != nil {
		return err
	}
	return nil
}
