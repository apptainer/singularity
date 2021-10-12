// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package unix

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
)

// Listen wraps net.Listen to handle 108 characters issue
func Listen(path string) (net.Listener, error) {
	socket := path

	if len(path) >= 108 {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current working directory: %s", err)
		}
		defer os.Chdir(cwd)

		dir := filepath.Dir(path)
		socket = filepath.Base(path)

		if err := os.Chdir(dir); err != nil {
			return nil, fmt.Errorf("failed to go into %s: %s", dir, err)
		}
	}

	return net.Listen("unix", socket)
}

// Dial wraps net.Dial to handle 108 characters issue
func Dial(path string) (net.Conn, error) {
	socket := path

	if len(path) >= 108 {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current working directory: %s", err)
		}
		defer os.Chdir(cwd)

		dir := filepath.Dir(path)
		socket = filepath.Base(path)

		if err := os.Chdir(dir); err != nil {
			return nil, fmt.Errorf("failed to go into %s: %s", dir, err)
		}
	}

	return net.Dial("unix", socket)
}

// CreateSocket creates an unix socket and returns connection listener.
func CreateSocket(path string) (net.Listener, error) {
	oldmask := syscall.Umask(0o177)
	defer syscall.Umask(oldmask)
	return Listen(path)
}

// WriteSocket writes data over unix socket
func WriteSocket(path string, data []byte) error {
	c, err := Dial(path)
	if err != nil {
		return fmt.Errorf("failed to connect to %s socket: %s", path, err)
	}
	defer c.Close()

	if _, err := c.Write(data); err != nil {
		return fmt.Errorf("failed to send data over socket: %s", err)
	}

	return nil
}
