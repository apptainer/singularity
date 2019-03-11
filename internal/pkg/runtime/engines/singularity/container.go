// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/cgroups"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/singularity/rpc/client"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/internal/pkg/util/fs/files"
	"github.com/sylabs/singularity/internal/pkg/util/fs/layout"
	"github.com/sylabs/singularity/internal/pkg/util/fs/layout/layer/overlay"
	"github.com/sylabs/singularity/internal/pkg/util/fs/layout/layer/underlay"
	"github.com/sylabs/singularity/internal/pkg/util/fs/mount"
	"github.com/sylabs/singularity/internal/pkg/util/user"
	"github.com/sylabs/singularity/pkg/image"
	"github.com/sylabs/singularity/pkg/network"
	"github.com/sylabs/singularity/pkg/util/fs/proc"
	"github.com/sylabs/singularity/pkg/util/loop"
	"golang.org/x/crypto/ssh/terminal"
)

// defaultCNIConfPath is the default directory to CNI network configuration files
var defaultCNIConfPath = filepath.Join(buildcfg.SYSCONFDIR, "singularity", "network")

// defaultCNIPluginPath is the default directory to CNI plugins executables
var defaultCNIPluginPath = filepath.Join(buildcfg.LIBEXECDIR, "singularity", "cni")

type container struct {
	engine           *EngineOperations
	rpcOps           *client.RPC
	session          *layout.Session
	sessionLayerType string
	sessionFsType    string
	sessionSize      int
	userNS           bool
	pidNS            bool
	utsNS            bool
	netNS            bool
	ipcNS            bool
	mountInfoPath    string
	skippedMount     []string
	checkDest        []string
	suidFlag         uintptr
	devSourcePath    string
}

func create(engine *EngineOperations, rpcOps *client.RPC, pid int) error {
	var err error

	c := &container{
		engine:           engine,
		rpcOps:           rpcOps,
		sessionLayerType: "none",
		sessionFsType:    engine.EngineConfig.File.MemoryFSType,
		mountInfoPath:    fmt.Sprintf("/proc/%d/mountinfo", pid),
		skippedMount:     make([]string, 0),
		checkDest:        make([]string, 0),
		suidFlag:         syscall.MS_NOSUID,
	}

	cwd := engine.EngineConfig.GetCwd()
	if err := os.Chdir(cwd); err != nil {
		return fmt.Errorf("can't change directory to %s: %s", cwd, err)
	}

	if engine.EngineConfig.OciConfig.Linux != nil {
		for _, namespace := range engine.EngineConfig.OciConfig.Linux.Namespaces {
			switch namespace.Type {
			case specs.UserNamespace:
				c.userNS = true
			case specs.PIDNamespace:
				c.pidNS = true
			case specs.UTSNamespace:
				c.utsNS = true
			case specs.NetworkNamespace:
				c.netNS = true
			case specs.IPCNamespace:
				c.ipcNS = true
			}
		}
	}

	if os.Geteuid() != 0 {
		c.sessionSize = int(engine.EngineConfig.File.SessiondirMaxSize)
	} else if engine.EngineConfig.GetAllowSUID() && !c.userNS {
		c.suidFlag = 0
	}

	p := &mount.Points{}
	system := &mount.System{Points: p, Mount: c.mount}

	if err := c.setupSessionLayout(system); err != nil {
		return err
	}

	if err := system.RunAfterTag(mount.LayerTag, c.addIdentityMount); err != nil {
		return err
	}
	if err := system.RunAfterTag(mount.RootfsTag, c.addActionsMount); err != nil {
		return err
	}

	if err := c.addRootfsMount(system); err != nil {
		return err
	}
	if err := c.addKernelMount(system); err != nil {
		return err
	}
	if err := c.addDevMount(system); err != nil {
		return err
	}
	if err := c.addHostMount(system); err != nil {
		return err
	}
	if err := c.addBindsMount(system); err != nil {
		return err
	}
	if err := c.addHomeMount(system); err != nil {
		return err
	}
	if err := c.addUserbindsMount(system); err != nil {
		return err
	}
	if err := c.addTmpMount(system); err != nil {
		return err
	}
	if err := c.addScratchMount(system); err != nil {
		return err
	}
	if err := c.addCwdMount(system); err != nil {
		return err
	}
	if err := c.addLibsMount(system); err != nil {
		return err
	}
	if err := c.addResolvConfMount(system); err != nil {
		return err
	}
	if err := c.addHostnameMount(system); err != nil {
		return err
	}

	sylog.Debugf("Mount all")
	if err := system.MountAll(); err != nil {
		return err
	}

	sylog.Debugf("Chroot into %s\n", c.session.FinalPath())
	_, err = c.rpcOps.Chroot(c.session.FinalPath(), "pivot")
	if err != nil {
		sylog.Debugf("Fallback to move/chroot")
		_, err = c.rpcOps.Chroot(c.session.FinalPath(), "move")
		if err != nil {
			return fmt.Errorf("chroot failed: %s", err)
		}
	}

	if c.netNS {
		if os.Geteuid() == 0 && !c.userNS {
			/* hold a reference to container network namespace for cleanup */
			f, err := syscall.Open("/proc/"+strconv.Itoa(pid)+"/ns/net", os.O_RDONLY, 0)
			if err != nil {
				return fmt.Errorf("can't open network namespace: %s", err)
			}
			nspath := fmt.Sprintf("/proc/%d/fd/%d", os.Getpid(), f)
			networks := strings.Split(engine.EngineConfig.GetNetwork(), ",")

			cniPath := &network.CNIPath{}

			if engine.EngineConfig.File.CniConfPath != "" {
				cniPath.Conf = engine.EngineConfig.File.CniConfPath
			} else {
				cniPath.Conf = defaultCNIConfPath
			}
			if engine.EngineConfig.File.CniPluginPath != "" {
				cniPath.Plugin = engine.EngineConfig.File.CniPluginPath
			} else {
				cniPath.Plugin = defaultCNIPluginPath
			}

			setup, err := network.NewSetup(networks, strconv.Itoa(pid), nspath, cniPath)
			if err != nil {
				return fmt.Errorf("%s", err)
			}
			netargs := engine.EngineConfig.GetNetworkArgs()
			if err := setup.SetArgs(netargs); err != nil {
				return fmt.Errorf("%s", err)
			}

			setup.SetEnvPath("/bin:/sbin:/usr/bin:/usr/sbin")

			if err := setup.AddNetworks(); err != nil {
				return fmt.Errorf("%s", err)
			}

			engine.EngineConfig.Network = setup
		} else if engine.EngineConfig.GetNetwork() != "none" {
			return fmt.Errorf("Network requires root permissions or --network=none argument as user")
		}
	}

	if os.Geteuid() == 0 {
		path := engine.EngineConfig.GetCgroupsPath()
		if path != "" {
			cgroupPath := filepath.Join("/singularity", strconv.Itoa(pid))
			manager := &cgroups.Manager{Pid: pid, Path: cgroupPath}
			if err := manager.ApplyFromFile(path); err != nil {
				return fmt.Errorf("Failed to apply cgroups ressources restriction: %s", err)
			}
			engine.EngineConfig.Cgroups = manager
		}
	}

	sylog.Debugf("Chdir into / to avoid errors\n")
	err = syscall.Chdir("/")
	if err != nil {
		return fmt.Errorf("change directory failed: %s", err)
	}

	return nil
}

func (c *container) setupSIFOverlay(img *image.Image, writable bool) error {
	// Determine if overlay partitions exists
	overlayPart := 0
	overlayImg := c.engine.EngineConfig.GetOverlayImage()
	imglist := c.engine.EngineConfig.GetImageList()

	for _, p := range img.Partitions[1:] {
		if p.Type == image.EXT3 || p.Type == image.SQUASHFS {
			imgCopy := *img
			imgCopy.Type = int(p.Type)
			imgCopy.Partitions = []image.Section{p}
			imglist = append(imglist, imgCopy)
			overlayImg = append(overlayImg, imgCopy.Path)
			overlayPart++
		}
	}

	c.engine.EngineConfig.SetOverlayImage(overlayImg)
	c.engine.EngineConfig.SetImageList(imglist)

	if overlayPart == 0 && writable {
		return fmt.Errorf("no SIF writable overlay partition found")
	}

	return nil
}

// setupSessionLayout will create the session layout according to the capabilities of Singularity
// on the system. It will first attempt to use "overlay", followed by "underlay", and if neither
// are available it will not use either. If neither are used, we will not be able to bind mount
// to non-existent paths within the container
func (c *container) setupSessionLayout(system *mount.System) error {
	writableTmpfs := c.engine.EngineConfig.GetWritableTmpfs()
	overlayEnabled := false

	sessionPath, err := filepath.EvalSymlinks(buildcfg.SESSIONDIR)
	if err != nil {
		return fmt.Errorf("failed to resolved session directory %s: %s", buildcfg.SESSIONDIR, err)
	}

	if enabled, _ := proc.HasFilesystem("overlay"); enabled && !c.userNS {
		switch c.engine.EngineConfig.File.EnableOverlay {
		case "yes", "try":
			overlayEnabled = true
		}
	}

	imgObject, err := c.loadImage(c.engine.EngineConfig.GetImage(), true)
	if err != nil {
		return fmt.Errorf("while loading image object: %s", err)
	}

	if c.engine.EngineConfig.GetWritableImage() && !writableTmpfs {
		sylog.Debugf("Image is writable, not attempting to use overlay or underlay\n")
		if imgObject.Type == image.SIF {
			err = c.setupSIFOverlay(imgObject, c.engine.EngineConfig.GetWritableImage())
			if err == nil {
				return c.setupOverlayLayout(system, sessionPath)
			}
			sylog.Warningf("While attempting to set up SIFOverlay: %s", err)
		}
		return c.setupDefaultLayout(system, sessionPath)
	}

	if overlayEnabled {
		sylog.Debugf("Attempting to use overlayfs (enable overlay = %v)\n", c.engine.EngineConfig.File.EnableOverlay)
		if imgObject.Type == image.SIF {
			err = c.setupSIFOverlay(imgObject, c.engine.EngineConfig.GetWritableImage())
			if err == nil {
				return c.setupOverlayLayout(system, sessionPath)
			}
			sylog.Warningf("While attempting to set up SIFOverlay: %s", err)
		}
		return c.setupOverlayLayout(system, sessionPath)
	}

	if writableTmpfs {
		sylog.Warningf("Ignoring --writable-tmpfs as it requires overlay support")
	}

	if c.engine.EngineConfig.File.EnableUnderlay {
		sylog.Debugf("Attempting to use underlay (enable underlay = yes)\n")
		return c.setupUnderlayLayout(system, sessionPath)
	}

	sylog.Debugf("Not attempting to use underlay or overlay\n")
	return c.setupDefaultLayout(system, sessionPath)
}

// setupOverlayLayout sets up the session with overlay filesystem
func (c *container) setupOverlayLayout(system *mount.System, sessionPath string) (err error) {
	sylog.Debugf("Creating overlay SESSIONDIR layout\n")
	if c.session, err = layout.NewSession(sessionPath, c.sessionFsType, c.sessionSize, system, overlay.New()); err != nil {
		return err
	}

	if err := c.addOverlayMount(system); err != nil {
		return err
	}

	c.sessionLayerType = "overlay"
	return system.RunAfterTag(mount.LayerTag, c.setPropagationMount)
}

// setupUnderlayLayout sets up the session with underlay "filesystem"
func (c *container) setupUnderlayLayout(system *mount.System, sessionPath string) (err error) {
	sylog.Debugf("Creating underlay SESSIONDIR layout\n")
	if c.session, err = layout.NewSession(sessionPath, c.sessionFsType, c.sessionSize, system, underlay.New()); err != nil {
		return err
	}

	c.sessionLayerType = "underlay"
	return system.RunAfterTag(mount.LayerTag, c.setPropagationMount)
}

// setupDefaultLayout sets up the session without overlay or underlay
func (c *container) setupDefaultLayout(system *mount.System, sessionPath string) (err error) {
	sylog.Debugf("Creating default SESSIONDIR layout\n")
	if c.session, err = layout.NewSession(sessionPath, c.sessionFsType, c.sessionSize, system, nil); err != nil {
		return err
	}

	c.sessionLayerType = "none"
	return system.RunAfterTag(mount.RootfsTag, c.setPropagationMount)
}

// isLayerEnabled returns whether or not overlay or underlay system
// is enabled
func (c *container) isLayerEnabled() bool {
	sylog.Debugf("Using Layer system: %v\n", c.sessionLayerType)
	if c.sessionLayerType == "none" {
		return false
	}

	return true
}

func (c *container) mount(point *mount.Point) error {
	if _, err := mount.GetOffset(point.InternalOptions); err == nil {
		if err := c.mountImage(point); err != nil {
			return fmt.Errorf("can't mount image %s: %s", point.Source, err)
		}
	} else {
		if err := c.mountGeneric(point); err != nil {
			flags, _ := mount.ConvertOptions(point.Options)
			if flags&syscall.MS_REMOUNT != 0 {
				return fmt.Errorf("can't remount %s: %s", point.Destination, err)
			}
			if point.Type != "" {
				if point.Source == "devpts" {
					sylog.Verbosef("Couldn't mount devpts filesystem, continuing with PTY allocation functionality disabled")
				} else {
					// mount error for other filesystems is considered fatal
					return fmt.Errorf("can't mount %s filesystem to %s: %s", point.Type, point.Destination, err)
				}
			}
			sylog.Verbosef("can't mount %s: %s", point.Source, err)
			return nil
		}
	}
	return nil
}

func (c *container) setPropagationMount(system *mount.System) error {
	pflags := uintptr(syscall.MS_REC)

	if c.engine.EngineConfig.File.MountSlave {
		sylog.Debugf("Set RPC mount propagation flag to SLAVE")
		pflags |= syscall.MS_SLAVE
	} else {
		sylog.Debugf("Set RPC mount propagation flag to PRIVATE")
		pflags |= syscall.MS_PRIVATE
	}

	if _, err := c.rpcOps.Mount("", "/", "", pflags, ""); err != nil {
		return err
	}
	return nil
}

func (c *container) checkMounted(dest string) string {
	if dest[0] != '/' {
		return ""
	}

	minfo, err := proc.ParseMountInfo(c.mountInfoPath)
	if err != nil {
		return ""
	}

	p, err := filepath.EvalSymlinks(dest)
	if err != nil {
		return ""
	}

	finalPath := c.session.FinalPath()
	rootfsPath := c.session.RootFsPath()
	sessionPath := c.session.Path()

	for {
		if p == finalPath || p == rootfsPath || p == sessionPath || p == "/" {
			break
		}
		for _, childs := range minfo {
			for _, child := range childs {
				if p == child {
					return child
				}
			}
		}
		p = filepath.Dir(p)
	}
	return ""
}

// mount any generic mount (not loop dev)
func (c *container) mountGeneric(mnt *mount.Point) (err error) {
	flags, opts := mount.ConvertOptions(mnt.Options)
	optsString := strings.Join(opts, ",")
	sessionPath := c.session.Path()
	remount := mount.HasRemountFlag(flags)
	propagation := mount.HasPropagationFlag(flags)
	source := mnt.Source
	dest := ""

	if flags&syscall.MS_BIND != 0 && !remount {
		if _, err := os.Stat(source); os.IsNotExist(err) {
			c.skippedMount = append(c.skippedMount, mnt.Destination)
			sylog.Debugf("Skipping mount, host source %s doesn't exist", source)
			return nil
		}
	}

	if !strings.HasPrefix(mnt.Destination, sessionPath) {
		dest = fs.EvalRelative(mnt.Destination, c.session.FinalPath())

		dest = filepath.Join(c.session.FinalPath(), dest)

		if _, err := os.Stat(dest); os.IsNotExist(err) {
			c.skippedMount = append(c.skippedMount, mnt.Destination)
			sylog.Debugf("Skipping mount, %s doesn't exist in container", dest)
			return nil
		}
	} else {
		dest = mnt.Destination
		if _, err := os.Stat(dest); os.IsNotExist(err) {
			return fmt.Errorf("destination %s doesn't exist", dest)
		}
	}

	if remount || propagation {
		for _, skipped := range c.skippedMount {
			if skipped == mnt.Destination {
				return nil
			}
		}
		sylog.Debugf("Remounting %s\n", dest)
	} else {
		for _, d := range c.checkDest {
			if d == mnt.Destination {
				mounted := c.checkMounted(dest)
				if mounted != "" {
					c.skippedMount = append(c.skippedMount, mnt.Destination)
					sylog.Debugf("Skipping mount %s, %s already mounted", dest, mounted)
					return nil
				}
				break
			}
		}
		sylog.Debugf("Mounting %s to %s\n", source, dest)

		// in stage 1 we changed current working directory to
		// sandbox image directory, just pass "." as source argument to
		// be sure RPC mount the right sandbox image
		if dest == c.session.RootFsPath() && flags&syscall.MS_BIND != 0 {
			source = "."
		}

		// overlay requires root filesystem UID/GID since upper/work
		// directories are owned by root
		if mnt.Type == "overlay" {
			c.rpcOps.SetFsID(0, 0)
			defer c.rpcOps.SetFsID(os.Getuid(), os.Getgid())
		}
	}
	_, err = c.rpcOps.Mount(source, dest, mnt.Type, flags, optsString)
	return err
}

// mount image via loop
func (c *container) mountImage(mnt *mount.Point) error {
	maxDevices := int(c.engine.EngineConfig.File.MaxLoopDevices)
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

	attachFlag := os.O_RDWR
	loopFlags := uint32(loop.FlagsAutoClear)

	if flags&syscall.MS_RDONLY == 1 {
		loopFlags |= loop.FlagsReadOnly
		attachFlag = os.O_RDONLY
	}

	info := &loop.Info64{
		Offset:    offset,
		SizeLimit: sizelimit,
		Flags:     loopFlags,
	}

	shared := c.engine.EngineConfig.File.SharedLoopDevices
	number, err := c.rpcOps.LoopDevice(mnt.Source, attachFlag, *info, maxDevices, shared)
	if err != nil {
		return fmt.Errorf("failed to find loop device: %s", err)
	}

	path := fmt.Sprintf("/dev/loop%d", number)
	sylog.Debugf("Mounting loop device %s to %s\n", path, mnt.Destination)
	_, err = c.rpcOps.Mount(path, mnt.Destination, mnt.Type, flags, optsString)
	if err != nil {
		return fmt.Errorf("failed to mount %s filesystem: %s", mnt.Type, err)
	}

	return nil
}

func (c *container) loadImage(path string, rootfs bool) (*image.Image, error) {
	list := c.engine.EngineConfig.GetImageList()

	if len(list) == 0 {
		return nil, fmt.Errorf("no root filesystem found in %s", path)
	}

	if rootfs {
		img := list[0]
		if img.File == nil {
			return &img, nil
		}
		img.File = os.NewFile(img.Fd, img.Path)
		if img.File == nil {
			return nil, fmt.Errorf("can't find image %s", path)
		}
		return &img, nil
	}
	for _, img := range list[1:] {
		p, err := image.ResolvePath(path)
		if err != nil {
			return nil, err
		}
		if p == img.Path {
			if img.File == nil {
				return &img, nil
			}
			img.File = os.NewFile(img.Fd, img.Path)
			if img.File == nil {
				return nil, fmt.Errorf("can't find image %s", path)
			}
			return &img, nil
		}
	}

	return nil, fmt.Errorf("no image found with path %s", path)
}

func (c *container) addRootfsMount(system *mount.System) error {
	flags := uintptr(c.suidFlag | syscall.MS_NODEV)
	rootfs := c.engine.EngineConfig.GetImage()

	imageObject, err := c.loadImage(rootfs, true)
	if err != nil {
		return err
	}

	if !imageObject.Writable {
		sylog.Debugf("Mount rootfs in read-only mode")
		flags |= syscall.MS_RDONLY
	} else {
		sylog.Debugf("Mount rootfs in read-write mode")
	}

	mountType := ""
	offset := imageObject.Partitions[0].Offset
	size := imageObject.Partitions[0].Size

	switch imageObject.Partitions[0].Type {
	case image.SQUASHFS:
		mountType = "squashfs"
	case image.EXT3:
		mountType = "ext3"
	case image.SANDBOX:
		sylog.Debugf("Mounting directory rootfs: %v\n", rootfs)
		flags |= syscall.MS_BIND
		if err := system.Points.AddBind(mount.RootfsTag, rootfs, c.session.RootFsPath(), flags); err != nil {
			return err
		}
		if !c.userNS {
			system.Points.AddRemount(mount.RootfsTag, c.session.RootFsPath(), flags)
		}
		return nil
	}

	sylog.Debugf("Mounting block [%v] image: %v\n", mountType, rootfs)
	if err := system.Points.AddImage(
		mount.RootfsTag,
		imageObject.Source,
		c.session.RootFsPath(),
		mountType,
		flags,
		offset,
		size,
	); err != nil {
		return err
	}

	if imageObject.Writable {
		return system.Points.AddPropagation(mount.DevTag, c.session.RootFsPath(), syscall.MS_UNBINDABLE)
	}

	return nil
}

func (c *container) overlayUpperWork(system *mount.System) error {
	ov := c.session.Layer.(*overlay.Overlay)

	u := ov.GetUpperDir()
	w := ov.GetWorkDir()

	if fs.IsLink(u) {
		return fmt.Errorf("symlink detected, upper overlay %s must be a directory", u)
	}
	if fs.IsLink(w) {
		return fmt.Errorf("symlink detected, work overlay %s must be a directory", w)
	}

	c.rpcOps.SetFsID(0, 0)
	defer c.rpcOps.SetFsID(os.Getuid(), os.Getgid())

	if !fs.IsDir(u) {
		if _, err := c.rpcOps.Mkdir(u, 0755); err != nil {
			return fmt.Errorf("failed to create %s directory: %s", u, err)
		}
	}
	if !fs.IsDir(w) {
		if _, err := c.rpcOps.Mkdir(w, 0755); err != nil {
			return fmt.Errorf("failed to create %s directory: %s", w, err)
		}
	}

	return nil
}

func (c *container) addOverlayMount(system *mount.System) error {
	nb := 0
	ov := c.session.Layer.(*overlay.Overlay)
	hasUpper := false

	if c.engine.EngineConfig.GetWritableTmpfs() {
		sylog.Debugf("Setup writable tmpfs overlay")

		if err := c.session.AddDir("/tmpfs/upper"); err != nil {
			return err
		}
		if err := c.session.AddDir("/tmpfs/work"); err != nil {
			return err
		}

		upper, _ := c.session.GetPath("/tmpfs/upper")
		work, _ := c.session.GetPath("/tmpfs/work")

		if err := ov.SetUpperDir(upper); err != nil {
			return fmt.Errorf("failed to add overlay upper: %s", err)
		}
		if err := ov.SetWorkDir(work); err != nil {
			return fmt.Errorf("failed to add overlay upper: %s", err)
		}

		tmpfsPath := filepath.Dir(upper)

		flags := uintptr(c.suidFlag | syscall.MS_NODEV)

		if err := system.Points.AddBind(mount.PreLayerTag, tmpfsPath, tmpfsPath, flags); err != nil {
			return fmt.Errorf("failed to add %s temporary filesystem: %s", tmpfsPath, err)
		}

		if err := system.Points.AddRemount(mount.PreLayerTag, tmpfsPath, flags); err != nil {
			return fmt.Errorf("failed to add %s temporary filesystem: %s", tmpfsPath, err)
		}

		hasUpper = true
	}

	for _, img := range c.engine.EngineConfig.GetOverlayImage() {
		splitted := strings.SplitN(img, ":", 2)

		imageObject, err := c.loadImage(splitted[0], false)
		if err != nil {
			return fmt.Errorf("failed to open overlay image %s: %s", splitted[0], err)
		}

		sessionDest := fmt.Sprintf("/overlay-images/%d", nb)
		if err := c.session.AddDir(sessionDest); err != nil {
			return fmt.Errorf("failed to create session directory for overlay: %s", err)
		}
		dst, _ := c.session.GetPath(sessionDest)
		nb++

		src := imageObject.Source
		offset := imageObject.Partitions[0].Offset
		size := imageObject.Partitions[0].Size

		switch imageObject.Type {
		case image.EXT3:
			flags := uintptr(c.suidFlag | syscall.MS_NODEV)

			if !imageObject.Writable {
				flags |= syscall.MS_RDONLY
				ov.AddLowerDir(filepath.Join(dst, "upper"))
			}

			err = system.Points.AddImage(mount.PreLayerTag, src, dst, "ext3", flags, offset, size)
			if err != nil {
				return fmt.Errorf("while adding ext3 image: %s", err)
			}
			flags &^= syscall.MS_RDONLY
		case image.SQUASHFS:
			flags := uintptr(c.suidFlag | syscall.MS_NODEV | syscall.MS_RDONLY)
			err = system.Points.AddImage(mount.PreLayerTag, src, dst, "squashfs", flags, offset, size)
			if err != nil {
				return err
			}
			ov.AddLowerDir(dst)
		case image.SANDBOX:
			if os.Geteuid() != 0 {
				return fmt.Errorf("only root user can use sandbox as overlay")
			}

			flags := uintptr(c.suidFlag | syscall.MS_NODEV)
			err = system.Points.AddBind(mount.PreLayerTag, imageObject.Path, dst, flags)
			if err != nil {
				return fmt.Errorf("while adding sandbox image: %s", err)
			}
			system.Points.AddRemount(mount.PreLayerTag, dst, flags)

			if !imageObject.Writable {
				if fs.IsDir(filepath.Join(imageObject.Path, "upper")) {
					ov.AddLowerDir(filepath.Join(dst, "upper"))
				} else {
					ov.AddLowerDir(dst)
				}
			}
		default:
			return fmt.Errorf("unknown image format")
		}

		err = system.Points.AddPropagation(mount.DevTag, dst, syscall.MS_UNBINDABLE)
		if err != nil {
			return err
		}

		if imageObject.Writable && !hasUpper {
			upper := filepath.Join(dst, "upper")
			work := filepath.Join(dst, "work")

			if err := ov.SetUpperDir(upper); err != nil {
				return fmt.Errorf("failed to add overlay upper: %s", err)
			}
			if err := ov.SetWorkDir(work); err != nil {
				return fmt.Errorf("failed to add overlay upper: %s", err)
			}

			hasUpper = true
		}
	}

	if hasUpper {
		if err := system.RunAfterTag(mount.PreLayerTag, c.overlayUpperWork); err != nil {
			return err
		}
	}

	return system.Points.AddPropagation(mount.DevTag, c.session.FinalPath(), syscall.MS_UNBINDABLE)
}

func (c *container) addKernelMount(system *mount.System) error {
	var err error
	bindFlags := uintptr(syscall.MS_BIND | syscall.MS_NOSUID | syscall.MS_NODEV | syscall.MS_REC)

	sylog.Debugf("Checking configuration file for 'mount proc'")
	if c.engine.EngineConfig.File.MountProc {
		sylog.Debugf("Adding proc to mount list\n")
		if c.pidNS {
			err = system.Points.AddFS(mount.KernelTag, "/proc", "proc", syscall.MS_NOSUID|syscall.MS_NODEV, "")
		} else {
			err = system.Points.AddBind(mount.KernelTag, "/proc", "/proc", bindFlags)
			if err == nil {
				if !c.userNS {
					system.Points.AddRemount(mount.KernelTag, "/proc", bindFlags)
				}
			}
		}
		if err != nil {
			return fmt.Errorf("unable to add proc to mount list: %s", err)
		}
		sylog.Verbosef("Default mount: /proc:/proc")
	} else {
		sylog.Verbosef("Skipping /proc mount")
	}

	sylog.Debugf("Checking configuration file for 'mount sys'")
	if c.engine.EngineConfig.File.MountSys {
		sylog.Debugf("Adding sysfs to mount list\n")
		if !c.userNS {
			err = system.Points.AddFS(mount.KernelTag, "/sys", "sysfs", syscall.MS_NOSUID|syscall.MS_NODEV, "")
		} else {
			err = system.Points.AddBind(mount.KernelTag, "/sys", "/sys", bindFlags)
			if err == nil {
				if !c.userNS {
					system.Points.AddRemount(mount.KernelTag, "/sys", bindFlags)
				}
			}
		}
		if err != nil {
			return fmt.Errorf("unable to add sys to mount list: %s", err)
		}
		sylog.Verbosef("Default mount: /sys:/sys")
	} else {
		sylog.Verbosef("Skipping /sys mount")
	}
	return nil
}

func (c *container) addSessionDevAt(srcpath string, atpath string, system *mount.System) error {
	fi, err := os.Lstat(srcpath)
	if err != nil {
		return err
	}

	switch mode := fi.Mode(); {
	case mode&os.ModeSymlink != 0:
		target, err := os.Readlink(srcpath)
		if err != nil {
			return err
		}
		if err := c.session.AddSymlink(atpath, target); err != nil {
			return fmt.Errorf("failed to create symlink %s", atpath)
		}

		dst, _ := c.session.GetPath(atpath)

		sylog.Debugf("Adding symlink device %s to %s at %s", srcpath, target, dst)

		return nil
	case mode.IsDir():
		if err := c.session.AddDir(atpath); err != nil {
			return fmt.Errorf("failed to add %s session dir: %s", atpath, err)
		}
	default:
		if err := c.session.AddFile(atpath, nil); err != nil {
			return fmt.Errorf("failed to add %s session file: %s", atpath, err)
		}
	}

	dst, _ := c.session.GetPath(atpath)

	sylog.Debugf("Mounting device %s at %s", srcpath, dst)

	if err := system.Points.AddBind(mount.DevTag, srcpath, dst, syscall.MS_BIND); err != nil {
		return fmt.Errorf("failed to add %s mount: %s", srcpath, err)
	}
	return nil
}

func (c *container) addSessionDev(devpath string, system *mount.System) error {
	return c.addSessionDevAt(devpath, devpath, system)
}

func (c *container) addSessionDevMount(system *mount.System) error {
	if c.devSourcePath == "" {
		c.devSourcePath, _ = c.session.GetPath("/dev")
	}
	err := system.Points.AddBind(mount.DevTag, c.devSourcePath, "/dev", syscall.MS_BIND|syscall.MS_REC)
	if err != nil {
		return fmt.Errorf("unable to add dev to mount list: %s", err)
	}
	return nil
}

func (c *container) addDevMount(system *mount.System) error {
	sylog.Debugf("Checking configuration file for 'mount dev'")

	if c.engine.EngineConfig.File.MountDev == "minimal" || c.engine.EngineConfig.GetContain() {
		sylog.Debugf("Creating temporary staged /dev")
		if err := c.session.AddDir("/dev"); err != nil {
			return fmt.Errorf("failed to add /dev session directory: %s", err)
		}
		sylog.Debugf("Creating temporary staged /dev/shm")
		if err := c.session.AddDir("/dev/shm"); err != nil {
			return fmt.Errorf("failed to add /dev/shm session directory: %s", err)
		}
		devshmPath, _ := c.session.GetPath("/dev/shm")
		flags := uintptr(syscall.MS_NOSUID | syscall.MS_NODEV)
		err := system.Points.AddFS(mount.DevTag, devshmPath, c.engine.EngineConfig.File.MemoryFSType, flags, "mode=1777")
		if err != nil {
			return fmt.Errorf("failed to add /dev/shm temporary filesystem: %s", err)
		}

		if c.ipcNS {
			sylog.Debugf("Creating temporary staged /dev/mqueue")
			if err := c.session.AddDir("/dev/mqueue"); err != nil {
				return fmt.Errorf("failed to add /dev/mqueue session directory: %s", err)
			}
			mqueuePath, _ := c.session.GetPath("/dev/mqueue")
			flags := uintptr(syscall.MS_NOSUID | syscall.MS_NODEV)
			err := system.Points.AddFS(mount.DevTag, mqueuePath, "mqueue", flags, "")
			if err != nil {
				return fmt.Errorf("failed to add /dev/mqueue filesystem: %s", err)
			}
		}

		if c.engine.EngineConfig.File.MountDevPts {
			if _, err := os.Stat("/dev/pts/ptmx"); os.IsNotExist(err) {
				return fmt.Errorf("Multiple devpts instances unsupported and /dev/pts configured")
			}

			sylog.Debugf("Creating temporary staged /dev/pts")
			if err := c.session.AddDir("/dev/pts"); err != nil {
				return fmt.Errorf("failed to add /dev/pts session directory: %s", err)
			}

			options := "mode=0620,newinstance,ptmxmode=0666"
			if !c.userNS {
				group, err := user.GetGrNam("tty")
				if err != nil {
					return fmt.Errorf("Problem resolving 'tty' group GID: %s", err)
				}
				options = fmt.Sprintf("%s,gid=%d", options, group.GID)

			} else {
				sylog.Debugf("Not setting /dev/pts filesystem gid: user namespace enabled")
			}
			sylog.Debugf("Mounting devpts for staged /dev/pts")
			devptsPath, _ := c.session.GetPath("/dev/pts")
			err = system.Points.AddFS(mount.DevTag, devptsPath, "devpts", syscall.MS_NOSUID|syscall.MS_NOEXEC, options)
			if err != nil {
				return fmt.Errorf("failed to add devpts filesystem: %s", err)
			}
			// add additional PTY allocation symlink
			if err := c.session.AddSymlink("/dev/ptmx", "/dev/pts/ptmx"); err != nil {
				return fmt.Errorf("failed to create /dev/ptmx symlink: %s", err)
			}

		}
		// add /dev/console mount pointing to original tty if there is one
		for fd := 0; fd <= 2; fd++ {
			if !terminal.IsTerminal(fd) {
				continue
			}
			// Found a tty on stdin, stdout, or stderr.
			// Bind mount it at /dev/console.
			// readlink() from /proc/self/fd/N isn't as reliable as
			//  ttyname() (e.g. it doesn't work in docker), but
			//  there is no golang ttyname() so use this for now
			//  and also check the device that docker uses,
			//  /dev/console.
			procfd := fmt.Sprintf("/proc/self/fd/%d", fd)
			ttylink, err := os.Readlink(procfd)
			if err != nil {
				return err
			}

			if _, err := os.Stat(ttylink); err != nil {
				// Check if in a system like docker
				//  using /dev/console already
				consinfo := new(syscall.Stat_t)
				conserr := syscall.Stat("/dev/console", consinfo)
				fdinfo := new(syscall.Stat_t)
				fderr := syscall.Fstat(fd, fdinfo)
				if conserr == nil &&
					fderr == nil &&
					consinfo.Ino == fdinfo.Ino &&
					consinfo.Rdev == fdinfo.Rdev {
					sylog.Debugf("Fd %d is tty pointing to nonexistent %s but /dev/console is good", fd, ttylink)
					ttylink = "/dev/console"

				} else {
					sylog.Debugf("Fd %d is tty but %s doesn't exist, skipping", fd, ttylink)
					continue
				}
			}
			sylog.Debugf("Fd %d is tty %s, binding to /dev/console", fd, ttylink)
			if err := c.addSessionDevAt(ttylink, "/dev/console", system); err != nil {
				return err
			}
			// and also add a /dev/tty
			if err := c.addSessionDev("/dev/tty", system); err != nil {
				return err
			}
			break
		}
		if err := c.addSessionDev("/dev/null", system); err != nil {
			return err
		}
		if err := c.addSessionDev("/dev/zero", system); err != nil {
			return err
		}
		if err := c.addSessionDev("/dev/random", system); err != nil {
			return err
		}
		if err := c.addSessionDev("/dev/urandom", system); err != nil {
			return err
		}
		if c.engine.EngineConfig.GetNv() {
			files, err := ioutil.ReadDir("/dev")
			if err != nil {
				return fmt.Errorf("failed to read /dev directory: %s", err)
			}
			for _, file := range files {
				if strings.HasPrefix(file.Name(), "nvidia") {
					if err := c.addSessionDev(filepath.Join("/dev", file.Name()), system); err != nil {
						return err
					}
				}
			}
		}

		if err := c.addSessionDev("/dev/fd", system); err != nil {
			return err
		}
		if err := c.addSessionDev("/dev/stdin", system); err != nil {
			return err
		}
		if err := c.addSessionDev("/dev/stdout", system); err != nil {
			return err
		}
		if err := c.addSessionDev("/dev/stderr", system); err != nil {
			return err
		}

		// devices could be added in addUserbindsMount so bind session dev
		// after that all devices have been added to the mount point list
		if err := system.RunAfterTag(mount.LayerTag, c.addSessionDevMount); err != nil {
			return err
		}
	} else if c.engine.EngineConfig.File.MountDev == "yes" {
		sylog.Debugf("Adding dev to mount list\n")
		err := system.Points.AddBind(mount.DevTag, "/dev", "/dev", syscall.MS_BIND|syscall.MS_REC)
		if err != nil {
			return fmt.Errorf("unable to add dev to mount list: %s", err)
		}
		sylog.Verbosef("Default mount: /dev:/dev")
	} else if c.engine.EngineConfig.File.MountDev == "no" {
		sylog.Verbosef("Not mounting /dev inside the container, disallowed by configuration")
	}
	return nil
}

func (c *container) addHostMount(system *mount.System) error {
	if !c.engine.EngineConfig.File.MountHostfs {
		sylog.Debugf("Not mounting host file systems per configuration")
		return nil
	}

	info, err := proc.ParseMountInfo("/proc/self/mountinfo")
	if err != nil {
		return err
	}
	flags := uintptr(syscall.MS_BIND | c.suidFlag | syscall.MS_NODEV | syscall.MS_REC)
	for _, child := range info["/"] {
		if strings.HasPrefix(child, "/proc") {
			sylog.Debugf("Skipping /proc based file system")
			continue
		} else if strings.HasPrefix(child, "/sys") {
			sylog.Debugf("Skipping /sys based file system")
			continue
		} else if strings.HasPrefix(child, "/dev") {
			sylog.Debugf("Skipping /dev based file system")
			continue
		} else if strings.HasPrefix(child, "/run") {
			sylog.Debugf("Skipping /run based file system")
			continue
		} else if strings.HasPrefix(child, "/boot") {
			sylog.Debugf("Skipping /boot based file system")
			continue
		} else if strings.HasPrefix(child, "/var") {
			sylog.Debugf("Skipping /var based file system")
			continue
		}
		sylog.Debugf("Adding %s to mount list\n", child)
		if err := system.Points.AddBind(mount.HostfsTag, child, child, flags); err != nil {
			return fmt.Errorf("unable to add %s to mount list: %s", child, err)
		}
		system.Points.AddRemount(mount.HostfsTag, child, flags)
	}
	return nil
}

func (c *container) addBindsMount(system *mount.System) error {
	flags := uintptr(syscall.MS_BIND | c.suidFlag | syscall.MS_NODEV | syscall.MS_REC)

	if c.engine.EngineConfig.GetContain() {
		sylog.Debugf("Skipping bind mounts as contain was requested")
		return nil
	}

	for _, bindpath := range c.engine.EngineConfig.File.BindPath {
		splitted := strings.Split(bindpath, ":")
		src := splitted[0]
		dst := ""
		if len(splitted) > 1 {
			dst = splitted[1]
		} else {
			dst = src
		}

		sylog.Verbosef("Found 'bind path' = %s, %s", src, dst)
		err := system.Points.AddBind(mount.BindsTag, src, dst, flags)
		if err != nil {
			return fmt.Errorf("unable to add %s to mount list: %s", src, err)
		}
	}

	return nil
}

// getHomePaths returns the source and destination path of the requested home mount
func (c *container) getHomePaths() (source string, dest string, err error) {
	if c.engine.EngineConfig.GetCustomHome() {
		dest = filepath.Clean(c.engine.EngineConfig.GetHomeDest())
		source, err = filepath.Abs(filepath.Clean(c.engine.EngineConfig.GetHomeSource()))
	} else {
		pw, err := user.GetPwUID(uint32(os.Getuid()))
		if err == nil {
			dest = pw.Dir
			source = pw.Dir
		}
	}

	return source, dest, err
}

// addHomeStagingDir adds and mounts home directory in session staging directory
func (c *container) addHomeStagingDir(system *mount.System, source string, dest string) (string, error) {
	flags := uintptr(syscall.MS_BIND | c.suidFlag | syscall.MS_NODEV | syscall.MS_REC)
	homeStage := ""

	if err := c.session.AddDir(dest); err != nil {
		return "", fmt.Errorf("failed to add %s as session directory: %s", source, err)
	}

	homeStage, _ = c.session.GetPath(dest)

	if !c.engine.EngineConfig.GetContain() || c.engine.EngineConfig.GetCustomHome() {
		sylog.Debugf("Staging home directory (%v) at %v\n", source, homeStage)

		if err := system.Points.AddBind(mount.HomeTag, source, homeStage, flags); err != nil {
			return "", fmt.Errorf("unable to add %s to mount list: %s", source, err)
		}
		system.Points.AddRemount(mount.HomeTag, homeStage, flags)
	} else {
		sylog.Debugf("Using session directory for home directory")
	}

	return homeStage, nil
}

// addHomeLayer adds the home mount when using either overlay or underlay
func (c *container) addHomeLayer(system *mount.System, source, dest string) error {
	flags := uintptr(syscall.MS_BIND | c.suidFlag | syscall.MS_NODEV | syscall.MS_REC)

	if err := system.Points.AddBind(mount.HomeTag, source, dest, flags); err != nil {
		return fmt.Errorf("unable to add home to mount list: %s", err)
	}

	return system.Points.AddRemount(mount.HomeTag, dest, flags)
}

// addHomeNoLayer is responsible for staging the home directory and adding the base
// directory of the staged home into the container when overlay/underlay are unavailable
func (c *container) addHomeNoLayer(system *mount.System, source, dest string) error {
	flags := uintptr(syscall.MS_BIND | c.suidFlag | syscall.MS_NODEV | syscall.MS_REC)

	homeBase := fs.RootDir(dest)
	if homeBase == "." {
		return fmt.Errorf("could not identify staged home directory base: %s", dest)
	}

	homeStageBase, _ := c.session.GetPath(homeBase)
	sylog.Verbosef("Mounting staged home directory base (%v) into container at %v\n", homeStageBase, filepath.Join(c.session.FinalPath(), homeBase))
	if err := system.Points.AddBind(mount.FinalTag, homeStageBase, homeBase, flags); err != nil {
		return fmt.Errorf("unable to add %s to mount list: %s", homeStageBase, err)
	}

	return nil
}

// addHomeMount is responsible for adding the home directory mount using the proper method
func (c *container) addHomeMount(system *mount.System) error {
	if c.engine.EngineConfig.GetNoHome() {
		sylog.Debugf("Skipping home directory mount by user request.")
		return nil
	}

	if !c.engine.EngineConfig.File.MountHome {
		sylog.Debugf("Skipping home dir mounting (per config)")
		return nil
	}

	// check if user attempt to mount a custom home when not allowed to
	if c.engine.EngineConfig.GetCustomHome() && !c.engine.EngineConfig.File.UserBindControl {
		return fmt.Errorf("Not mounting user requested home: user bind control is disallowed")
	}

	source, dest, err := c.getHomePaths()
	if err != nil {
		return fmt.Errorf("unable to get home source/destination: %v", err)
	}

	stagingDir, err := c.addHomeStagingDir(system, source, dest)
	if err != nil {
		return err
	}

	sylog.Debugf("Adding home directory mount [%v:%v] to list using layer: %v\n", stagingDir, dest, c.sessionLayerType)
	if !c.isLayerEnabled() {
		return c.addHomeNoLayer(system, stagingDir, dest)
	}
	return c.addHomeLayer(system, stagingDir, dest)
}

func (c *container) addUserbindsMount(system *mount.System) error {
	devicesMounted := 0
	devPrefix := "/dev"
	userBindControl := c.engine.EngineConfig.File.UserBindControl
	flags := uintptr(syscall.MS_BIND | c.suidFlag | syscall.MS_NODEV | syscall.MS_REC)

	if len(c.engine.EngineConfig.GetBindPath()) == 0 {
		return nil
	}

	for _, b := range c.engine.EngineConfig.GetBindPath() {
		splitted := strings.Split(b, ":")

		src, err := filepath.Abs(splitted[0])
		if err != nil {
			sylog.Warningf("Can't determine absolute path of %s bind point", splitted[0])
			continue
		}
		dst := src
		if len(splitted) > 1 {
			dst = splitted[1]
		}
		if len(splitted) > 2 {
			if splitted[2] == "ro" {
				flags |= syscall.MS_RDONLY
			} else if splitted[2] != "rw" {
				sylog.Warningf("Not mounting requested %s bind point, invalid mount option %s", src, splitted[2])
			}
		}

		// special case for /dev mount to override default mount behaviour
		// with --contain option or 'mount dev = minimal'
		if strings.HasPrefix(src, devPrefix) {
			if c.engine.EngineConfig.File.MountDev == "minimal" || c.engine.EngineConfig.GetContain() {
				if strings.HasPrefix(src, "/dev/shm/") || strings.HasPrefix(src, "/dev/mqueue/") {
					sylog.Warningf("Skipping %s bind mount: not allowed", src)
				} else {
					if src != devPrefix {
						if err := c.addSessionDev(src, system); err != nil {
							sylog.Warningf("Skipping %s bind mount: %s", src, err)
						}
					} else {
						system.Points.RemoveByTag(mount.DevTag)
						c.devSourcePath = devPrefix
					}
					sylog.Debugf("Adding device %s to mount list\n", src)
				}
				devicesMounted++
			} else if c.engine.EngineConfig.File.MountDev == "yes" {
				sylog.Warningf("Skipping %s bind mount: /dev is already mounted", src)
			} else {
				sylog.Warningf("Skipping %s bind mount: disallowed by configuration", src)
			}
			continue
		} else if !userBindControl {
			continue
		}

		sylog.Debugf("Adding %s to mount list\n", src)

		if err := system.Points.AddBind(mount.UserbindsTag, src, dst, flags); err != nil && err == mount.ErrMountExists {
			sylog.Warningf("destination %s already in mount list: %s", src, err)
		} else if err != nil {
			return fmt.Errorf("unable to add %s to mount list: %s", src, err)
		} else {
			system.Points.AddRemount(mount.UserbindsTag, dst, flags)
			flags &^= syscall.MS_RDONLY
		}
	}

	sylog.Debugf("Checking for 'user bind control' in configuration file")
	if !userBindControl && devicesMounted == 0 {
		sylog.Warningf("Ignoring user bind request: user bind control disabled by system administrator")
	}

	return nil
}

func (c *container) addTmpMount(system *mount.System) error {
	sylog.Debugf("Checking for 'mount tmp' in configuration file")
	if !c.engine.EngineConfig.File.MountTmp {
		sylog.Verbosef("Skipping tmp dir mounting (per config)")
		return nil
	}
	tmpSource := "/tmp"
	vartmpSource := "/var/tmp"

	if c.engine.EngineConfig.GetContain() {
		workdir := c.engine.EngineConfig.GetWorkdir()
		if workdir != "" {
			if !c.engine.EngineConfig.File.UserBindControl {
				sylog.Warningf("User bind control is disabled by system administrator")
				return nil
			}

			vartmpSource = "var_tmp"

			workdir, err := filepath.Abs(filepath.Clean(workdir))
			if err != nil {
				sylog.Warningf("Can't determine absolute path of workdir %s", workdir)
			}

			tmpSource = filepath.Join(workdir, tmpSource)
			vartmpSource = filepath.Join(workdir, vartmpSource)

			if err := fs.Mkdir(tmpSource, os.ModeSticky|0777); err != nil && !os.IsExist(err) {
				return fmt.Errorf("failed to create %s: %s", tmpSource, err)
			}
			if err := fs.Mkdir(vartmpSource, os.ModeSticky|0777); err != nil && !os.IsExist(err) {
				return fmt.Errorf("failed to create %s: %s", vartmpSource, err)
			}
		} else {
			if _, err := c.session.GetPath(tmpSource); err != nil {
				if err := c.session.AddDir(tmpSource); err != nil {
					return err
				}
				if err := c.session.Chmod(tmpSource, os.ModeSticky|0777); err != nil {
					return err
				}
			}
			if _, err := c.session.GetPath(vartmpSource); err != nil {
				if err := c.session.AddDir(vartmpSource); err != nil {
					return err
				}
				if err := c.session.Chmod(vartmpSource, os.ModeSticky|0777); err != nil {
					return err
				}
			}
			tmpSource, _ = c.session.GetPath(tmpSource)
			vartmpSource, _ = c.session.GetPath(vartmpSource)
		}
	}
	flags := uintptr(syscall.MS_BIND | c.suidFlag | syscall.MS_NODEV | syscall.MS_REC)

	if err := system.Points.AddBind(mount.TmpTag, tmpSource, "/tmp", flags); err == nil {
		system.Points.AddRemount(mount.TmpTag, "/tmp", flags)
		sylog.Verbosef("Default mount: /tmp:/tmp")
	} else {
		return fmt.Errorf("could not mount container's /tmp directory: %s %s", err, tmpSource)
	}
	if err := system.Points.AddBind(mount.TmpTag, vartmpSource, "/var/tmp", flags); err == nil {
		system.Points.AddRemount(mount.TmpTag, "/var/tmp", flags)
		sylog.Verbosef("Default mount: /var/tmp:/var/tmp")
	} else {
		return fmt.Errorf("could not mount container's /var/tmp directory: %s", err)
	}
	return nil
}

func (c *container) addScratchMount(system *mount.System) error {
	hasWorkdir := false

	scratchdir := c.engine.EngineConfig.GetScratchDir()
	if len(scratchdir) == 0 {
		sylog.Debugf("Not mounting scratch directory: Not requested")
		return nil
	} else if len(scratchdir) == 1 {
		scratchdir = strings.Split(filepath.Clean(scratchdir[0]), ",")
	}
	if !c.engine.EngineConfig.File.UserBindControl {
		sylog.Verbosef("Not mounting scratch: user bind control disabled by system administrator")
		return nil
	}
	workdir := c.engine.EngineConfig.GetWorkdir()
	sourceDir := ""
	if workdir != "" {
		hasWorkdir = true
		sourceDir = filepath.Clean(workdir) + "/scratch"
	} else {
		sourceDir = c.session.Path()
	}
	if hasWorkdir {
		if err := fs.MkdirAll(sourceDir, 0750); err != nil {
			return fmt.Errorf("could not create scratch working directory %s: %s", sourceDir, err)
		}
	}
	for _, dir := range scratchdir {
		fullSourceDir := ""

		if hasWorkdir {
			fullSourceDir = filepath.Join(sourceDir, filepath.Base(dir))
			if err := fs.MkdirAll(fullSourceDir, 0750); err != nil && !os.IsExist(err) {
				return fmt.Errorf("could not create scratch working directory %s: %s", sourceDir, err)
			}
		} else {
			src := filepath.Join("/scratch", dir)
			if err := c.session.AddDir(src); err != nil {
				return fmt.Errorf("could not create scratch working directory %s: %s", sourceDir, err)
			}
			fullSourceDir, _ = c.session.GetPath(src)
		}
		flags := uintptr(syscall.MS_BIND | c.suidFlag | syscall.MS_NODEV | syscall.MS_REC)
		if err := system.Points.AddBind(mount.ScratchTag, fullSourceDir, dir, flags); err != nil {
			return fmt.Errorf("could not bind scratch directory %s into container: %s", fullSourceDir, err)
		}
		system.Points.AddRemount(mount.ScratchTag, dir, flags)
	}
	return nil
}

func (c *container) addCwdMount(system *mount.System) error {
	cwd := ""

	if c.engine.EngineConfig.GetContain() {
		sylog.Verbosef("Not mounting current directory: container was requested")
		return nil
	}
	if !c.engine.EngineConfig.File.UserBindControl {
		sylog.Warningf("Not mounting current directory: user bind control is disabled by system administrator")
		return nil
	}
	if c.engine.EngineConfig.OciConfig.Process == nil {
		return nil
	}
	cwd = c.engine.EngineConfig.OciConfig.Process.Cwd
	if err := os.Chdir(cwd); err != nil {
		if os.IsNotExist(err) {
			sylog.Debugf("Container working directory %s doesn't exist, will retry after chroot", cwd)
		} else {
			sylog.Warningf("Could not set container working directory %s: %s", cwd, err)
		}
		return nil
	}
	current, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not obtain current directory path: %s", err)
	}
	switch current {
	case "/", "/etc", "/bin", "/mnt", "/usr", "/var", "/opt", "/sbin", "/lib", "/lib64":
		sylog.Verbosef("Not mounting CWD within operating system directory: %s", current)
		return nil
	}
	if strings.HasPrefix(current, "/sys") || strings.HasPrefix(current, "/proc") || strings.HasPrefix(current, "/dev") {
		sylog.Verbosef("Not mounting CWD within virtual directory: %s", current)
		return nil
	}
	flags := uintptr(syscall.MS_BIND | c.suidFlag | syscall.MS_NODEV | syscall.MS_REC)
	if err := system.Points.AddBind(mount.CwdTag, current, cwd, flags); err == nil {
		system.Points.AddRemount(mount.CwdTag, cwd, flags)
		c.checkDest = append(c.checkDest, cwd)
		sylog.Verbosef("Default mount: %v: to the container", cwd)
	} else {
		sylog.Warningf("Could not bind CWD to container %s: %s", current, err)
	}
	return nil
}

func (c *container) addLibsMount(system *mount.System) error {
	sylog.Debugf("Checking for 'user bind control' in configuration file")
	if !c.engine.EngineConfig.File.UserBindControl {
		sylog.Warningf("Ignoring libraries bind request: user bind control disabled by system administrator")
		return nil
	}

	flags := uintptr(syscall.MS_BIND | syscall.MS_NOSUID | syscall.MS_NODEV | syscall.MS_RDONLY | syscall.MS_REC)

	containerDir := "/.singularity.d/libs"
	sessionDir := "/libs"

	if err := c.session.AddDir(sessionDir); err != nil {
		return err
	}

	libraries := c.engine.EngineConfig.GetLibrariesPath()

	for _, lib := range libraries {
		sylog.Debugf("Add library %s to mount list", lib)

		file := filepath.Base(lib)
		sessionFile := filepath.Join(sessionDir, file)

		if err := c.session.AddFile(sessionFile, []byte{}); err != nil {
			return err
		}

		sessionFilePath, _ := c.session.GetPath(sessionFile)

		err := system.Points.AddBind(mount.FilesTag, lib, sessionFilePath, flags)
		if err != nil {
			return fmt.Errorf("unable to add %s to mount list: %s", lib, err)
		}

		system.Points.AddRemount(mount.FilesTag, sessionFilePath, flags)
	}

	if len(libraries) > 0 {
		sessionDirPath, _ := c.session.GetPath(sessionDir)

		err := system.Points.AddBind(mount.FilesTag, sessionDirPath, containerDir, flags)
		if err != nil {
			return fmt.Errorf("unable to add %s to mount list: %s", sessionDirPath, err)
		}

		return system.Points.AddRemount(mount.FilesTag, containerDir, flags)
	}

	return nil
}

func (c *container) addIdentityMount(system *mount.System) error {
	if os.Geteuid() == 0 && c.engine.EngineConfig.GetTargetUID() == 0 {
		sylog.Verbosef("Not updating passwd/group files, running as root!")
		return nil
	}

	rootfs := c.session.RootFsPath()
	defer c.session.Update()

	uid := os.Getuid()
	if uid == 0 && c.engine.EngineConfig.GetTargetUID() != 0 {
		uid = c.engine.EngineConfig.GetTargetUID()
	}

	if c.engine.EngineConfig.File.ConfigPasswd {
		passwd := filepath.Join(rootfs, "/etc/passwd")
		_, home, err := c.getHomePaths()
		if err != nil {
			sylog.Warningf("%s", err)
		} else {
			content, err := files.Passwd(passwd, home, uid)
			if err != nil {
				sylog.Warningf("%s", err)
			} else {
				if err := c.session.AddFile("/etc/passwd", content); err != nil {
					sylog.Warningf("failed to add passwd session file: %s", err)
				}
				passwd, _ = c.session.GetPath("/etc/passwd")

				sylog.Debugf("Adding /etc/passwd to mount list\n")
				err = system.Points.AddBind(mount.FilesTag, passwd, "/etc/passwd", syscall.MS_BIND)
				if err != nil {
					return fmt.Errorf("unable to add /etc/passwd to mount list: %s", err)
				}
				sylog.Verbosef("Default mount: /etc/passwd:/etc/passwd")
			}
		}
	} else {
		sylog.Verbosef("Skipping bind of the host's /etc/passwd")
	}

	if c.engine.EngineConfig.File.ConfigGroup {
		group := filepath.Join(rootfs, "/etc/group")
		content, err := files.Group(group, uid, c.engine.EngineConfig.GetTargetGID())
		if err != nil {
			sylog.Warningf("%s", err)
		} else {
			if err := c.session.AddFile("/etc/group", content); err != nil {
				sylog.Warningf("failed to add group session file: %s", err)
			}
			group, _ = c.session.GetPath("/etc/group")

			sylog.Debugf("Adding /etc/group to mount list\n")
			err = system.Points.AddBind(mount.FilesTag, group, "/etc/group", syscall.MS_BIND)
			if err != nil {
				return fmt.Errorf("unable to add /etc/group to mount list: %s", err)
			}
			sylog.Verbosef("Default mount: /etc/group:/etc/group")
		}
	} else {
		sylog.Verbosef("Skipping bind of the host's /etc/group")
	}

	return nil
}

func (c *container) addResolvConfMount(system *mount.System) error {
	resolvConf := "/etc/resolv.conf"

	if c.engine.EngineConfig.File.ConfigResolvConf {
		var err error
		var content []byte

		dns := c.engine.EngineConfig.GetDNS()

		if dns == "" {
			r, err := os.Open(resolvConf)
			if err != nil {
				return err
			}
			content, err = ioutil.ReadAll(r)
			if err != nil {
				return err
			}
		} else {
			dns = strings.Replace(dns, " ", "", -1)
			content, err = files.ResolvConf(strings.Split(dns, ","))
			if err != nil {
				return err
			}
		}
		if err := c.session.AddFile(resolvConf, content); err != nil {
			sylog.Warningf("failed to add resolv.conf session file: %s", err)
		}
		sessionFile, _ := c.session.GetPath(resolvConf)

		sylog.Debugf("Adding %s to mount list\n", resolvConf)
		err = system.Points.AddBind(mount.FilesTag, sessionFile, resolvConf, syscall.MS_BIND)
		if err != nil {
			return fmt.Errorf("unable to add %s to mount list: %s", resolvConf, err)
		}
		sylog.Verbosef("Default mount: /etc/resolv.conf:/etc/resolv.conf")
	} else {
		sylog.Verbosef("Skipping bind of the host's %s", resolvConf)
	}
	return nil
}

func (c *container) addHostnameMount(system *mount.System) error {
	hostnameFile := "/etc/hostname"

	if c.utsNS {
		hostname := c.engine.EngineConfig.GetHostname()
		if hostname != "" {
			sylog.Debugf("Set container hostname %s", hostname)

			content, err := files.Hostname(hostname)
			if err != nil {
				return fmt.Errorf("unable to add %s to hostname file: %s", hostname, err)
			}
			if err := c.session.AddFile(hostnameFile, content); err != nil {
				return fmt.Errorf("failed to add hostname session file: %s", err)
			}
			sessionFile, _ := c.session.GetPath(hostnameFile)

			sylog.Debugf("Adding %s to mount list\n", hostnameFile)
			err = system.Points.AddBind(mount.FilesTag, sessionFile, hostnameFile, syscall.MS_BIND)
			if err != nil {
				return fmt.Errorf("unable to add %s to mount list: %s", hostnameFile, err)
			}
			sylog.Verbosef("Default mount: /etc/hostname:/etc/hostname")
			if _, err := c.rpcOps.SetHostname(hostname); err != nil {
				return fmt.Errorf("failed to set container hostname: %s", err)
			}
		}
	} else {
		sylog.Debugf("Skipping hostname mount, not virtualizing UTS namespace on user request")
	}
	return nil
}

func (c *container) addActionsMount(system *mount.System) error {
	hostDir := filepath.Join(buildcfg.SYSCONFDIR, "/singularity/actions")
	containerDir := "/.singularity.d/actions"
	flags := uintptr(syscall.MS_BIND | syscall.MS_RDONLY | syscall.MS_NOSUID | syscall.MS_NODEV)

	actionsDir := filepath.Join(c.session.RootFsPath(), containerDir)
	if !fs.IsDir(actionsDir) {
		sylog.Debugf("Ignoring actions mount, %s doesn't exist", actionsDir)
		return nil
	}

	err := system.Points.AddBind(mount.BindsTag, hostDir, containerDir, flags)
	if err != nil {
		return fmt.Errorf("unable to add %s to mount list: %s", containerDir, err)
	}

	return system.Points.AddRemount(mount.BindsTag, containerDir, flags)
}
