package workflows

import (
	"fmt"

	"github.com/singularityware/singularity/src/pkg/sylog"
	runtime "github.com/singularityware/singularity/src/pkg/workflows"
	singularity "github.com/singularityware/singularity/src/runtime/workflows/workflows/singularity"
	singularityConfig "github.com/singularityware/singularity/src/runtime/workflows/workflows/singularity/config"
)

var engines map[string]*runtime.RuntimeEngine

// Instanciate a runtime engine based on json configuration
func NewRuntimeEngine(name string, jsonConfig []byte) (*runtime.RuntimeEngine, error) {
	var engine *runtime.RuntimeEngine

	engine = engines[name]

	if engine == nil {
		return nil, fmt.Errorf("no runtime engine named %s found", name)
	}
	if err := engine.SetConfig(jsonConfig); err != nil {
		return nil, fmt.Errorf("json parsing failed: %v", err)
	}
	return engine, nil
}

// Register a runtime engine
func registerRuntimeEngine(engine *runtime.RuntimeEngine, name string) {
	if engines == nil {
		engines = make(map[string]*runtime.RuntimeEngine)
	}
	engines[name] = engine
	engine.RuntimeConfig = engine.InitConfig()
	if engine.RuntimeConfig == nil {
		sylog.Fatalf("failed to initialize %s engine\n", name)
	}
}

func init() {
	// initialize singularity engine
	e := &singularity.RuntimeEngine{}
	registerRuntimeEngine(&runtime.RuntimeEngine{Runtime: e}, singularityConfig.Name)
}
