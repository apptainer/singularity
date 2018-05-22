// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package runtime

import (
	"net"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	config "github.com/singularityware/singularity/src/pkg/workflows/config"
)

// Engine describes the runtime engine
type Engine struct {
	*config.RuntimeConfig
	Runtime
}

// CLI describes the runtime CLI
type CLI struct {
	*config.RuntimeConfig
	OCIRuntime
}

// OCIRuntime describes the interface for an OCI runtime
type OCIRuntime interface {
	State(id string) *specs.State
	Create(id string, bundle string)
	Start(id string)
	Kill(id string, signal int)
	Delete(id string)
}

// Runtime operations
type Runtime interface {
	// intialize configuration and return it/
	InitConfig() *config.RuntimeConfig
	// call in stage1
	CheckConfig() error
	// are we run as instance
	IsRunAsInstance() bool
	// call in child stage2
	CreateContainer(rpcConn net.Conn) error
	// call in parent stage2 before waiting stage2 child
	PrestartProcess() error
	// call in parent stage2 after stage2 child exit
	StartProcess() error
	// call in smaster once container is created
	MonitorContainer() error
	// call in smaster for container cleanup
	CleanupContainer() error
}
