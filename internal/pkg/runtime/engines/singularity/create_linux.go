// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
	"path/filepath"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/singularity/rpc/client"
	singularityConfig "github.com/sylabs/singularity/pkg/runtime/engines/singularity/config"
)

// CreateContainer creates a container
func (engine *EngineOperations) CreateContainer(pid int, rpcConn net.Conn) error {
	if engine.CommonConfig.EngineName != singularityConfig.Name {
		return fmt.Errorf("engineName configuration doesn't match runtime name")
	}

	if engine.EngineConfig.GetInstanceJoin() {
		return nil
	}

	configurationFile := buildcfg.SYSCONFDIR + "/singularity/singularity.conf"
	if err := config.Parser(configurationFile, engine.EngineConfig.File); err != nil {
		return fmt.Errorf("Unable to parse singularity.conf file: %s", err)
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
			nstype       string
			ns           specs.LinuxNamespaceType
			checkEnabled bool
		}{
			{"pid", specs.PIDNamespace, false},
			{"uts", specs.UTSNamespace, false},
			{"ipc", specs.IPCNamespace, false},
			{"mnt", specs.MountNamespace, false},
			{"cgroup", specs.CgroupNamespace, false},
			{"net", specs.NetworkNamespace, false},
			{"user", specs.UserNamespace, true},
		}

		path := fmt.Sprintf("/proc/%d/ns", pid)
		ppid := os.Getpid()

		for _, n := range namespaces {
			has, err := rpcOps.HasNamespace(ppid, n.nstype)
			if err == nil && (has || n.checkEnabled) {
				enabled := false
				if n.checkEnabled {
					if engine.EngineConfig.OciConfig.Linux != nil {
						for _, namespace := range engine.EngineConfig.OciConfig.Linux.Namespaces {
							if n.ns == namespace.Type {
								enabled = true
								break
							}
						}
					}
				}
				if has || enabled {
					nspath := filepath.Join(path, n.nstype)
					engine.EngineConfig.OciConfig.AddOrReplaceLinuxNamespace(string(n.ns), nspath)
				}
			} else if err != nil {
				return fmt.Errorf("failed to check %s root and container namespace: %s", n.ns, err)
			}
		}
	}

	return create(engine, rpcOps, pid)
}
