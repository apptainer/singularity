// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"os"
	"syscall"
)

// PrestartProcess runs pre-start tasks
func (engine *EngineOperations) PrestartProcess() error {
	/* seccomp setup goes here */
	return nil
}

// StartProcess starts the process
func (engine *EngineOperations) StartProcess() error {
	os.Setenv("PS1", "shell> ")

	os.Chdir("/")

	args := engine.CommonConfig.OciConfig.Process.Args
	env := engine.CommonConfig.OciConfig.Process.Env

	err := syscall.Exec(args[0], args, env)
	if err != nil {
		return fmt.Errorf("exec %s failed: %s", args[0], err)
	}
	return nil
}
