// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package workflows

import (
	"fmt"

	"github.com/singularityware/singularity/src/pkg/sylog"
	runtime "github.com/singularityware/singularity/src/pkg/workflows"
	singularity "github.com/singularityware/singularity/src/runtime/workflows/workflows/singularity"
	singularityConfig "github.com/singularityware/singularity/src/runtime/workflows/workflows/singularity/config"
)

var engines map[string]*runtime.Engine

// NewRuntimeEngine instantiates a runtime engine based on JSON configuration
func NewRuntimeEngine(name string, jsonConfig []byte) (*runtime.Engine, error) {
	var engine *runtime.Engine

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
func registerRuntimeEngine(engine *runtime.Engine, name string) {
	if engines == nil {
		engines = make(map[string]*runtime.Engine)
	}
	engines[name] = engine
	engine.RuntimeConfig = engine.InitConfig()
	if engine.RuntimeConfig == nil {
		sylog.Fatalf("failed to initialize %s engine\n", name)
	}
}

func init() {
	// initialize singularity engine
	e := &singularity.Engine{}
	registerRuntimeEngine(&runtime.Engine{Runtime: e}, singularityConfig.Name)
}
