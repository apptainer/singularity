package tools

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
