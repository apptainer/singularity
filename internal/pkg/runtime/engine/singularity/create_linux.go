// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"context"
	"fmt"
	"net"
	"net/rpc"

	"github.com/sylabs/singularity/internal/pkg/runtime/engine/singularity/rpc/client"
	singularityConfig "github.com/sylabs/singularity/pkg/runtime/engine/singularity/config"
)

// CreateContainer is called from master process to prepare container
// environment, e.g. perform mount operations, setup network, etc.
//
// Additional privileges required for setup may be gained when running
// in suid flow. However, when a user namespace is requested and it is not
// a hybrid workflow (e.g. fakeroot), then there is no privileged saved uid
// and thus no additional privileges can be gained.
//
// Specifically in singularity engine, additional privileges are gained during
// network setup (see container.prepareNetworkSetup) in fakeroot flow. The rest
// of the setup (e.g. mount operations) where privileges may be required is performed
// by calling RPC server methods (see internal/app/starter/rpc_linux.go for details).
func (e *EngineOperations) CreateContainer(ctx context.Context, pid int, rpcConn net.Conn) error {
	if e.CommonConfig.EngineName != singularityConfig.Name {
		return fmt.Errorf("engineName configuration doesn't match runtime name")
	}

	if e.EngineConfig.GetInstanceJoin() {
		return nil
	}

	rpcOps := &client.RPC{
		Client: rpc.NewClient(rpcConn),
		Name:   e.CommonConfig.EngineName,
	}
	if rpcOps.Client == nil {
		return fmt.Errorf("failed to initialize RPC client")
	}

	return create(ctx, e, rpcOps, pid)
}
