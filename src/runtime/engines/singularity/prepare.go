// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"net"

	"github.com/singularityware/singularity/src/pkg/sylog"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// PrepareConfig checks and prepares the runtime engine config
func (engine *EngineOperations) PrepareConfig(masterConn net.Conn) error {
	if engine.CommonConfig.EngineName != Name {
		return fmt.Errorf("incorrect engine")
	}

	if !engine.EngineConfig.File.AllowPidNs && engine.CommonConfig.OciConfig.Linux != nil {
		namespaces := engine.CommonConfig.OciConfig.Linux.Namespaces
		for i, ns := range namespaces {
			if ns.Type == specs.PIDNamespace {
				sylog.Debugf("Not virtualizing PID namespace by configuration")
				engine.CommonConfig.OciConfig.Linux.Namespaces = append(namespaces[:i], namespaces[i+1:]...)
				break
			}
		}
	}

	engine.CommonConfig.OciConfig.SetProcessNoNewPrivileges(true)
	return nil
}
