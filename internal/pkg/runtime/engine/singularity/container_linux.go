// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"context"
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
	"github.com/sylabs/singularity/internal/pkg/plugin"
	"github.com/sylabs/singularity/internal/pkg/runtime/engine/singularity/rpc/client"
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
	singularitycallback "github.com/sylabs/singularity/pkg/plugin/callback/runtime/engine/singularity"
	singularity "github.com/sylabs/singularity/pkg/runtime/engine/singularity/config"
	"github.com/sylabs/singularity/pkg/sylog"
	"github.com/sylabs/singularity/pkg/util/fs/proc"
	"github.com/sylabs/singularity/pkg/util/gpu"
	"github.com/sylabs/singularity/pkg/util/loop"
	"github.com/sylabs/singularity/pkg/util/namespaces"
	"github.com/sylabs/singularity/pkg/util/singularityconf"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/sys/unix"
)

// global variables used by master process only at various steps:
// - setup
// - cleanup
// - post start process
var cryptDev string
var networkSetup *network.Setup
var cgroupManager *cgroups.Manager
var imageDriver image.Driver
var umountPoints []string

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

func create(ctx context.Context, engine *EngineOperations, rpcOps *client.RPC, pid int) error {
	var err error

	if len(engine.EngineConfig.GetImageList()) == 0 {
		return fmt.Errorf("no root filesystem image provided")
	}

	configurationFile := buildcfg.SINGULARITY_CONF_FILE
	if buildcfg.SINGULARITY_SUID_INSTALL == 0 || os.Geteuid() == 0 {
		configFile := engine.EngineConfig.GetConfigurationFile()
		if configFile != "" {
			configurationFile = configFile
		}
	}

	engine.EngineConfig.File, err = singularityconf.Parse(configurationFile)
	if err != nil {
		return fmt.Errorf("unable to parse singularity.conf file: %s", err)
	}

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
		c.sessionSize = int(c.engine.EngineConfig.File.SessiondirMaxSize)
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

	// load image driver plugins
	callbackType := (singularitycallback.RegisterImageDriver)(nil)
	callbacks, err := plugin.LoadCallbacks(callbackType)
	if err != nil {
		return fmt.Errorf("while loading plugins callbacks '%T': %s", callbackType, err)
	}
	for _, callback := range callbacks {
		if err := callback.(singularitycallback.RegisterImageDriver)(c.userNS); err != nil {
			return fmt.Errorf("while registering image driver: %s", err)
		}
	}

	driverName := c.engine.EngineConfig.File.ImageDriver
	imageDriver = image.GetDriver(driverName)
	if driverName != "" && imageDriver == nil {
		return fmt.Errorf("%q: no such image driver", driverName)
	}

	p := &mount.Points{}
	system := &mount.System{Points: p, Mount: c.mount}

	if err := c.setupSessionLayout(system); err != nil {
		return err
	}

	if err := c.setupImageDriver(system); err != nil {
		return err
	}

	umountPoints = append(umountPoints, c.session.RootFsPath())

	if c.session.FinalPath() != c.session.RootFsPath() {
		umountPoints = append(umountPoints, c.session.FinalPath())
	}

	if err := system.RunAfterTag(mount.SessionTag, c.addMountInfo); err != nil {
		return err
	}
	if err := system.RunBeforeTag(mount.CwdTag, c.addCwdMount); err != nil {
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

	if err := c.addRootfsMount(system); err != nil {
		return err
	}
	if err := c.addImageBindMount(system); err != nil {
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
	usernsFd, err := c.addFuseMount(system)
	if err != nil {
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
		if err := networkSetup(ctx); err != nil {
			return err
		}
	}

	if os.Geteuid() == 0 && !c.userNS {
		path := engine.EngineConfig.GetCgroupsPath()
		if path != "" {
			cgroupPath := filepath.Join("/singularity", strconv.Itoa(pid))
			cgroupManager = &cgroups.Manager{Pid: pid, Path: cgroupPath}
			if err := cgroupManager.ApplyFromFile(path); err != nil {
				return fmt.Errorf("failed to apply cgroups resources restriction: %s", err)
			}
		}
	}

	sylog.Debugf("Chdir into / to avoid errors\n")
	err = syscall.Chdir("/")
	if err != nil {
		return fmt.Errorf("change directory failed: %s", err)
	}

	if err := engine.runFuseDrivers(false, usernsFd); err != nil {
		return fmt.Errorf("while running FUSE drivers: %s", err)
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

// setupImageDriver prepare the image driver configured in singularity.conf
// to start it after the session setup.
func (c *container) setupImageDriver(system *mount.System) error {
	if imageDriver == nil {
		return nil
	}

	const sessionPath = "/driver"

	fuseDriver := imageDriver.Features()&image.FuseFeature != 0

	if err := c.session.AddDir(sessionPath); err != nil {
		return fmt.Errorf("while creating session driver directory: %s", err)
	}
	sp, _ := c.session.GetPath(sessionPath)

	params := &image.DriverParams{
		SessionPath: sp,
		UsernsFd:    -1,
		FuseFd:      -1,
		Config:      c.engine.CommonConfig,
	}

	if c.userNS {
		fds, err := c.getFuseFdFromRPC(nil)
		if err != nil {
			return fmt.Errorf("while getting /proc/self/ns/user file descriptor: %s", err)
		}
		params.UsernsFd = fds[0]
	}

	if fuseDriver {
		fuseFd, fuseRPCFd, err := c.openFuseFdFromRPC()
		if err != nil {
			return fmt.Errorf("while requesting /dev/fuse file descriptor from RPC: %s", err)
		}
		params.FuseFd = fuseFd
		system.RunAfterTag(mount.SessionTag, func(system *mount.System) error {
			defer unix.Close(params.FuseFd)

			fakeroot := c.engine.EngineConfig.GetFakeroot()
			fakerootHybrid := fakeroot && os.Geteuid() != 0

			uid := os.Getuid()
			gid := os.Getgid()
			rootmode := syscall.S_IFDIR & syscall.S_IFMT

			// as fakeroot can change UID/GID, we allow others users
			// to access FUSE mount point
			allowOther := ""
			if fakeroot {
				allowOther = ",allow_other"
			}
			if fakerootHybrid {
				uid = 0
				gid = 0

				// with hybrid workflow this process is actually running as the
				// user but outside of the fakeroot user namespace, it means that
				// the FUSE kernel code prevent us from accessing the mount point
				// where images (rootfs, overlay ...) resides if we are not in the
				// user namespace, so we redirect session VFS calls via RPC in order
				// to be in the user namespace when dealing with filesystem related calls.
				c.session.VFS = c.rpcOps

				if params.UsernsFd != -1 {
					defer unix.Close(params.UsernsFd)
				}
			} else {
				if params.UsernsFd != -1 {
					unix.Close(params.UsernsFd)
					params.UsernsFd = -1
				}
			}

			opts := fmt.Sprintf("fd=%d,rootmode=%o,user_id=%d,group_id=%d%s",
				fuseRPCFd,
				rootmode,
				uid,
				gid,
				allowOther,
			)

			sylog.Debugf("Add FUSE mount for image driver with options %s", opts)
			err := c.rpcOps.Mount("fuse", sp, "fuse", syscall.MS_NOSUID|syscall.MS_NODEV, opts)
			if err != nil {
				return fmt.Errorf("while mounting fuse image driver: %s", err)
			}

			umountPoints = append(umountPoints, sp)

			sylog.Debugf("Starting image driver %s", c.engine.EngineConfig.File.ImageDriver)
			if err := imageDriver.Start(params); err != nil {
				return fmt.Errorf("failed to start driver: %s", err)
			}

			return nil
		})
		return nil
	}

	system.RunAfterTag(mount.SessionTag, func(system *mount.System) error {
		if params.UsernsFd != -1 {
			defer unix.Close(params.UsernsFd)
		}
		sylog.Debugf("Starting image driver %s", c.engine.EngineConfig.File.ImageDriver)
		if err := imageDriver.Start(params); err != nil {
			return fmt.Errorf("failed to start driver: %s", err)
		}
		return nil
	})

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

// addMountinfo handles the case where hidepid is set on /proc mount
// point preventing this process from accessing /proc/<rpc_pid>/mountinfo
// without error, so we bind mount /proc/self/mountinfo from RPC process
// to a session file and read mount information from there.
func (c *container) addMountInfo(system *mount.System) error {
	const (
		mountinfo = "/mountinfo"
		self      = "/proc/self/mountinfo"
	)

	c.mountInfoPath = filepath.Join(c.session.Path(), mountinfo)
	if err := fs.Touch(c.mountInfoPath); err != nil {
		return fmt.Errorf("while creating %s: %s", c.mountInfoPath, err)
	}

	if err := c.rpcOps.Mount(self, c.mountInfoPath, "", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("while mounting %s to %s: %s", c.mountInfoPath, self, err)
	}

	return nil
}

func (c *container) chdirFinal(system *mount.System) error {
	if _, err := c.rpcOps.Chdir(c.session.FinalPath()); err != nil {
		return err
	}
	return nil
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
			if imageDriver != nil && c.engine.EngineConfig.File.EnableOverlay == "driver" {
				if imageDriver.Features()&image.OverlayFeature != 0 {
					params := &image.MountParams{
						Source:     source,
						Target:     dest,
						Filesystem: mnt.Type,
						Flags:      flags,
						FSOptions:  opts,
					}
					return imageDriver.Mount(params, c.rpcOps.Mount)
				}
			}
		}
	}

mount:
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
		if !bindMount && !remount {
			if mnt.Type == "devpts" {
				sylog.Verbosef("Couldn't mount devpts filesystem, continuing with PTY allocation functionality disabled")
				return nil
			} else if mnt.Type == "overlay" && err == syscall.ESTALE {
				// overlay mount can return this error when a previous mount was
				// done with an upper layer and overlay inodes index is enabled
				// by default, see https://github.com/sylabs/singularity/issues/4539
				sylog.Verbosef("Overlay mount failed with %s, mounting with index=off", err)
				optsString = fmt.Sprintf("%s,index=off", optsString)
				goto mount
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

		if mount.SkipOnError(mnt.InternalOptions) {
			sylog.Warningf("could not mount %s: %s", mnt.Source, err)
			c.skippedMount = append(c.skippedMount, mnt.Destination)
			return nil
		}
		return fmt.Errorf("could not mount %s: %s", mnt.Source, err)
	}

	return nil
}

// mount image via loop
func (c *container) mountImage(mnt *mount.Point) error {
	var key []byte

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

	mountType := mnt.Type

	if mountType == "encryptfs" {
		key, err = mount.GetKey(mnt.InternalOptions)
		if err != nil {
			return err
		}
	}

	if imageDriver != nil && imageDriver.Features()&image.ImageFeature != 0 {
		params := &image.MountParams{
			Source:     mnt.Source,
			Target:     mnt.Destination,
			Filesystem: mountType,
			Flags:      flags,
			Offset:     offset,
			Size:       sizelimit,
			Key:        key,
			FSOptions:  opts,
		}
		return imageDriver.Mount(params, c.rpcOps.Mount)
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

	if mountType == "encryptfs" {
		// pass the master processus ID only if a container IPC
		// namespace was requested because cryptsetup requires
		// to run in the host IPC namespace
		masterPid := 0
		if c.ipcNS {
			masterPid = os.Getpid()
		}

		cryptDev, err = c.rpcOps.Decrypt(offset, path, key, masterPid)

		if err != nil {
			return fmt.Errorf("unable to decrypt the file system: %s", err)
		}

		path = cryptDev

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

func (c *container) addRootfsMount(system *mount.System) error {
	flags := uintptr(c.suidFlag | syscall.MS_NODEV)
	rootfs := c.engine.EngineConfig.GetImage()

	imageObject := c.engine.EngineConfig.GetImageList()[0]
	part, err := imageObject.GetRootFsPartition()
	if err != nil {
		return fmt.Errorf("while getting root filesystem: %s", err)
	}

	if !imageObject.Writable {
		sylog.Debugf("Mount rootfs in read-only mode")
		flags |= syscall.MS_RDONLY
	} else {
		sylog.Debugf("Mount rootfs in read-write mode")
	}

	mountType := ""
	var key []byte

	sylog.Debugf("Image type is %v", part.Type)

	switch part.Type {
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
		if err := system.Points.AddRemount(mount.RootfsTag, c.session.RootFsPath(), flags); err != nil {
			return err
		}
		// re-apply mount propagation flag, on EL6 a kernel bug reset propagation flag
		// and may lead to crash (see https://github.com/sylabs/singularity/issues/4851)
		flags = syscall.MS_SLAVE
		if !c.engine.EngineConfig.File.MountSlave {
			flags = syscall.MS_PRIVATE
		}
		return system.Points.AddPropagation(mount.RootfsTag, c.session.RootFsPath(), flags)
	}

	sylog.Debugf("Mounting block [%v] image: %v\n", mountType, rootfs)
	if err := system.Points.AddImage(
		mount.RootfsTag,
		imageObject.Source,
		c.session.RootFsPath(),
		mountType,
		flags,
		part.Offset,
		part.Size,
		key,
	); err != nil {
		return err
	}

	if imageObject.Writable {
		return system.Points.AddPropagation(mount.SharedTag, c.session.RootFsPath(), syscall.MS_UNBINDABLE)
	}

	return nil
}

func (c *container) overlayUpperWork(system *mount.System) error {
	ov := c.session.Layer.(*overlay.Overlay)

	createUpperWork := func(path, label string) error {
		fi, err := c.rpcOps.Lstat(path)
		if os.IsNotExist(err) {
			if err := c.rpcOps.Mkdir(path, 0755); err != nil {
				return fmt.Errorf("failed to create %s directory: %s", path, err)
			}
		} else if err == nil && !fi.IsDir() {
			return fmt.Errorf("%s overlay %s must be a directory", label, path)
		} else if err != nil {
			return fmt.Errorf("could not setup writable overlay: %s", err)
		}
		return nil
	}

	if err := createUpperWork(ov.GetUpperDir(), "upper"); err != nil {
		sylog.Errorf("Could not create overlay upper dir. If using an overlay image ensure it contains 'upper' and 'work' directories")
		return err
	}
	if err := createUpperWork(ov.GetWorkDir(), "workdir"); err != nil {
		sylog.Errorf("Could not create overlay work dir. If using an overlay image ensure it contains 'upper' and 'work' directories")
		return err
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

	for _, img := range c.engine.EngineConfig.GetImageList() {
		overlays, err := img.GetOverlayPartitions()
		if err != nil {
			return fmt.Errorf("while opening overlay image %s: %s", img.Path, err)
		}
		for _, overlay := range overlays {
			sylog.Debugf("Using overlay partition in image %s", img.Path)

			sessionDest := fmt.Sprintf("/overlay-images/%d", nb)
			if err := c.session.AddDir(sessionDest); err != nil {
				return fmt.Errorf("failed to create session directory for overlay: %s", err)
			}
			dst, _ := c.session.GetPath(sessionDest)
			nb++

			src := img.Source
			offset := overlay.Offset
			size := overlay.Size

			switch overlay.Type {
			case image.EXT3:
				flags := uintptr(c.suidFlag | syscall.MS_NODEV)

				if !img.Writable {
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
				allowed := os.Geteuid() == 0

				if c.engine.EngineConfig.File.EnableOverlay == "driver" {
					if imageDriver != nil && imageDriver.Features()&image.OverlayFeature != 0 {
						allowed = true
					}
				}

				if !allowed {
					return fmt.Errorf("only root user can use sandbox as overlay")
				}

				flags := uintptr(c.suidFlag | syscall.MS_NODEV)
				err = system.Points.AddBind(mount.PreLayerTag, img.Path, dst, flags)
				if err != nil {
					return fmt.Errorf("while adding sandbox image: %s", err)
				}
				system.Points.AddRemount(mount.PreLayerTag, dst, flags)

				if !img.Writable {
					// check if the sandbox directory is located on a compatible
					// filesystem usable overlay lower directory
					if err := fsoverlay.CheckLower(img.Path); err != nil {
						return err
					}
					if fs.IsDir(filepath.Join(img.Path, "upper")) {
						ov.AddLowerDir(filepath.Join(dst, "upper"))
					} else {
						ov.AddLowerDir(dst)
					}
				} else {
					// check if the sandbox directory is located on a compatible
					// filesystem usable with overlay upper directory
					if err := fsoverlay.CheckUpper(img.Path); err != nil {
						return err
					}
				}
			default:
				return fmt.Errorf("%s: overlay image with unknown format", img.Path)
			}

			err = system.Points.AddPropagation(mount.SharedTag, dst, syscall.MS_UNBINDABLE)
			if err != nil {
				return err
			}

			if img.Writable && !hasUpper {
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
	}

	if hasUpper {
		if err := system.RunAfterTag(mount.PreLayerTag, c.overlayUpperWork); err != nil {
			return err
		}
	}

	return system.Points.AddPropagation(mount.SharedTag, c.session.FinalPath(), syscall.MS_UNBINDABLE)
}

func (c *container) addImageBindMount(system *mount.System) error {
	nb := 0
	imageList := c.engine.EngineConfig.GetImageList()

	for _, bind := range c.engine.EngineConfig.GetBindPath() {
		if bind.ImageSrc() == "" && bind.ID() == "" {
			continue
		} else if !c.engine.EngineConfig.File.UserBindControl {
			sylog.Warningf("Ignoring image bind mount request: user bind control disabled by system administrator")
			return nil
		}

		imagePath := bind.Source
		destination := bind.Destination
		id := 0
		imageSource := "/"

		if src := bind.ImageSrc(); src != "" {
			imageSource = src
		}

		if idStr := bind.ID(); idStr != "" {
			var err error

			id, err = strconv.Atoi(idStr)
			if err != nil {
				return fmt.Errorf("while parsing id bind option: %s", err)
			} else if id <= 0 {
				return fmt.Errorf("id number must be greater than 0")
			}
		}

		for _, img := range imageList {
			// img.Source is formatted like /proc/self/fd/X and ensure
			// we get the right path to the image with the associated
			// file descriptor
			if imagePath != img.Source {
				continue
			}

			data := (*image.Section)(nil)

			// id is only meaningful for SIF images
			if img.Type == image.SIF && id > 0 {
				partitions, err := img.GetAllPartitions()
				if err != nil {
					return fmt.Errorf("while getting partitions for %s: %s", img.Path, err)
				}
				for _, part := range partitions {
					if part.ID == uint32(id) {
						data = &part
						break
					}
				}
			} else {
				// take the first data partition found
				partitions, err := img.GetDataPartitions()
				if err != nil {
					return fmt.Errorf("while getting data partition for %s: %s", img.Path, err)
				}
				for _, part := range partitions {
					data = &part
					break
				}
			}

			if data == nil {
				return fmt.Errorf("no data partition found in %s", img.Path)
			}

			sessionDest := fmt.Sprintf("/data-images/%d", nb)
			if err := c.session.AddDir(sessionDest); err != nil {
				return fmt.Errorf("failed to create session directory for overlay: %s", err)
			}
			imgDest, _ := c.session.GetPath(sessionDest)
			nb++

			flags := uintptr(syscall.MS_NOSUID | syscall.MS_NODEV)
			fstype := ""

			switch data.Type {
			case image.EXT3:
				if !img.Writable {
					flags |= syscall.MS_RDONLY
				}
				fstype = "ext3"
			case image.SQUASHFS:
				flags |= syscall.MS_RDONLY
				fstype = "squashfs"
			default:
				return fmt.Errorf("could not use %s for image binding: not supported image format", img.Path)
			}

			err := system.Points.AddImage(
				mount.PreLayerTag,
				img.Source,
				imgDest,
				fstype,
				flags,
				data.Offset,
				data.Size,
				nil,
			)
			if err != nil {
				return fmt.Errorf("while adding data %s partition from %s: %s", fstype, img.Path, err)
			}

			src := filepath.Join(imgDest, imageSource)

			system.RunAfterTag(mount.PreLayerTag, func(*mount.System) error {
				if err := unix.Access(src, unix.R_OK); os.IsNotExist(err) {
					return fmt.Errorf("%s doesn't exist in image %s", imageSource, img.Path)
				}
				return nil
			})

			if err := system.Points.AddBind(mount.UserbindsTag, src, destination, syscall.MS_BIND); err != nil {
				return fmt.Errorf("while adding data bind %s -> %s: %s", src, destination, err)
			}
		}
	}

	return nil
}

func (c *container) addKernelMount(system *mount.System) error {
	var err error
	bindFlags := uintptr(syscall.MS_BIND | syscall.MS_NOSUID | syscall.MS_NODEV | syscall.MS_REC)

	sylog.Debugf("Checking configuration file for 'mount proc'")
	if c.engine.EngineConfig.File.MountProc && !c.engine.EngineConfig.GetNoProc() {
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
	if c.engine.EngineConfig.File.MountSys && !c.engine.EngineConfig.GetNoSys() {
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

	if c.engine.EngineConfig.File.MountDev == "no" || c.engine.EngineConfig.GetNoDev() {
		sylog.Verbosef("Not mounting /dev inside the container, disallowed by configuration")
	} else if c.engine.EngineConfig.File.MountDev == "minimal" || c.engine.EngineConfig.GetContain() {
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
		err := system.Points.AddFS(mount.DevTag, devshmPath, c.sessionFsType, flags, "mode=1777")
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

		if c.engine.EngineConfig.File.MountDevPts && !c.engine.EngineConfig.GetNoDevPts() {
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
			devs, err := gpu.NvidiaDevices(true)
			if err != nil {
				return fmt.Errorf("failed to get nvidia devices: %v", err)
			}
			for _, dev := range devs {
				if err := c.addSessionDev(dev, system); err != nil {
					return err
				}
			}
		}

		if c.engine.EngineConfig.GetRocm() {
			devs, err := gpu.RocmDevices(true)
			if err != nil {
				return fmt.Errorf("failed to get rocm devices: %v", err)
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
	}
	return nil
}

func (c *container) addHostMount(system *mount.System) error {
	if !c.engine.EngineConfig.File.MountHostfs || c.engine.EngineConfig.GetNoHostfs() {
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

	const (
		hostsPath     = "/etc/hosts"
		localtimePath = "/etc/localtime"
	)

	if c.engine.EngineConfig.GetContain() {
		hosts := hostsPath

		// handle special case for /etc/hosts as it is required,
		// if no network namespace was requested we simply bind
		// /etc/hosts from host, if network namespace is requested
		// we create a minimal default hosts for localhost resolution
		if !c.netNS {
			sylog.Debugf("Binding /etc/hosts and /etc/localtime only with contain")
		} else {
			sylog.Debugf("Skipping bind mounts as contain was requested")

			sylog.Verbosef("Binding staging /etc/hosts as contain is set")
			if err := c.session.AddFile(hostsPath, files.DefaultHosts()); err != nil {
				return fmt.Errorf("while adding /etc/hosts staging file: %s", err)
			}
			hosts, _ = c.session.GetPath(hostsPath)
		}

		// #5465 If hosts/localtime mount fails, it should not be fatal so skip-on-error
		if err := system.Points.AddBind(mount.BindsTag, hosts, hostsPath, flags, "skip-on-error"); err != nil {
			return fmt.Errorf("unable to add %s to mount list: %s", hosts, err)
		}
		if err := system.Points.AddRemount(mount.BindsTag, hostsPath, flags); err != nil {
			return fmt.Errorf("unable to add %s for remount: %s", hostsPath, err)
		}
		if err := system.Points.AddBind(mount.BindsTag, localtimePath, localtimePath, flags, "skip-on-error"); err != nil {
			return fmt.Errorf("unable to add %s to mount list: %s", localtimePath, err)
		}
		if err := system.Points.AddRemount(mount.BindsTag, localtimePath, flags); err != nil {
			return fmt.Errorf("unable to add %s for remount: %s", localtimePath, err)
		}
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

		// #5465 If hosts/localtime mount fails, it should not be fatal so skip-on-error
		bindOpt := ""
		if src == localtimePath || src == hostsPath {
			bindOpt = "skip-on-error"
		}

		err := system.Points.AddBind(mount.BindsTag, src, dst, flags, bindOpt)
		if err != nil {
			return fmt.Errorf("unable to add %s to mount list: %s", src, err)
		}
		if err := system.Points.AddRemount(mount.BindsTag, dst, flags); err != nil {
			return fmt.Errorf("unable to add %s for remount: %s", dst, err)
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
		pw, err := user.CurrentOriginal()
		if err == nil {
			source = pw.Dir
			if c.engine.EngineConfig.GetFakeroot() {
				dest = "/root"
			} else {
				dest = pw.Dir
			}
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

	// issue #5228 - don't attempt to mount a '/' home dir like 'nobody' has
	if dest == "/" {
		sylog.Warningf("Skipping impossible home directory mount to '/'")
		return nil
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
	const devPrefix = "/dev"
	defaultFlags := uintptr(syscall.MS_BIND | c.suidFlag | syscall.MS_NODEV | syscall.MS_REC)

	for _, b := range c.engine.EngineConfig.GetBindPath() {
		// ignore image bind
		if b.ID() != "" || b.ImageSrc() != "" {
			continue
		}

		flags := defaultFlags
		source := b.Source
		dst := b.Destination

		src, err := filepath.Abs(source)
		if err != nil {
			sylog.Warningf("Can't determine absolute path of %s bind point", source)
			continue
		}
		if b.Readonly() {
			flags |= syscall.MS_RDONLY
		}

		// special case for /dev mount to override default mount behavior
		// with --contain option or 'mount dev = minimal'
		if strings.HasPrefix(dst, devPrefix) && strings.HasPrefix(src, devPrefix) {
			if dst != src {
				sylog.Warningf("Skipping %s bind mount: source and destination must be identical when binding to %s", src, devPrefix)
				continue
			}
			if c.engine.EngineConfig.File.MountDev == "no" || c.engine.EngineConfig.GetNoDev() {
				sylog.Warningf("Skipping %s bind mount: disallowed by configuration", src)
				continue
			} else if c.engine.EngineConfig.File.MountDev == "minimal" || c.engine.EngineConfig.GetContain() {
				// "--bind /dev" bind case
				if src == devPrefix {
					system.Points.RemoveByTag(mount.DevTag)
					c.devSourcePath = devPrefix
					sylog.Debugf("Adding %[1]s host bind mount, resetting container mount list for %[1]s\n", devPrefix)
					continue
				}
				_, err := c.session.GetPath(src)
				if err == nil {
					sylog.Warningf("Skipping %s bind mount: already mounted", src)
					continue
				}
				if err := c.addSessionDev(src, system); err != nil {
					sylog.Warningf("Skipping %s bind mount: %s", src, err)
				}
				sylog.Debugf("Adding device %s to mount list\n", src)
				continue
			}
			// proceed with normal binds below if 'mount dev = yes'
			// or '--contain' wasn't requested
		}
		if !c.engine.EngineConfig.File.UserBindControl {
			sylog.Warningf("Ignoring %s bind mount: user bind control disabled by system administrator", src)
			continue
		}

		sylog.Debugf("Adding %s to mount list\n", src)

		if err := system.Points.AddBind(mount.UserbindsTag, src, dst, flags); err == mount.ErrMountExists {
			sylog.Warningf("While bind mounting '%s:%s': %s", src, dst, err)
		} else if err != nil {
			return fmt.Errorf("unable to add %s to mount list: %s", src, err)
		} else {
			fi, err := os.Stat(src)
			if err == nil && fi.IsDir() {
				c.session.OverrideDir(dst, src)
			}
			system.Points.AddRemount(mount.UserbindsTag, dst, flags)
		}
	}

	return nil
}

func (c *container) addTmpMount(system *mount.System) error {
	const (
		tmpPath    = "/tmp"
		varTmpPath = "/var/tmp"
	)

	sylog.Debugf("Checking for 'mount tmp' in configuration file")
	if !c.engine.EngineConfig.File.MountTmp || c.engine.EngineConfig.GetNoTmp() {
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

func (c *container) isMounted(dest string) bool {
	sylog.Debugf("Checking if %s is already mounted", dest)

	if !filepath.IsAbs(dest) {
		sylog.Debugf("%s is not an absolute path", dest)
		return false
	}

	entries, err := proc.GetMountInfoEntry(c.mountInfoPath)
	if err != nil {
		sylog.Debugf("Could not get %s entries: %s", c.mountInfoPath, err)
		return false
	}

	for _, e := range entries {
		if e.Point == dest {
			return true
		}
	}

	return false
}

func (c *container) addCwdMount(system *mount.System) error {
	if c.engine.EngineConfig.GetContain() {
		sylog.Verbosef("Not mounting current directory: contain was requested")
		return nil
	}
	if !c.engine.EngineConfig.File.UserBindControl {
		sylog.Warningf("Not mounting current directory: user bind control is disabled by system administrator")
		return nil
	}
	if c.engine.EngineConfig.GetNoCwd() {
		sylog.Debugf("Skipping current directory mount by user request.")
		return nil
	}
	cwd := c.engine.EngineConfig.GetCwd()
	if cwd == "" {
		sylog.Warningf("No current working directory set: skipping mount")
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

	dest := fs.EvalRelative(cwd, c.session.FinalPath())
	dest = filepath.Join(c.session.FinalPath(), dest)

	fi, err := c.rpcOps.Stat(dest)
	if err != nil {
		if os.IsNotExist(err) {
			sylog.Verbosef("Not mounting CWD, %s doesn't exist within container", cwd)
		}
		sylog.Verbosef("Not mounting CWD, while getting %s information: %s", cwd, err)
		return nil
	}
	cst := fi.Sys().(*syscall.Stat_t)

	var hst syscall.Stat_t
	if err := syscall.Stat(cwd, &hst); err != nil {
		return err
	}
	// same ino/dev, the current working directory is available within the container
	if hst.Dev == cst.Dev && hst.Ino == cst.Ino {
		sylog.Verbosef("%s found within container", cwd)
		return nil
	} else if c.isMounted(dest) {
		sylog.Verbosef("Not mounting CWD (already mounted in container): %s", cwd)
		return nil
	}

	flags := uintptr(syscall.MS_BIND | c.suidFlag | syscall.MS_NODEV | syscall.MS_REC)
	if err := system.Points.AddBind(mount.CwdTag, cwd, cwd, flags); err != nil {
		return fmt.Errorf("could not bind cwd directory %s into container: %s", cwd, err)
	}
	return system.Points.AddRemount(mount.CwdTag, cwd, flags)
}

func (c *container) addLibsMount(system *mount.System) error {
	libraries := c.engine.EngineConfig.GetLibrariesPath()

	sylog.Debugf("Checking for 'user bind control' in configuration file")
	if !c.engine.EngineConfig.File.UserBindControl {
		msg := "Ignoring libraries bind request: user bind control disabled by system administrator"
		if len(libraries) > 0 {
			sylog.Warningf(msg)
		} else {
			sylog.Verbosef(msg)
		}
		return nil
	}

	flags := uintptr(syscall.MS_BIND | syscall.MS_NOSUID | syscall.MS_NODEV | syscall.MS_RDONLY | syscall.MS_REC)

	containerDir := "/.singularity.d/libs"
	sessionDir := "/libs"

	if err := c.session.AddDir(sessionDir); err != nil {
		return err
	}

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
	files := c.engine.EngineConfig.GetFilesPath()

	sylog.Debugf("Checking for 'user bind control' in configuration file")
	if !c.engine.EngineConfig.File.UserBindControl {
		msg := "Ignoring binaries bind request: user bind control disabled by system administrator"
		if len(files) > 0 {
			sylog.Warningf(msg)
		} else {
			sylog.Verbosef(msg)
		}
		return nil
	}

	flags := uintptr(syscall.MS_BIND | syscall.MS_NOSUID | syscall.MS_NODEV | syscall.MS_RDONLY | syscall.MS_REC)

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

func (c *container) prepareNetworkSetup(system *mount.System, pid int) (func(context.Context) error, error) {
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
	networkSetup = setup

	netargs := c.engine.EngineConfig.GetNetworkArgs()
	if err := networkSetup.SetArgs(netargs); err != nil {
		return nil, fmt.Errorf("error while setting network arguments: %s", err)
	}

	return func(ctx context.Context) error {
		if fakeroot {
			// prevent port hijacking between user processes
			if err := networkSetup.SetPortProtection(fakerootNet, 0); err != nil {
				return err
			}
			if euid != 0 {
				priv.Escalate()
				defer priv.Drop()
			}
		}

		networkSetup.SetEnvPath("/bin:/sbin:/usr/bin:/usr/sbin")

		if err := networkSetup.AddNetworks(ctx); err != nil {
			return fmt.Errorf("%s", err)
		}
		return nil
	}, nil
}

// getFuseFdFromRPC returns fuse file descriptors from RPC server based on
// the file descriptor list provided in argument, it also returns an
// additional file descriptor corresponding to /proc/self/ns/user.
// You can set an empty file descriptor list to just get /proc/self/ns/user
// file descriptor.
func (c *container) getFuseFdFromRPC(fds []int) ([]int, error) {
	socketPair := c.engine.EngineConfig.GetUnixSocketPair()

	if err := c.rpcOps.SendFuseFd(socketPair[1], fds); err != nil {
		return nil, fmt.Errorf("while requesting file descriptors send: %s", err)
	}

	bufSpace := (len(fds) + 1) * 4
	buf := make([]byte, unix.CmsgSpace(bufSpace))
	_, _, _, _, err := unix.Recvmsg(socketPair[0], nil, buf, 0)
	if err != nil {
		return nil, fmt.Errorf("while receiving file descriptors: %s", err)
	}

	msgs, err := unix.ParseSocketControlMessage(buf)
	if err != nil {
		return nil, fmt.Errorf("while parsing socket control message: %s", err)
	}

	newfds := make([]int, 0, 1)

	for _, msg := range msgs {
		pfds, err := unix.ParseUnixRights(&msg)
		if err != nil {
			return nil, fmt.Errorf("while getting file descriptor: %s", err)
		}
		newfds = append(newfds, pfds...)
	}

	if len(newfds) != len(fds)+1 {
		return nil, fmt.Errorf("got %d file descriptors instead of %d", len(newfds), len(fds)+1)
	}

	return newfds, nil
}

// openFuseFdFromRPC returns fuse file descriptor opened by RPC server,
// the first returned argument corresponds to the file descriptor to use
// by the caller while the second argument corresponds to the file
// descriptor used by the RPC server.
func (c *container) openFuseFdFromRPC() (int, int, error) {
	socketPair := c.engine.EngineConfig.GetUnixSocketPair()

	fuseRPCFd, err := c.rpcOps.OpenSendFuseFd(socketPair[1])
	if err != nil {
		return -1, -1, fmt.Errorf("while requesting a fuse file descriptor open/send: %s", err)
	}

	bufSpace := 4
	buf := make([]byte, unix.CmsgSpace(bufSpace))
	_, _, _, _, err = unix.Recvmsg(socketPair[0], nil, buf, 0)
	if err != nil {
		return -1, -1, fmt.Errorf("while receiving file descriptors: %s", err)
	}

	msgs, err := unix.ParseSocketControlMessage(buf)
	if err != nil {
		return -1, -1, fmt.Errorf("while parsing socket control message: %s", err)
	}

	fuseFd := -1

	for _, msg := range msgs {
		fds, err := unix.ParseUnixRights(&msg)
		if err != nil {
			return -1, -1, fmt.Errorf("while getting file descriptor: %s", err)
		}
		fuseFd = fds[0]
		break
	}

	return fuseFd, fuseRPCFd, nil
}

// addFuseMount transforms the plugin configuration into a series of
// mount requests for FUSE filesystems
func (c *container) addFuseMount(system *mount.System) (int, error) {
	fakeroot := c.engine.EngineConfig.GetFakeroot()
	fakerootHybrid := fakeroot && os.Geteuid() != 0

	uid := os.Getuid()
	gid := os.Getgid()

	// as fakeroot can change UID/GID, we allow others users
	// to access FUSE mount point
	allowOther := ""
	if fakeroot {
		allowOther = ",allow_other"
	}
	if fakerootHybrid {
		uid = 0
		gid = 0
	}

	fds := make([]int, 0)
	usernsFd := -1

	fuseMounts := c.engine.EngineConfig.GetFuseMount()

	for _, fuseMount := range fuseMounts {
		if fuseMount.FromContainer || !c.userNS {
			continue
		}
		fds = append(fds, fuseMount.Fd)
	}

	if len(fds) > 0 {
		newfds, err := c.getFuseFdFromRPC(fds)
		if err != nil {
			return usernsFd, err
		}

		// the additional file descriptor is for /proc/self/ns/user which is
		// always passed by RPC server, this file descriptor is returned by
		// this function
		usernsFd = newfds[len(newfds)-1]
		newfds = newfds[0 : len(newfds)-1]

		for i, fd := range newfds {
			if err := unix.Dup3(fd, fds[i], unix.O_CLOEXEC); err != nil {
				return usernsFd, fmt.Errorf("could not duplicate file descriptor: %s", err)
			}
			unix.Close(fd)
		}
	}

	for i := range fuseMounts {
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
		opts := fmt.Sprintf("fd=%d,rootmode=%o,user_id=%d,group_id=%d%s",
			fuseMounts[i].Fd,
			rootmode,
			uid,
			gid,
			allowOther,
		)

		// mount file system in three steps: first create a
		// dedicated session directory for each FUSE filesystem
		// and use that as the mount point.
		fuseDir := fmt.Sprintf("/fuse/%d", i)
		if err := c.session.AddDir(fuseDir); err != nil {
			return usernsFd, err
		}
		fuseDir, _ = c.session.GetPath(fuseDir)

		sylog.Debugf("Add FUSE mount for %s with options %s", fuseMounts[i].MountPoint, opts)
		err := system.Points.AddFS(
			mount.BindsTag,
			fuseDir,
			"fuse",
			syscall.MS_NOSUID|syscall.MS_NODEV,
			opts,
		)
		if err != nil {
			sylog.Debugf("Calling AddFS: %+v\n", err)
			return usernsFd, err
		}

		// with fakeroot and hybrid workflow we are not running
		// in the container user namespace so we need to enter
		// into it and executing FUSE program from there, otherwise
		// the program could not communicate correctly through
		// /dev/fuse file descriptor
		if !fuseMounts[i].FromContainer && fakerootHybrid {
			// all FUSE programs when executed have /dev/fuse file
			// descriptor as /dev/fd/3 and /proc/<container_pid>/ns/user
			// if provided as /dev/fd/4.
			// /dev/fd/4 is used by nsenter to join container user namespace
			// and execute FUSE program inside the container user namespace
			nsenter := []string{"nsenter", "--user=/dev/fd/4", "-F", "--preserve-credentials"}
			fuseMounts[i].Program = append(nsenter, fuseMounts[i].Program...)
		}

		// second, add a bind-mount the session directory into
		// the destination mount point inside the container.
		if err := system.Points.AddBind(mount.OtherTag, fuseDir, fuseMounts[i].MountPoint, 0); err != nil {
			return usernsFd, err
		}
	}

	return usernsFd, nil
}

func (c *container) getBindFlags(source string, defaultFlags uintptr) (uintptr, error) {
	addFlags := uintptr(0)

	// case where there is a single bind, we doesn't need
	// to apply mount flags from source directory/file
	if defaultFlags == syscall.MS_BIND || defaultFlags == syscall.MS_BIND|syscall.MS_REC {
		return defaultFlags, nil
	}

	var stfs unix.Statfs_t

	// use statfs to retrieve mount options or fallback to /proc/self/mountinfo
	// in case of failure
	if err := unix.Statfs(source, &stfs); err != nil {
		entries, err := proc.GetMountInfoEntry(c.mountInfoPath)
		if err != nil {
			return 0, fmt.Errorf("error while reading %s: %s", c.mountInfoPath, err)
		}

		e, err := proc.FindParentMountEntry(source, entries)
		if err != nil {
			return 0, fmt.Errorf("while searching parent mount point entry for %s: %s", source, err)
		}
		addFlags, _ = mount.ConvertOptions(e.Options)
	} else {
		// clear bits MS_REMOUNT (32) and MS_BIND/ST_RELATIME (4096)
		addFlags = uintptr(stfs.Flags &^ 4128)
	}

	if addFlags&syscall.MS_RDONLY != 0 && defaultFlags&syscall.MS_RDONLY == 0 {
		if !strings.HasPrefix(source, buildcfg.SESSIONDIR) {
			sylog.Verbosef("Could not mount %s as read-write: mounted read-only", source)
		}
	}

	return defaultFlags | addFlags, nil
}
