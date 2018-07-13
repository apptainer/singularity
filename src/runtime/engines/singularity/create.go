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
	"strings"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/singularityware/singularity/src/pkg/buildcfg"
	"github.com/singularityware/singularity/src/pkg/image"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/fs/layout"
	"github.com/singularityware/singularity/src/pkg/util/fs/mount"
	"github.com/singularityware/singularity/src/pkg/util/loop"
	"github.com/singularityware/singularity/src/runtime/engines/singularity/rpc/client"
	"github.com/sylabs/sif/pkg/sif"
)

type mountTest struct {
	engine  *EngineOperations
	rpcOps  *client.RPC
	session *layout.Session
	overlay *layout.Overlay
}

func (m *mountTest) localMount(point *mount.Point) error {
	uid := os.Getuid()

	syscall.Setresuid(uid, 0, uid)
	defer syscall.Setresuid(uid, uid, 0)

	if _, err := mount.GetOffset(point.InternalOptions); err == nil {
		if err := mountImage(point); err != nil {
			return err
		}
	} else {
		if err := mountGeneric(nil, point); err != nil {
			return err
		}
	}
	return nil
}

func (m *mountTest) rpcMount(point *mount.Point) error {
	if err := mountGeneric(m.rpcOps, point); err != nil {
		return err
	}
	return nil
}

func (m *mountTest) createSessionLayout(system *mount.System) error {
	overlay, err := layout.NewOverlay(m.session)
	if err != nil {
		return err
	}

	m.overlay = overlay

	lowerdir := fmt.Sprintf("%s:%s", overlay.Path(), buildcfg.CONTAINER_MOUNTDIR)
	err = system.Points.AddOverlay(mount.OverlayLowerDirTag, buildcfg.CONTAINER_FINALDIR, syscall.MS_NOSUID|syscall.MS_NODEV, lowerdir, "", "")
	if err != nil {
		return err
	}
	return nil
}

func (m *mountTest) switchMount(system *mount.System) error {
	system.Mount = m.rpcMount
	return nil
}

func (m *mountTest) createOverlayTmp(system *mount.System) error {
	points := system.Points.GetByTag(mount.RootfsTag)
	if len(points) != 1 {
		return fmt.Errorf("no root fs image found")
	}
	return m.overlay.CreateLayout(points[0].Destination, system.Points)
}

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
	session, err := layout.NewSession(buildcfg.SESSIONDIR)
	if err != nil {
		return err
	}
	sessionPath, _ := session.GetPath("/")

	err = p.AddFS(mount.SessionTag, sessionPath, "tmpfs", syscall.MS_NOSUID|syscall.MS_NODEV, "mode=1777")
	if err != nil {
		return err
	}

	mt := &mountTest{engine: engine, rpcOps: rpcOps, session: session}

	system := &mount.System{Points: p, Mount: mt.localMount}
	if err := system.RunAfterTag(mount.SessionTag, mt.createSessionLayout); err != nil {
		return err
	}
	if err := system.RunAfterTag(mount.RootfsTag, mt.switchMount); err != nil {
		return err
	}
	if err := system.RunAfterTag(mount.OverlayTag, mt.createOverlayTmp); err != nil {
		return err
	}

	if err := engine.addRootfs(p); err != nil {
		return err
	}

	sylog.Debugf("Adding proc to mount list\n")
	if pidNS {
		err = p.AddFS(mount.KernelTag, "/proc", "proc", syscall.MS_NOSUID, "")
	} else {
		err = p.AddBind(mount.KernelTag, "/proc", "/proc", syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_REC)
	}
	if err != nil {
		return fmt.Errorf("unable to add proc to mount list: %s", err)
	}

	sylog.Debugf("Adding sysfs to mount list\n")
	if !userNS {
		err = p.AddFS(mount.KernelTag, "/sys", "sysfs", syscall.MS_NOSUID, "")
	} else {
		err = p.AddBind(mount.KernelTag, "/sys", "/sys", syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_REC)
	}
	if err != nil {
		return fmt.Errorf("unable to add sys to mount list: %s", err)
	}

	sylog.Debugf("Adding home to mount list\n")
	err = p.AddBind(mount.HomeTag, "/home", "/home", syscall.MS_BIND)
	if err != nil {
		return fmt.Errorf("unable to add home to mount list: %s", err)
	}

	sylog.Debugf("Adding dev to mount list\n")
	err = p.AddBind(mount.DevTag, "/dev", "/dev", syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_REC)
	if err != nil {
		return fmt.Errorf("unable to add dev to mount list: %s", err)
	}

	sylog.Debugf("Adding /etc/passwd to mount list\n")
	err = p.AddBind(mount.FilesTag, "/etc/passwd", "/etc/passwd", syscall.MS_BIND)
	if err != nil {
		return fmt.Errorf("unable to add /etc/passwd to mount list: %s", err)
	}

	sylog.Debugf("Adding /etc/group to mount list\n")
	err = p.AddBind(mount.FilesTag, "/etc/group", "/etc/group", syscall.MS_BIND)
	if err != nil {
		return fmt.Errorf("unable to add /etc/group to mount list: %s", err)
	}

	for _, b := range engine.EngineConfig.GetBindPath() {
		splitted := strings.Split(b, ":")
		l := len(splitted)
		if l == 1 {
			sylog.Debugf("Adding %s to mount list\n", splitted[0])
			if err := p.AddBind(mount.UserbindsTag, splitted[0], splitted[0], syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_REC); err != nil {
				return fmt.Errorf("unabled to %s to mount list: %s", splitted[0], err)
			}
		} else {
			sylog.Debugf("Adding %s to mount list\n", splitted[0])
			if err := p.AddBind(mount.UserbindsTag, splitted[0], splitted[1], syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_REC); err != nil {
				return fmt.Errorf("unabled to %s to mount list: %s", splitted[0], err)
			}
		}
	}

	sylog.Debugf("Add")
	if err := system.MountAll(); err != nil {
		return err
	}

	sylog.Debugf("Chdir into %s\n", buildcfg.CONTAINER_FINALDIR)
	err = syscall.Chdir(buildcfg.CONTAINER_FINALDIR)
	if err != nil {
		return fmt.Errorf("change directory failed: %s", err)
	}

	sylog.Debugf("Chroot into %s\n", buildcfg.CONTAINER_FINALDIR)
	_, err = rpcOps.Chroot(buildcfg.CONTAINER_FINALDIR)
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
		return p.AddBind(mount.RootfsTag, rootfs, buildcfg.CONTAINER_MOUNTDIR, syscall.MS_BIND|flags)
	}

	sylog.Debugf("Mounting block [%v] image: %v\n", mountType, rootfs)
	return p.AddImage(mount.RootfsTag, rootfs, buildcfg.CONTAINER_MOUNTDIR, mountType, flags, imageObject.Offset, imageObject.Size)
}

// mount any generic mount (not loop dev)
func mountGeneric(rpcOps *client.RPC, mnt *mount.Point) (err error) {
	flags, opts := mount.ConvertOptions(mnt.Options)
	optsString := strings.Join(opts, ",")

	dest := ""
	if mnt.Destination != buildcfg.CONTAINER_FINALDIR && mnt.Destination != buildcfg.CONTAINER_MOUNTDIR && mnt.Destination != buildcfg.SESSIONDIR {
		dest = buildcfg.CONTAINER_FINALDIR + mnt.Destination
	} else {
		dest = mnt.Destination
	}
	sylog.Debugf("Mounting %s to %s\n", mnt.Source, dest)
	if rpcOps != nil {
		_, err = rpcOps.Mount(mnt.Source, dest, mnt.Type, flags, optsString)
	} else {
		err = syscall.Mount(mnt.Source, dest, mnt.Type, flags, optsString)
	}
	return err
}

// mount image via loop
func mountImage(mnt *mount.Point) error {
	flags, opts := mount.ConvertOptions(mnt.Options)
	optsString := strings.Join(opts, ",")

	offset, err := mount.GetOffset(mnt.InternalOptions)
	if err != nil {
		return err
	}

	sizelimit, err := mount.GetSizeLimit(mnt.InternalOptions)
	if err != nil {
		return err
	}

	info := &loop.Info64{
		Offset:    offset,
		SizeLimit: sizelimit,
		Flags:     loop.FlagsAutoClear,
	}

	loopdev := new(loop.Device)

	number := 0

	if err := loopdev.Attach(mnt.Source, os.O_RDONLY, &number); err != nil {
		return err
	}
	if err := loopdev.SetStatus(info); err != nil {
		return err
	}

	path := fmt.Sprintf("/dev/loop%d", number)
	sylog.Debugf("Mounting loop device %s to %s\n", path, mnt.Destination)
	err = syscall.Mount(path, mnt.Destination, mnt.Type, flags, optsString)
	if err != nil {
		return fmt.Errorf("failed to mount %s filesystem: %s", mnt.Type, err)
	}

	return nil
}
