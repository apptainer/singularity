// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"os"
	"runtime"
	"syscall"

	"github.com/singularityware/singularity/src/pkg/security"
)

// StartProcess starts the process
func (engine *EngineOperations) StartProcess() error {
	os.Setenv("PS1", "shell> ")

	os.Chdir("/")

	// lock the current thread to apply security configuration
	runtime.LockOSThread()

	if err := security.Configure(&engine.CommonConfig.OciConfig.Spec); err != nil {
		return fmt.Errorf("failed to apply security configuration: %s", err)
	}

	args := engine.CommonConfig.OciConfig.Process.Args
	env := engine.CommonConfig.OciConfig.Process.Env

	err := syscall.Exec(args[0], args, env)
	if err != nil {
		return fmt.Errorf("exec %s failed: %s", args[0], err)
	}
	return nil
}
