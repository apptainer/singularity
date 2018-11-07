// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package unix

import (
	"fmt"
	"net"
)

// CreateSocket creates an unix socket and returns connection listener.
func CreateSocket(path string) (net.Listener, error) {
	return net.Listen("unix", path)
}

// WriteSocket writes data over unix socket
func WriteSocket(path string, data []byte) error {
	c, err := net.Dial("unix", path)

	if err != nil {
		return fmt.Errorf("failed to connect to %s socket: %s", path, err)
	}
	defer c.Close()

	if _, err := c.Write(data); err != nil {
		return fmt.Errorf("failed to send data over socket: %s", err)
	}

	return nil
}
