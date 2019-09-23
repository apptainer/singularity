// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	fakerootutil "github.com/sylabs/singularity/internal/pkg/fakeroot"
	"github.com/sylabs/singularity/internal/pkg/instance"
	"github.com/sylabs/singularity/internal/pkg/runtime/engine/config"
	"github.com/sylabs/singularity/internal/pkg/runtime/engine/config/starter"
	"github.com/sylabs/singularity/internal/pkg/security"
	"github.com/sylabs/singularity/internal/pkg/security/seccomp"
	"github.com/sylabs/singularity/internal/pkg/syecl"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/internal/pkg/util/fs/overlay"
	"github.com/sylabs/singularity/internal/pkg/util/mainthread"
	"github.com/sylabs/singularity/internal/pkg/util/user"
	"github.com/sylabs/singularity/pkg/image"
	singularityConfig "github.com/sylabs/singularity/pkg/runtime/engines/singularity/config"
	"github.com/sylabs/singularity/pkg/util/capabilities"
	"github.com/sylabs/singularity/pkg/util/fs/proc"
	"golang.org/x/sys/unix"
)

var nsProcName = map[specs.LinuxNamespaceType]string{
	specs.PIDNamespace:     "pid",
	specs.UTSNamespace:     "uts",
	specs.IPCNamespace:     "ipc",
	specs.MountNamespace:   "mnt",
	specs.CgroupNamespace:  "cgroup",
	specs.NetworkNamespace: "net",
	specs.UserNamespace:    "user",
}

// PrepareConfig is called during stage1 to validate and prepare
// container configuration. It is responsible for singularity
// configuration file parsing, handling user input, reading capabilities,
// and checking what namespaces are required.
//
// No additional privileges can be gained as any of them are already
// dropped by the time PrepareConfig is called.
func (e *EngineOperations) PrepareConfig(starterConfig *starter.Config) error {
	if e.CommonConfig.EngineName != singularityConfig.Name {
		return fmt.Errorf("incorrect engine")
	}

	if e.EngineConfig.OciConfig.Generator.Config != &e.EngineConfig.OciConfig.Spec {
		return fmt.Errorf("bad engine configuration provided")
	}

	configurationFile := buildcfg.SINGULARITY_CONF_FILE
	if err := config.Parser(configurationFile, e.EngineConfig.File); err != nil {
		return fmt.Errorf("unable to parse singularity.conf file: %s", err)
	}

	if !e.EngineConfig.File.AllowSetuid && starterConfig.GetIsSUID() {
		return fmt.Errorf("suid workflow disabled by administrator")
	}

	if starterConfig.GetIsSUID() {
		// check for ownership of singularity.conf
		if !fs.IsOwner(configurationFile, 0) {
			return fmt.Errorf("%s must be owned by root", configurationFile)
		}
		// check for ownership of capability.json
		if !fs.IsOwner(buildcfg.CAPABILITY_FILE, 0) {
			return fmt.Errorf("%s must be owned by root", buildcfg.CAPABILITY_FILE)
		}
		// check for ownership of ecl.toml
		if !fs.IsOwner(buildcfg.ECL_FILE, 0) {
			return fmt.Errorf("%s must be owned by root", buildcfg.ECL_FILE)
		}
	}

	// Save the current working directory if not set
	if e.EngineConfig.GetCwd() == "" {
		if pwd, err := os.Getwd(); err == nil {
			e.EngineConfig.SetCwd(pwd)
		} else {
			sylog.Warningf("can't determine current working directory")
			e.EngineConfig.SetCwd("/")
		}
	}

	if e.EngineConfig.OciConfig.Process == nil {
		e.EngineConfig.OciConfig.Process = &specs.Process{}
	}
	if e.EngineConfig.OciConfig.Process.Capabilities == nil {
		e.EngineConfig.OciConfig.Process.Capabilities = &specs.LinuxCapabilities{}
	}
	if len(e.EngineConfig.OciConfig.Process.Args) == 0 {
		return fmt.Errorf("container process arguments not found")
	}

	uid := e.EngineConfig.GetTargetUID()
	gids := e.EngineConfig.GetTargetGID()

	if os.Getuid() == 0 && (uid != 0 || len(gids) > 0) {
		starterConfig.SetTargetUID(uid)
		starterConfig.SetTargetGID(gids)
		e.EngineConfig.OciConfig.SetProcessNoNewPrivileges(true)
	}

	if e.EngineConfig.GetInstanceJoin() {
		if err := e.prepareInstanceJoinConfig(starterConfig); err != nil {
			return err
		}
	} else {
		if err := e.prepareContainerConfig(starterConfig); err != nil {
			return err
		}
		if err := e.loadImages(starterConfig); err != nil {
			return err
		}
	}

	starterConfig.SetMasterPropagateMount(true)
	starterConfig.SetNoNewPrivs(e.EngineConfig.OciConfig.Process.NoNewPrivileges)

	if e.EngineConfig.OciConfig.Process != nil && e.EngineConfig.OciConfig.Process.Capabilities != nil {
		starterConfig.SetCapabilities(capabilities.Permitted, e.EngineConfig.OciConfig.Process.Capabilities.Permitted)
		starterConfig.SetCapabilities(capabilities.Effective, e.EngineConfig.OciConfig.Process.Capabilities.Effective)
		starterConfig.SetCapabilities(capabilities.Inheritable, e.EngineConfig.OciConfig.Process.Capabilities.Inheritable)
		starterConfig.SetCapabilities(capabilities.Bounding, e.EngineConfig.OciConfig.Process.Capabilities.Bounding)
		starterConfig.SetCapabilities(capabilities.Ambient, e.EngineConfig.OciConfig.Process.Capabilities.Ambient)
	}

	// determine if engine need to propagate signals across processes
	e.checkSignalPropagation()

	// We must call this here because at this point we haven't
	// spawned the master process nor the RPC server. The assumption
	// is that this function runs in stage 1 and that even if it's a
	// separate process, it's created in such a way that it's
	// sharing its file descriptor table with the wrapper / stage 2.
	//
	// At this point we do not have elevated privileges. We assume
	// that the user running singularity has access to /dev/fuse
	// (typically it's 0666, or 0660 belonging to a group that
	// allows the user to read and write to it).
	if err := openDevFuse(e, starterConfig); err != nil {
		return err
	}

	return nil
}

// prepareUserCaps is responsible for checking that user's requested
// capabilities are authorized.
func (e *EngineOperations) prepareUserCaps(enforced bool) error {
	commonCaps := make([]string, 0)
	commonUnauthorizedCaps := make([]string, 0)

	e.EngineConfig.OciConfig.SetProcessNoNewPrivileges(true)

	file, err := os.OpenFile(buildcfg.CAPABILITY_FILE, os.O_RDONLY, 0644)
	if err != nil {
		return fmt.Errorf("while opening capability config file: %s", err)
	}
	defer file.Close()

	capConfig, err := capabilities.ReadFrom(file)
	if err != nil {
		return fmt.Errorf("while parsing capability config data: %s", err)
	}

	pw, err := user.Current()
	if err != nil {
		return err
	}

	caps, ignoredCaps := capabilities.Split(e.EngineConfig.GetAddCaps())
	if len(ignoredCaps) > 0 {
		sylog.Warningf("won't add unknown capability: %s", strings.Join(ignoredCaps, ","))
	}
	caps = append(caps, e.EngineConfig.OciConfig.Process.Capabilities.Permitted...)

	if enforced {
		authorizedCaps, unauthorizedCaps := capConfig.CheckUserCaps(pw.Name, caps)
		if len(authorizedCaps) > 0 {
			sylog.Debugf("User capabilities %s added", strings.Join(authorizedCaps, ","))
			commonCaps = authorizedCaps
		}
		if len(unauthorizedCaps) > 0 {
			commonUnauthorizedCaps = append(commonUnauthorizedCaps, unauthorizedCaps...)
		}

		groups, err := os.Getgroups()
		if err != nil {
			return err
		}

		for _, g := range groups {
			gr, err := user.GetGrGID(uint32(g))
			if err != nil {
				sylog.Debugf("Ignoring group %d: %s", g, err)
				continue
			}
			authorizedCaps, unauthorizedCaps := capConfig.CheckGroupCaps(gr.Name, caps)
			if len(authorizedCaps) > 0 {
				sylog.Debugf("%s group capabilities %s added", gr.Name, strings.Join(authorizedCaps, ","))
				commonCaps = append(commonCaps, authorizedCaps...)
			}
			if len(unauthorizedCaps) > 0 {
				commonUnauthorizedCaps = append(commonUnauthorizedCaps, unauthorizedCaps...)
			}
		}
	} else {
		commonCaps = caps
	}

	commonCaps = capabilities.RemoveDuplicated(commonCaps)
	commonUnauthorizedCaps = capabilities.RemoveDuplicated(commonUnauthorizedCaps)

	// remove authorized capabilities from unauthorized capabilities list
	// to end with the really unauthorized capabilities
	for _, c := range commonCaps {
		for i := len(commonUnauthorizedCaps) - 1; i >= 0; i-- {
			if commonUnauthorizedCaps[i] == c {
				commonUnauthorizedCaps = append(commonUnauthorizedCaps[:i], commonUnauthorizedCaps[i+1:]...)
				break
			}
		}
	}
	if len(commonUnauthorizedCaps) > 0 {
		sylog.Warningf("not authorized to add capability: %s", strings.Join(commonUnauthorizedCaps, ","))
	}

	caps, ignoredCaps = capabilities.Split(e.EngineConfig.GetDropCaps())
	if len(ignoredCaps) > 0 {
		sylog.Warningf("won't drop unknown capability: %s", strings.Join(ignoredCaps, ","))
	}
	for _, cap := range caps {
		for i, c := range commonCaps {
			if c == cap {
				sylog.Debugf("Capability %s dropped", cap)
				commonCaps = append(commonCaps[:i], commonCaps[i+1:]...)
				break
			}
		}
	}

	e.EngineConfig.OciConfig.Process.Capabilities.Permitted = commonCaps
	e.EngineConfig.OciConfig.Process.Capabilities.Effective = commonCaps
	e.EngineConfig.OciConfig.Process.Capabilities.Inheritable = commonCaps
	e.EngineConfig.OciConfig.Process.Capabilities.Bounding = commonCaps
	e.EngineConfig.OciConfig.Process.Capabilities.Ambient = commonCaps

	return nil
}

// prepareRootCaps is responsible for setting root capabilities
// based on capability/configuration files and requested capabilities.
func (e *EngineOperations) prepareRootCaps() error {
	commonCaps := make([]string, 0)
	defaultCapabilities := e.EngineConfig.File.RootDefaultCapabilities

	uid := e.EngineConfig.GetTargetUID()
	gids := e.EngineConfig.GetTargetGID()

	if uid != 0 || len(gids) > 0 {
		defaultCapabilities = "no"
	}

	// is no-privs/keep-privs set on command line
	if e.EngineConfig.GetNoPrivs() {
		sylog.Debugf("--no-privs requested, no new privileges enabled")
		defaultCapabilities = "no"
	} else if e.EngineConfig.GetKeepPrivs() {
		sylog.Debugf("--keep-privs requested")
		defaultCapabilities = "full"
	}

	sylog.Debugf("Root %s capabilities", defaultCapabilities)

	// set default capabilities based on configuration file directive
	switch defaultCapabilities {
	case "full":
		e.EngineConfig.OciConfig.SetupPrivileged(true)
		commonCaps = e.EngineConfig.OciConfig.Process.Capabilities.Permitted
	case "file":
		file, err := os.OpenFile(buildcfg.CAPABILITY_FILE, os.O_RDONLY, 0644)
		if err != nil {
			return fmt.Errorf("while opening capability config file: %s", err)
		}
		defer file.Close()

		capConfig, err := capabilities.ReadFrom(file)
		if err != nil {
			return fmt.Errorf("while parsing capability config data: %s", err)
		}

		commonCaps = append(commonCaps, capConfig.ListUserCaps("root")...)

		groups, err := os.Getgroups()
		if err != nil {
			return fmt.Errorf("while getting groups: %s", err)
		}

		for _, g := range groups {
			gr, err := user.GetGrGID(uint32(g))
			if err != nil {
				sylog.Debugf("Ignoring group %d: %s", g, err)
				continue
			}
			caps := capConfig.ListGroupCaps(gr.Name)
			commonCaps = append(commonCaps, caps...)
			sylog.Debugf("%s group capabilities %s added", gr.Name, strings.Join(caps, ","))
		}
	default:
		e.EngineConfig.OciConfig.SetProcessNoNewPrivileges(true)
	}

	caps, ignoredCaps := capabilities.Split(e.EngineConfig.GetAddCaps())
	if len(ignoredCaps) > 0 {
		sylog.Warningf("won't add unknown capability: %s", strings.Join(ignoredCaps, ","))
	}
	for _, cap := range caps {
		found := false
		for _, c := range commonCaps {
			if c == cap {
				found = true
				break
			}
		}
		if !found {
			sylog.Debugf("Root capability %s added", cap)
			commonCaps = append(commonCaps, cap)
		}
	}

	commonCaps = capabilities.RemoveDuplicated(commonCaps)

	caps, ignoredCaps = capabilities.Split(e.EngineConfig.GetDropCaps())
	if len(ignoredCaps) > 0 {
		sylog.Warningf("won't add unknown capability: %s", strings.Join(ignoredCaps, ","))
	}
	for _, cap := range caps {
		for i, c := range commonCaps {
			if c == cap {
				sylog.Debugf("Root capability %s dropped", cap)
				commonCaps = append(commonCaps[:i], commonCaps[i+1:]...)
				break
			}
		}
	}

	e.EngineConfig.OciConfig.Process.Capabilities.Permitted = commonCaps
	e.EngineConfig.OciConfig.Process.Capabilities.Effective = commonCaps
	e.EngineConfig.OciConfig.Process.Capabilities.Inheritable = commonCaps
	e.EngineConfig.OciConfig.Process.Capabilities.Bounding = commonCaps
	e.EngineConfig.OciConfig.Process.Capabilities.Ambient = commonCaps

	return nil
}

func keepAutofsMount(source string, autoFsPoints []string) (int, error) {
	resolved, err := filepath.EvalSymlinks(source)
	if err != nil {
		return -1, err
	}

	for _, p := range autoFsPoints {
		if strings.HasPrefix(resolved, p) {
			sylog.Debugf("Open file descriptor for %s", resolved)
			f, err := os.Open(resolved)
			if err != nil {
				return -1, err
			}
			return int(f.Fd()), nil
		}
	}

	return -1, fmt.Errorf("no mount point")
}

func (e *EngineOperations) prepareAutofs(starterConfig *starter.Config) error {
	const mountInfoPath = "/proc/self/mountinfo"

	entries, err := proc.GetMountInfoEntry(mountInfoPath)
	if err != nil {
		return fmt.Errorf("while parsing %s: %s", mountInfoPath, err)
	}
	autoFsPoints := make([]string, 0)
	for _, e := range entries {
		if e.FSType == "autofs" {
			sylog.Debugf("Found %q as autofs mount point", e.Point)
			autoFsPoints = append(autoFsPoints, e.Point)
		}
	}
	if len(autoFsPoints) == 0 {
		sylog.Debugf("No autofs mount point found")
		return nil
	}

	fds := make([]int, 0)

	if e.EngineConfig.File.UserBindControl {
		for _, b := range e.EngineConfig.GetBindPath() {
			splitted := strings.Split(b, ":")

			fd, err := keepAutofsMount(splitted[0], autoFsPoints)
			if err != nil {
				sylog.Debugf("Could not keep file descriptor for user bind path %s: %s", splitted[0], err)
				continue
			}
			fds = append(fds, fd)
		}
	}

	if !e.EngineConfig.GetContain() {
		for _, bindpath := range e.EngineConfig.File.BindPath {
			splitted := strings.Split(bindpath, ":")

			fd, err := keepAutofsMount(splitted[0], autoFsPoints)
			if err != nil {
				sylog.Debugf("Could not keep file descriptor for bind path %s: %s", splitted[0], err)
				continue
			}
			fds = append(fds, fd)
		}

		// check home source directory
		dir := e.EngineConfig.GetHomeSource()
		fd, err := keepAutofsMount(dir, autoFsPoints)
		if err != nil {
			sylog.Debugf("Could not keep file descriptor for home directory %s: %s", dir, err)
		} else {
			fds = append(fds, fd)
		}

		// check the current working directory
		dir = e.EngineConfig.GetCwd()
		fd, err = keepAutofsMount(dir, autoFsPoints)
		if err != nil {
			sylog.Debugf("Could not keep file descriptor for current working directory %s: %s", dir, err)
		} else {
			fds = append(fds, fd)
		}
	} else {
		// check workdir
		dir := e.EngineConfig.GetWorkdir()
		fd, err := keepAutofsMount(dir, autoFsPoints)
		if err != nil {
			sylog.Debugf("Could not keep file descriptor for workdir %s: %s", dir, err)
		} else {
			fds = append(fds, fd)
		}
	}

	for _, f := range fds {
		if err := starterConfig.KeepFileDescriptor(f); err != nil {
			return err
		}
	}

	e.EngineConfig.SetOpenFd(fds)

	return nil
}

// prepareContainerConfig is responsible for getting and applying
// user supplied configuration for container creation.
func (e *EngineOperations) prepareContainerConfig(starterConfig *starter.Config) error {
	// always set mount namespace
	e.EngineConfig.OciConfig.AddOrReplaceLinuxNamespace(specs.MountNamespace, "")

	// if PID namespace is not allowed remove it from namespaces
	if !e.EngineConfig.File.AllowPidNs && e.EngineConfig.OciConfig.Linux != nil {
		namespaces := e.EngineConfig.OciConfig.Linux.Namespaces
		for i, ns := range namespaces {
			if ns.Type == specs.PIDNamespace {
				sylog.Debugf("Not virtualizing PID namespace by configuration")
				e.EngineConfig.OciConfig.Linux.Namespaces = append(namespaces[:i], namespaces[i+1:]...)
				break
			}
		}
	}

	if os.Getuid() == 0 {
		if err := e.prepareRootCaps(); err != nil {
			return err
		}
	} else {
		enforced := starterConfig.GetIsSUID()
		if err := e.prepareUserCaps(enforced); err != nil {
			return err
		}
	}

	if e.EngineConfig.File.MountSlave {
		starterConfig.SetMountPropagation("rslave")
	} else {
		starterConfig.SetMountPropagation("rprivate")
	}

	if e.EngineConfig.GetFakeroot() {
		if !starterConfig.GetIsSUID() {
			// no SUID workflow, check if newuidmap/newgidmap are present
			sylog.Verbosef("Fakeroot requested with unprivileged workflow, fallback to newuidmap/newgidmap")
			sylog.Debugf("Search for newuidmap binary")
			if err := starterConfig.SetNewUIDMapPath(); err != nil {
				return err
			}
			sylog.Debugf("Search for newgidmap binary")
			if err := starterConfig.SetNewGIDMapPath(); err != nil {
				return err
			}
		}

		uid := uint32(os.Getuid())
		gid := uint32(os.Getgid())

		e.EngineConfig.OciConfig.AddLinuxUIDMapping(uid, 0, 1)
		idRange, err := fakerootutil.GetIDRange(fakerootutil.SubUIDFile, uid)
		if err != nil {
			return fmt.Errorf("could not use fakeroot: %s", err)
		}
		e.EngineConfig.OciConfig.AddLinuxUIDMapping(idRange.HostID, idRange.ContainerID, idRange.Size)
		starterConfig.AddUIDMappings(e.EngineConfig.OciConfig.Linux.UIDMappings)

		e.EngineConfig.OciConfig.AddLinuxGIDMapping(gid, 0, 1)
		idRange, err = fakerootutil.GetIDRange(fakerootutil.SubGIDFile, uid)
		if err != nil {
			return fmt.Errorf("could not use fakeroot: %s", err)
		}
		e.EngineConfig.OciConfig.AddLinuxGIDMapping(idRange.HostID, idRange.ContainerID, idRange.Size)
		starterConfig.AddGIDMappings(e.EngineConfig.OciConfig.Linux.GIDMappings)

		e.EngineConfig.OciConfig.SetupPrivileged(true)

		e.EngineConfig.OciConfig.AddOrReplaceLinuxNamespace(specs.UserNamespace, "")

		starterConfig.SetHybridWorkflow(true)
		starterConfig.SetAllowSetgroups(true)

		starterConfig.SetTargetUID(0)
		starterConfig.SetTargetGID([]int{0})
	}

	starterConfig.SetBringLoopbackInterface(true)

	starterConfig.SetInstance(e.EngineConfig.GetInstance())

	starterConfig.SetNsFlagsFromSpec(e.EngineConfig.OciConfig.Linux.Namespaces)

	// user namespace ID mappings
	if e.EngineConfig.OciConfig.Linux != nil {
		if err := starterConfig.AddUIDMappings(e.EngineConfig.OciConfig.Linux.UIDMappings); err != nil {
			return err
		}
		if err := starterConfig.AddGIDMappings(e.EngineConfig.OciConfig.Linux.GIDMappings); err != nil {
			return err
		}
	}

	param := security.GetParam(e.EngineConfig.GetSecurity(), "selinux")
	if param != "" {
		sylog.Debugf("Applying SELinux context %s", param)
		e.EngineConfig.OciConfig.SetProcessSelinuxLabel(param)
	}
	param = security.GetParam(e.EngineConfig.GetSecurity(), "apparmor")
	if param != "" {
		sylog.Debugf("Applying Apparmor profile %s", param)
		e.EngineConfig.OciConfig.SetProcessApparmorProfile(param)
	}
	param = security.GetParam(e.EngineConfig.GetSecurity(), "seccomp")
	if param != "" {
		sylog.Debugf("Applying seccomp rule from %s", param)
		generator := &e.EngineConfig.OciConfig.Generator
		if err := seccomp.LoadProfileFromFile(param, generator); err != nil {
			return err
		}
	}

	// open file descriptors (autofs bug path)
	return e.prepareAutofs(starterConfig)
}

// prepareInstanceJoinConfig is responsible for getting and
// applying configuration to join a running instance.
func (e *EngineOperations) prepareInstanceJoinConfig(starterConfig *starter.Config) error {
	name := instance.ExtractName(e.EngineConfig.GetImage())
	file, err := instance.Get(name, instance.SingSubDir)
	if err != nil {
		return err
	}

	uid := os.Getuid()
	gid := os.Getgid()
	suidRequired := uid != 0 && !file.UserNs

	// basic checks:
	// 1. a user must not use SUID workflow to join an instance
	//    started with user namespace
	// 2. a user must use SUID workflow to join an instance
	//    started without user namespace
	if starterConfig.GetIsSUID() && !suidRequired {
		return fmt.Errorf("joining user namespace with suid workflow is not allowed")
	} else if !starterConfig.GetIsSUID() && suidRequired {
		return fmt.Errorf("a setuid installation is required to join this instance")
	}

	// Pid and PPid are stored in instance file and can be controlled
	// by users, check to make sure these values are sane
	if file.Pid <= 1 || file.PPid <= 1 {
		return fmt.Errorf("bad instance process ID found")
	}

	// instance configuration holding configuration read
	// from instance file
	instanceEngineConfig := singularityConfig.NewConfig()

	// extract engine configuration from instance file, the whole content
	// of this file can't be trusted
	instanceConfig := &config.Common{
		EngineConfig: instanceEngineConfig,
	}
	if err := json.Unmarshal(file.Config, instanceConfig); err != nil {
		return err
	}

	// configuration may be altered, be sure to not panic
	if instanceEngineConfig.OciConfig.Linux == nil {
		instanceEngineConfig.OciConfig.Linux = &specs.Linux{}
	}

	// go into /proc/<pid> directory to open namespaces inodes
	// relative to current working directory while joining
	// namespaces within C starter code as changing directory
	// here will also affects starter process thanks to
	// SetWorkingDirectoryFd call.
	// Additionally it would prevent TOCTOU races and symlink
	// usage.
	// And if instance process exits during checks or while
	// entering in namespace, we would get a "no such process"
	// error because current working directory would point to a
	// deleted inode: "/proc/self/cwd -> /proc/<pid> (deleted)"
	path := filepath.Join("/proc", strconv.Itoa(file.Pid))
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_DIRECTORY, 0)
	if err != nil {
		return fmt.Errorf("could not open proc directory %s: %s", path, err)
	}
	if err := mainthread.Fchdir(fd); err != nil {
		return err
	}
	// will set starter (via fchdir too) in the same proc directory
	// in order to open namespace inodes with relative paths for the
	// right process
	starterConfig.SetWorkingDirectoryFd(fd)

	// enforce checks while joining an instance process with SUID workflow
	// since instance file is stored in user home directory, we can't trust
	// its content when using SUID workflow
	if suidRequired {
		// check if instance is running with user namespace enabled
		// by reading /proc/pid/uid_map
		_, hid, err := proc.ReadIDMap("uid_map")

		// if the error returned is "no such file or directory" it means
		// that user namespaces are not supported, just skip this check
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to read user namespace mapping: %s", err)
		} else if err == nil && hid > 0 {
			// a host uid greater than 0 means user namespace is in use for this process
			return fmt.Errorf("trying to join an instance running with user namespace enabled")
		}

		// read "/proc/pid/root" link of instance process must return
		// a permission denied error.
		// This is the "sinit" process (PID 1 in container) and it inherited
		// setuid bit, so most of "/proc/pid" entries are owned by root:root
		// like "/proc/pid/root" link even if the process has dropped all
		// privileges and run with user UID/GID. So we expect a "permission denied"
		// error when reading link.
		if _, err := mainthread.Readlink("root"); !os.IsPermission(err) {
			return fmt.Errorf("trying to join a wrong instance process")
		}
		// Since we could be tricked to join namespaces of a root owned process,
		// we will get UID/GID information of task directory to be sure it belongs
		// to the user currently joining the instance. Also ensure that a user won't
		// be able to join other user's instances.
		fi, err := os.Stat("task")
		if err != nil {
			return fmt.Errorf("error while getting information for instance task directory: %s", err)
		}
		st := fi.Sys().(*syscall.Stat_t)
		if st.Uid != uint32(uid) || st.Gid != uint32(gid) {
			return fmt.Errorf("instance process owned by %d:%d instead of %d:%d", st.Uid, st.Gid, uid, gid)
		}

		ppid := -1

		// read "/proc/pid/status" to check if instance process
		// is neither orphaned or faked
		f, err := os.Open("status")
		if err != nil {
			return fmt.Errorf("could not open status: %s", err)
		}

		for s := bufio.NewScanner(f); s.Scan(); {
			if n, _ := fmt.Sscanf(s.Text(), "PPid:\t%d", &ppid); n == 1 {
				break
			}
		}
		f.Close()

		// check that Ppid/Pid read from instance file are "somewhat" valid
		// processes
		if ppid <= 1 || ppid != file.PPid {
			return fmt.Errorf("orphaned (or faked) instance process")
		}

		// read "/proc/ppid/root" link of parent instance process must return
		// a permission denied error (same logic than "sinit" process).
		// Also we don't use absolute path because we want to return an error
		// if current working directory is deleted meaning that instance process
		// exited.
		path := filepath.Join("..", strconv.Itoa(file.PPid), "root")
		if _, err := mainthread.Readlink(path); !os.IsPermission(err) {
			return fmt.Errorf("trying to join a wrong instance process")
		}
		// "/proc/ppid/task" directory must be owned by user UID/GID
		path = filepath.Join("..", strconv.Itoa(file.PPid), "task")
		fi, err = os.Stat(path)
		if err != nil {
			return fmt.Errorf("error while getting information for parent task directory: %s", err)
		}
		st = fi.Sys().(*syscall.Stat_t)
		if st.Uid != uint32(uid) || st.Gid != uint32(gid) {
			return fmt.Errorf("parent instance process owned by %d:%d instead of %d:%d", st.Uid, st.Gid, uid, gid)
		}
	}

	path, err = filepath.Abs("comm")
	if err != nil {
		return fmt.Errorf("failed to determine absolute path for comm: %s", err)
	}

	// we must read "sinit\n"
	b, err := ioutil.ReadFile("comm")
	if err != nil {
		return fmt.Errorf("failed to read %s: %s", path, err)
	}
	// check that we are currently joining sinit process
	if "sinit" != strings.Trim(string(b), "\n") {
		return fmt.Errorf("sinit not found in %s, wrong instance process", path)
	}

	// tell starter that we are joining an instance
	starterConfig.SetNamespaceJoinOnly(true)

	// update namespaces path relative to /proc/<pid>
	// since starter process is in /proc/<pid> directory
	for i := range instanceEngineConfig.OciConfig.Linux.Namespaces {
		// ignore unknown namespaces
		t := instanceEngineConfig.OciConfig.Linux.Namespaces[i].Type
		if _, ok := nsProcName[t]; !ok {
			continue
		}
		// set namespace relative path
		instanceEngineConfig.OciConfig.Linux.Namespaces[i].Path = filepath.Join("ns", nsProcName[t])
	}

	// store namespace paths in starter configuration that will
	// be passed via a shared memory area and used by starter C code
	// once this process exit
	if err := starterConfig.SetNsPathFromSpec(instanceEngineConfig.OciConfig.Linux.Namespaces); err != nil {
		return err
	}

	// duplicate instance capabilities
	if instanceEngineConfig.OciConfig.Process != nil && instanceEngineConfig.OciConfig.Process.Capabilities != nil {
		e.EngineConfig.OciConfig.Process.Capabilities.Permitted = instanceEngineConfig.OciConfig.Process.Capabilities.Permitted
		e.EngineConfig.OciConfig.Process.Capabilities.Effective = instanceEngineConfig.OciConfig.Process.Capabilities.Effective
		e.EngineConfig.OciConfig.Process.Capabilities.Inheritable = instanceEngineConfig.OciConfig.Process.Capabilities.Inheritable
		e.EngineConfig.OciConfig.Process.Capabilities.Bounding = instanceEngineConfig.OciConfig.Process.Capabilities.Bounding
		e.EngineConfig.OciConfig.Process.Capabilities.Ambient = instanceEngineConfig.OciConfig.Process.Capabilities.Ambient
	}

	// check if user is authorized to set those capabilities and remove
	// unauthorized capabilities from current set according to capability
	// configuration file
	if uid == 0 {
		if err := e.prepareRootCaps(); err != nil {
			return err
		}
	} else {
		if err := e.prepareUserCaps(suidRequired); err != nil {
			return err
		}
	}

	// set UID/GID for the fakeroot context
	if instanceEngineConfig.GetFakeroot() {
		starterConfig.SetTargetUID(0)
		starterConfig.SetTargetGID([]int{0})
	}

	// restore HOME environment variable to match the
	// one set during instance start
	e.EngineConfig.OciConfig.AddProcessEnv("HOME", instanceEngineConfig.GetHomeDest())

	// restore apparmor profile or apply a new one if provided
	param := security.GetParam(e.EngineConfig.GetSecurity(), "apparmor")
	if param != "" {
		sylog.Debugf("Applying Apparmor profile %s", param)
		e.EngineConfig.OciConfig.SetProcessApparmorProfile(param)
	} else {
		e.EngineConfig.OciConfig.SetProcessApparmorProfile(instanceEngineConfig.OciConfig.Process.ApparmorProfile)
	}

	// restore selinux context or apply a new one if provided
	param = security.GetParam(e.EngineConfig.GetSecurity(), "selinux")
	if param != "" {
		sylog.Debugf("Applying SELinux context %s", param)
		e.EngineConfig.OciConfig.SetProcessSelinuxLabel(param)
	} else {
		e.EngineConfig.OciConfig.SetProcessSelinuxLabel(instanceEngineConfig.OciConfig.Process.SelinuxLabel)
	}

	// restore seccomp filter or apply a new one if provided
	param = security.GetParam(e.EngineConfig.GetSecurity(), "seccomp")
	if param != "" {
		sylog.Debugf("Applying seccomp rule from %s", param)
		generator := &e.EngineConfig.OciConfig.Generator
		if err := seccomp.LoadProfileFromFile(param, generator); err != nil {
			return err
		}
	} else {
		if e.EngineConfig.OciConfig.Linux == nil {
			e.EngineConfig.OciConfig.Linux = &specs.Linux{}
		}
		e.EngineConfig.OciConfig.Linux.Seccomp = instanceEngineConfig.OciConfig.Linux.Seccomp
	}

	// only root user can set this value based on instance file
	// and always set to true for normal users or if instance file
	// returned a wrong configuration
	if uid == 0 && instanceEngineConfig.OciConfig.Process != nil {
		e.EngineConfig.OciConfig.Process.NoNewPrivileges = instanceEngineConfig.OciConfig.Process.NoNewPrivileges
	} else {
		e.EngineConfig.OciConfig.Process.NoNewPrivileges = true
	}

	return nil
}

// openDevFuse is a helper function that opens /dev/fuse once for each
// plugin that wants to mount a FUSE filesystem.
func openDevFuse(e *EngineOperations, starterConfig *starter.Config) error {
	for _, name := range e.EngineConfig.GetPluginFuseMounts() {
		fd, err := syscall.Open("/dev/fuse", syscall.O_RDWR, 0)
		if err != nil {
			sylog.Debugf("Calling open: %+v\n", err)
			return err
		}

		err = e.EngineConfig.SetPluginFuseFd(name, fd)
		if err != nil {
			sylog.Debugf("Unable to setup plugin %s fd: %+v\n", name, err)
			return err
		}

		starterConfig.KeepFileDescriptor(fd)
	}

	return nil
}

func (e *EngineOperations) checkSignalPropagation() {
	// obtain the process group ID of the associated controlling
	// terminal (if there's one).
	pgrp := 0
	for i := 0; i <= 2; i++ {
		// The two possible errors:
		// - EBADF will return 0 as process group
		// - ENOTTY will also return 0 as process group
		pgrp, _ = unix.IoctlGetInt(i, unix.TIOCGPGRP)
		// based on kernel source a 0 value for process group
		// theorically be set but really not sure it can happen
		// with linux tty behavior
		if pgrp != 0 {
			break
		}
	}
	// cases we need to propagate signals to container process:
	// - when pgrp == 0 because container won't run in a terminal
	// - when process group is different from the process group controlling terminal
	if pgrp == 0 || (pgrp > 0 && pgrp != syscall.Getpgrp()) {
		e.EngineConfig.SetSignalPropagation(true)
	}
}

// setSessionLayer will test if overlay is supported/allowed.
func (e *EngineOperations) setSessionLayer(img *image.Image) error {
	e.EngineConfig.SetSessionLayer(singularityConfig.DefaultLayer)

	writableTmpfs := e.EngineConfig.GetWritableTmpfs()
	writableImage := e.EngineConfig.GetWritableImage()
	hasOverlayImage := len(e.EngineConfig.GetOverlayImage()) > 0

	if writableImage && hasOverlayImage {
		return fmt.Errorf("you could not use --overlay in conjunction with --writable")
	}

	// NEED FIX: on ubuntu until 4.15 kernel it was possible to mount overlay
	// with the current workflow, since 4.18 we get an operation not permitted
	for _, ns := range e.EngineConfig.OciConfig.Linux.Namespaces {
		if ns.Type == specs.UserNamespace {
			if !e.EngineConfig.File.EnableUnderlay {
				sylog.Debugf("Not attempting to use underlay with user namespace: disabled by configuration ('enable underlay = no')")
				return nil
			}
			if !writableImage {
				sylog.Debugf("Using underlay layer: user namespace requested")
				e.EngineConfig.SetSessionLayer(singularityConfig.UnderlayLayer)
				return nil
			}
			sylog.Debugf("Not attempting to use overlay or underlay: writable flag requested")
			return nil
		}
	}

	// starter was forced to load overlay module, now check if there
	// is an overlay entry in /proc/filesystems
	if has, _ := proc.HasFilesystem("overlay"); has {
		sylog.Debugf("Overlay seems supported and allowed by kernel")
		switch e.EngineConfig.File.EnableOverlay {
		case "yes", "try":
			e.EngineConfig.SetSessionLayer(singularityConfig.OverlayLayer)

			// a SIF image may contain one or more overlay partition
			// check there is at least one ext3 overlay partition
			// to validate overlay with writable flag
			hasSIFOverlay := false

			if img.Type == image.SIF {
				for _, p := range img.Partitions[1:] {
					if p.Type == image.EXT3 {
						hasSIFOverlay = true
						break
					}
				}
			}

			if !writableImage || hasSIFOverlay {
				sylog.Debugf("Attempting to use overlayfs (enable overlay = %v)\n", e.EngineConfig.File.EnableOverlay)
				return nil
			}

			sylog.Debugf("Not attempting to use overlay or underlay: writable flag requested")
			e.EngineConfig.SetSessionLayer(singularityConfig.DefaultLayer)
			return nil
		default:
			if hasOverlayImage {
				return fmt.Errorf("overlay images requires 'enable overlay = yes': set to 'no' by administrator")
			}
			if writableTmpfs {
				return fmt.Errorf("--writable-tmpfs requires 'enable overlay = yes': set to 'no' by administrator")
			}
			sylog.Debugf("Could not use overlay, disabled by configuration ('enable overlay = no')")
		}
	} else {
		if writableTmpfs {
			return fmt.Errorf("--writable-tmpfs requires overlay kernel support: your kernel doesn't support it")
		}
		if hasOverlayImage {
			return fmt.Errorf("overlay images requires overlay kernel support: your kernel doesn't support it")
		}
	}

	// if --writable wasn't set, use underlay if possible
	if !writableImage && e.EngineConfig.File.EnableUnderlay {
		sylog.Debugf("Attempting to use underlay (enable underlay = yes)\n")
		e.EngineConfig.SetSessionLayer(singularityConfig.UnderlayLayer)
		return nil
	} else if writableImage {
		sylog.Debugf("Not attempting to use overlay or underlay: writable flag requested")
		return nil
	}

	sylog.Debugf("Not attempting to use overlay or underlay: both disabled by administrator")
	return nil
}

func (e *EngineOperations) loadImages(starterConfig *starter.Config) error {
	images := make([]image.Image, 0)

	// load rootfs image
	writable := e.EngineConfig.GetWritableImage()
	img, err := e.loadImage(e.EngineConfig.GetImage(), writable)
	if err != nil {
		return err
	}

	if !img.HasRootFs() {
		return fmt.Errorf("no root filesystem partition found in image %s", e.EngineConfig.GetImage())
	}

	if writable && !img.Writable {
		return fmt.Errorf("could not use %s for writing, you don't have write permissions", img.Path)
	}

	if err := e.setSessionLayer(img); err != nil {
		return err
	}

	sessionLayer := e.EngineConfig.GetSessionLayer()

	// first image is always the root filesystem
	images = append(images, *img)
	writableOverlayPath := ""
	overlayPartitions := []string{}

	// sandbox are handled differently for security reasons
	if img.Type == image.SANDBOX {
		if img.Path == "/" {
			return fmt.Errorf("/ as sandbox is not authorized")
		}

		if err := overlay.CheckLower(img.Path); overlay.IsIncompatible(err) {
			// a warning message would be better but on some systems it
			// could annoy users, make it verbose instead
			sylog.Verbosef("Fallback to underlay layer: %s", err)
			e.EngineConfig.SetSessionLayer(singularityConfig.UnderlayLayer)
			return nil
		} else if err != nil {
			return fmt.Errorf("while checking image compatibility with overlay: %s", err)
		}

		// C starter code will position current working directory
		starterConfig.SetWorkingDirectoryFd(int(img.Fd))
	} else if img.Type == image.SIF {
		// query the ECL module, proceed if an ecl config file is found
		ecl, err := syecl.LoadConfig(buildcfg.ECL_FILE)
		if err == nil {
			if err = ecl.ValidateConfig(); err != nil {
				return err
			}
			_, err := ecl.ShouldRunFp(img.File)
			if err != nil {
				return err
			}
		}
		// load overlay partition if we use overlay layer
		if sessionLayer == singularityConfig.OverlayLayer {
			// look for potential overlay partition in SIF image
			// and inject them into the overlay list
			for _, p := range img.Partitions[1:] {
				if p.Type == image.EXT3 || p.Type == image.SQUASHFS {
					imgCopy := *img
					imgCopy.Type = int(p.Type)
					imgCopy.Partitions = []image.Section{p}
					images = append(images, imgCopy)
					overlayPartitions = append(overlayPartitions, imgCopy.Path)
					if img.Writable && p.Type == image.EXT3 {
						writableOverlayPath = img.Path
					}
				}
			}
		}
		// SIF image open for writing without writable
		// overlay partition, assuming that the root
		// filesystem is squashfs or encrypted squashfs
		if img.Writable && img.Partitions[0].Type != image.EXT3 && writableOverlayPath == "" {
			return fmt.Errorf("no SIF writable overlay partition found in %s", img.Path)
		}
	}

	if err := starterConfig.KeepFileDescriptor(int(img.Fd)); err != nil {
		return err
	}

	// lock all ext3 partitions if any to prevent concurrent writes
	for _, part := range img.Partitions {
		if part.Type == image.EXT3 {
			if err := img.LockSection(part); err != nil {
				return fmt.Errorf("error while locking ext3 partition from %s: %s", img.Path, err)
			}
		}
	}

	// load overlay images
	for _, overlayImg := range e.EngineConfig.GetOverlayImage() {
		writableOverlay := true

		splitted := strings.SplitN(overlayImg, ":", 2)
		if len(splitted) == 2 {
			if splitted[1] == "ro" {
				writableOverlay = false
			}
		}

		img, err := e.loadImage(splitted[0], writableOverlay)
		if err != nil {
			return fmt.Errorf("failed to open overlay image %s: %s", splitted[0], err)
		}

		for _, part := range img.Partitions {
			// lock all ext3 partitions if any to prevent concurrent writes
			if part.Type == image.EXT3 {
				if err := img.LockSection(part); err != nil {
					return fmt.Errorf("error while locking ext3 overlay partition from %s: %s", img.Path, err)
				}
			}

			if writableOverlay && img.Writable {
				if writableOverlayPath != "" {
					return fmt.Errorf(
						"you can't specify more than one writable overlay, "+
							"%s contains a writable overlay, requires to use '--overlay %s:ro'",
						writableOverlayPath, img.Path,
					)
				}
				writableOverlayPath = img.Path
			}
		}

		if err := starterConfig.KeepFileDescriptor(int(img.Fd)); err != nil {
			return err
		}
		images = append(images, *img)
	}

	if e.EngineConfig.GetWritableTmpfs() && writableOverlayPath != "" {
		return fmt.Errorf("you can't specify --writable-tmpfs with another writable overlay image (%s)", writableOverlayPath)
	}

	e.EngineConfig.SetOverlayImage(append(e.EngineConfig.GetOverlayImage(), overlayPartitions...))
	e.EngineConfig.SetImageList(images)

	return nil
}

func (e *EngineOperations) loadImage(path string, writable bool) (*image.Image, error) {
	imgObject, err := image.Init(path, writable)
	if err != nil {
		return nil, err
	}

	link, err := mainthread.Readlink(imgObject.Source)
	if err != nil {
		return nil, err
	}

	if link != imgObject.Path {
		return nil, fmt.Errorf("resolved path %s doesn't match with opened path %s", imgObject.Path, link)
	}

	if len(e.EngineConfig.File.LimitContainerPaths) != 0 {
		if authorized, err := imgObject.AuthorizedPath(e.EngineConfig.File.LimitContainerPaths); err != nil {
			return nil, err
		} else if !authorized {
			return nil, fmt.Errorf("singularity image is not in an allowed configured path")
		}
	}
	if len(e.EngineConfig.File.LimitContainerGroups) != 0 {
		if authorized, err := imgObject.AuthorizedGroup(e.EngineConfig.File.LimitContainerGroups); err != nil {
			return nil, err
		} else if !authorized {
			return nil, fmt.Errorf("singularity image is not owned by required group(s)")
		}
	}
	if len(e.EngineConfig.File.LimitContainerOwners) != 0 {
		if authorized, err := imgObject.AuthorizedOwner(e.EngineConfig.File.LimitContainerOwners); err != nil {
			return nil, err
		} else if !authorized {
			return nil, fmt.Errorf("singularity image is not owned by required user(s)")
		}
	}

	switch imgObject.Type {
	case image.SANDBOX:
		if !e.EngineConfig.File.AllowContainerDir {
			return nil, fmt.Errorf("configuration disallows users from running sandbox based containers")
		}
	case image.EXT3:
		if !e.EngineConfig.File.AllowContainerExtfs {
			return nil, fmt.Errorf("configuration disallows users from running extFS based containers")
		}
	case image.SQUASHFS:
		if !e.EngineConfig.File.AllowContainerSquashfs {
			return nil, fmt.Errorf("configuration disallows users from running squashFS based containers")
		}
	}
	return imgObject, nil
}
