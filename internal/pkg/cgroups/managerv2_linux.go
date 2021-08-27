// Copyright (c) 2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cgroups

import (
	"fmt"
	"path"

	cgroupsv2 "github.com/containerd/cgroups/v2"
	"github.com/hpcng/singularity/pkg/sylog"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

const mountPoint = "/sys/fs/cgroup"

// ManagerV2 manages a cgroup 'Group', containing process 'Pid' for a v2  unified cgroups hierarchy.
type ManagerV2 struct {
	group  string
	pid    int
	cgroup *cgroupsv2.Manager
}

func (m *ManagerV2) load() (err error) {
	if m.group != "" {
		return m.loadFromGroup()
	}
	return m.loadFromPid()
}

func (m *ManagerV2) loadFromPid() (err error) {
	if m.pid == 0 {
		return fmt.Errorf("cannot load from pid - no process ID specified")
	}
	group, err := cgroupsv2.PidGroupPath(m.pid)
	if err != nil {
		return fmt.Errorf("could not find group for pid %d: %v", m.pid, err)
	}
	m.cgroup, err = cgroupsv2.LoadManager(mountPoint, group)
	return err
}

func (m *ManagerV2) loadFromGroup() (err error) {
	if m.group == "" {
		return fmt.Errorf("cannot load from group - no group specified")
	}
	m.cgroup, err = cgroupsv2.LoadManager(mountPoint, m.group)
	return err
}

func (m *ManagerV2) GetVersion() int {
	return 2
}

// GetCgroupRootPath returns cgroup root path
func (m *ManagerV2) GetCgroupRootPath() string {
	if m.group == "" {
		return ""
	}
	return path.Join(mountPoint, m.group)
}

// ApplyFromSpec applies a cgroups configuration from an OCI LinuxResources spec
// struct, creating a new group if necessary, and places the process with
// Manager.Pid into the cgroup. The `Unified` key for native v2 cgroup
// specifications is not yet supported.
func (m *ManagerV2) ApplyFromSpec(spec *specs.LinuxResources) (err error) {
	if len(spec.Unified) > 0 {
		sylog.Warningf("Unified cgroup resource specifications are not supported, and will not be applied.")
	}
	if m.group == "" {
		return fmt.Errorf("group must be specified when creating a cgroup")
	}
	if m.pid == 0 {
		return fmt.Errorf("pid must be specified when creating a cgroup")
	}

	s := spec
	if s == nil {
		s = &specs.LinuxResources{}
	}

	// translate the LinuxResources cgroups v1 / OCI spec to v2 Resources
	res := cgroupsv2.ToResources(s)
	// v1 device restrictions have to manually be brought across into the v2
	// Resources struct, as ToResources(s) doesn't do this. They will then be
	// converted to ebpf programs and attached when the cgroup is created.
	res.Devices = v2FixDevices(s.Devices)

	// creates cgroup
	m.cgroup, err = cgroupsv2.NewManager(mountPoint, m.group, res)
	if err != nil {
		return err
	}

	return m.cgroup.AddProc(uint64(m.pid))
}

// ApplyFromFile applies a cgroup configuration from a toml file, creating a new
// group if necessary, and places the process with Manager.Pid into the cgroup.
// The `Unified` key for native v2 cgroup specifications is not yet supported.
func (m *ManagerV2) ApplyFromFile(path string) error {
	spec, err := readSpecFromFile(path)
	if err != nil {
		return err
	}
	return m.ApplyFromSpec(&spec)
}

// UpdateFromSpec updates the existing managed cgroup using configuration from
// an OCI LinuxResources spec struct. The `Unified` key for native v2 cgroup
// specifications is not yet supported.
func (m *ManagerV2) UpdateFromSpec(spec *specs.LinuxResources) (err error) {
	if len(spec.Unified) > 0 {
		sylog.Warningf("Unified cgroup resource specifications are not supported, and will not be applied.")
	}
	if m.group == "" {
		if m.pid == 0 {
			return fmt.Errorf("pid must be provided if group is not known")
		}
		m.group, err = cgroupsv2.PidGroupPath(m.pid)
		if err != nil {
			return fmt.Errorf("could not find group for pid %d: %v", m.pid, err)
		}
	}

	s := spec
	if s == nil {
		s = &specs.LinuxResources{}
	}

	// translate the LinuxResources cgroupsv1 / OCI spec to v2 Resources
	res := cgroupsv2.ToResources(s)
	// v1 device restrictions have to manually be brought across into the v2 Resources struct,
	// as ToResources doesn't do this. They will then be converted to ebpf programs and attached.
	res.Devices = v2FixDevices(s.Devices)

	// updates existing cgroup
	m.cgroup, err = cgroupsv2.NewManager(mountPoint, m.group, res)
	if err != nil {
		return err
	}

	return err
}

// UpdateFromFile updates the existing managed cgroup using configuration
// from a toml file.
func (m *ManagerV2) UpdateFromFile(path string) error {
	spec, err := readSpecFromFile(path)
	if err != nil {
		return err
	}
	return m.UpdateFromSpec(&spec)
}

// Remove deletes the managed cgroup.
func (m *ManagerV2) Remove() (err error) {
	// deletes subgroup
	return m.cgroup.Delete()
}

func (m *ManagerV2) AddProc(pid int) (err error) {
	if m.cgroup == nil {
		if err := m.load(); err != nil {
			return err
		}
	}
	return m.cgroup.AddProc(uint64(pid))
}

// Pause freezes processes in the managed cgroup.
func (m *ManagerV2) Pause() (err error) {
	if m.cgroup == nil {
		if err := m.load(); err != nil {
			return err
		}
	}
	return m.cgroup.Freeze()
}

// Resume unfreezes process in the managed cgroup.
func (m *ManagerV2) Resume() (err error) {
	if m.cgroup == nil {
		if err := m.load(); err != nil {
			return err
		}
	}
	return m.cgroup.Thaw()
}

// v2FixDevices modifies device entries to use an explicit, rather than implied
// wildcard.
//
// containerd/cgroups v1 device handling accepts:
//    "" for type, which is replaced as "a"
//    nil for major/minor, which is replaced as -1
//
// containerd/cgroups v2 will not handle the "" and nil, and the explicit
// wildcard is needed.
func v2FixDevices(devs []specs.LinuxDeviceCgroup) []specs.LinuxDeviceCgroup {
	for i, d := range devs {
		if d.Type == "" {
			d.Type = "a"
		}
		if d.Major == nil {
			d.Major = wildcard
		}
		if d.Minor == nil {
			d.Minor = wildcard
		}
		devs[i] = d
	}
	return devs
}
