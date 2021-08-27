// Copyright (c) 2018-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cgroups

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/containerd/cgroups"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// ManagerV1 manages a cgroup 'Path', containing process 'Pid' for a v1 cgroups hierarchy.
type ManagerV1 struct {
	path   string
	pid    int
	cgroup cgroups.Cgroup
}

func (m *ManagerV1) GetVersion() int {
	return 1
}

func (m *ManagerV1) load() (err error) {
	if m.path != "" {
		return m.loadFromPath()
	}
	return m.loadFromPid()
}

func (m *ManagerV1) loadFromPid() (err error) {
	if m.pid == 0 {
		return fmt.Errorf("cannot load from pid - no process ID specified")
	}
	path := cgroups.PidPath(m.pid)
	m.cgroup, err = cgroups.Load(cgroups.V1, path)
	return err
}

func (m *ManagerV1) loadFromPath() (err error) {
	if m.path == "" {
		return fmt.Errorf("cannot load from path - no path specified")
	}
	path := cgroups.StaticPath(m.path)
	m.cgroup, err = cgroups.Load(cgroups.V1, path)
	return err
}

// GetCgroupRootPath returns the path to the root of the cgroup on the
// filesystem.
func (m *ManagerV1) GetCgroupRootPath() string {
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

// ApplyFromSpec applies a cgroups configuration from an OCI LinuxResources
// spec struct, creating a new group if necessary, and places the process
// with Manager.Pid into the cgroup.
func (m *ManagerV1) ApplyFromSpec(spec *specs.LinuxResources) (err error) {
	var path cgroups.Path

	if !filepath.IsAbs(m.path) {
		return fmt.Errorf("cgroup path must be an absolute path")
	}

	path = cgroups.StaticPath(m.path)

	s := spec
	if s == nil {
		s = &specs.LinuxResources{}
	}

	// creates cgroup
	m.cgroup, err = cgroups.New(cgroups.V1, path, s)
	if err != nil {
		return err
	}

	return m.cgroup.Add(cgroups.Process{Pid: m.pid})
}

// ApplyFromFile applies a cgroup configuration from a toml file, creating a
// new group if necessary, and places the process with Manager.Pid into the
// cgroup.
func (m *ManagerV1) ApplyFromFile(path string) error {
	spec, err := readSpecFromFile(path)
	if err != nil {
		return err
	}
	return m.ApplyFromSpec(&spec)
}

// UpdateFromSpec updates the existing managed cgroup using configuration
// from an OCI LinuxResources spec struct.
func (m *ManagerV1) UpdateFromSpec(spec *specs.LinuxResources) (err error) {
	if m.cgroup == nil {
		if err = m.load(); err != nil {
			return
		}
	}
	err = m.cgroup.Update(spec)
	return
}

// UpdateFromFile updates the existing managed cgroup using configuration
// from a toml file.
func (m *ManagerV1) UpdateFromFile(path string) error {
	spec, err := readSpecFromFile(path)
	if err != nil {
		return err
	}
	return m.UpdateFromSpec(&spec)
}

func (m *ManagerV1) AddProc(pid int) (err error) {
	if m.cgroup == nil {
		if err := m.load(); err != nil {
			return err
		}
	}
	return m.cgroup.Add(cgroups.Process{Pid: pid})
}

// Remove deletes the managed cgroup.
func (m *ManagerV1) Remove() error {
	// deletes subgroup
	return m.cgroup.Delete()
}

// Pause freezes processes in the managed cgroup.
func (m *ManagerV1) Pause() error {
	if m.cgroup == nil {
		if err := m.load(); err != nil {
			return err
		}
	}
	return m.cgroup.Freeze()
}

// Resume unfreezes process in the managed cgroup.
func (m *ManagerV1) Resume() error {
	if m.cgroup == nil {
		if err := m.load(); err != nil {
			return err
		}
	}
	return m.cgroup.Thaw()
}
