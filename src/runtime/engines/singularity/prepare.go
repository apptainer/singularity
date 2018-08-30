// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"encoding/json"
	"fmt"
	"net"
	"os"

	"github.com/singularityware/singularity/src/pkg/instance"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/capabilities"
	"github.com/singularityware/singularity/src/runtime/engines/common/config"
	"github.com/singularityware/singularity/src/runtime/engines/common/config/starter"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// prepareContainerConfig is responsible for getting and applying user supplied
// configuration for container creation
func (e *EngineOperations) prepareContainerConfig(starterConfig *starter.Config) error {
	// always set mount namespace
	e.CommonConfig.OciConfig.AddOrReplaceLinuxNamespace(specs.MountNamespace, "")

	// if PID namespace is not allowed remove it from namespaces
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

	if os.Getuid() == 0 {
		if e.EngineConfig.File.RootDefaultCapabilities == "full" {
			e.CommonConfig.OciConfig.SetupPrivileged(true)

			commonCaps := e.CommonConfig.OciConfig.Process.Capabilities.Permitted

			caps, _ := capabilities.Split(e.EngineConfig.GetDropCaps())
			for _, cap := range caps {
				for i, c := range commonCaps {
					if c == cap {
						commonCaps = append(commonCaps[:i], commonCaps[i+1:]...)
						break
					}
				}
			}

			e.CommonConfig.OciConfig.Process.Capabilities.Permitted = commonCaps
			e.CommonConfig.OciConfig.Process.Capabilities.Effective = commonCaps
			e.CommonConfig.OciConfig.Process.Capabilities.Inheritable = commonCaps
			e.CommonConfig.OciConfig.Process.Capabilities.Bounding = commonCaps
			e.CommonConfig.OciConfig.Process.Capabilities.Ambient = commonCaps
		}
	} else {
		e.CommonConfig.OciConfig.SetProcessNoNewPrivileges(true)
	}

	if e.EngineConfig.File.MountSlave {
		starterConfig.SetMountPropagation("slave")
	} else {
		starterConfig.SetMountPropagation("private")
	}

	starterConfig.SetInstance(e.EngineConfig.GetInstance())

	starterConfig.SetNsFlagsFromSpec(e.CommonConfig.OciConfig.Linux.Namespaces)

	// user namespace ID mappings
	if e.CommonConfig.OciConfig.Linux != nil {
		starterConfig.AddUIDMappings(e.CommonConfig.OciConfig.Linux.UIDMappings)
		starterConfig.AddGIDMappings(e.CommonConfig.OciConfig.Linux.GIDMappings)
	}

	return nil
}

// prepareInstanceJoinConfig is responsible for getting and applying configuration
// to join a running instance
func (e *EngineOperations) prepareInstanceJoinConfig(starterConfig *starter.Config) error {
	name := instance.ExtractName(e.EngineConfig.GetImage())
	file, err := instance.Get(name)
	if err != nil {
		return err
	}

	// check if SUID workflow is really used with a privileged instance
	if !file.PrivilegedPath() && starterConfig.GetIsSUID() {
		return fmt.Errorf("try to join unprivileged instance with SUID workflow")
	}

	// extract configuration from instance file
	instanceConfig := &config.Common{
		EngineConfig: NewConfig(),
	}
	if err := json.Unmarshal(file.Config, instanceConfig); err != nil {
		return err
	}

	// set namespaces to join
	starterConfig.SetNsPathFromSpec(instanceConfig.OciConfig.Linux.Namespaces)

	if e.CommonConfig.OciConfig.Process == nil {
		e.CommonConfig.OciConfig.Process = &specs.Process{}
	}
	if e.CommonConfig.OciConfig.Process.Capabilities == nil {
		e.CommonConfig.OciConfig.Process.Capabilities = &specs.LinuxCapabilities{}
	}

	// duplicate instance capabilities
	if instanceConfig.OciConfig.Process != nil && instanceConfig.OciConfig.Process.Capabilities != nil {
		e.CommonConfig.OciConfig.Process.Capabilities.Permitted = instanceConfig.OciConfig.Process.Capabilities.Permitted
		e.CommonConfig.OciConfig.Process.Capabilities.Effective = instanceConfig.OciConfig.Process.Capabilities.Effective
		e.CommonConfig.OciConfig.Process.Capabilities.Inheritable = instanceConfig.OciConfig.Process.Capabilities.Inheritable
		e.CommonConfig.OciConfig.Process.Capabilities.Bounding = instanceConfig.OciConfig.Process.Capabilities.Bounding
		e.CommonConfig.OciConfig.Process.Capabilities.Ambient = instanceConfig.OciConfig.Process.Capabilities.Ambient
	}

	e.CommonConfig.OciConfig.Process.NoNewPrivileges = instanceConfig.OciConfig.Process.NoNewPrivileges

	return nil
}

// PrepareConfig checks and prepares the runtime engine config
func (e *EngineOperations) PrepareConfig(masterConn net.Conn, starterConfig *starter.Config) error {
	if e.CommonConfig.EngineName != Name {
		return fmt.Errorf("incorrect engine")
	}

	if !e.EngineConfig.File.AllowSetuid && starterConfig.GetIsSUID() {
		return fmt.Errorf("SUID workflow disabled by administrator")
	}

	if e.EngineConfig.GetInstanceJoin() {
		if err := e.prepareInstanceJoinConfig(starterConfig); err != nil {
			return err
		}
	} else {
		if err := e.prepareContainerConfig(starterConfig); err != nil {
			return err
		}
	}

	starterConfig.SetNoNewPrivs(e.CommonConfig.OciConfig.Process.NoNewPrivileges)

	if e.CommonConfig.OciConfig.Process != nil && e.CommonConfig.OciConfig.Process.Capabilities != nil {
		starterConfig.SetCapabilities(capabilities.Permitted, e.CommonConfig.OciConfig.Process.Capabilities.Permitted)
		starterConfig.SetCapabilities(capabilities.Effective, e.CommonConfig.OciConfig.Process.Capabilities.Effective)
		starterConfig.SetCapabilities(capabilities.Inheritable, e.CommonConfig.OciConfig.Process.Capabilities.Inheritable)
		starterConfig.SetCapabilities(capabilities.Bounding, e.CommonConfig.OciConfig.Process.Capabilities.Bounding)
		starterConfig.SetCapabilities(capabilities.Ambient, e.CommonConfig.OciConfig.Process.Capabilities.Ambient)
	}
	return nil
}
