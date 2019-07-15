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
	const (
		// rangeSize corresponds to the size of UID/GID allocated for each users
		rangeSize = uint64(65536)
		// minBase is the minimal base ID allocation authorized by configuration
		minBase = uint64(131072)
		// maxBase is the maximal base ID allocation authorized by configuration
		maxBase = uint64(4294901760)
	)
	userinfo, err := user.GetPwUID(uint32(os.Getuid()))
	if err != nil {
		return nil, err
	}
	if base%rangeSize != 0 {
		return nil, fmt.Errorf("fakeroot base id is not a multiple of %d", rangeSize)
	} else if base < minBase || base > maxBase {
		return nil, fmt.Errorf("fakeroot base id is not set between %d and %d", minBase, maxBase)
	}

	// root user is always authorized and has a 1:1 mapping
	if userinfo.UID == 0 {
		return &specs.LinuxIDMapping{
			ContainerID: 1,
			HostID:      1,
			Size:        uint32(rangeSize),
		}, nil
	}

	for i, name := range allowedUsers {
		if userinfo.Name == name {
			return &specs.LinuxIDMapping{
				ContainerID: 1,
				HostID:      uint32(base) + uint32(i*int(rangeSize)),
				Size:        uint32(rangeSize),
			}, nil
		}
	}

	msg := "you are not allowed to use fakeroot as you are not listed in 'fakeroot allowed users' in singularity.conf"
	return nil, fmt.Errorf(msg)
}
