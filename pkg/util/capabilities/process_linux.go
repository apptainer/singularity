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
func getProcessCapabilities() ([2]unix.CapUserData, error) {
	var data [2]unix.CapUserData
	var header unix.CapUserHeader

	header.Version = unix.LINUX_CAPABILITY_VERSION_3

	if err := unix.Capget(&header, &data[0]); err != nil {
		return data, fmt.Errorf("while getting capability: %s", err)
	}

	return data, nil
}

// GetProcessEffective returns effective capabilities for
// the current process.
func GetProcessEffective() (uint64, error) {
	data, err := getProcessCapabilities()
	if err != nil {
		return 0, err
	}
	return uint64(data[0].Effective) | uint64(data[1].Effective)<<32, nil
}

// GetProcessPermitted returns permitted capabilities for
// the current process.
func GetProcessPermitted() (uint64, error) {
	data, err := getProcessCapabilities()
	if err != nil {
		return 0, err
	}
	return uint64(data[0].Permitted) | uint64(data[1].Permitted)<<32, nil
}

// GetProcessInheritable returns inheritable capabilities for
// the current process.
func GetProcessInheritable() (uint64, error) {
	data, err := getProcessCapabilities()
	if err != nil {
		return 0, err
	}
	return uint64(data[0].Inheritable) | uint64(data[1].Inheritable)<<32, nil
}

// SetProcessEffective set effective capabilities for the
// the current process and returns previous effective set.
func SetProcessEffective(caps uint64) (uint64, error) {
	var data [2]unix.CapUserData
	var header unix.CapUserHeader

	header.Version = unix.LINUX_CAPABILITY_VERSION_3

	data, err := getProcessCapabilities()
	if err != nil {
		return 0, err
	}

	oldEffective := uint64(data[0].Effective) | uint64(data[1].Effective)<<32

	data[0].Effective = uint32(caps)
	data[1].Effective = uint32(caps >> 32)

	effective := uint64(data[0].Effective) | uint64(data[1].Effective)<<32
	permitted := uint64(data[0].Permitted) | uint64(data[1].Permitted)<<32

	for i := 0; i <= len(Map); i++ {
		if effective&uint64(1<<i) != 0 {
			if permitted&uint64(1<<i) != 0 {
				continue
			}
			strCap := "UNKNOWN"
			for _, cap := range Map {
				if uint(i) == cap.Value {
					strCap = cap.Name
					break
				}
			}
			err := fmt.Sprintf("%s is not in the permitted capability set", strCap)
			return 0, fmt.Errorf("while setting effective capabilities: %s", err)
		}
	}

	if err := unix.Capset(&header, &data[0]); err != nil {
		return 0, fmt.Errorf("while setting effective capabilities: %s", err)
	}

	return oldEffective, nil
}
