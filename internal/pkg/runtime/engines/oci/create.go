// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oci

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/sylabs/singularity/pkg/ociruntime"
	"github.com/sylabs/singularity/pkg/util/namespaces"
	"github.com/sylabs/singularity/pkg/util/sysctl"
	"github.com/sylabs/singularity/pkg/util/unix"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/internal/pkg/cgroups"
	"github.com/sylabs/singularity/internal/pkg/instance"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/oci/rpc/client"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/internal/pkg/util/fs/mount"
	"github.com/sylabs/singularity/pkg/util/fs/proc"
)

var symlinkDevices = []struct {
	old string
	new string
}{
	{"/proc/self/fd", "/dev/fd"},
	{"/proc/kcore", "/dev/core"},
	{"pts/ptmx", "/dev/ptmx"},
	{"/proc/self/fd/0", "/dev/stdin"},
	{"/proc/self/fd/1", "/dev/stdout"},
	{"/proc/self/fd/2", "/dev/stderr"},
}

type device struct {
	major int64
	minor int64
	path  string
	mode  os.FileMode
	uid   int
	gid   int
}

var devices = []device{
	{1, 7, "/dev/full", syscall.S_IFCHR | 0666, 0, 0},
	{1, 3, "/dev/null", syscall.S_IFCHR | 0666, 0, 0},
	{1, 8, "/dev/random", syscall.S_IFCHR | 0666, 0, 0},
	{5, 0, "/dev/tty", syscall.S_IFCHR | 0666, 0, 0},
	{1, 9, "/dev/urandom", syscall.S_IFCHR | 0666, 0, 0},
	{1, 5, "/dev/zero", syscall.S_IFCHR | 0666, 0, 0},
}

func int64ptr(i int) *int64 {
	t := int64(i)
	return &t
}

var cgroupDevices = []specs.LinuxDeviceCgroup{
	{
		Allow:  true,
		Type:   "c",
		Major:  int64ptr(1),
		Minor:  int64ptr(7),
		Access: "rw",
	},
	{
		Allow:  true,
		Type:   "c",
		Major:  int64ptr(1),
		Minor:  int64ptr(3),
		Access: "rw",
	},
	{
		Allow:  true,
		Type:   "c",
		Major:  int64ptr(1),
		Minor:  int64ptr(8),
		Access: "rw",
	},
	{
		Allow:  true,
		Type:   "c",
		Major:  int64ptr(5),
		Minor:  int64ptr(0),
		Access: "rw",
	},
	{
		Allow:  true,
		Type:   "c",
		Major:  int64ptr(1),
		Minor:  int64ptr(9),
		Access: "rw",
	},
	{
		Allow:  true,
		Type:   "c",
		Major:  int64ptr(1),
		Minor:  int64ptr(5),
		Access: "rw",
	},
	{
		Allow:  true,
		Type:   "c",
		Major:  int64ptr(136),
		Access: "rwm",
	},
	{
		Allow:  true,
		Type:   "c",
		Major:  int64ptr(5),
		Minor:  int64ptr(1),
		Access: "rw",
	},
	{
		Allow:  true,
		Type:   "c",
		Major:  int64ptr(5),
		Minor:  int64ptr(2),
		Access: "rw",
	},
}

type container struct {
	engine      *EngineOperations
	rpcOps      *client.RPC
	rootfs      string
	rpcRoot     string
	userNS      bool
	utsNS       bool
	mntNS       bool
	devIndex    int
	cgroupIndex int
}

var statusChan = make(chan string, 1)

func (engine *EngineOperations) createState(pid int) error {
	engine.EngineConfig.Lock()
	defer engine.EngineConfig.Unlock()

	name := engine.CommonConfig.ContainerID

	file, err := instance.Add(name, true, instance.OciSubDir)
	if err != nil {
		return err
	}

	engine.EngineConfig.State.Version = specs.Version
	engine.EngineConfig.State.Bundle = engine.EngineConfig.GetBundlePath()
	engine.EngineConfig.State.ID = engine.CommonConfig.ContainerID
	engine.EngineConfig.State.Pid = pid
	engine.EngineConfig.State.Status = ociruntime.Creating
	engine.EngineConfig.State.Annotations = engine.EngineConfig.OciConfig.Annotations

	file.Config, err = json.Marshal(engine.CommonConfig)
	if err != nil {
		return err
	}

	file.User = "root"
	file.Pid = pid
	file.PPid = os.Getpid()
	file.Image = filepath.Join(engine.EngineConfig.GetBundlePath(), engine.EngineConfig.OciConfig.Root.Path)

	if err := file.Update(); err != nil {
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

	return nil
}

func (engine *EngineOperations) updateState(status string) error {
	engine.EngineConfig.Lock()
	defer engine.EngineConfig.Unlock()

	file, err := instance.Get(engine.CommonConfig.ContainerID, instance.OciSubDir)
	if err != nil {
		return err
	}
	// do nothing if already stopped
	if engine.EngineConfig.State.Status == ociruntime.Stopped {
		return nil
	}
	oldStatus := engine.EngineConfig.State.Status
	engine.EngineConfig.State.Status = status

	t := time.Now().UnixNano()

	switch status {
	case ociruntime.Created:
		if engine.EngineConfig.State.CreatedAt == nil {
			engine.EngineConfig.State.CreatedAt = &t
		}
	case ociruntime.Running:
		if engine.EngineConfig.State.StartedAt == nil {
			engine.EngineConfig.State.StartedAt = &t
		}
	case ociruntime.Stopped:
		if engine.EngineConfig.State.FinishedAt == nil {
			engine.EngineConfig.State.FinishedAt = &t
		}
	}

	file.Config, err = json.Marshal(engine.CommonConfig)
	if err != nil {
		return err
	}

	if err := file.Update(); err != nil {
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

	// send running or stopped status right after container creation
	// to notify that container process started
	if statusChan != nil && oldStatus == ociruntime.Created &&
		(status == ociruntime.Running || status == ociruntime.Stopped) {
		statusChan <- status
	}
	return nil
}

// one shot function to wait on running or stopped status
func (engine *EngineOperations) waitStatusUpdate() {
	if statusChan == nil {
		return
	}
	// block until status update is sent
	<-statusChan
	// close channel and set it to nil
	close(statusChan)
	statusChan = nil
}

// CreateContainer creates a container
func (engine *EngineOperations) CreateContainer(pid int, rpcConn net.Conn) error {
	var err error

	if engine.CommonConfig.EngineName != Name {
		return fmt.Errorf("engineName configuration doesn't match runtime name")
	}

	rpcOps := &client.RPC{}
	rpcOps.Client = rpc.NewClient(rpcConn)
	rpcOps.Name = engine.CommonConfig.EngineName

	if rpcOps.Client == nil {
		return fmt.Errorf("failed to initialize RPC client")
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
		engine:      engine,
		rpcOps:      rpcOps,
		rootfs:      resolvedRootfs,
		rpcRoot:     fmt.Sprintf("/proc/%d/root", pid),
		cgroupIndex: -1,
		devIndex:    -1,
	}

	for _, ns := range engine.EngineConfig.OciConfig.Linux.Namespaces {
		switch ns.Type {
		case specs.UserNamespace:
			c.userNS = true
		case specs.UTSNamespace:
			c.utsNS = true
		case specs.MountNamespace:
			c.mntNS = true
		}
	}

	p := &mount.Points{}
	if engine.EngineConfig.OciConfig.Linux.MountLabel != "" {
		if err := p.SetContext(engine.EngineConfig.OciConfig.Linux.MountLabel); err != nil {
			return err
		}
	}

	system := &mount.System{Points: p, Mount: c.mount}

	for i, point := range engine.EngineConfig.OciConfig.Config.Mounts {
		// cgroup creation
		if point.Type == "cgroup" {
			c.cgroupIndex = i
			continue
		}
		// dev creation
		if point.Destination == "/dev" && point.Type == "tmpfs" {
			c.devIndex = i
		}
	}

	if err := c.addDevices(system); err != nil {
		return err
	}

	if err := c.addCgroups(pid, system); err != nil {
		return err
	}

	// import OCI mount spec
	if err := system.Points.ImportFromSpec(engine.EngineConfig.OciConfig.Config.Mounts); err != nil {
		return err
	}

	if err := c.addRootfsMount(system); err != nil {
		return err
	}

	if err := system.RunAfterTag(mount.KernelTag, c.addDefaultDevices); err != nil {
		return err
	}

	if err := system.RunAfterTag(mount.KernelTag, c.addAllPaths); err != nil {
		return err
	}

	if err := proc.SetOOMScoreAdj(pid, engine.EngineConfig.OciConfig.Process.OOMScoreAdj); err != nil {
		return err
	}

	if err := namespaces.Enter(pid, "ipc"); err != nil {
		return err
	}
	if err := namespaces.Enter(pid, "net"); err != nil {
		return err
	}

	for key, value := range engine.EngineConfig.OciConfig.Linux.Sysctl {
		if err := sysctl.Set(key, value); err != nil {
			return err
		}
	}

	if err := namespaces.Enter(os.Getpid(), "ipc"); err != nil {
		return err
	}
	if err := namespaces.Enter(os.Getpid(), "net"); err != nil {
		return err
	}

	sylog.Debugf("Mount all")
	if err := system.MountAll(); err != nil {
		return err
	}

	if c.utsNS && engine.EngineConfig.OciConfig.Hostname != "" {
		if _, err := rpcOps.SetHostname(engine.EngineConfig.OciConfig.Hostname); err != nil {
			return err
		}
	}

	// update namespaces configuration path
	namespaces := []struct {
		nstype       string
		ns           specs.LinuxNamespaceType
		checkEnabled bool
	}{
		{"pid", specs.PIDNamespace, false},
		{"uts", specs.UTSNamespace, false},
		{"ipc", specs.IPCNamespace, false},
		{"mnt", specs.MountNamespace, false},
		{"cgroup", specs.CgroupNamespace, false},
		{"net", specs.NetworkNamespace, false},
		{"user", specs.UserNamespace, true},
	}

	path := fmt.Sprintf("/proc/%d/ns", pid)

	for _, n := range namespaces {
		has, err := proc.HasNamespace(pid, n.nstype)
		if err == nil && (has || n.checkEnabled) {
			enabled := false
			if n.checkEnabled {
				if engine.EngineConfig.OciConfig.Linux != nil {
					for _, namespace := range engine.EngineConfig.OciConfig.Linux.Namespaces {
						if n.ns == namespace.Type {
							enabled = true
							break
						}
					}
				}
			}
			if has || enabled {
				nspath := filepath.Join(path, n.nstype)
				engine.EngineConfig.OciConfig.AddOrReplaceLinuxNamespace(string(n.ns), nspath)
			}
		} else if err != nil {
			return fmt.Errorf("failed to check %s root and container namespace: %s", n.ns, err)
		}
	}

	method := "pivot"
	if !c.mntNS {
		method = "chroot"
	}

	_, err = rpcOps.Chroot(c.rootfs, method)
	if err != nil {
		return fmt.Errorf("chroot failed: %s", err)
	}

	if engine.EngineConfig.SlavePts != -1 {
		if err := syscall.Close(engine.EngineConfig.SlavePts); err != nil {
			return fmt.Errorf("failed to close slave part: %s", err)
		}
	}
	if engine.EngineConfig.OutputStreams[0] != -1 {
		if err := syscall.Close(engine.EngineConfig.OutputStreams[1]); err != nil {
			return fmt.Errorf("failed to close write output stream: %s", err)
		}
	}
	if engine.EngineConfig.ErrorStreams[0] != -1 {
		if err := syscall.Close(engine.EngineConfig.ErrorStreams[1]); err != nil {
			return fmt.Errorf("failed to close write error stream: %s", err)
		}
	}
	if engine.EngineConfig.InputStreams[0] != -1 {
		if err := syscall.Close(engine.EngineConfig.InputStreams[1]); err != nil {
			return fmt.Errorf("failed to close write input stream: %s", err)
		}
	}

	return nil
}

func (c *container) addCgroups(pid int, system *mount.System) error {
	name := c.engine.CommonConfig.ContainerID
	cgroupsPath := c.engine.EngineConfig.OciConfig.Linux.CgroupsPath

	if !filepath.IsAbs(cgroupsPath) {
		if cgroupsPath == "" {
			cgroupsPath = filepath.Join("/singularity-oci", name)
		} else {
			cgroupsPath = filepath.Join("/", cgroupsPath)
		}
	}

	c.engine.EngineConfig.OciConfig.Linux.CgroupsPath = cgroupsPath

	manager := &cgroups.Manager{Path: cgroupsPath, Pid: pid}

	if err := manager.ApplyFromSpec(c.engine.EngineConfig.OciConfig.Linux.Resources); err != nil {
		return fmt.Errorf("Failed to apply cgroups ressources restriction: %s", err)
	}

	if c.cgroupIndex >= 0 {
		m := c.engine.EngineConfig.OciConfig.Config.Mounts[c.cgroupIndex]
		c.engine.EngineConfig.OciConfig.Config.Mounts = append(
			c.engine.EngineConfig.OciConfig.Config.Mounts[:c.cgroupIndex],
			c.engine.EngineConfig.OciConfig.Config.Mounts[c.cgroupIndex+1:]...,
		)

		cgroupRootPath := manager.GetCgroupRootPath()
		if cgroupRootPath == "" {
			return fmt.Errorf("failed to determine cgroup root path")
		}

		flags, opt := mount.ConvertOptions(m.Options)
		options := strings.Join(opt, ",")

		readOnly := false
		if flags&syscall.MS_RDONLY != 0 {
			readOnly = true
			flags &^= uintptr(syscall.MS_RDONLY)
		}

		hasMode := false
		for _, o := range opt {
			if strings.HasPrefix(o, "mode=") {
				hasMode = true
				break
			}
		}
		if !hasMode {
			options += ",mode=755"
		}

		if err := system.Points.AddFS(mount.OtherTag, m.Destination, "tmpfs", flags, options); err != nil {
			return err
		}

		createSymlinks := func(*mount.System) error {
			cgroupPath := filepath.Join(c.rpcRoot, c.rootfs, m.Destination)
			if _, err := os.Stat(filepath.Join(cgroupPath, "cpu")); err != nil && os.IsNotExist(err) {
				if _, err := c.rpcOps.Symlink("cpu,cpuacct", filepath.Join(c.rootfs, m.Destination, "cpu")); err != nil {
					return err
				}
				if _, err := c.rpcOps.Symlink("cpu,cpuacct", filepath.Join(c.rootfs, m.Destination, "cpuacct")); err != nil {
					return err
				}
			}

			if _, err := os.Stat(filepath.Join(cgroupPath, "net_cls")); err != nil && os.IsNotExist(err) {
				if _, err := c.rpcOps.Symlink("net_cls,net_prio", filepath.Join(c.rootfs, m.Destination, "net_cls")); err != nil {
					return err
				}
				if _, err := c.rpcOps.Symlink("net_cls,net_prio", filepath.Join(c.rootfs, m.Destination, "net_prio")); err != nil {
					return err
				}
			}
			return nil
		}

		if err := system.RunAfterTag(mount.OtherTag, createSymlinks); err != nil {
			return err
		}

		f, err := os.Open(fmt.Sprintf("/proc/%d/cgroup", pid))
		if err != nil {
			return err
		}
		defer f.Close()

		flags |= uintptr(syscall.MS_BIND)
		if readOnly {
			flags |= syscall.MS_RDONLY
		}

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			cgroupLine := strings.Split(scanner.Text(), ":")
			if strings.HasPrefix(cgroupLine[1], "name=") {
				cgroupLine[1] = strings.Replace(cgroupLine[1], "name=", "", 1)
			}
			if cgroupLine[1] != "" {
				source := filepath.Join(cgroupRootPath, cgroupLine[1], cgroupLine[2])
				dest := filepath.Join(m.Destination, cgroupLine[1])
				if err := system.Points.AddBind(mount.OtherTag, source, dest, flags); err != nil {
					return err
				}
				if readOnly {
					if err := system.Points.AddRemount(mount.OtherTag, dest, flags); err != nil {
						return err
					}
				}
			}
		}

		if readOnly {
			if err := system.Points.AddRemount(mount.FinalTag, m.Destination, flags); err != nil {
				return err
			}
		}
	}

	c.engine.EngineConfig.Cgroups = manager

	return nil
}

func (c *container) addAllPaths(system *mount.System) error {
	// add masked path
	if err := c.addMaskedPathsMount(system); err != nil {
		return err
	}

	// add read-only path
	if !c.userNS {
		if err := c.addReadonlyPathsMount(system); err != nil {
			return err
		}
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

	rootfsPath := filepath.Join(c.rpcRoot, c.rootfs)

	devPath := filepath.Join(rootfsPath, fs.EvalRelative("/dev", rootfsPath))
	if _, err := os.Lstat(devPath); os.IsNotExist(err) {
		if err := os.Mkdir(devPath, 0755); err != nil {
			return err
		}
	}

	for _, symlink := range symlinkDevices {
		path := filepath.Join(rootfsPath, symlink.new)
		if _, err := os.Lstat(path); os.IsNotExist(err) {
			if c.userNS {
				path = filepath.Join(c.rootfs, symlink.new)
				if _, err := c.rpcOps.Symlink(symlink.old, path); err != nil {
					return err
				}
			} else {
				if err := os.Symlink(symlink.old, path); err != nil {
					return err
				}
			}
		}
	}

	if c.engine.EngineConfig.OciConfig.Process.Terminal {
		path := filepath.Join(rootfsPath, "dev", "console")
		if _, err := os.Lstat(path); os.IsNotExist(err) {
			if c.userNS {
				if _, err := c.rpcOps.Touch(filepath.Join(c.rootfs, "dev", "console")); err != nil {
					return err
				}
			} else {
				if err := fs.Touch(path); err != nil {
					return err
				}
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
	}

	for _, device := range devices {
		dev := int((device.major << 8) | (device.minor & 0xff) | ((device.minor & 0xfff00) << 12))
		path := filepath.Join(rootfsPath, device.path)
		if _, err := os.Lstat(path); os.IsNotExist(err) {
			if c.userNS {
				path = filepath.Join(c.rootfs, device.path)
				if _, err := os.Stat(device.path); os.IsNotExist(err) {
					sylog.Debugf("skipping mount, %s doesn't exists", device.path)
					continue
				}
				if _, err := c.rpcOps.Touch(path); err != nil {
					return err
				}
				if _, err := c.rpcOps.Mount(device.path, path, "", syscall.MS_BIND, ""); err != nil {
					return err
				}
			} else {
				if err := syscall.Mknod(path, uint32(device.mode), dev); err != nil {
					return fmt.Errorf("mknod: %s", err)
				}
				if device.uid != 0 || device.gid != 0 {
					if err := os.Chown(path, device.uid, device.gid); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func (c *container) addDevices(system *mount.System) error {
	for _, d := range c.engine.EngineConfig.OciConfig.Linux.Devices {
		var dev device

		if d.Path == "" {
			return fmt.Errorf("device path required")
		}
		dev.path = d.Path

		if d.FileMode != nil {
			dev.mode = *d.FileMode
		} else {
			dev.mode = 0644
		}

		switch d.Type {
		case "c", "u":
			dev.mode |= syscall.S_IFCHR
			dev.major = d.Major
			dev.minor = d.Minor
		case "b":
			dev.mode |= syscall.S_IFBLK
			dev.major = d.Major
			dev.minor = d.Minor
		case "p":
			dev.mode |= syscall.S_IFIFO
		default:
			return fmt.Errorf("device type unknown for %s", d.Path)
		}

		if d.UID != nil {
			dev.uid = int(*d.UID)
		}
		if d.GID != nil {
			dev.gid = int(*d.GID)
		}

		devices = append(devices, dev)
	}

	if c.devIndex >= 0 {
		m := &c.engine.EngineConfig.OciConfig.Config.Mounts[c.devIndex]

		flags, _ := mount.ConvertOptions(m.Options)

		flags |= uintptr(syscall.MS_BIND)
		if flags&syscall.MS_RDONLY != 0 {
			if err := system.Points.AddRemount(mount.FinalTag, m.Destination, flags); err != nil {
				return err
			}
			for i := len(m.Options) - 1; i >= 0; i-- {
				if m.Options[i] == "ro" {
					m.Options = append(m.Options[:i], m.Options[i+1:]...)
					break
				}
			}
		}

		if c.engine.EngineConfig.OciConfig.Linux.Resources == nil {
			c.engine.EngineConfig.OciConfig.Linux.Resources = &specs.LinuxResources{}
		}

		c.engine.EngineConfig.OciConfig.Linux.Resources.Devices = append(c.engine.EngineConfig.OciConfig.Linux.Resources.Devices, cgroupDevices...)
	}

	return nil
}

func (c *container) addMaskedPathsMount(system *mount.System) error {
	paths := c.engine.EngineConfig.OciConfig.Linux.MaskedPaths

	dir, err := instance.GetDirPrivileged(c.engine.CommonConfig.ContainerID, instance.OciSubDir)
	if err != nil {
		return err
	}
	nullPath := filepath.Join(dir, "null")

	if _, err := os.Stat(nullPath); os.IsNotExist(err) {
		oldmask := syscall.Umask(0)
		defer syscall.Umask(oldmask)

		if err := os.Mkdir(nullPath, 0755); err != nil {
			return err
		}
	}

	for _, path := range paths {
		relativePath := filepath.Join(c.rootfs, path)
		rpcPath := filepath.Join(c.rpcRoot, relativePath)
		fi, err := os.Stat(rpcPath)
		if err != nil {
			sylog.Debugf("ignoring masked path %s: %s", path, err)
			continue
		}
		if fi.IsDir() {
			if err := system.Points.AddBind(mount.OtherTag, nullPath, relativePath, syscall.MS_BIND); err != nil {
				return err
			}
		} else if err := system.Points.AddBind(mount.OtherTag, "/dev/null", relativePath, syscall.MS_BIND); err != nil {
			return err
		}
	}
	return nil
}

func (c *container) addReadonlyPathsMount(system *mount.System) error {
	paths := c.engine.EngineConfig.OciConfig.Linux.ReadonlyPaths

	for _, path := range paths {
		relativePath := filepath.Join(c.rootfs, path)
		rpcPath := filepath.Join(c.rpcRoot, relativePath)
		_, err := os.Stat(rpcPath)
		if err != nil {
			sylog.Debugf("ignoring read-only path %s: %s", path, err)
			continue
		}
		if err := system.Points.AddBind(mount.OtherTag, relativePath, relativePath, syscall.MS_BIND|syscall.MS_RDONLY); err != nil {
			return err
		}
		if err := system.Points.AddRemount(mount.OtherTag, relativePath, syscall.MS_BIND|syscall.MS_RDONLY); err != nil {
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
		rootfsPath := filepath.Join(c.rpcRoot, c.rootfs)
		relativeDest := fs.EvalRelative(dest, c.rootfs)
		procDest := filepath.Join(rootfsPath, relativeDest)

		dest = filepath.Join(c.rootfs, relativeDest)

		sylog.Debugf("Checking if %s exists", procDest)
		if _, err := os.Stat(procDest); os.IsNotExist(err) && !ignore {
			oldmask := syscall.Umask(0)
			defer syscall.Umask(oldmask)

			if point.Type != "" {
				sylog.Debugf("Creating %s", procDest)
				if c.userNS {
					if _, err := c.rpcOps.MkdirAll(dest, 0755); err != nil {
						return err
					}
				} else {
					if err := os.MkdirAll(procDest, 0755); err != nil {
						return err
					}
				}
			} else {
				var st syscall.Stat_t

				dir := filepath.Dir(procDest)
				if _, err := os.Stat(dir); os.IsNotExist(err) {
					sylog.Debugf("Creating parent %s", dir)
					if c.userNS {
						if _, err := c.rpcOps.Mkdir(filepath.Dir(dest), 0755); err != nil {
							return err
						}
					} else {
						if err := os.MkdirAll(dir, 0755); err != nil {
							return err
						}
					}
				}

				if err := syscall.Stat(source, &st); err != nil {
					sylog.Debugf("Ignoring %s: %s", source, err)
					return nil
				}
				switch st.Mode & syscall.S_IFMT {
				case syscall.S_IFDIR:
					sylog.Debugf("Creating dir %s", filepath.Base(procDest))
					if c.userNS {
						if _, err := c.rpcOps.Mkdir(dest, 0755); err != nil {
							return err
						}
					} else {
						if err := os.Mkdir(procDest, 0755); err != nil {
							return err
						}
					}
				case syscall.S_IFREG:
					sylog.Debugf("Creating file %s", filepath.Base(procDest))
					if c.userNS {
						if _, err := c.rpcOps.Touch(dest); err != nil {
							return err
						}
					} else {
						if err := fs.Touch(procDest); err != nil {
							return err
						}
					}
				}
			}
		}
	} else {
		procDest := filepath.Join(c.rpcRoot, dest)

		sylog.Debugf("Checking if %s exists", procDest)
		if _, err := os.Stat(procDest); os.IsNotExist(err) {
			sylog.Warningf("destination %s doesn't exist", dest)
			return nil
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
