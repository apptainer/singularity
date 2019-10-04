// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package imgbuild

import (
	"fmt"
	"os"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/internal/pkg/runtime/engine"
	"github.com/sylabs/singularity/internal/pkg/runtime/engine/config/starter"
	imgbuildConfig "github.com/sylabs/singularity/internal/pkg/runtime/engine/imgbuild/config"
	"github.com/sylabs/singularity/internal/pkg/runtime/engine/singularity/rpc/server"
	"github.com/sylabs/singularity/pkg/runtime/engine/config"
	"github.com/sylabs/singularity/pkg/util/capabilities"
)

// EngineOperations is a Singularity runtime engine that implements engine.Operations.
// Basically, this is the core of `singularity build` command.
type EngineOperations struct {
	CommonConfig *config.Common               `json:"-"`
	EngineConfig *imgbuildConfig.EngineConfig `json:"engineConfig"`
}

// InitConfig stores the parsed config.Common inside the engine.
//
// Since this method simply stores config.Common, it does not matter
// whether or not there are any elevated privileges during this call.
func (e *EngineOperations) InitConfig(cfg *config.Common) {
	e.CommonConfig = cfg
}

// Config returns a pointer to imgbuildConfig.EngineConfig literal
// as a config.EngineConfig interface. This pointer gets stored in
// the Engine.Common field.
//
// Since this method simply returns a zero value of the concrete
// EngineConfig, it does not matter whether or not there are any elevated
// privileges during this call.
func (e *EngineOperations) Config() config.EngineConfig {
	return e.EngineConfig
}

// PrepareConfig is called during stage1 to validate and prepare
// build container configuration.
//
// No additional privileges can be gained as any of them are already
// dropped by the time PrepareConfig is called.
//
// Note that imgbuild engine is called by root user or fakeroot engine,
// so technically this method may already be run with escalated privileges.
func (e *EngineOperations) PrepareConfig(starterConfig *starter.Config) error {
	if e.EngineConfig.OciConfig.Generator.Config != &e.EngineConfig.OciConfig.Spec {
		return fmt.Errorf("bad engine configuration provided")
	}
	if starterConfig.GetIsSUID() {
		return fmt.Errorf("imgbuild engine can't run with SUID workflow")
	}
	if os.Getuid() != 0 {
		return fmt.Errorf("unable to run imgbuild engine as non-root user or without --fakeroot")
	}

	e.EngineConfig.OciConfig.SetProcessNoNewPrivileges(true)
	starterConfig.SetNoNewPrivs(e.EngineConfig.OciConfig.Process.NoNewPrivileges)

	e.EngineConfig.OciConfig.SetupPrivileged(true)

	e.EngineConfig.OciConfig.AddOrReplaceLinuxNamespace(specs.MountNamespace, "")

	if e.EngineConfig.OciConfig.Linux != nil {
		starterConfig.SetNsFlagsFromSpec(e.EngineConfig.OciConfig.Linux.Namespaces)
	}
	if e.EngineConfig.OciConfig.Process != nil && e.EngineConfig.OciConfig.Process.Capabilities != nil {
		starterConfig.SetCapabilities(capabilities.Permitted, e.EngineConfig.OciConfig.Process.Capabilities.Permitted)
		starterConfig.SetCapabilities(capabilities.Effective, e.EngineConfig.OciConfig.Process.Capabilities.Effective)
		starterConfig.SetCapabilities(capabilities.Inheritable, e.EngineConfig.OciConfig.Process.Capabilities.Inheritable)
		starterConfig.SetCapabilities(capabilities.Bounding, e.EngineConfig.OciConfig.Process.Capabilities.Bounding)
		starterConfig.SetCapabilities(capabilities.Ambient, e.EngineConfig.OciConfig.Process.Capabilities.Ambient)
	}

	starterConfig.SetMountPropagation("rslave")
	starterConfig.SetMasterPropagateMount(true)

	return nil
}

func init() {
	engine.RegisterOperations(
		imgbuildConfig.Name,
		&EngineOperations{
			EngineConfig: &imgbuildConfig.EngineConfig{},
		},
	)
	engine.RegisterRPCMethods(
		imgbuildConfig.Name,
		new(server.Methods),
	)
}
