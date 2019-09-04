// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package fakeroot

import (
	"fmt"
	"net"
	"os"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/opencontainers/runtime-tools/generate"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	fakerootutil "github.com/sylabs/singularity/internal/pkg/fakeroot"
	"github.com/sylabs/singularity/internal/pkg/runtime/engine"
	"github.com/sylabs/singularity/internal/pkg/runtime/engine/config"
	"github.com/sylabs/singularity/internal/pkg/runtime/engine/config/starter"
	fakerootConfig "github.com/sylabs/singularity/internal/pkg/runtime/engine/fakeroot/config"
	"github.com/sylabs/singularity/internal/pkg/security/seccomp"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	singularity "github.com/sylabs/singularity/pkg/runtime/engines/singularity/config"
	"github.com/sylabs/singularity/pkg/util/capabilities"
	"github.com/sylabs/singularity/pkg/util/fs/proc"
)

// EngineOperations is a Singularity fakeroot runtime engine that implements engine.Operations.
type EngineOperations struct {
	CommonConfig *config.Common               `json:"-"`
	EngineConfig *fakerootConfig.EngineConfig `json:"engineConfig"`
}

// InitConfig stores the parsed config.Common inside the engine.
//
// Since this method simply stores config.Common, it does not matter
// whether or not there are any elevated privileges during this call.
func (e *EngineOperations) InitConfig(cfg *config.Common) {
	e.CommonConfig = cfg
}

// Config returns a pointer to a fakerootConfig.EngineConfig
// literal as a config.EngineConfig interface. This pointer
// gets stored in the engine.Engine.Common field.
//
// Since this method simply returns a zero value of the concrete
// EngineConfig, it does not matter whether or not there are any elevated
// privileges during this call.
func (e *EngineOperations) Config() config.EngineConfig {
	return e.EngineConfig
}

// PrepareConfig is called during stage1 to validate and prepare
// container configuration. It is responsible for singularity
// configuration file parsing, reading capabilities, configuring
// UID/GID mappings, etc.
//
// No additional privileges can be gained as any of them are already
// dropped by the time PrepareConfig is called.
func (e *EngineOperations) PrepareConfig(starterConfig *starter.Config) error {
	g := generate.Generator{Config: &specs.Spec{}}

	configurationFile := buildcfg.SINGULARITY_CONF_FILE

	// check for ownership of singularity.conf
	if starterConfig.GetIsSUID() && !fs.IsOwner(configurationFile, 0) {
		return fmt.Errorf("%s must be owned by root", configurationFile)
	}

	fileConfig := &singularity.FileConfig{}
	if err := config.Parser(configurationFile, fileConfig); err != nil {
		return fmt.Errorf("unable to parse singularity.conf file: %s", err)
	}

	if starterConfig.GetIsSUID() {
		if !fileConfig.AllowSetuid {
			return fmt.Errorf("fakeroot requires to set 'allow setuid = yes' in %s", configurationFile)
		}
	} else {
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

	g.AddOrReplaceLinuxNamespace(specs.UserNamespace, "")
	g.AddOrReplaceLinuxNamespace(specs.MountNamespace, "")
	g.AddOrReplaceLinuxNamespace(string(specs.PIDNamespace), "")

	uid := uint32(os.Getuid())
	gid := uint32(os.Getgid())

	g.AddLinuxUIDMapping(uid, 0, 1)
	idRange, err := fakerootutil.GetIDRange(fakerootutil.SubUIDFile, uid)
	if err != nil {
		return fmt.Errorf("could not use fakeroot: %s", err)
	}
	g.AddLinuxUIDMapping(idRange.HostID, idRange.ContainerID, idRange.Size)
	starterConfig.AddUIDMappings(g.Config.Linux.UIDMappings)

	g.AddLinuxGIDMapping(gid, 0, 1)
	idRange, err = fakerootutil.GetIDRange(fakerootutil.SubGIDFile, uid)
	if err != nil {
		return fmt.Errorf("could not use fakeroot: %s", err)
	}
	g.AddLinuxGIDMapping(idRange.HostID, idRange.ContainerID, idRange.Size)
	starterConfig.AddGIDMappings(g.Config.Linux.GIDMappings)

	starterConfig.SetHybridWorkflow(true)
	starterConfig.SetAllowSetgroups(true)

	starterConfig.SetTargetUID(0)
	starterConfig.SetTargetGID([]int{0})

	if g.Config.Linux != nil {
		starterConfig.SetNsFlagsFromSpec(g.Config.Linux.Namespaces)
	}

	g.SetupPrivileged(true)

	starterConfig.SetCapabilities(capabilities.Permitted, g.Config.Process.Capabilities.Permitted)
	starterConfig.SetCapabilities(capabilities.Effective, g.Config.Process.Capabilities.Effective)
	starterConfig.SetCapabilities(capabilities.Inheritable, g.Config.Process.Capabilities.Inheritable)
	starterConfig.SetCapabilities(capabilities.Bounding, g.Config.Process.Capabilities.Bounding)
	starterConfig.SetCapabilities(capabilities.Ambient, g.Config.Process.Capabilities.Ambient)

	return nil
}

// CreateContainer does nothing for the fakeroot engine.
func (e *EngineOperations) CreateContainer(pid int, rpcConn net.Conn) error {
	return nil
}

// fakerootSeccompProfile returns a seccomp filter allowing to
// set the return value to 0 for mknod and mknodat syscalls. It
// allows build bootstrap like yum to work with fakeroot.
func fakerootSeccompProfile() *specs.LinuxSeccomp {
	syscalls := []specs.LinuxSyscall{
		{
			Names:  []string{"mknod", "mknodat"},
			Action: specs.ActErrno,
		},
	}
	return &specs.LinuxSeccomp{
		DefaultAction: specs.ActAllow,
		Syscalls:      syscalls,
	}
}

// StartProcess is called during stage2 after RPC server finished
// environment preparation. This is the container process itself.
// It will execute command in the fakeroot context.
//
// This will be executed as a fake root user in a new user
// namespace (PrepareConfig will set both).
func (e *EngineOperations) StartProcess(masterConn net.Conn) error {
	const (
		mountInfo    = "/proc/self/mountinfo"
		selinuxMount = "/sys/fs/selinux"
	)

	if e.EngineConfig == nil {
		return fmt.Errorf("bad fakeroot engine configuration provided")
	}
	if e.EngineConfig.Home == "" {
		return fmt.Errorf("a user home directory is required to bind it on top of /root directory")
	}

	// simple trick to bind user home directory on top of /root
	err := syscall.Mount(e.EngineConfig.Home, "/root", "", syscall.MS_BIND|syscall.MS_REC, "")
	if err != nil {
		return fmt.Errorf("failed to mount %s to /root: %s", e.EngineConfig.Home, err)
	}
	err = syscall.Mount("proc", "/proc", "proc", syscall.MS_NOSUID|syscall.MS_NOEXEC|syscall.MS_NODEV, "")
	if err != nil {
		return fmt.Errorf("failed to mount proc filesystem: %s", err)
	}

	// fix potential issue with SELinux (https://github.com/sylabs/singularity/issues/4038)
	mounts, err := proc.ParseMountInfo(mountInfo)
	if err != nil {
		return fmt.Errorf("while parsing %s: %s", mountInfo, err)
	}
	for _, m := range mounts["/sys"] {
		if m == selinuxMount {
			flags := uintptr(syscall.MS_BIND | syscall.MS_REMOUNT | syscall.MS_RDONLY)
			err = syscall.Mount("", selinuxMount, "", flags, "")
			if err != nil {
				return fmt.Errorf("while remount %s read-only: %s", selinuxMount, err)
			}
			break
		}
	}

	args := e.EngineConfig.Args
	if len(args) == 0 {
		return fmt.Errorf("no command to execute provided")
	}
	env := e.EngineConfig.Envs
	if seccomp.Enabled() {
		if err := seccomp.LoadSeccompConfig(fakerootSeccompProfile(), false, 0); err != nil {
			sylog.Warningf("could not apply seccomp filter, some bootstrap may not work correctly")
		}
	}
	return syscall.Exec(args[0], args, env)
}

// MonitorContainer is called from master once the container has
// been spawned. It will block until the container exists.
//
// Additional privileges may be gained when running hybrid flow.
//
// Particularly here no additional privileges are gained as monitor does
// not need them for wait4 and kill syscalls.
func (e *EngineOperations) MonitorContainer(pid int, signals chan os.Signal) (syscall.WaitStatus, error) {
	var status syscall.WaitStatus

	for {
		s := <-signals
		switch s {
		case syscall.SIGCHLD:
			if wpid, err := syscall.Wait4(pid, &status, syscall.WNOHANG, nil); err != nil {
				return status, fmt.Errorf("error while waiting child: %s", err)
			} else if wpid != pid {
				continue
			}
			return status, nil
		default:
			if err := syscall.Kill(pid, s.(syscall.Signal)); err != nil {
				return status, fmt.Errorf("interrupted by signal %s", s.String())
			}
		}
	}
}

// CleanupContainer does nothing for the fakeroot engine.
func (e *EngineOperations) CleanupContainer(fatal error, status syscall.WaitStatus) error {
	return nil
}

// PostStartProcess does nothing for the fakeroot engine.
func (e *EngineOperations) PostStartProcess(pid int) error {
	return nil
}

func init() {
	engine.RegisterOperations(
		fakerootConfig.Name,
		&EngineOperations{
			EngineConfig: &fakerootConfig.EngineConfig{},
		},
	)
}
