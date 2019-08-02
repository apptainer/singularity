// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"net"
	"net/rpc"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/singularity/rpc/client"
	singularityConfig "github.com/sylabs/singularity/pkg/runtime/engines/singularity/config"
)

// CreateContainer creates a container
func (e *EngineOperations) CreateContainer(pid int, rpcConn net.Conn) error {
	if e.CommonConfig.EngineName != singularityConfig.Name {
		return fmt.Errorf("engineName configuration doesn't match runtime name")
	}

	if e.EngineConfig.GetInstanceJoin() {
		return nil
	}

	configurationFile := buildcfg.SYSCONFDIR + "/singularity/singularity.conf"
	if err := config.Parser(configurationFile, e.EngineConfig.File); err != nil {
		return fmt.Errorf("unable to parse singularity.conf file: %s", err)
	}

	rpcOps := &client.RPC{
		Client: rpc.NewClient(rpcConn),
		Name:   e.CommonConfig.EngineName,
	}
	if rpcOps.Client == nil {
		return fmt.Errorf("failed to initialize RPC client")
	}

	return create(e, rpcOps, pid)
}
