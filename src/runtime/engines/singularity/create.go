// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"net"
	"net/rpc"
	"path/filepath"

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
		namespaces := []struct {
			nstype string
			ns     specs.LinuxNamespaceType
		}{
			{"pid", specs.PIDNamespace},
			{"uts", specs.UTSNamespace},
			{"ipc", specs.IPCNamespace},
			{"mnt", specs.MountNamespace},
			{"cgroup", specs.CgroupNamespace},
			{"net", specs.NetworkNamespace},
			{"user", specs.UserNamespace},
		}

		path := fmt.Sprintf("/proc/%d/ns", pid)

		for _, n := range namespaces {
			has, err := rpcOps.HasNamespace(n.nstype)
			if err == nil && has {
				nspath := filepath.Join(path, n.nstype)
				engine.EngineConfig.OciConfig.AddOrReplaceLinuxNamespace(string(n.ns), nspath)
			} else if err != nil {
				return fmt.Errorf("failed to check %s root and container namespace: %s", n.ns, err)
			}
		}
	}

	return create(engine, rpcOps, pid)
}
