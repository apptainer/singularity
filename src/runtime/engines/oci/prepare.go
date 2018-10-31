// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oci

import (
	"fmt"
	"net"
	"os"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/src/pkg/util/capabilities"
	"github.com/sylabs/singularity/src/runtime/engines/config/starter"
)

// PrepareConfig checks and prepares the runtime engine config
func (e *EngineOperations) PrepareConfig(masterConn net.Conn, starterConfig *starter.Config) error {
	if e.CommonConfig.EngineName != Name {
		return fmt.Errorf("incorrect engine")
	}

	if starterConfig.GetIsSUID() {
		return fmt.Errorf("SUID workflow disabled by administrator")
	}

	if e.EngineConfig.OciConfig.Process == nil {
		return fmt.Errorf("empty OCI process configuration")
	}

	// reset state config that could be passed to engine
	e.EngineConfig.State = specs.State{}

	var gids []int
	uid := int(e.EngineConfig.OciConfig.Process.User.UID)

	gids = append(gids, int(e.EngineConfig.OciConfig.Process.User.GID))
	for _, g := range e.EngineConfig.OciConfig.Process.User.AdditionalGids {
		gids = append(gids, int(g))
	}

	starterConfig.SetTargetUID(uid)
	starterConfig.SetTargetGID(gids)
	starterConfig.SetInstance(true)

	if e.EngineConfig.OciConfig.Linux != nil {
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
		starterConfig.AddUIDMappings(e.EngineConfig.OciConfig.Linux.UIDMappings)
		starterConfig.AddGIDMappings(e.EngineConfig.OciConfig.Linux.GIDMappings)
	} else {
		return fmt.Errorf("empty OCI linux configuration")
	}

	starterConfig.SetNoNewPrivs(e.EngineConfig.OciConfig.Process.NoNewPrivileges)
	starterConfig.SetMountPropagation("private")

	if e.EngineConfig.OciConfig.Process.Capabilities != nil {
		starterConfig.SetCapabilities(capabilities.Permitted, e.EngineConfig.OciConfig.Process.Capabilities.Permitted)
		starterConfig.SetCapabilities(capabilities.Effective, e.EngineConfig.OciConfig.Process.Capabilities.Effective)
		starterConfig.SetCapabilities(capabilities.Inheritable, e.EngineConfig.OciConfig.Process.Capabilities.Inheritable)
		starterConfig.SetCapabilities(capabilities.Bounding, e.EngineConfig.OciConfig.Process.Capabilities.Bounding)
		starterConfig.SetCapabilities(capabilities.Ambient, e.EngineConfig.OciConfig.Process.Capabilities.Ambient)
	}

	return nil
}
