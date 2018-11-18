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

const (
	singularity = "/singularity"
)

// Manager manage container cgroup resources restriction
type Manager struct {
	ParentPath   string
	Name         string
	Pid          int
	parentCgroup cgroups.Cgroup
	childCgroup  cgroups.Cgroup
}

// GetCgroupRootPath returns cgroup root path
func (m *Manager) GetCgroupRootPath() string {
	path := ""

	if m.childCgroup == nil {
		return path
	}

	for _, sub := range m.childCgroup.Subsystems() {
		processes, err := m.childCgroup.Processes(sub.Name(), false)
		if len(processes) == 0 || err != nil {
			continue
		}
		process := processes[0]
		cgroupPath := strings.Split(process.Path, string(sub.Name()))[0]
		return filepath.Clean(cgroupPath)
	}

	return path
}

// ApplyFromSpec applies cgroups ressources restriction from OCI specification
func (m *Manager) ApplyFromSpec(spec *specs.LinuxResources) (err error) {
	var path cgroups.Path

	if m.ParentPath == "" {
		path = cgroups.StaticPath(singularity)
	} else {
		if !filepath.IsAbs(m.ParentPath) {
			return fmt.Errorf("parent path must be an absolute path")
		}
		path = cgroups.StaticPath(m.ParentPath)
	}

	// creates singularity group
	m.parentCgroup, err = cgroups.New(cgroups.V1, path, &specs.LinuxResources{})
	if err != nil {
		return err
	}

	s := spec
	if s == nil {
		s = &specs.LinuxResources{}
	}

	// creates subgroup in singularity group
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
	return m.childCgroup.Delete()
}
