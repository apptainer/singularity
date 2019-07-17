// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package fakeroot

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/internal/pkg/util/user"
)

const (
	// SubUIDFile is the path to /etc/subuid file
	SubUIDFile = "/etc/subuid"
	// SubGIDFile is the path to /etc/subgid file
	SubGIDFile = "/etc/subgid"
)

// GetIDRange determines UID/GID mappings based on configuration
// file provided in path.
func GetIDRange(path string, uid uint32) (*specs.LinuxIDMapping, error) {
	const validRangeCount = 65536
	var line int
	var entries []string

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open %s: %s", path, err)
	}
	defer f.Close()

	userinfo, err := user.GetPwUID(uid)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve user with UID %d: %s", uid, err)
	}
	uidStr := strconv.FormatUint(uint64(uid), 10)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line++
		splitted := strings.Split(scanner.Text(), ":")
		switch splitted[0] {
		case userinfo.Name, uidStr:
			size, err := strconv.ParseUint(splitted[2], 10, 32)
			if err != nil {
				return nil, fmt.Errorf("could not convert %s: %s", splitted[2], err)
			}
			// fakeroot requires a range count of 65536
			if size != validRangeCount {
				entries = append(entries, strconv.Itoa(line))
				continue
			}
			hostID, err := strconv.ParseUint(splitted[1], 10, 32)
			if err != nil {
				return nil, fmt.Errorf("could not convert %s: %s", splitted[1], err)
			}
			return &specs.LinuxIDMapping{
				ContainerID: 1,
				HostID:      uint32(hostID),
				Size:        uint32(size),
			}, nil
		}
	}
	if len(entries) > 0 {
		return nil, fmt.Errorf(
			"entry for user %s found in %s at line %s but all with a range count different from %d",
			userinfo.Name, f.Name(), strings.Join(entries, ", "), validRangeCount,
		)
	}
	return nil, fmt.Errorf("user %s not found in %s", userinfo.Name, f.Name())
}
