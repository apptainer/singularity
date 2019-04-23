// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build build_engine

package imgbuild

import (
	"fmt"
	"syscall"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config/starter"
	imgbuildConfig "github.com/sylabs/singularity/internal/pkg/runtime/engines/imgbuild/config"
	"github.com/sylabs/singularity/pkg/util/capabilities"
)

// EngineOperations implements the engines.EngineOperations interface for
// the image build process
type EngineOperations struct {
	CommonConfig *config.Common               `json:"-"`
	EngineConfig *imgbuildConfig.EngineConfig `json:"engineConfig"`
}

// InitConfig initializes engines config internals
func (e *EngineOperations) InitConfig(cfg *config.Common) {
	e.CommonConfig = cfg
}

// Config returns the EngineConfig
func (e *EngineOperations) Config() config.EngineConfig {
	return e.EngineConfig
}

// PrepareConfig validates/prepares EngineConfig setup
func (e *EngineOperations) PrepareConfig(starterConfig *starter.Config) error {
	e.EngineConfig.OciConfig.SetProcessNoNewPrivileges(true)
	starterConfig.SetNoNewPrivs(e.EngineConfig.OciConfig.Process.NoNewPrivileges)

	if syscall.Getuid() != 0 {
		return fmt.Errorf("unable to run imgbuild engine as non-root user")
	}

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
	starterConfig.SetSharedMount(true)

	return nil
}
