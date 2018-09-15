// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"net"
	"net/rpc"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/singularityware/singularity/src/runtime/engines/singularity/rpc/client"
)

// CreateContainer creates a container
func (engine *EngineOperations) CreateContainer(pid int, rpcConn net.Conn) error {
	if engine.CommonConfig.EngineName != Name {
		return fmt.Errorf("engineName configuration doesn't match runtime name")
	}

	if engine.EngineConfig.GetInstanceJoin() {
		return nil
	}

	rpcOps := &client.RPC{
		Client: rpc.NewClient(rpcConn),
		Name:   engine.CommonConfig.EngineName,
	}
	if rpcOps.Client == nil {
		return fmt.Errorf("failed to initialiaze RPC client")
	}

	if engine.EngineConfig.GetInstance() {
		if engine.EngineConfig.OciConfig.Linux != nil {
			for i, namespace := range engine.EngineConfig.OciConfig.Linux.Namespaces {
				nstype := ""

				switch namespace.Type {
				case specs.PIDNamespace:
					nstype = "pid"
				case specs.UTSNamespace:
					nstype = "uts"
				case specs.IPCNamespace:
					nstype = "ipc"
				case specs.MountNamespace:
					nstype = "mnt"
				case specs.CgroupNamespace:
					nstype = "cgroup"
				case specs.NetworkNamespace:
					nstype = "net"
				case specs.UserNamespace:
					nstype = "user"
				}

				if nstype != "" {
					path := fmt.Sprintf("/proc/%d/ns/%s", pid, nstype)
					engine.EngineConfig.OciConfig.Linux.Namespaces[i].Path = path
				}
			}
		}
	}

	return create(engine, rpcOps, pid)
}
