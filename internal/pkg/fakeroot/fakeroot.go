// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package fakeroot

import (
	"fmt"
	"os"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/internal/pkg/util/user"
)

// GetIDRange returns the allocated ID range based on base ID
// and a list of allowed users
func GetIDRange(base uint64, allowedUsers []string) (*specs.LinuxIDMapping, error) {
	userinfo, err := user.GetPwUID(uint32(os.Getuid()))
	if err != nil {
		return nil, err
	}
	if base%65536 != 0 {
		return nil, fmt.Errorf("fakeroot base id is not a multiple of 65536")
	} else if base < 65536 || base > 4294901760 {
		return nil, fmt.Errorf("fakeroot base id is not set between 65536 and 4294901760")
	}

	// root user is always authorized and has a 1:1 mapping
	if userinfo.UID == 0 {
		return &specs.LinuxIDMapping{
			ContainerID: 1,
			HostID:      1,
			Size:        65535,
		}, nil
	}

	for i, name := range allowedUsers {
		if userinfo.Name == name {
			return &specs.LinuxIDMapping{
				ContainerID: 1,
				HostID:      uint32(base) + uint32(i*65536),
				Size:        65535,
			}, nil
		}
	}

	return nil, fmt.Errorf("you are not allowed to use fakeroot")
}
