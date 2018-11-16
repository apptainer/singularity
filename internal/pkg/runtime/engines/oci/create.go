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
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/internal/pkg/util/unix"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/internal/pkg/cgroups"
	"github.com/sylabs/singularity/internal/pkg/instance"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/singularity/rpc/client"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/fs/mount"
	"github.com/sylabs/singularity/internal/pkg/util/fs/proc"
)

type container struct {
	engine     *EngineOperations
	rpcOps     *client.RPC
	rootfs     string
	rpcRoot    string
	bindDev    bool
	bindCgroup bool
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

	if engine.EngineConfig.State.Annotations == nil {
		engine.EngineConfig.State.Annotations = make(map[string]string)
	}

	file.Config, err = json.Marshal(engine.CommonConfig)
	if err != nil {
		return err
	}

	file.User = "root"
	file.Pid = pid
	file.PPid = os.Getpid()
	file.Image = filepath.Join(engine.EngineConfig.GetBundlePath(), engine.EngineConfig.OciConfig.Root.Path)

	socketPath := engine.EngineConfig.SyncSocket

	if socketPath != "" {
		data, err := json.Marshal(engine.EngineConfig.State)
		if err != nil {
			sylog.Warningf("failed to marshal state data: %s", err)
		} else if err := unix.WriteSocket(socketPath, data); err != nil {
			sylog.Warningf("%s", err)
		}
	}

	return file.Update()
}

func (engine *EngineOperations) updateState(status string) error {
	file, err := instance.Get(engine.CommonConfig.ContainerID)
	if err != nil {
		return err
	}

	engine.EngineConfig.State.Status = status

	t := time.Now().UnixNano()

	switch status {
	case "created":
		engine.EngineConfig.State.Annotations["io.sylabs.runtime.oci.created_at"] = strconv.FormatInt(t, 10)
	case "running":
		engine.EngineConfig.State.Annotations["io.sylabs.runtime.oci.starter_at"] = strconv.FormatInt(t, 10)
	case "stopped":
		engine.EngineConfig.State.Annotations["io.sylabs.runtime.oci.finished_at"] = strconv.FormatInt(t, 10)
	}

	file.Config, err = json.Marshal(engine.CommonConfig)
	if err != nil {
		return err
	}

	socketPath := engine.EngineConfig.SyncSocket

	if socketPath != "" {
		data, err := json.Marshal(engine.EngineConfig.State)
		if err != nil {
			sylog.Warningf("failed to marshal state data: %s", err)
		} else if err := unix.WriteSocket(socketPath, data); err != nil {
			sylog.Warningf("%s", err)
		}
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

	rootfs := engine.EngineConfig.OciConfig.Root.Path

	if !filepath.IsAbs(rootfs) {
		rootfs = filepath.Join(engine.EngineConfig.GetBundlePath(), rootfs)
	}

	resolvedRootfs, err := filepath.EvalSymlinks(rootfs)
	if err != nil {
		return fmt.Errorf("failed to resolve %s path: %s", rootfs, err)
	}

	c := &container{
		engine:  engine,
		rpcOps:  rpcOps,
		rootfs:  resolvedRootfs,
		rpcRoot: fmt.Sprintf("/proc/%d/root", pid),
	}

	p := &mount.Points{}
	if engine.EngineConfig.OciConfig.Linux.MountLabel != "" {
		if err := p.SetContext(engine.EngineConfig.OciConfig.Linux.MountLabel); err != nil {
			return err
		}
	}

	system := &mount.System{Points: p, Mount: c.mount}

	manager := &cgroups.Manager{Pid: pid, Name: engine.CommonConfig.ContainerID}
	if err := manager.ApplyFromSpec(engine.EngineConfig.OciConfig.Linux.Resources); err != nil {
		return fmt.Errorf("Failed to apply cgroups ressources restriction: %s", err)
	}
	engine.EngineConfig.Cgroups = manager

	// import OCI mount spec
	if err := system.Points.ImportFromSpec(engine.EngineConfig.OciConfig.Config.Mounts); err != nil {
		return err
	}

	for _, point := range system.Points.GetByTag(mount.DevTag) {
		if point.Destination == "/dev" && point.Type == "" {
			flags, _ := mount.ConvertOptions(point.Options)
			if flags&syscall.MS_REC != 0 {
				c.bindDev = true
			}
			break
		}
	}

	for _, point := range system.Points.GetByTag(mount.KernelTag) {
		if point.Type == "cgroup" {
			c.bindCgroup = true
			break
		}
	}

	if err := c.addRootfsMount(system); err != nil {
		return err
	}

	if !c.bindDev {
		if err := system.RunAfterTag(mount.DevTag, c.addDevices); err != nil {
			return err
		}
	}

	if err := system.RunAfterTag(mount.RootfsTag, c.addAllPaths); err != nil {
		return err
	}

	sylog.Debugf("Mount all")
	if err := system.MountAll(); err != nil {
		return err
	}

	_, err = rpcOps.Chroot(c.rootfs, true)
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

func (c *container) addAllPaths(system *mount.System) error {
	// add masked path
	if err := c.addMaskedPathsMount(system); err != nil {
		return err
	}

	// add read-only path
	if err := c.addReadonlyPathsMount(system); err != nil {
		return err
	}

	return nil
}

func (c *container) addRootfsMount(system *mount.System) error {
	flags := uintptr(syscall.MS_BIND)

	if c.engine.EngineConfig.OciConfig.Root.Readonly {
		sylog.Debugf("Mounted read-only")
		flags |= syscall.MS_RDONLY
	}

	parentRootfs, err := proc.ParentMount(c.rootfs)
	if err != nil {
		return err
	}

	sylog.Debugf("Parent rootfs: %s", parentRootfs)

	if _, err := c.rpcOps.Mount("", parentRootfs, "", syscall.MS_PRIVATE, ""); err != nil {
		return err
	}
	if err := system.Points.AddBind(mount.RootfsTag, c.rootfs, c.rootfs, flags); err != nil {
		return err
	}
	if flags&syscall.MS_RDONLY != 0 {
		return system.Points.AddRemount(mount.FinalTag, c.rootfs, flags)
	}

	return nil
}

func (c *container) addDefaultDevices(system *mount.System) error {
	oldmask := syscall.Umask(0)
	defer syscall.Umask(oldmask)

	devPath := filepath.Join(c.rpcRoot, c.rootfs, "dev")
	if _, err := os.Stat(devPath); os.IsNotExist(err) {
		if err := os.Mkdir(devPath, 0755); err != nil {
			return err
		}
	}

	for _, symlink := range []struct {
		old string
		new string
	}{
		{"/proc/self/fd", "/dev/fd"},
		{"/proc/kcore", "/dev/core"},
		{"pts/ptmx", "/dev/ptmx"},
		{"/proc/self/fd/0", "/dev/stdin"},
		{"/proc/self/fd/1", "/dev/stdout"},
		{"/proc/self/fd/2", "/dev/stderr"},
	} {
		path := filepath.Join(c.rpcRoot, c.rootfs, symlink.new)
		if err := os.Symlink(symlink.old, path); err != nil {
			return err
		}
	}

	if c.engine.EngineConfig.OciConfig.Process.Terminal {
		path := filepath.Join(c.rpcRoot, c.rootfs, "dev", "console")
		if err := fs.Touch(path); err != nil {
			return err
		}
		path = fmt.Sprintf("/proc/self/fd/%d", c.engine.EngineConfig.SlavePts)
		console, err := os.Readlink(path)
		if err != nil {
			return err
		}
		if err := system.Points.AddBind(mount.OtherTag, console, "/dev/console", syscall.MS_BIND); err != nil {
			return err
		}
	}

	for _, charDevice := range []struct {
		major uint16
		minor uint16
		path  string
	}{
		{1, 7, "/dev/full"},
		{1, 3, "/dev/null"},
		{1, 8, "/dev/random"},
		{5, 0, "/dev/tty"},
		{1, 9, "/dev/urandom"},
		{1, 5, "/dev/zero"},
	} {
		dev := int((charDevice.major << 8) | charDevice.minor)
		path := filepath.Join(c.rpcRoot, c.rootfs, charDevice.path)
		if err := syscall.Mknod(path, syscall.S_IFCHR|0666, dev); err != nil {
			return err
		}
	}
	return nil
}

func (c *container) addDevices(system *mount.System) error {
	if err := c.addDefaultDevices(system); err != nil {
		return err
	}
	return nil
}

func (c *container) addMaskedPathsMount(system *mount.System) error {
	paths := c.engine.EngineConfig.OciConfig.Linux.MaskedPaths

	nullPath := filepath.Join(c.engine.EngineConfig.GetBundlePath(), "null")

	if _, err := os.Stat(nullPath); os.IsNotExist(err) {
		oldmask := syscall.Umask(0)
		defer syscall.Umask(oldmask)

		if err := os.Mkdir(nullPath, 0755); err != nil {
			return err
		}
	}

	for _, path := range paths {
		fi, err := os.Stat(path)
		if err != nil {
			sylog.Debugf("ignoring masked path %s: %s", path, err)
			continue
		}
		if fi.IsDir() {
			if err := system.Points.AddBind(mount.OtherTag, nullPath, path, syscall.MS_BIND); err != nil {
				return err
			}
		} else if err := system.Points.AddBind(mount.OtherTag, "/dev/null", path, syscall.MS_BIND); err != nil {
			return err
		}
	}
	return nil
}

func (c *container) addReadonlyPathsMount(system *mount.System) error {
	paths := c.engine.EngineConfig.OciConfig.Linux.ReadonlyPaths

	for _, path := range paths {
		if err := system.Points.AddBind(mount.OtherTag, path, path, syscall.MS_BIND|syscall.MS_RDONLY); err != nil {
			return err
		}
		if err := system.Points.AddRemount(mount.OtherTag, path, syscall.MS_BIND|syscall.MS_RDONLY); err != nil {
			return err
		}
	}
	return nil
}

func (c *container) mount(point *mount.Point) error {
	source := point.Source
	dest := point.Destination
	flags, opts := mount.ConvertOptions(point.Options)
	optsString := strings.Join(opts, ",")
	ignore := false

	if flags&syscall.MS_REMOUNT != 0 {
		ignore = true
	}

	if !strings.HasPrefix(dest, c.rootfs) {
		dest = filepath.Join(c.rootfs, dest)

		procDest := filepath.Join(c.rpcRoot, dest)

		sylog.Debugf("Checking if %s exists", procDest)
		if _, err := os.Stat(procDest); os.IsNotExist(err) && !ignore {
			oldmask := syscall.Umask(0)
			defer syscall.Umask(oldmask)

			if point.Type != "" {
				sylog.Debugf("Creating %s", procDest)
				if err := os.MkdirAll(procDest, 0755); err != nil {
					return err
				}
			} else {
				var st syscall.Stat_t

				dir := filepath.Dir(procDest)
				if _, err := os.Stat(dir); os.IsNotExist(err) {
					sylog.Debugf("Creating parent %s", dir)
					if err := os.MkdirAll(dir, 0755); err != nil {
						return err
					}
				}

				if err := syscall.Stat(source, &st); err != nil {
					sylog.Debugf("Ignoring %s: %s", source, err)
					return nil
				}
				switch st.Mode & syscall.S_IFMT {
				case syscall.S_IFDIR:
					sylog.Debugf("Creating dir %s", filepath.Base(procDest))
					if err := os.Mkdir(procDest, 0755); err != nil {
						return err
					}
				case syscall.S_IFREG:
					sylog.Debugf("Creating file %s", filepath.Base(procDest))
					if err := fs.Touch(procDest); err != nil {
						return err
					}
				}
			}
		}
	} else {
		procDest := filepath.Join(c.rpcRoot, dest)

		sylog.Debugf("Checking if %s exists", procDest)
		if _, err := os.Stat(procDest); os.IsNotExist(err) {
			return fmt.Errorf("destination %s doesn't exist", dest)
		}
	}

	if ignore {
		sylog.Debugf("(re)mount %s", dest)
	} else {
		sylog.Debugf("Mount %s to %s : %s [%s]", source, dest, point.Type, optsString)
	}

	_, err := c.rpcOps.Mount(source, dest, point.Type, flags, optsString)
	if err != nil {
		sylog.Debugf("RPC mount error: %s", err)
	}

	return err
}
