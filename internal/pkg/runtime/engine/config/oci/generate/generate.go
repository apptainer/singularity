// Copyright 2015 The Linux Foundation.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.
//
// This file contains modified code originally taken from:
// github.com/opencontainers/runtime-tools/generate/config.go

package generate

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/pkg/util/capabilities"
	"golang.org/x/sys/unix"
)

// Generator represents a generator for a OCI runtime config.
type Generator struct {
	Config *specs.Spec
}

// New returns a generator associated to the OCI specification
// passed in parameter or an empty OCI specification if parameter
// is nil.
func New(spec *specs.Spec) *Generator {
	if spec == nil {
		return &Generator{
			Config: &specs.Spec{
				Version: specs.Version,
			},
		}
	}
	return &Generator{
		Config: spec,
	}
}

func (g *Generator) initLinux() {
	if g.Config.Linux != nil {
		return
	}
	g.Config.Linux = &specs.Linux{}
}

func (g *Generator) initProcess() {
	if g.Config.Process != nil {
		return
	}
	g.Config.Process = &specs.Process{}
}

func (g *Generator) initProcessCapabilities() {
	g.initProcess()
	if g.Config.Process.Capabilities != nil {
		return
	}
	g.Config.Process.Capabilities = &specs.LinuxCapabilities{}
}

func (g *Generator) initRoot() {
	if g.Config.Root != nil {
		return
	}
	g.Config.Root = &specs.Root{}
}

func (g *Generator) initLinuxNamespaces() {
	g.initLinux()
	if g.Config.Linux.Namespaces != nil {
		return
	}
	g.Config.Linux.Namespaces = make([]specs.LinuxNamespace, 0)
}

// AddProcessEnv adds or replaces a container process environment variable.
func (g *Generator) AddProcessEnv(env, value string) {
	g.initProcess()

	kenv := fmt.Sprintf("%s=", env)
	l := len(kenv)

	for i, e := range g.Config.Process.Env {
		if len(e) >= l && e[:l] == kenv {
			g.Config.Process.Env[i] = kenv + value
			return
		}
	}

	g.Config.Process.Env = append(g.Config.Process.Env, kenv+value)
}

// RemoveProcessEnv removes a container process environment variable.
func (g *Generator) RemoveProcessEnv(env string) {
	g.initProcess()

	kenv := fmt.Sprintf("%s=", env)
	l := len(kenv)

	for i, e := range g.Config.Process.Env {
		if len(e) >= l && e[:l] == kenv {
			g.Config.Process.Env = append(g.Config.Process.Env[:i], g.Config.Process.Env[i+1:]...)
			return
		}
	}
}

// AddOrReplaceLinuxNamespace adds or updates a container process namespace.
func (g *Generator) AddOrReplaceLinuxNamespace(ns specs.LinuxNamespaceType, path string) {
	switch ns {
	case specs.NetworkNamespace:
	case specs.MountNamespace:
	case specs.UTSNamespace:
	case specs.UserNamespace:
	case specs.CgroupNamespace:
	case specs.IPCNamespace:
	case specs.PIDNamespace:
	default:
		return
	}

	g.initLinuxNamespaces()

	namespace := specs.LinuxNamespace{
		Type: ns,
		Path: path,
	}

	for i, n := range g.Config.Linux.Namespaces {
		if n.Type == ns {
			g.Config.Linux.Namespaces[i] = namespace
			return
		}
	}

	g.Config.Linux.Namespaces = append(g.Config.Linux.Namespaces, namespace)
}

// SetProcessArgs sets container process arguments.
func (g *Generator) SetProcessArgs(args []string) {
	g.initProcess()
	g.Config.Process.Args = args
}

// SetProcessCwd sets container process working directory.
func (g *Generator) SetProcessCwd(cwd string) {
	g.initProcess()
	g.Config.Process.Cwd = cwd
}

// SetProcessTerminal sets if container process terminal or not.
func (g *Generator) SetProcessTerminal(b bool) {
	g.initProcess()
	g.Config.Process.Terminal = b
}

// SetProcessCwd sets container root filesystem path.
func (g *Generator) SetRootPath(path string) {
	g.initRoot()
	g.Config.Root.Path = path
}

// AddMount adds a mount for container environment setup.
func (g *Generator) AddMount(mnt specs.Mount) {
	g.Config.Mounts = append(g.Config.Mounts, mnt)
}

// AddLinuxUIDMapping adds a UID mapping.
func (g *Generator) AddLinuxUIDMapping(host, container, size uint32) {
	g.initLinux()

	idMapping := specs.LinuxIDMapping{
		HostID:      host,
		ContainerID: container,
		Size:        size,
	}

	g.Config.Linux.UIDMappings = append(g.Config.Linux.UIDMappings, idMapping)
}

// AddLinuxGIDMapping adds a GID mapping.
func (g *Generator) AddLinuxGIDMapping(host, container, size uint32) {
	g.initLinux()

	idMapping := specs.LinuxIDMapping{
		HostID:      host,
		ContainerID: container,
		Size:        size,
	}

	g.Config.Linux.GIDMappings = append(g.Config.Linux.GIDMappings, idMapping)
}

// AddProcessRlimits adds a container process rlimit.
func (g *Generator) AddProcessRlimits(rType string, rHard uint64, rSoft uint64) {
	g.initProcess()

	for i, rlimit := range g.Config.Process.Rlimits {
		if rlimit.Type == rType {
			g.Config.Process.Rlimits[i].Hard = rHard
			g.Config.Process.Rlimits[i].Soft = rSoft
			return
		}
	}

	newRlimit := specs.POSIXRlimit{
		Type: rType,
		Hard: rHard,
		Soft: rSoft,
	}

	g.Config.Process.Rlimits = append(g.Config.Process.Rlimits, newRlimit)
}

// SetupPrivileged sets requirements for a container process with all
// privileges.
func (g *Generator) SetupPrivileged(privileged bool) {
	if !privileged {
		return
	}

	// Add all capabilities, we don't need to check for the
	// latest capability available as it's handled automatically
	// by the starter
	var allCapability []string
	for capStr := range capabilities.Map {
		allCapability = append(allCapability, capStr)
	}

	g.initLinux()
	g.initProcessCapabilities()

	g.Config.Process.Capabilities.Bounding = allCapability
	g.Config.Process.Capabilities.Effective = allCapability
	g.Config.Process.Capabilities.Inheritable = allCapability
	g.Config.Process.Capabilities.Permitted = allCapability
	g.Config.Process.Capabilities.Ambient = allCapability

	g.Config.Process.SelinuxLabel = ""
	g.Config.Process.ApparmorProfile = ""
	g.Config.Linux.Seccomp = nil
}

// SetProcessNoNewPrivileges sets g.Config.Process.NoNewPrivileges.
func (g *Generator) SetProcessNoNewPrivileges(b bool) {
	g.initProcess()
	g.Config.Process.NoNewPrivileges = b
}

// SetProcessSelinuxLabel sets container process SELinux execution label.
func (g *Generator) SetProcessSelinuxLabel(label string) {
	g.initProcess()
	g.Config.Process.SelinuxLabel = label
}

// SetProcessApparmorProfile sets container process AppArmor profile.
func (g *Generator) SetProcessApparmorProfile(prof string) {
	g.initProcess()
	g.Config.Process.ApparmorProfile = prof
}

// Save writes the configuration into w.
func (g *Generator) Save(w io.Writer) (err error) {
	var data []byte

	if g.Config.Linux != nil {
		buf, err := json.Marshal(g.Config.Linux)
		if err != nil {
			return err
		}
		if string(buf) == "{}" {
			g.Config.Linux = nil
		}
	}

	data, err = json.MarshalIndent(g.Config, "", "\t")
	if err != nil {
		return err
	}

	_, err = w.Write(data)
	if err != nil {
		return err
	}

	return nil
}

// SaveToFile writes the configuration into a file.
func (g *Generator) SaveToFile(path string) error {
	flags := os.O_RDWR | os.O_CREATE | os.O_TRUNC | unix.O_NOFOLLOW

	f, err := os.OpenFile(path, flags, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	return g.Save(f)
}
