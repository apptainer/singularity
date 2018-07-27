// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package imgbuild

import (
	"net"
	"os"
	"syscall"

	"github.com/singularityware/singularity/src/pkg/build"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/runtime/engines/common/config"
)

// EngineOperations implements the engines.EngineOperations interface for
// the image build process
type EngineOperations struct {
	CommonConfig *config.Common      `json:"-"`
	EngineConfig *build.EngineConfig `json:"engineConfig"`
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
func (e *EngineOperations) PrepareConfig(masterConn net.Conn) error {
	e.CommonConfig.OciConfig.SetProcessNoNewPrivileges(true)

	if syscall.Getuid() != 0 {
		sylog.Fatalf("Unable to run imgbuild engine as non-root user\n")
		os.Exit(1)
	}

	e.CommonConfig.OciConfig.SetupPrivileged(true)
	return nil
}

// IsRunAsInstance returns false
func (e *EngineOperations) IsRunAsInstance() bool {
	return false
}

// IsAllowSUID always returns false to not allow SUID workflow
func (e *EngineOperations) IsAllowSUID() bool {
	return false
}
