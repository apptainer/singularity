// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package command

import (
	"context"
	"fmt"

	uuid "github.com/satori/go.uuid"

	"github.com/sylabs/singularity/pkg/stest"
	"mvdan.cc/sh/v3/interp"
)

// uuidgen builtin
// usage:
// uuidgen
func uuidGen(ctx context.Context, mc interp.ModuleCtx, args []string) error {
	_, err := fmt.Fprintf(mc.Stdout, "%s\n", uuid.NewV4().String())
	return err
}

func init() {
	stest.RegisterCommandBuiltin("uuidgen", uuidGen)
}
