// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package wrapper

/*
#include <sys/types.h>
#include "startup/c/wrapper.h"
*/
// #cgo CFLAGS: -I../../../..
import "C"
import (
	"encoding/json"
	"fmt"
	"io"
	"syscall"
	"unsafe"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/singularityware/singularity/src/pkg/util/capabilities"
)

// CConfig is the common type for C.struct_cConfig
type CConfig *C.struct_cConfig

// Config represents structure to manipulate C wrapper configuration
type Config struct {
	config CConfig
}

// NewConfig takes a pointer to C wrapper configuration and returns a
// pointer to a Config
func NewConfig(config CConfig) *Config {
	return &Config{config}
}

// GetIsSUID returns if SUID workflow is enabled or not
func (c *Config) GetIsSUID() bool {
	if c.config.isSuid == 1 {
		return true
	}
	return false
}

// GetContainerPid returns container process ID
func (c *Config) GetContainerPid() int {
	return int(c.config.containerPid)
}

// SetInstance sets if wrapper should spawn instance or not
func (c *Config) SetInstance(instance bool) {
	if instance {
		c.config.isInstance = C.uchar(1)
	} else {
		c.config.isInstance = C.uchar(0)
	}
}

// GetInstance returns if container run as instance or not
func (c *Config) GetInstance() bool {
	if c.config.isInstance == 1 {
		return true
	}
	return false
}

// SetNoNewPrivs sets NO_NEW_PRIVS flag
func (c *Config) SetNoNewPrivs(noprivs bool) {
	if noprivs {
		c.config.noNewPrivs = C.uchar(1)
	} else {
		c.config.noNewPrivs = C.uchar(0)
	}
}

// GetNoNewPrivs returns if NO_NEW_PRIVS flag is set or not
func (c *Config) GetNoNewPrivs() bool {
	if c.config.noNewPrivs == 1 {
		return true
	}
	return false
}

// GetJSONConfSize returns size of JSON configuration sent
// by wrapper
func (c *Config) GetJSONConfSize() uint {
	return uint(c.config.jsonConfSize)
}

// WritePayload writes raw C configuration and payload passed in
// argument to the provided writer
func (c *Config) WritePayload(w io.Writer, payload interface{}) error {
	jsonConf, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %s", err)
	}

	c.config.jsonConfSize = C.uint(len(jsonConf))
	cconfPayload := C.GoBytes(unsafe.Pointer(c.config), C.sizeof_struct_cConfig)
	finalPayload := append(cconfPayload, jsonConf...)

	if n, err := w.Write(finalPayload); err != nil || n != len(finalPayload) {
		return fmt.Errorf("failed to write payload: %s", err)
	}
	return nil
}

// AddUIDMappings sets user namespace UID mapping
func (c *Config) AddUIDMappings(uids []specs.LinuxIDMapping) {
	for i, uid := range uids {
		c.config.uidMapping[i].containerID = C.uid_t(uid.ContainerID)
		c.config.uidMapping[i].hostID = C.uid_t(uid.HostID)
		c.config.uidMapping[i].size = C.uint(uid.Size)
	}
}

// AddGIDMappings sets user namespace GID mapping
func (c *Config) AddGIDMappings(gids []specs.LinuxIDMapping) {
	for i, gid := range gids {
		c.config.gidMapping[i].containerID = C.gid_t(gid.ContainerID)
		c.config.gidMapping[i].hostID = C.gid_t(gid.HostID)
		c.config.gidMapping[i].size = C.uint(gid.Size)
	}
}

// SetNsFlags sets namespaces flag directly from flags argument
func (c *Config) SetNsFlags(flags int) {
	c.config.nsFlags = C.uint(flags)
}

// SetNsFlagsFromSpec sets namespaces flag from OCI spec
func (c *Config) SetNsFlagsFromSpec(namespaces []specs.LinuxNamespace) {
	c.config.nsFlags = 0
	for _, namespace := range namespaces {
		switch namespace.Type {
		case specs.UserNamespace:
			c.config.nsFlags |= syscall.CLONE_NEWUSER
		case specs.IPCNamespace:
			c.config.nsFlags |= syscall.CLONE_NEWIPC
		case specs.UTSNamespace:
			c.config.nsFlags |= syscall.CLONE_NEWUTS
		case specs.PIDNamespace:
			c.config.nsFlags |= syscall.CLONE_NEWPID
		case specs.NetworkNamespace:
			c.config.nsFlags |= syscall.CLONE_NEWNET
		case specs.MountNamespace:
			c.config.nsFlags |= syscall.CLONE_NEWNS
		case specs.CgroupNamespace:
			c.config.nsFlags |= 0x2000000
		}
	}
}

// SetNsPid sets corresponding namespace to be joined
func (c *Config) SetNsPid(nstype specs.LinuxNamespaceType, pid int) {
	switch nstype {
	case specs.UserNamespace:
		c.config.userPid = C.pid_t(pid)
	case specs.IPCNamespace:
		c.config.ipcPid = C.pid_t(pid)
	case specs.UTSNamespace:
		c.config.utsPid = C.pid_t(pid)
	case specs.PIDNamespace:
		c.config.pidPid = C.pid_t(pid)
	case specs.NetworkNamespace:
		c.config.netPid = C.pid_t(pid)
	case specs.MountNamespace:
		c.config.mntPid = C.pid_t(pid)
	case specs.CgroupNamespace:
		c.config.cgroupPid = C.pid_t(pid)
	}
}

// SetCapabilities sets corresponding capability set identified by ctype
// from a capability string list identified by ctype
func (c *Config) SetCapabilities(ctype string, caps []string) {
	switch ctype {
	case capabilities.Permitted:
		c.config.capPermitted = 0
		for _, v := range caps {
			c.config.capPermitted |= C.ulonglong(1 << capabilities.Map[v].Value)
		}
	case capabilities.Effective:
		c.config.capEffective = 0
		for _, v := range caps {
			c.config.capEffective |= C.ulonglong(1 << capabilities.Map[v].Value)
		}
	case capabilities.Inheritable:
		c.config.capInheritable = 0
		for _, v := range caps {
			c.config.capInheritable |= C.ulonglong(1 << capabilities.Map[v].Value)
		}
	case capabilities.Bounding:
		c.config.capBounding = 0
		for _, v := range caps {
			c.config.capBounding |= C.ulonglong(1 << capabilities.Map[v].Value)
		}
	case capabilities.Ambient:
		c.config.capAmbient = 0
		for _, v := range caps {
			c.config.capAmbient |= C.ulonglong(1 << capabilities.Map[v].Value)
		}
	}
}
