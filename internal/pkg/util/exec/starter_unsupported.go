// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build !linux

package exec

import (
	"fmt"
	"os/exec"

	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config"
)

// LookStarterPath returns the starter binary path and also returns
// if the path point to the setuid starter executable
func LookStarterPath(suid bool, allowedSuid bool) (string, bool) {
	return "", false
}

// Starter execute starter binary with arguments and pass data over pipe
func Starter(command string, args []string, env []string, config *config.Common) error {
	return fmt.Errorf("unsupported on this platform")
}

// StarterCommand creates an exec.Command struct which will execute the starter binary
func StarterCommand(command string, args []string, env []string, config *config.Common) (*exec.Cmd, error) {
	return nil, fmt.Errorf("unsupported on this platform")
}

// SetStarterPipe sets the PIPE_EXEC_FD environment variable containing the JSON configuration data
func SetStarterPipe(config *config.Common) (string, error) {
	return "", fmt.Errorf("unsupported on this platform")
}
