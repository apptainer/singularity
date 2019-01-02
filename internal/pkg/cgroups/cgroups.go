// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cgroups

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/containerd/cgroups"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// Manager manage container cgroup resources restriction
type Manager struct {
	Path   string
	Pid    int
	cgroup cgroups.Cgroup
}

// GetCgroupRootPath returns cgroup root path
func (m *Manager) GetCgroupRootPath() string {
	if m.cgroup == nil {
		return ""
	}

	for _, sub := range m.cgroup.Subsystems() {
		processes, err := m.cgroup.Processes(sub.Name(), false)
		if len(processes) == 0 || err != nil {
			continue
		}
		process := processes[0]
		cgroupPath := strings.Split(process.Path, string(sub.Name()))[0]
		return filepath.Clean(cgroupPath)
	}

	return ""
}

// ApplyFromSpec applies cgroups ressources restriction from OCI specification
func (m *Manager) ApplyFromSpec(spec *specs.LinuxResources) (err error) {
	var path cgroups.Path

	if !filepath.IsAbs(m.Path) {
		return fmt.Errorf("cgroup path must be an absolute path")
	}

	path = cgroups.StaticPath(m.Path)

	s := spec
	if s == nil {
		s = &specs.LinuxResources{}
	}

	// creates cgroup
	m.cgroup, err = cgroups.New(cgroups.V1, path, s)
	if err != nil {
		return err
	}

	if err := m.cgroup.Add(cgroups.Process{Pid: m.Pid}); err != nil {
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

	// convert TOML structures to OCI JSON structures
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
	// deletes subgroup
	return m.cgroup.Delete()
}
