// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package exec

import (
	"fmt"
	"os/exec"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/sylog"
)

// Pipe execute a command with arguments and pass data over pipe
func Pipe(command string, args []string, env []string, data []byte) error {
	pipeEnv, err := SetPipe(data)
	if err != nil {
		return err
	}

	env = append(env, pipeEnv)
	err = syscall.Exec(command, args, env)
	if err != nil {
		return fmt.Errorf("command %s execution failed: %s", command, err)
	}

	return nil
}

// PipeCommand creates an exec.Command struct which will execute the starter binary
func PipeCommand(command string, args []string, env []string, data []byte) (*exec.Cmd, error) {
	pipeEnv, err := SetPipe(data)
	if err != nil {
		return nil, err
	}

	env = append(env, pipeEnv)

	c := &exec.Cmd{
		Path: command,
		Args: args,
		Env:  env,
	}
	return c, nil
}

// setPipe sets a pipe communication channel for JSON configuration data and returns
// the file descriptor of the read side
func setPipe(data []byte) (int, error) {
	fd, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM|syscall.SOCK_CLOEXEC, 0)
	if err != nil {
		return -1, fmt.Errorf("failed to create communication pipe: %s", err)
	}

	if curSize, err := syscall.GetsockoptInt(fd[0], syscall.SOL_SOCKET, syscall.SO_SNDBUF); err == nil {
		if curSize <= 65536 {
			sylog.Warningf("current buffer size is %d, you may encounter some issues", curSize)
			sylog.Warningf("the minimum recommended value is 65536, you can adjust this value with:")
			sylog.Warningf("\"echo 65536 > /proc/sys/net/core/wmem_default\"")
		}
	} else {
		return -1, fmt.Errorf("failed to determine current pipe size: %s", err)
	}

	pipeFd, err := syscall.Dup(fd[1])
	if err != nil {
		return -1, fmt.Errorf("failed to duplicate pipe file descriptor: %s", err)
	}

	if n, err := syscall.Write(fd[0], data); err != nil || n != len(data) {
		return -1, fmt.Errorf("failed to write data to stdin: %s", err)
	}

	return pipeFd, err
}

// SetPipe sets the PIPE_EXEC_FD environment variable containing the JSON configuration data
func SetPipe(data []byte) (string, error) {
	pipeFd, err := setPipe(data)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("PIPE_EXEC_FD=%d", pipeFd), nil
}
