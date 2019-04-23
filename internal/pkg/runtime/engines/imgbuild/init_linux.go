// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build build_engine

package imgbuild

import (
	"github.com/sylabs/singularity/internal/pkg/runtime/engines"
	imgbuildConfig "github.com/sylabs/singularity/internal/pkg/runtime/engines/imgbuild/config"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/singularity/rpc/server"
)

// Init registers runtime engine, this method is called
// from cmd/starter/main_linux.go
func Init(name string) error {
	if name != imgbuildConfig.Name {
		return nil
	}
	eOp := &EngineOperations{EngineConfig: &imgbuildConfig.EngineConfig{}}
	if err := engines.RegisterEngineOperations(imgbuildConfig.Name, eOp); err != nil {
		return err
	}
	return engines.RegisterEngineRPCMethods(imgbuildConfig.Name, new(server.Methods))
}
