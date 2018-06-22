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
	"strconv"
	"strings"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/singularityware/singularity/src/pkg/buildcfg"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/fs/mount"
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

	_, err := os.Stat(engine.EngineConfig.GetImage())
	if err != nil {
		return fmt.Errorf("stat on %s failed", engine.EngineConfig.GetImage())
	}

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

	p := &mount.Points{}
	if err := engine.addRootfs(p); err != nil {
		return err
	}

	sylog.Debugf("Adding proc to mount list\n")
	if pidNS {
		err = p.AddFS("proc", filepath.Join(buildcfg.CONTAINER_FINALDIR, "proc"), "proc", syscall.MS_NOSUID, "")
	} else {
		err = p.AddBind("proc", "/proc", filepath.Join(buildcfg.CONTAINER_FINALDIR, "proc"), syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_REC)
	}
	if err != nil {
		return fmt.Errorf("unable to add proc to mount list: %s", err)
	}

	sylog.Debugf("Adding sysfs to mount list\n")
	if !userNS {
		err = p.AddFS("sysfs", filepath.Join(buildcfg.CONTAINER_FINALDIR, "sys"), "sysfs", syscall.MS_NOSUID, "")
	} else {
		err = p.AddBind("sysfs", "/sys", filepath.Join(buildcfg.CONTAINER_FINALDIR, "sys"), syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_REC)
	}
	if err != nil {
		return fmt.Errorf("unable to add sys to mount list: %s", err)
	}

	sylog.Debugf("Adding home to mount list\n")
	err = p.AddBind("home", "/home", filepath.Join(buildcfg.CONTAINER_FINALDIR, "home"), syscall.MS_BIND)
	if err != nil {
		return fmt.Errorf("unable to add home to mount list: %s", err)
	}

	sylog.Debugf("Adding dev to mount list\n")
	err = p.AddBind("dev", "/dev", filepath.Join(buildcfg.CONTAINER_FINALDIR, "dev"), syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_REC)
	if err != nil {
		return fmt.Errorf("unable to add dev to mount list: %s", err)
	}

	sylog.Debugf("Adding /etc/passwd to mount list\n")
	err = p.AddBind("passwd", "/etc/passwd", filepath.Join(buildcfg.CONTAINER_FINALDIR, "etc/passwd"), syscall.MS_BIND)
	if err != nil {
		return fmt.Errorf("unable to add /etc/passwd to mount list: %s", err)
	}

	sylog.Debugf("Adding /etc/group to mount list\n")
	err = p.AddBind("group", "/etc/group", filepath.Join(buildcfg.CONTAINER_FINALDIR, "etc/group"), syscall.MS_BIND)
	if err != nil {
		return fmt.Errorf("unable to add /etc/group to mount list: %s", err)
	}

	sylog.Debugf("Adding user binds to mount list\n")
	err = p.Import(engine.CommonConfig.OciConfig.Spec.Mounts)
	if err != nil {
		return fmt.Errorf("unable to add user bind mounts to mount list: %s", err)
	}

	sylog.Debugf("Adding staging dir -> final dir to mount list\n")
	err = p.AddBind("final", buildcfg.CONTAINER_FINALDIR, buildcfg.SESSIONDIR, syscall.MS_BIND|syscall.MS_REC)
	if err != nil {
		return fmt.Errorf("unable to add final staging dir to mount list: %s", err)
	}

	if err := mountAll(rpcOps, p); err != nil {
		return err
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

// func add* are specific for engine to store rootfs info (could be extended to generic helper).
// these funcs create mount.Points struct full for later mount
func (engine *EngineOperations) addSifPartition(p *mount.Points, imageObject C.struct_image_object, id int, dest string, flags uintptr) error {
	// get sif common struct from sif[id] partition
	sifCommon := C.sif_getpart(&imageObject, C.int(id))

	partitionOffset := uint64(C.uint(sifCommon.fileoff))
	partitionSizelimit := uint64(C.uint(sifCommon.filelen))

	idString := strconv.Itoa(id)
	err := p.AddImage(idString, engine.EngineConfig.GetImage(), dest, "squashfs", flags, partitionOffset, partitionSizelimit)
	if err != nil {
		return err
	}

	return nil
}

func (engine *EngineOperations) addRootfs(p *mount.Points) error {
	imagePath := engine.EngineConfig.GetImage()
	imageObject := C.singularity_image_init(C.CString(imagePath), 0)
	var flags uintptr = syscall.MS_NOSUID | syscall.MS_RDONLY | syscall.MS_NODEV

	sylog.Debugf("img type: %v\n", C.singularity_image_type(&imageObject))

	switch C.singularity_image_type(&imageObject) {
	case 1:
		mountType := "squashfs"
		rootfsOffset := uint64(C.uint(imageObject.offset))
		rootfsFilelen := uint64(C.uint(imageObject.size))

		sylog.Debugf("Add squashfs as rootfs\n")
		err := p.AddImage("rootfs", imagePath, buildcfg.CONTAINER_FINALDIR, mountType, flags, rootfsOffset, rootfsFilelen)
		if err != nil {
			return err
		}
	case 2:
		mountType := "ext3"
		rootfsOffset := uint64(C.uint(imageObject.offset))
		rootfsFilelen := uint64(C.uint(imageObject.size))

		sylog.Debugf("Add ext3 as rootfs\n")
		err := p.AddImage("rootfs", imagePath, buildcfg.CONTAINER_FINALDIR, mountType, flags, rootfsOffset, rootfsFilelen)
		if err != nil {
			return err
		}
	case 3:
		sylog.Debugf("Add dir as rootfs\n")
		// Add directory rootfs as bind. Can directly return because a dir will not have multiple sif partition
		return p.AddBind("rootfs", imagePath, buildcfg.CONTAINER_FINALDIR, syscall.MS_BIND|flags)
	}

	sylog.Debugf("Adding SIF partition 2 to mount list at .singularity.d\n")
	err := engine.addSifPartition(p, imageObject, 2, filepath.Join(buildcfg.CONTAINER_FINALDIR, ".singularity.d"), flags)
	if err != nil {
		return fmt.Errorf("unable to add .singularity.d partition to mount list: %s", err)
	}

	return nil
}

func (engine *EngineOperations) addUserBinds(p *mount.Points) error {
	newMounts := []specs.Mount{}
	for _, mnt := range engine.CommonConfig.OciConfig.Spec.Mounts {
		if !strings.Contains(mnt.Destination, buildcfg.CONTAINER_FINALDIR) {
			mnt.Destination = filepath.Join(buildcfg.CONTAINER_FINALDIR, mnt.Destination)
		}

		newMounts = append(newMounts, mnt)
	}

	return p.Import(newMounts)
}

func mountAll(rpcOps *client.RPC, p *mount.Points) error {
	// first mount rootfs
	if err := mountRootfs(rpcOps, p); err != nil {
		return err
	}

	for _, mnt := range p.GetAll()[1:] { // rootfs is always idx=0, so skip that
		_, _, iopts := mount.ConvertOptions(mnt.Options)

		// if GetOffset succeeds, image needs a loop device
		if _, err := mount.GetOffset(iopts); err == nil {
			if err := mountImage(rpcOps, mnt); err != nil {
				return err
			}

			continue
		}

		if err := mountGeneric(rpcOps, mnt); err != nil {
			return err
		}
	}

	return nil
}

// mount rootfs partition
func mountRootfs(rpcOps *client.RPC, p *mount.Points) error {
	mnts := p.GetByName("rootfs")
	sylog.Debugf("len rootfs: %v\n", len(mnts))
	mnt := mnts[0]
	flags, _, _ := mount.ConvertOptions(mnt.Options)

	if flags&syscall.MS_BIND != 0 { // if bind mount, the rootfs is a directory
		return mountGeneric(rpcOps, mnt)
	}

	// rootfs is an image
	return mountImage(rpcOps, mnt)
}

// mount any generic mount (not loop dev)
func mountGeneric(rpcOps *client.RPC, mnt specs.Mount) error {
	flags, opts, _ := mount.ConvertOptions(mnt.Options)
	optsString := ""
	for _, opt := range opts {
		optsString = optsString + "," + opt
	}

	sylog.Debugf("Mounting %s to %s\n", mnt.Source, mnt.Destination)
	_, err := rpcOps.Mount(mnt.Source, mnt.Destination, mnt.Type, flags, optsString)
	return err
}

// mount image via loop
func mountImage(rpcOps *client.RPC, mnt specs.Mount) error {
	flags, opts, iopts := mount.ConvertOptions(mnt.Options)

	optsString := ""
	for _, opt := range opts {
		optsString = optsString + "," + opt
	}

	offset, err := mount.GetOffset(iopts)
	if err != nil {
		return err
	}

	sizelimit, err := mount.GetSizeLimit(iopts)
	if err != nil {
		return err
	}

	info := &loop.Info64{
		Offset:    offset,
		SizeLimit: sizelimit,
		Flags:     loop.FlagsAutoClear,
	}

	number, err := rpcOps.LoopDevice(mnt.Source, os.O_RDONLY, *info)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/dev/loop%d", number)
	sylog.Debugf("Mounting loop device %s to %s\n", path, mnt.Destination)
	_, err = rpcOps.Mount(path, mnt.Destination, mnt.Type, flags, optsString)
	if err != nil {
		return fmt.Errorf("failed to mount %s filesystem: %s", mnt.Type, err)
	}

	return nil
}
