// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package capabilities

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// getProcessCapabilities returns capabilities either effective,
// permitted or inheritable for the current process.
func getProcessCapabilities(capType string) (uint64, error) {
	var caps uint64
	var data [2]unix.CapUserData
	var header unix.CapUserHeader

	header.Version = unix.LINUX_CAPABILITY_VERSION_3

	if err := unix.Capget(&header, &data[0]); err != nil {
		return caps, fmt.Errorf("while getting capability: %s", err)
	}

	switch capType {
	case Effective:
		caps = uint64(data[0].Effective)
		caps |= uint64(data[1].Effective) << 32
	case Permitted:
		caps = uint64(data[0].Permitted)
		caps |= uint64(data[1].Permitted) << 32
	case Inheritable:
		caps = uint64(data[0].Inheritable)
		caps |= uint64(data[1].Inheritable) << 32
	}

	return caps, nil
}

// GetProcessEffective returns effective capabilities for
// the current process.
func GetProcessEffective() (uint64, error) {
	return getProcessCapabilities(Effective)
}

// GetProcessPermitted returns permitted capabilities for
// the current process.
func GetProcessPermitted() (uint64, error) {
	return getProcessCapabilities(Permitted)
}

// GetProcessInheritable returns inheritable capabilities for
// the current process.
func GetProcessInheritable() (uint64, error) {
	return getProcessCapabilities(Inheritable)
}
