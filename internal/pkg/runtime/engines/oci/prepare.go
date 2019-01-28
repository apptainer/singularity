// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oci

import (
	"fmt"
	"os"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/ociruntime"

	"github.com/kr/pty"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config/starter"
	"github.com/sylabs/singularity/pkg/util/capabilities"
)

// make master/slave as global variable to avoid GC close file descriptor
var (
	master *os.File
	slave  *os.File
)

func (e *EngineOperations) checkCapabilities() error {
	for _, cap := range e.EngineConfig.OciConfig.Process.Capabilities.Permitted {
		if _, ok := capabilities.Map[cap]; !ok {
			return fmt.Errorf("Unrecognized capabilities %s", cap)
		}
	}
	for _, cap := range e.EngineConfig.OciConfig.Process.Capabilities.Effective {
		if _, ok := capabilities.Map[cap]; !ok {
			return fmt.Errorf("Unrecognized capabilities %s", cap)
		}
	}
	for _, cap := range e.EngineConfig.OciConfig.Process.Capabilities.Inheritable {
		if _, ok := capabilities.Map[cap]; !ok {
			return fmt.Errorf("Unrecognized capabilities %s", cap)
		}
	}
	for _, cap := range e.EngineConfig.OciConfig.Process.Capabilities.Bounding {
		if _, ok := capabilities.Map[cap]; !ok {
			return fmt.Errorf("Unrecognized capabilities %s", cap)
		}
	}
	for _, cap := range e.EngineConfig.OciConfig.Process.Capabilities.Ambient {
		if _, ok := capabilities.Map[cap]; !ok {
			return fmt.Errorf("Unrecognized capabilities %s", cap)
		}
	}
	return nil
}

// PrepareConfig checks and prepares the runtime engine config
func (e *EngineOperations) PrepareConfig(starterConfig *starter.Config) error {
	if e.CommonConfig.EngineName != Name {
		return fmt.Errorf("incorrect engine")
	}

	if starterConfig.GetIsSUID() {
		return fmt.Errorf("SUID workflow disabled by administrator")
	}

	if e.EngineConfig.OciConfig.Process == nil {
		return fmt.Errorf("empty OCI process configuration")
	}

	if e.EngineConfig.OciConfig.Linux == nil {
		return fmt.Errorf("empty OCI linux configuration")
	}

	// reset state config that could be passed to engine
	e.EngineConfig.State = ociruntime.State{}

	var gids []int

	uid := int(e.EngineConfig.OciConfig.Process.User.UID)
	gid := e.EngineConfig.OciConfig.Process.User.GID

	gids = append(gids, int(gid))
	for _, g := range e.EngineConfig.OciConfig.Process.User.AdditionalGids {
		gids = append(gids, int(g))
	}

	starterConfig.SetTargetUID(uid)
	starterConfig.SetTargetGID(gids)

	if !e.EngineConfig.Exec {
		starterConfig.SetInstance(true)
	}

	userNS := false
	for _, ns := range e.EngineConfig.OciConfig.Linux.Namespaces {
		if ns.Type == specs.UserNamespace {
			userNS = true
			break
		}
	}
	if !userNS && os.Getuid() != 0 {
		return fmt.Errorf("you can't run without root privileges, use user namespace rather")
	}

	starterConfig.SetNsFlagsFromSpec(e.EngineConfig.OciConfig.Linux.Namespaces)
	if err := starterConfig.SetNsPathFromSpec(e.EngineConfig.OciConfig.Linux.Namespaces); err != nil {
		return err
	}

	if userNS {
		if len(e.EngineConfig.OciConfig.Linux.UIDMappings) == 0 {
			return fmt.Errorf("user namespace invoked without uid mapping")
		}
		if len(e.EngineConfig.OciConfig.Linux.GIDMappings) == 0 {
			return fmt.Errorf("user namespace invoked without gid mapping")
		}
		if err := starterConfig.AddUIDMappings(e.EngineConfig.OciConfig.Linux.UIDMappings); err != nil {
			return err
		}
		if err := starterConfig.AddGIDMappings(e.EngineConfig.OciConfig.Linux.GIDMappings); err != nil {
			return err
		}
	}

	if e.EngineConfig.OciConfig.Linux.RootfsPropagation != "" {
		starterConfig.SetMountPropagation(e.EngineConfig.OciConfig.Linux.RootfsPropagation)
	} else {
		starterConfig.SetMountPropagation("private")
	}

	starterConfig.SetNoNewPrivs(e.EngineConfig.OciConfig.Process.NoNewPrivileges)

	if e.EngineConfig.OciConfig.Process.Capabilities != nil {
		if err := e.checkCapabilities(); err != nil {
			return err
		}

		// force cap_sys_admin for seccomp and no_new_priv flag
		caps := append(e.EngineConfig.OciConfig.Process.Capabilities.Effective, "CAP_SYS_ADMIN")
		starterConfig.SetCapabilities(capabilities.Effective, caps)

		caps = append(e.EngineConfig.OciConfig.Process.Capabilities.Permitted, "CAP_SYS_ADMIN")
		starterConfig.SetCapabilities(capabilities.Permitted, caps)

		starterConfig.SetCapabilities(capabilities.Inheritable, e.EngineConfig.OciConfig.Process.Capabilities.Inheritable)
		starterConfig.SetCapabilities(capabilities.Bounding, e.EngineConfig.OciConfig.Process.Capabilities.Bounding)
		starterConfig.SetCapabilities(capabilities.Ambient, e.EngineConfig.OciConfig.Process.Capabilities.Ambient)
	}

	e.EngineConfig.MasterPts = -1
	e.EngineConfig.SlavePts = -1
	e.EngineConfig.OutputStreams = [2]int{-1, -1}
	e.EngineConfig.ErrorStreams = [2]int{-1, -1}
	e.EngineConfig.InputStreams = [2]int{-1, -1}

	if e.EngineConfig.GetLogFormat() == "" {
		sylog.Debugf("No log format specified, setting kubernetes log format by default")
		e.EngineConfig.SetLogFormat("kubernetes")
	}

	if !e.EngineConfig.Exec {
		if e.EngineConfig.OciConfig.Process.Terminal {
			var err error

			master, slave, err = pty.Open()
			if err != nil {
				return err
			}
			consoleSize := e.EngineConfig.OciConfig.Process.ConsoleSize
			if consoleSize != nil {
				var size pty.Winsize

				size.Cols = uint16(consoleSize.Width)
				size.Rows = uint16(consoleSize.Height)
				if err := pty.Setsize(slave, &size); err != nil {
					return err
				}
			}
			e.EngineConfig.MasterPts = int(master.Fd())
			e.EngineConfig.SlavePts = int(slave.Fd())
		} else {
			flags := syscall.O_CLOEXEC
			if err := syscall.Pipe2(e.EngineConfig.OutputStreams[0:], flags); err != nil {
				return err
			}
			if err := syscall.Pipe2(e.EngineConfig.ErrorStreams[0:], flags); err != nil {
				return err
			}
			if err := syscall.Pipe2(e.EngineConfig.InputStreams[0:], flags); err != nil {
				return err
			}
		}
	} else {
		starterConfig.SetJoinMount(true)
	}

	return nil
}
