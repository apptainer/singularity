// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

/*
#include <fcntl.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <unistd.h>
#include <uuid/uuid.h>

#include "util/config_parser.h"

#include "image/image.h"
#include "util/message.h"
#include "sif/list.h"
#include "sif/sif.h"
#include "sif/sifaccess.h"

// TODO return struct Sifpartition instead of Sifcommon
Sifcommon sif_getpart(struct image_object *image, int id) {
	Sifinfo sif;
	//Sifpartition *partdesc;

	if (sif_load(image->path, &sif, 1) < 0) {
		singularity_message(ERROR, "Unable to open sif\n");
	}

	Sifdescriptor *desc = sif_getdescid(&sif, id);

	return desc->cm;
}

*/
// #cgo CFLAGS: -I../../c/lib
// #cgo LDFLAGS: -L../../../../builddir/lib -lruntime -luuid
import "C"

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
	"path/filepath"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/singularityware/singularity/src/pkg/buildcfg"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/loop"
	"github.com/singularityware/singularity/src/runtime/engines/singularity/rpc/client"
)

// loopDevices is a mapping of loop device id -> associated dest in image. The
// dest should be given relative to the image root
var loopDevices map[int]string

func init() {
	loopDevices = make(map[int]string)
}

// CreateContainer creates a container
func (engine *EngineOperations) CreateContainer(rpcConn net.Conn) error {
	if engine.CommonConfig.EngineName != Name {
		return fmt.Errorf("engineName configuration doesn't match runtime name")
	}

	rpcOps := &client.RPC{
		Client: rpc.NewClient(rpcConn),
		Name:   engine.CommonConfig.EngineName,
	}
	if rpcOps.Client == nil {
		return fmt.Errorf("failed to initialiaze RPC client")
	}

	st, err := os.Stat(engine.EngineConfig.GetImage())
	if err != nil {
		return fmt.Errorf("stat on %s failed", engine.EngineConfig.GetImage())
	}

	rootfs := engine.EngineConfig.GetImage()

	userNS := false
	pidNS := false

	if engine.CommonConfig.OciConfig.Linux != nil {
		for _, namespace := range engine.CommonConfig.OciConfig.Linux.Namespaces {
			switch namespace.Type {
			case specs.UserNamespace:
				userNS = true
			case specs.PIDNamespace:
				pidNS = true
			}
		}
	}

	C.singularity_config_init()

	imageObject := C.singularity_image_init(C.CString(rootfs), 0)

	info := new(loop.Info64)
	mountType := ""

	switch C.singularity_image_type(&imageObject) {
	case 1:
		mountType = "squashfs"
		info.Offset = uint64(C.uint(imageObject.offset))
		info.SizeLimit = uint64(C.uint(imageObject.size))
	case 2:
		mountType = "ext3"
		info.Offset = uint64(C.uint(imageObject.offset))
		info.SizeLimit = uint64(C.uint(imageObject.size))
	}

	// Mount SIF default system partition
	if st.IsDir() == false && !userNS {
		var number int
		info.Flags = loop.FlagsAutoClear
		number, err = rpcOps.LoopDevice(rootfs, os.O_RDONLY, *info)
		if err != nil {
			return err
		}

		path := fmt.Sprintf("/dev/loop%d", number)
		sylog.Debugf("Mounting loop device %s\n", path)
		_, err = rpcOps.Mount(path, buildcfg.CONTAINER_FINALDIR, mountType, syscall.MS_NOSUID|syscall.MS_RDONLY|syscall.MS_NODEV, "errors=remount-ro")
		if err != nil {
			return fmt.Errorf("failed to mount %s filesystem: %s", mountType, err)
		}
	} else {
		sylog.Debugf("Mounting image directory %s\n", rootfs)
		_, err = rpcOps.Mount(rootfs, buildcfg.CONTAINER_FINALDIR, "", syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_RDONLY|syscall.MS_NODEV, "errors=remount-ro")
		if err != nil {
			return fmt.Errorf("failed to mount directory filesystem %s: %s", rootfs, err)
		}
	}

	if pidNS {
		sylog.Debugf("Mounting proc at %s\n", filepath.Join(buildcfg.CONTAINER_FINALDIR, "proc"))
		_, err = rpcOps.Mount("proc", filepath.Join(buildcfg.CONTAINER_FINALDIR, "proc"), "proc", syscall.MS_NOSUID, "")
		if err != nil {
			return fmt.Errorf("mount proc failed: %s", err)
		}
	} else {
		sylog.Debugf("Mounting proc at %s\n", filepath.Join(buildcfg.CONTAINER_FINALDIR, "proc"))
		_, err = rpcOps.Mount("/proc", filepath.Join(buildcfg.CONTAINER_FINALDIR, "proc"), "", syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_REC, "")
		if err != nil {
			return fmt.Errorf("mount proc failed: %s", err)
		}
	}
	if !userNS {
		sylog.Debugf("Mounting sysfs at %s\n", filepath.Join(buildcfg.CONTAINER_FINALDIR, "sys"))
		_, err = rpcOps.Mount("sysfs", filepath.Join(buildcfg.CONTAINER_FINALDIR, "sys"), "sysfs", syscall.MS_NOSUID, "")
		if err != nil {
			return fmt.Errorf("mount sys failed: %s", err)
		}
	} else {
		sylog.Debugf("Mounting sysfs at %s\n", filepath.Join(buildcfg.CONTAINER_FINALDIR, "sys"))
		_, err = rpcOps.Mount("/sys", filepath.Join(buildcfg.CONTAINER_FINALDIR, "sys"), "", syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_REC, "")
		if err != nil {
			return fmt.Errorf("mount sys failed: %s", err)
		}
	}

	sylog.Debugf("Mounting home at %s\n", filepath.Join(buildcfg.CONTAINER_FINALDIR, "home"))
	_, err = rpcOps.Mount("/home", filepath.Join(buildcfg.CONTAINER_FINALDIR, "home"), "", syscall.MS_BIND, "")
	if err != nil {
		return fmt.Errorf("mount /home failed: %s", err)
	}

	sylog.Debugf("Mounting dev at %s\n", filepath.Join(buildcfg.CONTAINER_FINALDIR, "dev"))
	_, err = rpcOps.Mount("/dev", filepath.Join(buildcfg.CONTAINER_FINALDIR, "dev"), "", syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_REC, "")
	if err != nil {
		return fmt.Errorf("mount /dev failed: %s", err)
	}

	sylog.Debugf("Mounting /etc/passwd at %s\n", filepath.Join(buildcfg.CONTAINER_FINALDIR, "etc/passwd"))
	_, err = rpcOps.Mount("/etc/passwd", filepath.Join(buildcfg.CONTAINER_FINALDIR, "etc/passwd"), "", syscall.MS_BIND, "")
	if err != nil {
		return fmt.Errorf("mount /etc/passwd failed: %s", err)
	}

	sylog.Debugf("Mounting /etc/group at %s\n", filepath.Join(buildcfg.CONTAINER_FINALDIR, "etc/group"))
	_, err = rpcOps.Mount("/etc/group", filepath.Join(buildcfg.CONTAINER_FINALDIR, "etc/group"), "", syscall.MS_BIND, "")
	if err != nil {
		return fmt.Errorf("mount /etc/group failed: %s", err)
	}

	// Mount SINGULARITY.D partition
	id, err := loopSifPartition(rpcOps, rootfs, imageObject, 2)
	if err != nil {
		return fmt.Errorf("mount filesystem failed: %v", err)
	}

	loopDevices[id] = ".singularity.d"
	if err := mountLoops(rpcOps, loopDevices); err != nil {
		return fmt.Errorf("loop mounts failed: %v", err)
	}

	sylog.Debugf("Mounting staging dir %s into final dir %s\n", buildcfg.CONTAINER_FINALDIR, buildcfg.SESSIONDIR)
	_, err = rpcOps.Mount(buildcfg.CONTAINER_FINALDIR, buildcfg.SESSIONDIR, "", syscall.MS_BIND|syscall.MS_REC, "")
	if err != nil {
		return fmt.Errorf("mount staging directory failed: %s", err)
	}

	sylog.Debugf("Chdir into %s\n", buildcfg.SESSIONDIR)
	err = syscall.Chdir(buildcfg.SESSIONDIR)
	if err != nil {
		return fmt.Errorf("change directory failed: %s", err)
	}

	sylog.Debugf("Chroot into %s\n", buildcfg.SESSIONDIR)
	_, err = rpcOps.Chroot(buildcfg.SESSIONDIR)
	if err != nil {
		return fmt.Errorf("chroot failed: %s", err)
	}

	sylog.Debugf("Chdir into / to avoid errors\n")
	err = syscall.Chdir("/")
	if err != nil {
		return fmt.Errorf("change directory failed: %s", err)
	}
	if err := rpcOps.Client.Close(); err != nil {
		return fmt.Errorf("can't close connection with rpc server: %s", err)
	}

	return nil
}

/* Most of the following work will be able to be replace by PR #1625 once that's merged */

// loopSifPartition opens a loop device and mounts the partition described
// by id of the SIF file at imageObject->path [c struct]
func loopSifPartition(rpcOps *client.RPC, rootfs string, imageObject C.struct_image_object, id int) (int, error) {
	info := new(loop.Info64)

	// get sif common struct from sif[id] partition
	sifCommon := C.sif_getpart(&imageObject, C.int(id))

	info.Offset = uint64(C.uint(sifCommon.fileoff))
	info.SizeLimit = uint64(C.uint(sifCommon.filelen))
	info.Flags = loop.FlagsAutoClear

	// ask RPC server to open loop device
	return rpcOps.LoopDevice(rootfs, os.O_RDONLY, *info)
}

// mountSifLoop takes a loop dev, destination path (in img), mount type, and flags; and
// performs the mount operation
func mountSifLoop(rpcOps *client.RPC, id int, rdest, mountType string, flags uintptr) error {
	path := fmt.Sprintf("/dev/loop%d", id)
	sylog.Debugf("Mounting loop device %s\n", path)

	dest := filepath.Join(buildcfg.CONTAINER_FINALDIR, rdest)

	_, err := rpcOps.Mount(path, dest, mountType, flags, "errors=remount-ro")
	if err != nil {
		return fmt.Errorf("failed to mount %s filesystem: %s", mountType, err)
	}

	return nil
}

// mountLoops takes the map of loop devices and mounts at their respective destinations
func mountLoops(rpcOps *client.RPC, loops map[int]string) error {
	mountType := "squashfs" //assuming squashfs partition for now
	var mountFlags uintptr = syscall.MS_NOSUID | syscall.MS_RDONLY | syscall.MS_NODEV
	for id, dest := range loops {
		if err := mountSifLoop(rpcOps, id, dest, mountType, mountFlags); err != nil {
			return err
		}
	}

	return nil
}
