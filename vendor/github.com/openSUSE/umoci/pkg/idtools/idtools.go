/*
 * umoci: Umoci Modifies Open Containers' Images
 * Copyright (C) 2016, 2017, 2018 SUSE LLC.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package idtools

import (
	"strconv"
	"strings"

	rspec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
)

// ToHost translates a remapped container ID to an unmapped host ID using the
// provided ID mapping. If no mapping is provided, then the mapping is a no-op.
// If there is no mapping for the given ID an error is returned.
func ToHost(contID int, idMap []rspec.LinuxIDMapping) (int, error) {
	if idMap == nil {
		return contID, nil
	}

	for _, m := range idMap {
		if uint32(contID) >= m.ContainerID && uint32(contID) < m.ContainerID+m.Size {
			return int(m.HostID + (uint32(contID) - m.ContainerID)), nil
		}
	}

	return -1, errors.Errorf("container id %d cannot be mapped to a host id", contID)
}

// ToContainer takes an unmapped host ID and translates it to a remapped
// container ID using the provided ID mapping. If no mapping is provided, then
// the mapping is a no-op. If there is no mapping for the given ID an error is
// returned.
func ToContainer(hostID int, idMap []rspec.LinuxIDMapping) (int, error) {
	if idMap == nil {
		return hostID, nil
	}

	for _, m := range idMap {
		if uint32(hostID) >= m.HostID && uint32(hostID) < m.HostID+m.Size {
			return int(m.ContainerID + (uint32(hostID) - m.HostID)), nil
		}
	}

	return -1, errors.Errorf("host id %d cannot be mapped to a container id", hostID)
}

// ParseMapping takes a mapping string of the form "container:host[:size]" and
// returns the corresponding rspec.LinuxIDMapping. An error is returned if not
// enough fields are provided or are otherwise invalid. The default size is 1.
func ParseMapping(spec string) (rspec.LinuxIDMapping, error) {
	parts := strings.Split(spec, ":")

	var err error
	var hostID, contID, size int
	switch len(parts) {
	case 3:
		size, err = strconv.Atoi(parts[2])
		if err != nil {
			return rspec.LinuxIDMapping{}, errors.Wrap(err, "invalid size in mapping")
		}
	case 2:
		size = 1
	default:
		return rspec.LinuxIDMapping{}, errors.Errorf("invalid number of fields in mapping '%s': %d", spec, len(parts))
	}

	contID, err = strconv.Atoi(parts[0])
	if err != nil {
		return rspec.LinuxIDMapping{}, errors.Wrap(err, "invalid containerID in mapping")
	}

	hostID, err = strconv.Atoi(parts[1])
	if err != nil {
		return rspec.LinuxIDMapping{}, errors.Wrap(err, "invalid hostID in mapping")
	}

	return rspec.LinuxIDMapping{
		HostID:      uint32(hostID),
		ContainerID: uint32(contID),
		Size:        uint32(size),
	}, nil
}
