// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build oci_engine

package oci

import (
	"github.com/sylabs/singularity/internal/pkg/runtime/engines"
	ociserver "github.com/sylabs/singularity/internal/pkg/runtime/engines/oci/rpc/server"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/singularity/rpc/server"
)

// Init registers runtime engine, this method is called
// from cmd/starter/main_linux.go
func Init(name string) error {
	if name != Name {
		return nil
	}
	eOp := &EngineOperations{EngineConfig: &EngineConfig{}}
	if err := engines.RegisterEngineOperations(Name, eOp); err != nil {
		return err
	}
	ocimethods := new(ociserver.Methods)
	ocimethods.Methods = new(server.Methods)
	return engines.RegisterEngineRPCMethods(Name, ocimethods)
}
