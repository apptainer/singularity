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

	"github.com/singularityware/singularity/src/pkg/util/user"

	"github.com/singularityware/singularity/src/pkg/buildcfg"
	"github.com/singularityware/singularity/src/pkg/instance"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/capabilities"
	"github.com/singularityware/singularity/src/runtime/engines/common/config"
	"github.com/singularityware/singularity/src/runtime/engines/common/config/starter"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// prepareUserCaps is responsible for checking that user's requested
// capabilities are authorized
func (e *EngineOperations) prepareUserCaps() error {
	uid := os.Getuid()
	commonCaps := make([]string, 0)

	e.CommonConfig.OciConfig.SetProcessNoNewPrivileges(true)

	file, err := capabilities.Open(buildcfg.CAPABILITY_FILE, true)
	if err != nil {
		return err
	}
	defer file.Close()

	pw, err := user.GetPwUID(uint32(uid))
	if err != nil {
		return err
	}

	caps, _ := capabilities.Split(e.EngineConfig.GetAddCaps())
	authorizedCaps, _ := file.CheckUserCaps(pw.Name, caps)

	if len(authorizedCaps) > 0 {
		sylog.Debugf("User capabilities %v added", authorizedCaps)
		commonCaps = authorizedCaps
	}

	groups, err := os.Getgroups()
	for _, g := range groups {
		gr, err := user.GetGrGID(uint32(g))
		if err != nil {
			return err
		}
		authorizedCaps, _ := file.CheckGroupCaps(gr.Name, caps)
		if len(authorizedCaps) > 0 {
			sylog.Debugf("%s group capabilities %v added", gr.Name, authorizedCaps)
			commonCaps = append(commonCaps, authorizedCaps...)
		}
	}

	commonCaps = capabilities.RemoveDuplicated(commonCaps)

	caps, _ = capabilities.Split(e.EngineConfig.GetDropCaps())
	for _, cap := range caps {
		for i, c := range commonCaps {
			if c == cap {
				sylog.Debugf("Capability %s dropped", cap)
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

	return nil
}

// prepareRootCaps is responsible for setting root capabilities
// based on capability/configuration files and requested capabilities
func (e *EngineOperations) prepareRootCaps() error {
	commonCaps := make([]string, 0)
	defaultCapabilities := e.EngineConfig.File.RootDefaultCapabilities

	// is no-privs/keep-privs set on command line
	if e.EngineConfig.GetNoPrivs() {
		sylog.Debugf("--no-privs requested")
		defaultCapabilities = "no"
	} else if e.EngineConfig.GetKeepPrivs() {
		sylog.Debugf("--keep-privs requested")
		defaultCapabilities = "full"
	}

	sylog.Debugf("Root %s capabilities", defaultCapabilities)

	// set default capabilities based on configuration file directive
	switch defaultCapabilities {
	case "full":
		e.CommonConfig.OciConfig.SetupPrivileged(true)
		commonCaps = e.CommonConfig.OciConfig.Process.Capabilities.Permitted
	case "file":
		file, err := capabilities.Open(buildcfg.CAPABILITY_FILE, true)
		if err != nil {
			return err
		}
		defer file.Close()

		commonCaps = append(commonCaps, file.ListUserCaps("root")...)
		groups, err := os.Getgroups()
		for _, g := range groups {
			gr, err := user.GetGrGID(uint32(g))
			if err != nil {
				return err
			}
			caps := file.ListGroupCaps(gr.Name)
			commonCaps = append(commonCaps, caps...)
			sylog.Debugf("%s group capabilities %v added", gr.Name, caps)
		}
	}

	caps, _ := capabilities.Split(e.EngineConfig.GetAddCaps())
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

	caps, _ = capabilities.Split(e.EngineConfig.GetDropCaps())
	for _, cap := range caps {
		for i, c := range commonCaps {
			if c == cap {
				sylog.Debugf("Root capability %s dropped", cap)
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

	return nil
}

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
		if err := e.prepareRootCaps(); err != nil {
			return err
		}
	} else {
		if err := e.prepareUserCaps(); err != nil {
			return err
		}
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

	// duplicate instance capabilities
	if instanceConfig.OciConfig.Process != nil && instanceConfig.OciConfig.Process.Capabilities != nil {
		e.CommonConfig.OciConfig.Process.Capabilities.Permitted = instanceConfig.OciConfig.Process.Capabilities.Permitted
		e.CommonConfig.OciConfig.Process.Capabilities.Effective = instanceConfig.OciConfig.Process.Capabilities.Effective
		e.CommonConfig.OciConfig.Process.Capabilities.Inheritable = instanceConfig.OciConfig.Process.Capabilities.Inheritable
		e.CommonConfig.OciConfig.Process.Capabilities.Bounding = instanceConfig.OciConfig.Process.Capabilities.Bounding
		e.CommonConfig.OciConfig.Process.Capabilities.Ambient = instanceConfig.OciConfig.Process.Capabilities.Ambient
	}

	if os.Getuid() == 0 {
		if err := e.prepareRootCaps(); err != nil {
			return err
		}
	} else {
		if err := e.prepareUserCaps(); err != nil {
			return err
		}
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

	if e.CommonConfig.OciConfig.Process == nil {
		e.CommonConfig.OciConfig.Process = &specs.Process{}
	}
	if e.CommonConfig.OciConfig.Process.Capabilities == nil {
		e.CommonConfig.OciConfig.Process.Capabilities = &specs.LinuxCapabilities{}
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
