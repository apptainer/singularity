// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package imgbuild

import (
	"fmt"
	"net"
	"syscall"

	"github.com/singularityware/singularity/src/pkg/util/capabilities"
	"github.com/singularityware/singularity/src/runtime/engines/common/config"
	"github.com/singularityware/singularity/src/runtime/engines/common/config/starter"
)

// EngineOperations implements the engines.EngineOperations interface for
// the image build process
type EngineOperations struct {
	CommonConfig *config.Common `json:"-"`
	EngineConfig *EngineConfig  `json:"engineConfig"`
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
func (e *EngineOperations) PrepareConfig(masterConn net.Conn, starterConfig *starter.Config) error {
	e.CommonConfig.OciConfig.SetProcessNoNewPrivileges(true)
	starterConfig.SetNoNewPrivs(e.CommonConfig.OciConfig.Process.NoNewPrivileges)

	if syscall.Getuid() != 0 {
		return fmt.Errorf("unable to run imgbuild engine as non-root user")
	}

	if starterConfig.GetIsSUID() {
		return fmt.Errorf("%s don't allow SUID workflow", e.CommonConfig.EngineName)
	}

	e.CommonConfig.OciConfig.SetupPrivileged(true)

	if e.CommonConfig.OciConfig.Linux != nil {
		starterConfig.SetNsFlagsFromSpec(e.CommonConfig.OciConfig.Linux.Namespaces)
	}
	if e.CommonConfig.OciConfig.Process != nil && e.CommonConfig.OciConfig.Process.Capabilities != nil {
		starterConfig.SetCapabilities(capabilities.Permitted, e.CommonConfig.OciConfig.Process.Capabilities.Permitted)
		starterConfig.SetCapabilities(capabilities.Effective, e.CommonConfig.OciConfig.Process.Capabilities.Effective)
		starterConfig.SetCapabilities(capabilities.Inheritable, e.CommonConfig.OciConfig.Process.Capabilities.Inheritable)
		starterConfig.SetCapabilities(capabilities.Bounding, e.CommonConfig.OciConfig.Process.Capabilities.Bounding)
		starterConfig.SetCapabilities(capabilities.Ambient, e.CommonConfig.OciConfig.Process.Capabilities.Ambient)
	}

	return nil
}
