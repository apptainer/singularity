// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oci

import (
	"encoding/json"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	specs "github.com/opencontainers/image-spec/specs-go"
	"github.com/sylabs/singularity/src/pkg/buildcfg"
	"github.com/sylabs/singularity/src/pkg/cgroups"
	"github.com/sylabs/singularity/src/pkg/instance"
	"github.com/sylabs/singularity/src/pkg/sylog"
	"github.com/sylabs/singularity/src/pkg/util/fs/layout"
	"github.com/sylabs/singularity/src/pkg/util/fs/layout/layer/overlay"
	"github.com/sylabs/singularity/src/pkg/util/fs/mount"
	"github.com/sylabs/singularity/src/runtime/engines/singularity/rpc/client"
)

type container struct {
	engine  *EngineOperations
	rpcOps  *client.RPC
	session *layout.Session
	rootfs  string
}

func (engine *EngineOperations) createState(pid int) error {
	name := engine.CommonConfig.ContainerID

	file, err := instance.Add(name, true)
	if err != nil {
		return err
	}

	engine.EngineConfig.State.Version = specs.Version
	engine.EngineConfig.State.Bundle = engine.EngineConfig.GetBundlePath()
	engine.EngineConfig.State.ID = engine.CommonConfig.ContainerID
	engine.EngineConfig.State.Pid = pid
	engine.EngineConfig.State.Status = "creating"

	file.Config, err = json.Marshal(engine.CommonConfig)
	if err != nil {
		return err
	}

	file.User = "root"
	file.Pid = pid
	file.PPid = os.Getpid()
	file.Image = filepath.Join(engine.EngineConfig.GetBundlePath(), engine.EngineConfig.OciConfig.Root.Path)

	return file.Update()
}

func (engine *EngineOperations) updateState(status string) error {
	file, err := instance.Get(engine.CommonConfig.ContainerID)
	if err != nil {
		return err
	}

	engine.EngineConfig.State.Status = status

	file.Config, err = json.Marshal(engine.CommonConfig)
	if err != nil {
		return err
	}

	return file.Update()
}

// CreateContainer creates a container
func (engine *EngineOperations) CreateContainer(pid int, rpcConn net.Conn) error {
	var err error

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

	if err := engine.createState(pid); err != nil {
		return err
	}

	rootfs := filepath.Join(engine.EngineConfig.GetBundlePath(), engine.EngineConfig.OciConfig.Root.Path)

	c := &container{
		engine: engine,
		rpcOps: rpcOps,
		rootfs: rootfs,
	}

	p := &mount.Points{}
	if engine.EngineConfig.OciConfig.Linux.MountLabel != "" {
		if err := p.SetContext(engine.EngineConfig.OciConfig.Linux.MountLabel); err != nil {
			return err
		}
	}

	system := &mount.System{Points: p, Mount: c.mount}

	// setup overlay layout sets up the session with overlay filesystem
	sylog.Debugf("Creating overlay SESSIONDIR layout\n")
	if c.session, err = layout.NewSession(buildcfg.SESSIONDIR, "tmpfs", -1, system, overlay.New()); err != nil {
		return err
	}

	manager := &cgroups.Manager{Pid: pid, Name: engine.CommonConfig.ContainerID}
	if err := manager.ApplyFromSpec(engine.EngineConfig.OciConfig.Linux.Resources); err != nil {
		return fmt.Errorf("Failed to apply cgroups ressources restriction: %s", err)
	}
	engine.EngineConfig.Cgroups = manager

	// import OCI mount spec
	if err := p.ImportFromSpec(engine.EngineConfig.OciConfig.Config.Mounts); err != nil {
		return err
	}

	// add masked path
	if err := p.AddMaskedPaths(engine.EngineConfig.OciConfig.Linux.MaskedPaths); err != nil {
		return err
	}

	// add read-only path
	if err := p.AddReadonlyPaths(engine.EngineConfig.OciConfig.Linux.ReadonlyPaths); err != nil {
		return err
	}

	if err := c.addOverlayMount(system); err != nil {
		return err
	}

	if err := c.addRootfsMount(system); err != nil {
		return err
	}

	sylog.Debugf("Mount all")
	if err := system.MountAll(); err != nil {
		return err
	}

	sylog.Debugf("Set RPC mount propagation flag to SLAVE")
	if _, err := rpcOps.Mount("", "/", "", syscall.MS_SLAVE|syscall.MS_REC, ""); err != nil {
		return err
	}

	_, err = rpcOps.Chroot(c.session.FinalPath(), true)
	if err != nil {
		return fmt.Errorf("chroot failed: %s", err)
	}

	if engine.EngineConfig.SlavePts != -1 {
		if err := syscall.Close(engine.EngineConfig.SlavePts); err != nil {
			return err
		}
	}

	return nil
}

func (c *container) addOverlayMount(system *mount.System) error {
	ov := c.session.Layer.(*overlay.Overlay)

	sylog.Debugf("Setup writable tmpfs overlay")

	if err := c.session.AddDir("/upper"); err != nil {
		return err
	}
	if err := c.session.AddDir("/work"); err != nil {
		return err
	}

	upper, _ := c.session.GetPath("/upper")
	work, _ := c.session.GetPath("/work")

	if err := ov.SetUpperDir(upper); err != nil {
		return fmt.Errorf("failed to add overlay upper: %s", err)
	}
	if err := ov.SetWorkDir(work); err != nil {
		return fmt.Errorf("failed to add overlay upper: %s", err)
	}

	return nil
}

func (c *container) addRootfsMount(system *mount.System) error {
	flags := uintptr(syscall.MS_BIND | syscall.MS_REC)
	if c.engine.EngineConfig.OciConfig.Root.Readonly {
		flags |= syscall.MS_RDONLY
	}
	if err := system.Points.AddBind(mount.RootfsTag, c.rootfs, c.session.RootFsPath(), flags); err != nil {
		return err
	}
	if flags&syscall.MS_RDONLY != 0 {
		return system.Points.AddRemount(mount.RootfsTag, c.session.RootFsPath(), flags)
	}
	return nil
}

func (c *container) mount(point *mount.Point) error {
	sylog.Debugf("mount %s to %s : %s %s", point.Source, point.Destination, point.Type, point.Options)
	source := point.Source
	dest := point.Destination
	flags, opts := mount.ConvertOptions(point.Options)
	optsString := strings.Join(opts, ",")
	sessionPath := c.session.Path()
	remount := false

	if flags&syscall.MS_REMOUNT != 0 {
		remount = true
	}

	if !strings.HasPrefix(dest, sessionPath) {
		dest = filepath.Join(c.session.FinalPath(), dest)

		if _, err := os.Stat(dest); os.IsNotExist(err) {
			if !remount {
				if point.Type != "" {
					if err := os.MkdirAll(dest, 0755); err != nil {
						return err
					}
				} else {
					dir := filepath.Dir(dest)
					if err := os.MkdirAll(dir, 0755); err != nil {
						return err
					}
				}
			}
		}
	} else {
		if _, err := os.Stat(dest); os.IsNotExist(err) {
			return fmt.Errorf("destination %s doesn't exist", dest)
		}
	}

	_, err := c.rpcOps.Mount(source, dest, point.Type, flags, optsString)
	return err
}
