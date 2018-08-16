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
func (e *EngineOperations) PrepareEngineConfig(masterConn net.Conn) error {
	e.CommonConfig.OciConfig.SetProcessNoNewPrivileges(true)
	e.CommonConfig.OciConfig.SetupPrivileged(true)

	if syscall.Getuid() != 0 {
		return fmt.Errorf("unable to run imgbuild engine as non-root user")
	}

	return nil
}

// PrepareStartupConfig ...
func (e *EngineOperations) PrepareStartupConfig(startupConfig *config.Startup) error {
	startupConfig.SetNoNewPrivs(e.CommonConfig.OciConfig.Process.NoNewPrivileges)

	if startupConfig.GetIsSUID() {
		return fmt.Errorf("%s don't allow SUID workflow", e.CommonConfig.EngineName)
	}

	if e.CommonConfig.OciConfig.Linux != nil {
		startupConfig.SetNsFlagsFromSpec(e.CommonConfig.OciConfig.Linux.Namespaces)
	}
	if e.CommonConfig.OciConfig.Process != nil && e.CommonConfig.OciConfig.Process.Capabilities != nil {
		startupConfig.SetCapabilities(capabilities.Permitted, e.CommonConfig.OciConfig.Process.Capabilities.Permitted)
		startupConfig.SetCapabilities(capabilities.Effective, e.CommonConfig.OciConfig.Process.Capabilities.Effective)
		startupConfig.SetCapabilities(capabilities.Inheritable, e.CommonConfig.OciConfig.Process.Capabilities.Inheritable)
		startupConfig.SetCapabilities(capabilities.Bounding, e.CommonConfig.OciConfig.Process.Capabilities.Bounding)
		startupConfig.SetCapabilities(capabilities.Ambient, e.CommonConfig.OciConfig.Process.Capabilities.Ambient)
	}

	return nil
}
