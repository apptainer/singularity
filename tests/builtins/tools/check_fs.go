package tools

import (
	"context"
	"fmt"

	"github.com/sylabs/singularity/pkg/util/fs/proc"

	"github.com/sylabs/singularity/pkg/stest"
	"mvdan.cc/sh/v3/interp"
)

// check-fs builtin
// usage:
// check-fs <filesystem>
func checkFs(ctx context.Context, mc interp.ModuleCtx, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("check-fs requires a filesystem argument")
	}
	has, err := proc.HasFilesystem(args[0])
	if err != nil || !has {
		return interp.ExitStatus(1)
	}
	return interp.ExitStatus(0)
}

func init() {
	stest.RegisterCommandBuiltin("check-fs", checkFs)
}
