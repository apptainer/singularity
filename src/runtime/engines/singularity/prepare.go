// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"net"

	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/capabilities"
	"github.com/singularityware/singularity/src/runtime/engines/common/config/wrapper"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// PrepareConfig checks and prepares the runtime engine config
func (e *EngineOperations) PrepareConfig(masterConn net.Conn, wrapperConfig *wrapper.Config) error {
	if e.CommonConfig.EngineName != Name {
		return fmt.Errorf("incorrect engine")
	}

	if !e.EngineConfig.File.AllowSetuid && wrapperConfig.GetIsSUID() {
		return fmt.Errorf("SUID workflow disabled by administrator")
	}

	if !e.EngineConfig.File.AllowPidNs && e.CommonConfig.OciConfig.Linux != nil {
		namespaces := e.CommonConfig.OciConfig.Linux.Namespaces
		for i, ns := range namespaces {
			if ns.Type == specs.PIDNamespace {
				sylog.Debugf("Not virtualizing PID namespace by configuration")
				e.CommonConfig.OciConfig.Linux.Namespaces = append(namespaces[:i], namespaces[i+1:]...)
				break
			}
		}
	}

	e.CommonConfig.OciConfig.SetProcessNoNewPrivileges(true)

	wrapperConfig.SetInstance(e.EngineConfig.GetInstance())
	wrapperConfig.SetNoNewPrivs(e.CommonConfig.OciConfig.Process.NoNewPrivileges)
	if e.EngineConfig.File.MountSlave {
		wrapperConfig.SetMountPropagation("slave")
	} else {
		wrapperConfig.SetMountPropagation("private")
	}

	if e.CommonConfig.OciConfig.Linux != nil {
		wrapperConfig.AddUIDMappings(e.CommonConfig.OciConfig.Linux.UIDMappings)
		wrapperConfig.AddGIDMappings(e.CommonConfig.OciConfig.Linux.GIDMappings)
		wrapperConfig.SetNsFlagsFromSpec(e.CommonConfig.OciConfig.Linux.Namespaces)
	}
	if e.CommonConfig.OciConfig.Process != nil && e.CommonConfig.OciConfig.Process.Capabilities != nil {
		wrapperConfig.SetCapabilities(capabilities.Permitted, e.CommonConfig.OciConfig.Process.Capabilities.Permitted)
		wrapperConfig.SetCapabilities(capabilities.Effective, e.CommonConfig.OciConfig.Process.Capabilities.Effective)
		wrapperConfig.SetCapabilities(capabilities.Inheritable, e.CommonConfig.OciConfig.Process.Capabilities.Inheritable)
		wrapperConfig.SetCapabilities(capabilities.Bounding, e.CommonConfig.OciConfig.Process.Capabilities.Bounding)
		wrapperConfig.SetCapabilities(capabilities.Ambient, e.CommonConfig.OciConfig.Process.Capabilities.Ambient)
	}
	return nil
}
