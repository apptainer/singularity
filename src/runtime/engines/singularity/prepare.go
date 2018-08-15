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
	"github.com/singularityware/singularity/src/runtime/engines/common/config"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// PrepareConfig checks and prepares the runtime engine config
func (e *EngineOperations) PrepareEngineConfig(masterConn net.Conn) error {
	if e.CommonConfig.EngineName != Name {
		return fmt.Errorf("incorrect engine")
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

	return nil
}

func (e *EngineOperations) PrepareStartupConfig(startupConfig *config.Startup) error {
	if !e.EngineConfig.File.AllowSetuid && startupConfig.GetIsSUID() {
		return fmt.Errorf("SUID workflow disabled by administrator")
	}

	startupConfig.SetInstance(e.EngineConfig.GetInstance())
	startupConfig.SetNoNewPrivs(e.CommonConfig.OciConfig.Process.NoNewPrivileges)

	if e.CommonConfig.OciConfig.Linux != nil {
		startupConfig.AddUIDMappings(e.CommonConfig.OciConfig.Linux.UIDMappings)
		startupConfig.AddGIDMappings(e.CommonConfig.OciConfig.Linux.UIDMappings)
		startupConfig.SetNsFlagsFromSpec(e.CommonConfig.OciConfig.Linux.Namespaces)
	}
	if e.CommonConfig.OciConfig.Process != nil && e.CommonConfig.OciConfig.Process.Capabilities != nil {
		startupConfig.SetCapabilities(capabilities.Permitted, e.CommonConfig.OciConfig.Process.Capabilities.Permitted)
		startupConfig.SetCapabilities(capabilities.Effective, e.CommonConfig.OciConfig.Process.Capabilities.Effective)
		startupConfig.SetCapabilities(capabilities.Inheritable, e.CommonConfig.OciConfig.Process.Capabilities.Inheritable)
		startupConfig.SetCapabilities(capabilities.Bounding, e.CommonConfig.OciConfig.Process.Capabilities.Bounding)
		startupConfig.SetCapabilities(capabilities.Ambient, e.CommonConfig.OciConfig.Process.Capabilities.Ambient)
	}

	return nil
}
