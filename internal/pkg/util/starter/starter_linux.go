// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package starter

import (
	"fmt"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"golang.org/x/sys/unix"
)

// sendData sets a socket communication channel between caller and starter
// binary in order to pass engine JSON configuration data to starter.
func sendData(data []byte) (int, error) {
	fd, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return -1, fmt.Errorf("failed to create socket communication pipe: %s", err)
	}

	if curSize, err := unix.GetsockoptInt(fd[0], unix.SOL_SOCKET, unix.SO_SNDBUF); err == nil {
		if curSize < 65536 {
			sylog.Warningf("current buffer size is %d, you may encounter some issues", curSize)
			sylog.Warningf("the minimum recommended value is 65536, you can adjust this value with:")
			sylog.Warningf("\"echo 65536 > /proc/sys/net/core/wmem_default\"")
		}
	} else {
		return -1, fmt.Errorf("failed to determine current socket buffer size: %s", err)
	}

	pipeFd, err := unix.Dup(fd[1])
	if err != nil {
		return -1, fmt.Errorf("failed to duplicate socket file descriptor: %s", err)
	}

	if n, err := unix.Write(fd[0], data); err != nil || n != len(data) {
		return -1, fmt.Errorf("failed to write data to socket: %s", err)
	}

	return pipeFd, nil
}
