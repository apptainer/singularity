/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package exec

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// Pipe execute a command with arguments and pass data over stdin
func Pipe(command string, args []string, env []string, data []byte) error {
	r, w, err := os.Pipe()

	if err != nil {
		return fmt.Errorf("failed to create pipe: %s", err)
	}

	if err := syscall.Close(0); err != nil {
		return fmt.Errorf("failed to close stdin: %s", err)
	}

	rfd := r.Fd()
	wfd := w.Fd()

	if err := syscall.Dup2(*(*int)(unsafe.Pointer(&rfd)), 0); err != nil {
		return fmt.Errorf("failed to duplicate pipe file descriptor: %s", err)
	}

	if n, err := syscall.Write(*(*int)(unsafe.Pointer(&wfd)), data); err != nil || n != len(data) {
		return fmt.Errorf("failed to write data to stdin: %s", err)
	}

	if _, _, err := syscall.Syscall(syscall.SYS_FCNTL, rfd, syscall.F_SETFD, syscall.FD_CLOEXEC); err != 0 {
		return fmt.Errorf("failed to set close-on-exec on read pipe: %s", err.Error())
	}

	if _, _, err := syscall.Syscall(syscall.SYS_FCNTL, wfd, syscall.F_SETFD, syscall.FD_CLOEXEC); err != 0 {
		return fmt.Errorf("failed to set close-on-exec on write pipe: %s", err.Error())
	}

	err = syscall.Exec(command, args, env)
	if err != nil {
		return fmt.Errorf("command %s execution failed: %s", command, err)
	}

	return nil
}
