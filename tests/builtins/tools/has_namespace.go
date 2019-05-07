package tools

import (
	"context"
	"fmt"

	uuid "github.com/satori/go.uuid"

	"github.com/sylabs/singularity/pkg/stest"
	"mvdan.cc/sh/v3/interp"
)

// has-namespace builtin
// usage:
// has-namespace
func hasNamespace(ctx context.Context, mc interp.ModuleCtx, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("has-namespace requires as argument: user, user-priv, mount, net, uts, ipc, pid or cgroup")
	}
	_, err := fmt.Fprintf(mc.Stdout, "%s\n", uuid.NewV4().String())
	return err
}

func init() {
	stest.RegisterCommandBuiltin("has-namespace", hasNamespace)
}
