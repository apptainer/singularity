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
	"github.com/sylabs/singularity/internal/pkg/runtime/engine/singularity/rpc/client"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/internal/pkg/util/fs/files"
	"github.com/sylabs/singularity/internal/pkg/util/fs/layout"
	"github.com/sylabs/singularity/internal/pkg/util/fs/layout/layer/overlay"
	"github.com/sylabs/singularity/internal/pkg/util/fs/layout/layer/underlay"
	"github.com/sylabs/singularity/internal/pkg/util/fs/mount"
	fsoverlay "github.com/sylabs/singularity/internal/pkg/util/fs/overlay"
	"github.com/sylabs/singularity/internal/pkg/util/mainthread"
	"github.com/sylabs/singularity/internal/pkg/util/priv"
	"github.com/sylabs/singularity/internal/pkg/util/user"
	"github.com/sylabs/singularity/pkg/image"
	"github.com/sylabs/singularity/pkg/network"
	singularity "github.com/sylabs/singularity/pkg/runtime/engines/singularity/config"
	"github.com/sylabs/singularity/pkg/util/fs/proc"
	"github.com/sylabs/singularity/pkg/util/loop"
	"github.com/sylabs/singularity/pkg/util/namespaces"
	"github.com/sylabs/singularity/pkg/util/nvidia"
	"golang.org/x/crypto/ssh/terminal"
)

// defaultCNIConfPath is the default directory to CNI network configuration files.
var defaultCNIConfPath = filepath.Join(buildcfg.SYSCONFDIR, "singularity", "network")

// defaultCNIPluginPath is the default directory to CNI plugins executables.
var defaultCNIPluginPath = filepath.Join(buildcfg.LIBEXECDIR, "singularity", "cni")

type lastMount struct {
	dest  string
	flags uintptr
}

type container struct {
	engine        *EngineOperations
	rpcOps        *client.RPC
	session       *layout.Session
	sessionFsType string
	sessionSize   int
	userNS        bool
	pidNS         bool
	utsNS         bool
	netNS         bool
	ipcNS         bool
	mountInfoPath string
	lastMount     lastMount
	skippedMount  []string
	suidFlag      uintptr
	devSourcePath string
}

func create(engine *EngineOperations, rpcOps *client.RPC, pid int) error {
	var err error

	c := &container{
		engine:        engine,
		rpcOps:        rpcOps,
		sessionFsType: engine.EngineConfig.File.MemoryFSType,
		mountInfoPath: fmt.Sprintf("/proc/%d/mountinfo", pid),
		skippedMount:  make([]string, 0),
		suidFlag:      syscall.MS_NOSUID,
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

	// user namespace was not requested but we need to check
	// if we are currently running in a user namespace and set
	// value accordingly to avoid remount errors while running
	// inside a user namespace
	if !c.userNS {
		c.userNS, _ = namespaces.IsInsideUserNamespace(os.Getpid())
	}

	p := &mount.Points{}
	system := &mount.System{Points: p, Mount: c.mount}

	if err := c.setupSessionLayout(system); err != nil {
		return err
	}

	if err := system.RunAfterTag(mount.SharedTag, c.addIdentityMount); err != nil {
		return err
	}
	// this call must occur just after all container layers are mounted
	// to prevent user binds to screw up session final directory and
	// consequently chroot
	if err := system.RunAfterTag(mount.SharedTag, c.chdirFinal); err != nil {
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
	if err := c.addFilesMount(system); err != nil {
		return err
	}
	if err := c.addResolvConfMount(system); err != nil {
		return err
	}
	if err := c.addHostnameMount(system); err != nil {
		return err
	}
	if err := c.addFuseMount(system); err != nil {
		return err
	}

	networkSetup, err := c.prepareNetworkSetup(system, pid)
	if err != nil {
		return err
	}

	sylog.Debugf("Mount all")
	if err := system.MountAll(); err != nil {
		return err
	}

	// chroot from RPC server current working directory since
	// it's already in final directory after chdirFinal call
	sylog.Debugf("Chroot into %s\n", c.session.FinalPath())
	_, err = c.rpcOps.Chroot(".", "pivot")
	if err != nil {
		sylog.Debugf("Fallback to move/chroot")
		_, err = c.rpcOps.Chroot(".", "move")
		if err != nil {
			return fmt.Errorf("chroot failed: %s", err)
		}
	}

	if networkSetup != nil {
		if err := networkSetup(); err != nil {
			return err
		}
	}

	if os.Geteuid() == 0 && !c.userNS {
		path := engine.EngineConfig.GetCgroupsPath()
		if path != "" {
			cgroupPath := filepath.Join("/singularity", strconv.Itoa(pid))
			manager := &cgroups.Manager{Pid: pid, Path: cgroupPath}
			if err := manager.ApplyFromFile(path); err != nil {
				return fmt.Errorf("failed to apply cgroups resources restriction: %s", err)
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

// setupSessionLayout will create the session layout according to the capabilities of Singularity
// on the system. It will first attempt to use "overlay", followed by "underlay", and if neither
// are available it will not use either. If neither are used, we will not be able to bind mount
// to non-existent paths within the container
func (c *container) setupSessionLayout(system *mount.System) error {
	var err error
	var sessionPath string

	sessionPath, err = filepath.EvalSymlinks(buildcfg.SESSIONDIR)
	if err != nil {
		return fmt.Errorf("failed to resolve session directory %s: %s", buildcfg.SESSIONDIR, err)
	}

	sessionLayer := c.engine.EngineConfig.GetSessionLayer()

	sylog.Debugf("Using Layer system: %s\n", sessionLayer)

	switch sessionLayer {
	case singularity.DefaultLayer:
		err = c.setupDefaultLayout(system, sessionPath)
	case singularity.OverlayLayer:
		err = c.setupOverlayLayout(system, sessionPath)
	case singularity.UnderlayLayer:
		err = c.setupUnderlayLayout(system, sessionPath)
	default:
		return fmt.Errorf("unknown session layer set: %s", sessionLayer)
	}

	if err != nil {
		return fmt.Errorf("while setting %s session layout: %s", sessionLayer, err)
	}

	return system.RunAfterTag(mount.SharedTag, c.setPropagationMount)
}

// setupOverlayLayout sets up the session with overlay filesystem
func (c *container) setupOverlayLayout(system *mount.System, sessionPath string) (err error) {
	sylog.Debugf("Creating overlay SESSIONDIR layout\n")
	if c.session, err = layout.NewSession(sessionPath, c.sessionFsType, c.sessionSize, system, overlay.New()); err != nil {
		return err
	}
	return c.addOverlayMount(system)
}

// setupUnderlayLayout sets up the session with underlay "filesystem"
func (c *container) setupUnderlayLayout(system *mount.System, sessionPath string) (err error) {
	sylog.Debugf("Creating underlay SESSIONDIR layout\n")
	c.session, err = layout.NewSession(sessionPath, c.sessionFsType, c.sessionSize, system, underlay.New())
	return err
}

// setupDefaultLayout sets up the session without overlay or underlay
func (c *container) setupDefaultLayout(system *mount.System, sessionPath string) (err error) {
	sylog.Debugf("Creating default SESSIONDIR layout\n")
	c.session, err = layout.NewSession(sessionPath, c.sessionFsType, c.sessionSize, system, nil)
	return err
}

// isLayerEnabled returns whether or not overlay or underlay system
// is enabled
func (c *container) isLayerEnabled() bool {
	return c.engine.EngineConfig.GetSessionLayer() != singularity.DefaultLayer
}

func (c *container) mount(point *mount.Point, system *mount.System) error {
	if _, err := mount.GetOffset(point.InternalOptions); err == nil {
		if err := c.mountImage(point); err != nil {
			return fmt.Errorf("while mounting image %s: %s", point.Source, err)
		}
	} else {
		tag := system.CurrentTag()
		if err := c.mountGeneric(point, tag); err != nil {
			return fmt.Errorf("while mounting %s: %s", point.Source, err)
		}
	}
	return nil
}

// setPropagationMount will apply propagation flag set by
// configuration directive, when applied master process
// won't see mount done by RPC server anymore. Typically
// called after SharedTag mounts
func (c *container) setPropagationMount(system *mount.System) error {
	pflags := uintptr(syscall.MS_REC)

	if c.engine.EngineConfig.File.MountSlave {
		sylog.Debugf("Set RPC mount propagation flag to SLAVE")
		pflags |= syscall.MS_SLAVE
	} else {
		sylog.Debugf("Set RPC mount propagation flag to PRIVATE")
		pflags |= syscall.MS_PRIVATE
	}

	if err := c.rpcOps.Mount("", "/", "", pflags, ""); err != nil {
		return err
	}

	return nil
}

func (c *container) chdirFinal(system *mount.System) error {
	if _, err := c.rpcOps.Chdir(c.session.FinalPath()); err != nil {
		return err
	}
	return nil
}

func (c *container) checkMounted(dest string) string {
	if dest[0] != '/' {
		return ""
	}

	entries, err := proc.GetMountInfoEntry(c.mountInfoPath)
	if err != nil {
		return ""
	}
	d, err := proc.FindParentMountEntry(dest, entries)
	if err != nil {
		return ""
	}

	finalPath := c.session.FinalPath()
	finalDest := filepath.Join(finalPath, dest)

	for _, e := range entries {
		if e.Dev == d.Dev && e.Point != "/" && strings.HasPrefix(finalDest, e.Point) {
			return e.Point
		}
	}

	return ""
}

// mount any generic mount (not loop dev)
func (c *container) mountGeneric(mnt *mount.Point, tag mount.AuthorizedTag) (err error) {
	flags, opts := mount.ConvertOptions(mnt.Options)
	optsString := strings.Join(opts, ",")
	sessionPath := c.session.Path()
	bindMount := flags&syscall.MS_BIND != 0
	remount := mount.HasRemountFlag(flags)
	propagation := mount.HasPropagationFlag(flags)
	source := mnt.Source
	dest := ""

	if bindMount {
		if !remount {
			if _, err := os.Stat(source); os.IsNotExist(err) {
				return fmt.Errorf("mount source %s doesn't exist", source)
			} else if err != nil {
				return fmt.Errorf("while getting stat for %s: %s", source, err)
			}

			// retrieve original mount flags from the parent mount point
			// where source is located on
			flags, err = c.getBindFlags(source, flags)
			if err != nil {
				return fmt.Errorf("while getting mount flags for %s: %s", source, err)
			}
			// save them for the remount step
			c.lastMount = lastMount{
				dest:  mnt.Destination,
				flags: flags,
			}
		} else if c.lastMount.dest == mnt.Destination {
			flags = c.lastMount.flags | flags
			c.lastMount.dest = ""
			c.lastMount.flags = 0
		}
	}

	if !strings.HasPrefix(mnt.Destination, sessionPath) {
		dest = fs.EvalRelative(mnt.Destination, c.session.FinalPath())
		dest = filepath.Join(c.session.FinalPath(), dest)
	} else {
		dest = mnt.Destination
	}

	if remount || propagation {
		for _, skipped := range c.skippedMount {
			if skipped == mnt.Destination {
				return nil
			}
		}
		sylog.Debugf("Remounting %s\n", dest)
	} else {
		if tag == mount.CwdTag {
			cwd := c.engine.EngineConfig.GetCwd()
			mounted := c.checkMounted(cwd)
			if mounted != "" {
				c.skippedMount = append(c.skippedMount, mnt.Destination)
				sylog.Verbosef("Skipping mount %s, %s already mounted", cwd, mounted)
				return nil
			}
		}
		sylog.Debugf("Mounting %s to %s\n", source, dest)

		// in stage 1 we changed current working directory to
		// sandbox image directory, just pass "." as source argument to
		// be sure RPC mount the right sandbox image
		if tag == mount.RootfsTag && dest == c.session.RootFsPath() {
			source = "."
		}

		// overlay requires root filesystem UID/GID since upper/work
		// directories are owned by root
		if tag == mount.LayerTag && mnt.Type == "overlay" {
			c.rpcOps.SetFsID(0, 0)
			defer c.rpcOps.SetFsID(os.Getuid(), os.Getgid())
		}
	}

	err = c.rpcOps.Mount(source, dest, mnt.Type, flags, optsString)
	if os.IsNotExist(err) {
		switch tag {
		case mount.KernelTag,
			mount.HostfsTag,
			mount.BindsTag,
			mount.CwdTag,
			mount.FilesTag,
			mount.TmpTag:
			c.skippedMount = append(c.skippedMount, mnt.Destination)
			sylog.Warningf("Skipping mount %s [%s]: %s doesn't exist in container", source, tag, mnt.Destination)
			return nil
		default:
			if c.engine.EngineConfig.GetWritableImage() {
				sylog.Warningf(
					"By using --writable, Singularity can't create %s destination automatically without overlay or underlay",
					mnt.Destination,
				)
			} else if !c.isLayerEnabled() {
				sylog.Warningf("No layer in use (overlay or underlay), check your configuration, "+
					"Singularity can't create %s destination automatically without overlay or underlay", mnt.Destination)
			}
			return fmt.Errorf("destination %s doesn't exist in container", mnt.Destination)
		}
	} else if err != nil {
		if !bindMount {
			if mnt.Source == "devpts" {
				sylog.Verbosef("Couldn't mount devpts filesystem, continuing with PTY allocation functionality disabled")
				return nil
			}
			// mount error for other filesystems is considered fatal
			return fmt.Errorf("can't mount %s filesystem to %s: %s", mnt.Type, mnt.Destination, err)
		}
		if remount {
			if os.IsPermission(err) && c.userNS {
				// when using user namespace we always try to apply mount flags with
				// remount, then if we get a permission denied error, we continue
				// execution by ignoring the error and warn user if the bind mount
				// need to be mounted read-only
				if flags&syscall.MS_RDONLY != 0 {
					sylog.Warningf("Could not remount %s read-only: %s", mnt.Destination, err)
				} else {
					sylog.Verbosef("Could not remount %s: %s", mnt.Destination, err)
				}
				return nil
			}
			return fmt.Errorf("could not remount %s: %s", mnt.Destination, err)
		}
		return fmt.Errorf("could not mount %s: %s", mnt.Source, err)
	}

	return nil
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

	sylog.Debugf("Mounting loop device %s to %s of type %s\n", path, mnt.Destination, mnt.Type)

	mountType := mnt.Type

	if mountType == "encryptfs" {
		key, err := mount.GetKey(mnt.InternalOptions)
		if err != nil {
			return err
		}

		// pass the master processus ID only if a container IPC
		// namespace was requested because cryptsetup requires
		// to run in the host IPC namespace
		masterPid := 0
		if c.ipcNS {
			masterPid = os.Getpid()
		}

		cryptDev, err := c.rpcOps.Decrypt(offset, path, key, masterPid)

		if err != nil {
			return fmt.Errorf("unable to decrypt the file system: %s", err)
		}

		path = cryptDev

		// Save this device to cleanup later
		c.engine.EngineConfig.CryptDev = cryptDev

		// Currently we only support encrypted squashfs file system
		mountType = "squashfs"
	}
	err = c.rpcOps.Mount(path, mnt.Destination, mountType, flags, optsString)
	switch err {
	case syscall.EINVAL:
		if mountType == "squashfs" {
			return fmt.Errorf(
				"kernel reported a bad superblock for %s image partition, "+
					"possible causes are that your kernel doesn't support "+
					"the compression algorithm or the image is corrupted",
				mountType)
		}
		return fmt.Errorf("%s image partition contains a bad superblock (corrupted image ?)", mountType)
	case syscall.ENODEV:
		return fmt.Errorf("%s filesystem seems not enabled and/or supported by your kernel", mountType)
	default:
		if err != nil {
			return fmt.Errorf("failed to mount %s filesystem: %s", mountType, err)
		}
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
	var key []byte

	sylog.Debugf("Image type is %v", imageObject.Partitions[0].Type)

	switch imageObject.Partitions[0].Type {
	case image.SQUASHFS:
		mountType = "squashfs"
	case image.EXT3:
		mountType = "ext3"
	case image.ENCRYPTSQUASHFS:
		mountType = "encryptfs"
		key = c.engine.EngineConfig.GetEncryptionKey()
	case image.SANDBOX:
		sylog.Debugf("Mounting directory rootfs: %v\n", rootfs)
		flags |= syscall.MS_BIND
		if err := system.Points.AddBind(mount.RootfsTag, rootfs, c.session.RootFsPath(), flags); err != nil {
			return err
		}
		return system.Points.AddRemount(mount.RootfsTag, c.session.RootFsPath(), flags)
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
		key,
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

			err = system.Points.AddImage(mount.PreLayerTag, src, dst, "ext3", flags, offset, size, nil)
			if err != nil {
				return fmt.Errorf("while adding ext3 image: %s", err)
			}
		case image.SQUASHFS:
			flags := uintptr(c.suidFlag | syscall.MS_NODEV | syscall.MS_RDONLY)
			err = system.Points.AddImage(mount.PreLayerTag, src, dst, "squashfs", flags, offset, size, nil)
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
				// check if the sandbox directory is located on a compatible
				// filesystem usable overlay lower directory
				if err := fsoverlay.CheckLower(imageObject.Path); err != nil {
					return err
				}
				if fs.IsDir(filepath.Join(imageObject.Path, "upper")) {
					ov.AddLowerDir(filepath.Join(dst, "upper"))
				} else {
					ov.AddLowerDir(dst)
				}
			} else {
				// check if the sandbox directory is located on a compatible
				// filesystem usable with overlay upper directory
				if err := fsoverlay.CheckUpper(imageObject.Path); err != nil {
					return err
				}
			}
		case image.SIF:
			return fmt.Errorf("%s: SIF image not supported as overlay image", imageObject.Path)
		default:
			return fmt.Errorf("%s: overlay image with unknown format", imageObject.Path)
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
				return fmt.Errorf("multiple devpts instances unsupported and /dev/pts configured")
			}

			sylog.Debugf("Creating temporary staged /dev/pts")
			if err := c.session.AddDir("/dev/pts"); err != nil {
				return fmt.Errorf("failed to add /dev/pts session directory: %s", err)
			}

			options := "mode=0620,newinstance,ptmxmode=0666"
			if !c.userNS {
				group, err := user.GetGrNam("tty")
				if err != nil {
					return fmt.Errorf("problem resolving 'tty' group gid: %s", err)
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
			ttylink, err := mainthread.Readlink(procfd)
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
			break
		}
		if err := c.addSessionDev("/dev/tty", system); err != nil {
			return err
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
			devs, err := nvidia.Devices(true)
			if err != nil {
				return fmt.Errorf("failed to get nvidia devices: %v", err)
			}
			for _, dev := range devs {
				if err := c.addSessionDev(dev, system); err != nil {
					return err
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
		if err := system.RunAfterTag(mount.SharedTag, c.addSessionDevMount); err != nil {
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

	info, err := proc.GetMountPointMap("/proc/self/mountinfo")
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
		pw, err := user.Current()
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

	bindSource := !c.engine.EngineConfig.GetContain() || c.engine.EngineConfig.GetCustomHome()

	// use the session home directory is the user home directory doesn't exist (issue #4208)
	if _, err := os.Stat(source); os.IsNotExist(err) {
		bindSource = false
	}

	if bindSource {
		sylog.Debugf("Staging home directory (%v) at %v\n", source, homeStage)

		if err := system.Points.AddBind(mount.HomeTag, source, homeStage, flags); err != nil {
			return "", fmt.Errorf("unable to add %s to mount list: %s", source, err)
		}
		system.Points.AddRemount(mount.HomeTag, homeStage, flags)
		c.session.OverrideDir(dest, source)
	} else {
		sylog.Debugf("Using session directory for home directory")
		c.session.OverrideDir(dest, homeStage)
	}

	return homeStage, nil
}

// addHomeLayer adds the home mount when using either overlay or underlay
func (c *container) addHomeLayer(system *mount.System, source, dest string) error {
	flags := uintptr(syscall.MS_BIND | c.suidFlag | syscall.MS_NODEV | syscall.MS_REC)

	if len(system.Points.GetByTag(mount.HomeTag)) > 0 {
		flags = uintptr(syscall.MS_BIND | syscall.MS_REC)
		if err := system.Points.AddBind(mount.HomeTag, source, dest, flags); err != nil {
			return fmt.Errorf("unable to add home to mount list: %s", err)
		}
		return nil
	}

	if err := system.Points.AddBind(mount.HomeTag, source, dest, flags); err != nil {
		return fmt.Errorf("unable to add home to mount list: %s", err)
	}

	return system.Points.AddRemount(mount.HomeTag, dest, flags)
}

// addHomeNoLayer is responsible for staging the home directory and adding the base
// directory of the staged home into the container when overlay/underlay are unavailable
func (c *container) addHomeNoLayer(system *mount.System, source, dest string) error {
	flags := uintptr(syscall.MS_BIND | syscall.MS_REC)

	homeBase := fs.RootDir(dest)
	if homeBase == "." {
		return fmt.Errorf("could not identify staged home directory base: %s", dest)
	}

	homeStageBase, _ := c.session.GetPath(homeBase)

	sylog.Verbosef("Mounting staged home directory base (%v) into container at %v\n", homeStageBase, filepath.Join(c.session.FinalPath(), homeBase))
	if err := system.Points.AddBind(mount.HomeTag, homeStageBase, homeBase, flags); err != nil {
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

	if !c.engine.EngineConfig.GetCustomHome() && !c.engine.EngineConfig.File.MountHome {
		sylog.Debugf("Skipping home dir mounting (per config)")
		return nil
	}

	// check if user attempt to mount a custom home when not allowed to
	if c.engine.EngineConfig.GetCustomHome() && !c.engine.EngineConfig.File.UserBindControl {
		return fmt.Errorf("not mounting user requested home: user bind control is disallowed")
	}

	source, dest, err := c.getHomePaths()
	if err != nil {
		return fmt.Errorf("unable to get home source/destination: %v", err)
	}

	stagingDir, err := c.addHomeStagingDir(system, source, dest)
	if err != nil {
		return err
	}

	sessionLayer := c.engine.EngineConfig.GetSessionLayer()
	sylog.Debugf("Adding home directory mount [%v:%v] to list using layer: %s\n", stagingDir, dest, sessionLayer)
	if !c.isLayerEnabled() {
		return c.addHomeNoLayer(system, stagingDir, dest)
	}
	return c.addHomeLayer(system, stagingDir, dest)
}

func (c *container) addUserbindsMount(system *mount.System) error {
	devicesMounted := 0
	devPrefix := "/dev"
	userBindControl := c.engine.EngineConfig.File.UserBindControl
	defaultFlags := uintptr(syscall.MS_BIND | c.suidFlag | syscall.MS_NODEV | syscall.MS_REC)

	if len(c.engine.EngineConfig.GetBindPath()) == 0 {
		return nil
	}

	for _, b := range c.engine.EngineConfig.GetBindPath() {
		flags := defaultFlags
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

		// special case for /dev mount to override default mount behavior
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

		if err := system.Points.AddBind(mount.UserbindsTag, src, dst, flags); err == mount.ErrMountExists {
			sylog.Warningf("destination %s already in mount list: %s", src, err)
		} else if err != nil {
			return fmt.Errorf("unable to add %s to mount list: %s", src, err)
		} else {
			c.session.OverrideDir(dst, src)
			system.Points.AddRemount(mount.UserbindsTag, dst, flags)
		}
	}

	sylog.Debugf("Checking for 'user bind control' in configuration file")
	if !userBindControl && devicesMounted == 0 {
		sylog.Warningf("Ignoring user bind request: user bind control disabled by system administrator")
	}

	return nil
}

func (c *container) addTmpMount(system *mount.System) error {
	const (
		tmpPath    = "/tmp"
		varTmpPath = "/var/tmp"
	)

	sylog.Debugf("Checking for 'mount tmp' in configuration file")
	if !c.engine.EngineConfig.File.MountTmp {
		sylog.Verbosef("Skipping tmp dir mounting (per config)")
		return nil
	}

	tmpSource := tmpPath
	vartmpSource := varTmpPath

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

	c.session.OverrideDir(tmpPath, tmpSource)
	c.session.OverrideDir(varTmpPath, vartmpSource)

	flags := uintptr(syscall.MS_BIND | c.suidFlag | syscall.MS_NODEV | syscall.MS_REC)

	if err := system.Points.AddBind(mount.TmpTag, tmpSource, tmpPath, flags); err == nil {
		system.Points.AddRemount(mount.TmpTag, tmpPath, flags)
		sylog.Verbosef("Default mount: %s:%s", tmpPath, tmpPath)
	} else {
		return fmt.Errorf("could not mount container's %s directory: %s", tmpPath, err)
	}

	if err := system.Points.AddBind(mount.TmpTag, vartmpSource, varTmpPath, flags); err == nil {
		system.Points.AddRemount(mount.TmpTag, varTmpPath, flags)
		sylog.Verbosef("Default mount: %s:%s", varTmpPath, varTmpPath)
	} else {
		return fmt.Errorf("could not mount container's %s directory: %s", varTmpPath, err)
	}
	return nil
}

func (c *container) addScratchMount(system *mount.System) error {
	const scratchSessionDir = "/scratch"

	scratchDir := c.engine.EngineConfig.GetScratchDir()
	if len(scratchDir) == 0 {
		sylog.Debugf("Not mounting scratch directory: Not requested")
		return nil
	} else if len(scratchDir) == 1 {
		scratchDir = strings.Split(filepath.Clean(scratchDir[0]), ",")
	}
	if !c.engine.EngineConfig.File.UserBindControl {
		sylog.Verbosef("Not mounting scratch: user bind control disabled by system administrator")
		return nil
	}

	workdir := c.engine.EngineConfig.GetWorkdir()
	hasWorkdir := workdir != ""

	if hasWorkdir {
		workdir = filepath.Clean(workdir)
		sourceDir := filepath.Join(workdir, scratchSessionDir)
		if err := fs.MkdirAll(sourceDir, 0750); err != nil {
			return fmt.Errorf("could not create scratch working directory %s: %s", sourceDir, err)
		}
	}

	for _, dir := range scratchDir {
		src := filepath.Join(scratchSessionDir, dir)
		if err := c.session.AddDir(src); err != nil {
			return fmt.Errorf("could not create scratch working directory %s: %s", src, err)
		}
		fullSourceDir, _ := c.session.GetPath(src)
		if hasWorkdir {
			fullSourceDir = filepath.Join(workdir, scratchSessionDir, dir)
			if err := fs.MkdirAll(fullSourceDir, 0750); err != nil {
				return fmt.Errorf("could not create scratch working directory %s: %s", fullSourceDir, err)
			}
		}
		c.session.OverrideDir(dir, fullSourceDir)

		flags := uintptr(syscall.MS_BIND | c.suidFlag | syscall.MS_NODEV | syscall.MS_REC)
		if err := system.Points.AddBind(mount.ScratchTag, fullSourceDir, dir, flags); err != nil {
			return fmt.Errorf("could not bind scratch directory %s into container: %s", fullSourceDir, err)
		}
		system.Points.AddRemount(mount.ScratchTag, dir, flags)
	}
	return nil
}

func (c *container) addCwdMount(system *mount.System) error {
	if c.engine.EngineConfig.GetContain() {
		sylog.Verbosef("Not mounting current directory: container was requested")
		return nil
	}
	if !c.engine.EngineConfig.File.UserBindControl {
		sylog.Warningf("Not mounting current directory: user bind control is disabled by system administrator")
		return nil
	}
	cwd := c.engine.EngineConfig.GetCwd()
	if cwd == "" {
		sylog.Warningf("Not current working directory set: skipping mount")
	}

	current, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		return fmt.Errorf("could not obtain current directory path: %s", err)
	}
	sylog.Debugf("Using %s as current working directory", cwd)

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
	if err := system.Points.AddBind(mount.CwdTag, cwd, current, flags); err == nil {
		system.Points.AddRemount(mount.CwdTag, current, flags)
		sylog.Verbosef("Default mount: %v: to the container", cwd)
	} else {
		sylog.Warningf("Could not bind CWD to container %s: %s", cwd, err)
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

func (c *container) addFilesMount(system *mount.System) error {
	sylog.Debugf("Checking for 'user bind control' in configuration file")
	if !c.engine.EngineConfig.File.UserBindControl {
		sylog.Warningf("Ignoring binaries bind request: user bind control disabled by system administrator")
		return nil
	}

	flags := uintptr(syscall.MS_BIND | syscall.MS_NOSUID | syscall.MS_NODEV | syscall.MS_RDONLY | syscall.MS_REC)

	files := c.engine.EngineConfig.GetFilesPath()

	for _, file := range files {
		sylog.Debugf("Adding file %s to mount list", file)

		splitted := strings.Split(file, ":")
		src := splitted[0]
		dst := splitted[0]
		if len(splitted) > 1 {
			dst = splitted[1]
		}
		err := system.Points.AddBind(mount.FilesTag, src, dst, flags)
		if err != nil {
			return fmt.Errorf("unable to add %s to mount list: %s", src, err)
		}

		system.Points.AddRemount(mount.FilesTag, dst, flags)
	}

	return nil
}

func (c *container) addIdentityMount(system *mount.System) error {
	if (os.Geteuid() == 0 && c.engine.EngineConfig.GetTargetUID() == 0) ||
		c.engine.EngineConfig.GetFakeroot() {
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
			r.Close()
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

func (c *container) prepareNetworkSetup(system *mount.System, pid int) (func() error, error) {
	const (
		fakerootNet  = "fakeroot"
		noneNet      = "none"
		procNetNs    = "/proc/self/ns/net"
		sessionNetNs = "/netns"
	)

	fakeroot := c.engine.EngineConfig.GetFakeroot()
	net := c.engine.EngineConfig.GetNetwork()
	euid := os.Geteuid()

	if !c.netNS || net == noneNet {
		return nil, nil
	} else if (c.userNS || euid != 0) && !fakeroot {
		return nil, fmt.Errorf("network requires root or --fakeroot, users need to specify --network=%s with --net", noneNet)
	}

	// we hold a reference to container network namespace
	// by binding /proc/self/ns/net (from the RPC server) inside the
	// session directory
	if err := c.session.AddFile(sessionNetNs, nil); err != nil {
		return nil, err
	}
	nspath, _ := c.session.GetPath(sessionNetNs)
	if err := system.Points.AddBind(mount.SharedTag, procNetNs, nspath, 0); err != nil {
		return nil, fmt.Errorf("could not hold network namespace reference: %s", err)
	}
	networks := strings.Split(c.engine.EngineConfig.GetNetwork(), ",")

	if fakeroot && euid != 0 && net != fakerootNet {
		// set as debug message to avoid annoying warning
		sylog.Debugf("only '%s' network is allowed for regular user, you requested '%s'", fakerootNet, net)
		networks = []string{fakerootNet}
	}

	cniPath := &network.CNIPath{}

	cniPath.Conf = c.engine.EngineConfig.File.CniConfPath
	if cniPath.Conf == "" {
		cniPath.Conf = defaultCNIConfPath
	}
	cniPath.Plugin = c.engine.EngineConfig.File.CniPluginPath
	if cniPath.Plugin == "" {
		cniPath.Plugin = defaultCNIPluginPath
	}

	setup, err := network.NewSetup(networks, strconv.Itoa(pid), nspath, cniPath)
	if err != nil {
		return nil, fmt.Errorf("network setup failed: %s", err)
	}
	netargs := c.engine.EngineConfig.GetNetworkArgs()
	if err := setup.SetArgs(netargs); err != nil {
		return nil, fmt.Errorf("error while setting network arguments: %s", err)
	}

	return func() error {
		if fakeroot {
			// prevent port hijacking between user processes
			if err := setup.SetPortProtection(fakerootNet, 0); err != nil {
				return err
			}
			if euid != 0 {
				priv.Escalate()
				defer priv.Drop()
			}
		}

		setup.SetEnvPath("/bin:/sbin:/usr/bin:/usr/sbin")

		if err := setup.AddNetworks(); err != nil {
			return fmt.Errorf("%s", err)
		}
		c.engine.EngineConfig.Network = setup
		return nil
	}, nil
}

// addFuseMount transforms the plugin configuration into a series of
// mount requests for FUSE filesystems
func (c *container) addFuseMount(system *mount.System) error {
	for i, name := range c.engine.EngineConfig.GetPluginFuseMounts() {
		var cfg struct {
			Fuse singularity.FuseInfo
		}

		if err := c.engine.EngineConfig.GetPluginConfig(name, &cfg); err != nil {
			sylog.Debugf("Failed getting plugin config: %+v\n", err)
			return err
		}

		// we cannot check this because the mount point might
		// not exist outside the container with the name that
		// it's going to have _inside_ the container, so we
		// simply assume that it is a directory.
		//
		// In a normal situation we would stat the mount point
		// to obtain the mode, and bitwise-and the result with
		// S_IFMT.
		rootmode := syscall.S_IFDIR & syscall.S_IFMT

		// we assume that the file descriptor we obtained by
		// opening /dev/fuse before is valid in the RPC server,
		// where the actual mount operation is going to be
		// executed.
		opts := fmt.Sprintf("fd=%d,rootmode=%o,user_id=%d,group_id=%d",
			cfg.Fuse.DevFuseFd,
			rootmode,
			os.Getuid(),
			os.Getgid())

		// mount file system in three steps: first create a
		// dedicated session directory for each FUSE filesystem
		// and use that as the mount point.
		fuseDir := fmt.Sprintf("/fuse/%d", i)
		if err := c.session.AddDir(fuseDir); err != nil {
			return err
		}
		fuseDir, _ = c.session.GetPath(fuseDir)
		sylog.Debugf("Add FUSE mount %s for %s with options %s", fuseDir, name, opts)
		if err := system.Points.AddFS(
			mount.BindsTag,
			fuseDir,
			"fuse",
			syscall.MS_NOSUID|syscall.MS_NODEV,
			opts); err != nil {
			sylog.Debugf("Calling AddFS: %+v\n", err)
			return err
		}

		// second, add a bind-mount the session directory into
		// the destination mount point inside the container.
		if err := system.Points.AddBind(mount.OtherTag, fuseDir, cfg.Fuse.MountPoint, 0); err != nil {
			return err
		}
	}

	return nil
}

func (c *container) getBindFlags(source string, defaultFlags uintptr) (uintptr, error) {
	addFlags := uintptr(0)

	// case where there is a single bind, we doesn't need
	// to apply mount flags from source directory/file
	if defaultFlags == syscall.MS_BIND || defaultFlags == syscall.MS_BIND|syscall.MS_REC {
		return defaultFlags, nil
	}

	entries, err := proc.GetMountInfoEntry(c.mountInfoPath)
	if err != nil {
		return 0, fmt.Errorf("error while reading %s: %s", c.mountInfoPath, err)
	}

	e, err := proc.FindParentMountEntry(source, entries)
	if err != nil {
		return 0, fmt.Errorf("while searching parent mount point entry for %s: %s", source, err)
	}
	addFlags, _ = mount.ConvertOptions(e.Options)

	if addFlags&syscall.MS_RDONLY != 0 && defaultFlags&syscall.MS_RDONLY == 0 {
		if !strings.HasPrefix(source, buildcfg.SESSIONDIR) {
			sylog.Verbosef("Could not mount %s as read-write: mounted read-only", source)
		}
	}

	return defaultFlags | addFlags, nil
}
