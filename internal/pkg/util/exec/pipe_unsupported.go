// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build !linux

package exec

import (
	"fmt"
	"os/exec"
)

// Pipe execute a command with arguments and pass data over pipe
func Pipe(command string, args []string, env []string, data []byte) error {
	return fmt.Errorf("unsupported on this platform")
}

// PipeCommand creates an exec.Command struct which will execute the starter binary
func PipeCommand(command string, args []string, env []string, data []byte) (*exec.Cmd, error) {
	return nil, fmt.Errorf("unsupported on this platform")
}

// SetPipe sets the PIPE_EXEC_FD environment variable containing the JSON configuration data
func SetPipe(data []byte) (string, error) {
	return "", fmt.Errorf("unsupported on this platform")
}
