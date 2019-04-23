// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build singularity_engine

package singularity

import (
	"github.com/sylabs/singularity/internal/pkg/runtime/engines"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/singularity/rpc/server"
	singularityConfig "github.com/sylabs/singularity/pkg/runtime/engines/singularity/config"
)

// Init registers runtime engine, this method is called
// from cmd/starter/main_linux.go
func Init(name string) error {
	if name != singularityConfig.Name {
		return nil
	}
	eOp := &EngineOperations{EngineConfig: singularityConfig.NewConfig()}
	if err := engines.RegisterEngineOperations(singularityConfig.Name, eOp); err != nil {
		return err
	}
	return engines.RegisterEngineRPCMethods(singularityConfig.Name, new(server.Methods))
}
