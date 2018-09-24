// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cgroups

import (
	"encoding/json"

	"github.com/containerd/cgroups"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

const singularity = "/singularity"

// Manager manage container cgroup resources restriction
type Manager struct {
	Name         string
	Pid          int
	parentCgroup cgroups.Cgroup
	childCgroup  cgroups.Cgroup
}

// ApplyFromSpec applies cgroups ressources restriction from OCI specification
func (m *Manager) ApplyFromSpec(spec *specs.LinuxResources) (err error) {
	path := cgroups.StaticPath(singularity)

	_, err = cgroups.New(cgroups.V1, path, &specs.LinuxResources{})
	if err != nil {
		return err
	}

	m.parentCgroup, err = cgroups.Load(cgroups.V1, path)
	if err != nil {
		return err
	}

	s := spec
	if s == nil {
		s = &specs.LinuxResources{}
	}

	m.childCgroup, err = m.parentCgroup.New(m.Name, s)
	if err != nil {
		return err
	}

	if err := m.childCgroup.Add(cgroups.Process{Pid: m.Pid}); err != nil {
		return err
	}

	return
}

// ApplyFromFile applies cgroups ressources restriction from TOML configuration
// file
func (m *Manager) ApplyFromFile(path string) error {
	var spec specs.LinuxResources

	conf, err := LoadConfig(path)
	if err != nil {
		return err
	}

	data, err := json.Marshal(conf)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &spec); err != nil {
		return err
	}

	return m.ApplyFromSpec(&spec)
}

// Remove removes ressources restriction for current managed process
func (m *Manager) Remove() error {
	m.parentCgroup.Add(cgroups.Process{Pid: m.Pid})
	return m.childCgroup.Delete()
}
