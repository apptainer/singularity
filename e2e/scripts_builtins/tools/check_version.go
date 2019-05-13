package tools

import (
	"context"
	"fmt"

	"github.com/blang/semver"

	"github.com/sylabs/singularity/pkg/stest"
	"mvdan.cc/sh/v3/interp"
)

// check-version builtin
// usage:
// check-version <version> <range>
func checkVersion(ctx context.Context, mc interp.ModuleCtx, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("check-version requires a version to compare against a range")
	}
	v, err := semver.Parse(args[0])
	if err != nil {
		return fmt.Errorf("check-version: %s", err)
	}
	vrange, err := semver.ParseRange(args[1])
	if err != nil {
		return fmt.Errorf("check-version: %s", err)
	}
	if !vrange(v) {
		return interp.ExitStatus(1)
	}
	return interp.ExitStatus(0)
}

func init() {
	stest.RegisterCommandBuiltin("check-version", checkVersion)
}
