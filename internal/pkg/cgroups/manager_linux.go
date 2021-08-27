// Copyright (c) 2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cgroups

import (
	"encoding/json"
	"path/filepath"
	"strconv"

	"github.com/containerd/cgroups"
	"github.com/hpcng/singularity/pkg/sylog"
	"github.com/opencontainers/runtime-spec/specs-go"
)

// Manager is used to work with cgroups resource restrictions. It is an
// interface satisfied by different implementations for v1 and v2 cgroups.
type Manager interface {
	// GetVersion returns the version of the cgroups interface in use by
	// the manager.
	GetVersion() int
	// GetCgroupRootPath returns the path to the root of the cgroup on the
	// filesystem.
	GetCgroupRootPath() string
	// ApplyFromFile applies a cgroup configuration from a toml file, creating a
	// new group if necessary, and places the process with Manager.Pid into the
	// cgroup.
	ApplyFromFile(path string) error
	// ApplyFromSpec applies a cgroups configuration from an OCI LinuxResources
	// spec struct, creating a new group if necessary, and places the process
	// with Manager.Pid into the cgroup.
	ApplyFromSpec(spec *specs.LinuxResources) error
	// UpdateFromFile updates the existing managed cgroup using configuration
	// from a toml file.
	UpdateFromFile(path string) error
	// UpdateFromSpec updates the existing managed cgroup using configuration
	// from an OCI LinuxResources spec struct.
	UpdateFromSpec(spec *specs.LinuxResources) error
	// AddProc adds the process with specified pid to the managed cgroup
	AddProc(pid int) error
	// Remove deletes the managed cgroup.
	Remove() error
	// Pause freezes processes in the managed cgroup.
	Pause() error
	// Resume unfreezes process in the managed cgroup.
	Resume() error
}

// NewManagerFromFile creates a Manager, applies the configuration at specPath, and adds pid to the cgroup.
// If a group name is supplied, it will be used by the manager.
// If group = "" then "/singularity/<pid>" is used as a default.
func NewManagerFromFile(specPath string, pid int, group string) (manager Manager, err error) {
	if group == "" {
		group = filepath.Join("/singularity", strconv.Itoa(pid))
	}
	if cgroups.Mode() == cgroups.Unified {
		sylog.Debugf("Applying cgroups v2 configuration")
		mgrv2 := ManagerV2{pid: pid, group: group}
		return &mgrv2, mgrv2.ApplyFromFile(specPath)
	}

	sylog.Debugf("Applying cgroups v1 configuration")
	mgrv1 := ManagerV1{pid: pid, path: group}
	return &mgrv1, mgrv1.ApplyFromFile(specPath)
}

// NewManagerFromFile creates a Manager, applies the configuration in spec, and adds pid to the cgroup.
// If a group name is supplied, it will be used by the manager.
// If group = "" then "/singularity/<pid>" is used as a default.
func NewManagerFromSpec(spec *specs.LinuxResources, pid int, group string) (manager Manager, err error) {
	if group == "" {
		group = filepath.Join("/singularity", strconv.Itoa(pid))
	}

	if cgroups.Mode() == cgroups.Unified {
		sylog.Debugf("Applying cgroups v2 configuration")
		mgrv2 := ManagerV2{pid: pid, group: group}
		return &mgrv2, mgrv2.ApplyFromSpec(spec)
	}

	sylog.Debugf("Applying cgroups v1 configuration")
	mgrv1 := ManagerV1{pid: pid, path: group}
	return &mgrv1, mgrv1.ApplyFromSpec(spec)
}

// GetManager returns a Manager for the provided cgroup name/path.
func GetManager(group string) (manager Manager, err error) {
	if cgroups.Mode() == cgroups.Unified {
		sylog.Debugf("Fetching cgroups v2 configuration")
		mgrv2 := ManagerV2{group: group}
		if err := mgrv2.loadFromGroup(); err != nil {
			return nil, err
		}
		return &mgrv2, nil
	}

	sylog.Debugf("Fetching cgroups v1 configuration")
	mgrv1 := ManagerV1{path: group}
	if err := mgrv1.loadFromPath(); err != nil {
		return nil, err
	}
	return &mgrv1, nil
}

// GetManagerFromPid returns a Manager for the cgroup that pid is a member of.
func GetManagerFromPid(pid int) (manager Manager, err error) {
	if cgroups.Mode() == cgroups.Unified {
		sylog.Debugf("Fetching cgroups v2 configuration")
		mgrv2 := ManagerV2{pid: pid}
		if err := mgrv2.loadFromPid(); err != nil {
			return nil, err
		}
		return &mgrv2, nil
	}

	sylog.Debugf("Fetching cgroups v1 configuration")
	mgrv1 := ManagerV1{pid: pid}
	if err := mgrv1.loadFromPid(); err != nil {
		return nil, err
	}
	return &mgrv1, nil
}

// readSpecFromFile loads a TOML file containing a specs.LinuxResources cgroups configuration.
func readSpecFromFile(path string) (spec specs.LinuxResources, err error) {
	conf, err := LoadConfig(path)
	if err != nil {
		return
	}

	// convert TOML structures to OCI JSON structures
	data, err := json.Marshal(conf)
	if err != nil {
		return
	}

	if err = json.Unmarshal(data, &spec); err != nil {
		return
	}

	return
}
