// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/singularityware/singularity/src/pkg/buildcfg"
	"github.com/singularityware/singularity/src/pkg/image"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/fs/mount"
	"github.com/singularityware/singularity/src/pkg/util/loop"
	"github.com/singularityware/singularity/src/runtime/engines/singularity/rpc/client"
	"github.com/sylabs/sif/pkg/sif"
)

var sharedLoopDevices bool
var maxLoopDevices uint

// CreateContainer creates a container
func (engine *EngineOperations) CreateContainer(pid int, rpcConn net.Conn) error {
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

	sharedLoopDevices = engine.EngineConfig.File.SharedLoopDevices
	maxLoopDevices = engine.EngineConfig.File.MaxLoopDevices

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

	return nil
}

func (engine *EngineOperations) addRootfs(p *mount.Points) error {
	var flags uintptr = syscall.MS_NOSUID | syscall.MS_RDONLY | syscall.MS_NODEV
	rootfs := engine.EngineConfig.GetImage()

	imageObject, err := image.Init(rootfs, false)
	if err != nil {
		return err
	}

	mountType := ""

	switch imageObject.Type {
	case image.SIF:
		// Load the SIF file
		fimg, err := sif.LoadContainerFp(imageObject.File, !imageObject.Writable)
		if err != nil {
			return err
		}

		// Get the default system partition image
		part, _, err := fimg.GetPartFromGroup(sif.DescrDefaultGroup)
		if err != nil {
			return err
		}

		// Check that this is a system partition
		parttype, err := part.GetPartType()
		if err != nil {
			return err
		}
		if parttype != sif.PartSystem {
			return fmt.Errorf("found partition is not system")
		}

		// record the fs type
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

		imageObject.Offset = uint64(part.Fileoff)
		imageObject.Size = uint64(part.Filelen)
	case image.SQUASHFS:
		mountType = "squashfs"
	case image.EXT3:
		mountType = "ext3"
	case image.SANDBOX:
		sylog.Debugf("Mounting directory rootfs: %v\n", rootfs)
		return p.AddBind("rootfs", rootfs, buildcfg.CONTAINER_FINALDIR, syscall.MS_BIND|flags)
	}

	sylog.Debugf("Mounting block [%v] image: %v\n", mountType, rootfs)
	return p.AddImage("rootfs", rootfs, buildcfg.CONTAINER_FINALDIR, mountType, flags, imageObject.Offset, imageObject.Size)
}

func (engine *EngineOperations) addUserBinds(p *mount.Points) error {
	newMounts := []specs.Mount{}
	for _, mnt := range engine.CommonConfig.OciConfig.Spec.Mounts {
		if !strings.Contains(mnt.Destination, buildcfg.CONTAINER_FINALDIR) {
			mnt.Destination = filepath.Join(buildcfg.CONTAINER_FINALDIR, mnt.Destination)
		}

		sylog.Debugf("Adding user bind request %s : %s to mount list\n")
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
	sylog.Debugf("Adding rootfs to mount list\n")
	mnt := p.GetByName("rootfs")[0]
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
	optsString := strings.Join(opts, ",")

	sylog.Debugf("Mounting %s to %s\n", mnt.Source, mnt.Destination)
	_, err := rpcOps.Mount(mnt.Source, mnt.Destination, mnt.Type, flags, optsString)
	return err
}

// mount image via loop
func mountImage(rpcOps *client.RPC, mnt specs.Mount) error {
	flags, opts, iopts := mount.ConvertOptions(mnt.Options)
	optsString := strings.Join(opts, ",")

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

	sylog.Debugf("Mounting %v to loop device from %v - %v\n", mnt.Source, offset, sizelimit)
	number, err := rpcOps.LoopDevice(mnt.Source, *info, sharedLoopDevices, maxLoopDevices)
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
