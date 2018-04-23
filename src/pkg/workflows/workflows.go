package runtime

import (
	"net"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	config "github.com/singularityware/singularity/src/pkg/workflows/config"
)

// Generic runtime engine
type RuntimeEngine struct {
	*config.RuntimeConfig
	Runtime
}

type RuntimeCLI struct {
	*config.RuntimeConfig
	OciRuntime
}

// OCI runtime operations
type OciRuntime interface {
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
