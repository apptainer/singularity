// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package starter

/*
#include <stdlib.h>
#include <string.h>
#include <sys/mman.h>
#include <sys/types.h>
#include "starter.h"
*/
// #cgo CFLAGS: -I../../../../../../cmd/starter/c/include
import "C"
import (
	"encoding/json"
	"fmt"
	"strings"
	"syscall"
	"unsafe"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/util/capabilities"
)

// CConfig is the common type for C.struct_cConfig
type CConfig *C.struct_cConfig

// Config represents structure to manipulate C starter configuration
type Config struct {
	config CConfig
	nsPath []byte
}

// NewConfig takes a pointer to C starter configuration and returns a
// pointer to a Config
func NewConfig(config CConfig) *Config {
	return &Config{config: config, nsPath: make([]byte, 1)}
}

// GetIsSUID returns if SUID workflow is enabled or not
func (c *Config) GetIsSUID() bool {
	return c.config.container.isSuid == 1
}

// GetContainerPid returns container process ID
func (c *Config) GetContainerPid() int {
	return int(c.config.container.pid)
}

// SetInstance sets if starter should spawn instance or not
func (c *Config) SetInstance(instance bool) {
	if instance {
		c.config.container.isInstance = C.uchar(1)
	} else {
		c.config.container.isInstance = C.uchar(0)
	}
}

// GetInstance returns if container run as instance or not
func (c *Config) GetInstance() bool {
	return c.config.container.isInstance == 1
}

// SetNoNewPrivs sets NO_NEW_PRIVS flag
func (c *Config) SetNoNewPrivs(noprivs bool) {
	if noprivs {
		c.config.container.noNewPrivs = C.uchar(1)
	} else {
		c.config.container.noNewPrivs = C.uchar(0)
	}
}

// GetNoNewPrivs returns if NO_NEW_PRIVS flag is set or not
func (c *Config) GetNoNewPrivs() bool {
	return c.config.container.noNewPrivs == 1
}

// SetSharedMount sets if master/container shares mount point
func (c *Config) SetSharedMount(shared bool) {
	if shared {
		c.config.container.sharedMount = C.uchar(1)
	} else {
		c.config.container.sharedMount = C.uchar(0)
	}
}

// GetSharedMount returns if master/container shares mount point or not
func (c *Config) GetSharedMount() bool {
	return c.config.container.sharedMount == 1
}

// SetJoinMount sets if container process join a mount namespace
func (c *Config) SetJoinMount(join bool) {
	if join {
		c.config.container.joinMount = C.uchar(1)
	} else {
		c.config.container.joinMount = C.uchar(0)
	}
}

// GetJoinMount returns if container process join a mount namespace
func (c *Config) GetJoinMount() bool {
	return c.config.container.joinMount == 1
}

// SetBringLoopbackInterface sets if starter bring loopback network interface
func (c *Config) SetBringLoopbackInterface(bring bool) {
	if bring {
		c.config.container.bringLoopbackInterface = C.uchar(1)
	} else {
		c.config.container.bringLoopbackInterface = C.uchar(0)
	}
}

// GetBringLoopbackInterface returns if starter bring loopback network interface
func (c *Config) GetBringLoopbackInterface() bool {
	return c.config.container.bringLoopbackInterface == 1
}

// SetMountPropagation sets root filesystem mount propagation
func (c *Config) SetMountPropagation(propagation string) {
	var flags uintptr

	switch propagation {
	case "shared", "rshared":
		flags = syscall.MS_SHARED
	case "slave", "rslave":
		flags = syscall.MS_SLAVE
	case "private", "rprivate":
		flags = syscall.MS_PRIVATE
	case "unbindable", "runbindable":
		flags = syscall.MS_UNBINDABLE
	}

	if strings.HasPrefix(propagation, "r") {
		flags |= syscall.MS_REC
	}
	c.config.container.mountPropagation = C.ulong(flags)
}

// GetJSONConfig returns pointer to JSON configuration
func (c *Config) GetJSONConfig() []byte {
	return C.GoBytes(unsafe.Pointer(&c.config.json.config[0]), C.int(c.config.json.size))
}

// WriteConfig writes raw C configuration
func (c *Config) Write(payload interface{}) error {
	jsonConf, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %s", err)
	}
	size := len(jsonConf)
	maxSize := C.MAX_JSON_SIZE - 1
	c.config.json.size = C.size_t(size)

	if size >= maxSize {
		return fmt.Errorf("json configuration too big %d > %d", size, maxSize)
	}

	json := C.CBytes(jsonConf)

	C.memcpy(unsafe.Pointer(&c.config.json.config[0]), json, c.config.json.size)
	C.free(json)

	return nil
}

// AddUIDMappings sets user namespace UID mapping.
func (c *Config) AddUIDMappings(uids []specs.LinuxIDMapping) error {
	uidMap := ""
	for i, uid := range uids {
		if i == 0 {
			c.SetTargetUID(int(uid.ContainerID))
		}
		uidMap = uidMap + fmt.Sprintf("%d %d %d\n", uid.ContainerID, uid.HostID, uid.Size)
	}

	l := len(uidMap)
	if l >= C.MAX_MAP_SIZE-1 {
		return fmt.Errorf("UID map too big")
	}

	if l > 0 {
		cpath := unsafe.Pointer(C.CString(uidMap))
		size := C.size_t(l)

		C.memcpy(unsafe.Pointer(&c.config.container.uidMap[0]), cpath, size)
		C.free(cpath)
	}

	return nil
}

// AddGIDMappings sets user namespace GID mapping
func (c *Config) AddGIDMappings(gids []specs.LinuxIDMapping) error {
	targetGids := make([]int, 0, len(gids))
	gidMap := ""
	for _, gid := range gids {
		targetGids = append(targetGids, int(gid.ContainerID))
		gidMap = gidMap + fmt.Sprintf("%d %d %d\n", gid.ContainerID, gid.HostID, gid.Size)
	}

	if len(targetGids) != 0 {
		c.SetTargetGID(targetGids)
	}

	l := len(gidMap)
	if l >= C.MAX_MAP_SIZE-1 {
		return fmt.Errorf("GID map too big")
	}

	if l > 0 {
		cpath := unsafe.Pointer(C.CString(gidMap))
		size := C.size_t(l)

		C.memcpy(unsafe.Pointer(&c.config.container.gidMap[0]), cpath, size)
		C.free(cpath)
	}

	return nil
}

// SetNsFlags sets namespaces flag directly from flags argument
func (c *Config) SetNsFlags(flags int) {
	c.config.namespace.flags = C.uint(flags)
}

// SetNsFlagsFromSpec sets namespaces flag from OCI spec
func (c *Config) SetNsFlagsFromSpec(namespaces []specs.LinuxNamespace) {
	c.config.namespace.flags = 0
	for _, namespace := range namespaces {
		if namespace.Path == "" {
			switch namespace.Type {
			case specs.UserNamespace:
				c.config.namespace.flags |= syscall.CLONE_NEWUSER
			case specs.IPCNamespace:
				c.config.namespace.flags |= syscall.CLONE_NEWIPC
			case specs.UTSNamespace:
				c.config.namespace.flags |= syscall.CLONE_NEWUTS
			case specs.PIDNamespace:
				c.config.namespace.flags |= syscall.CLONE_NEWPID
			case specs.NetworkNamespace:
				c.config.namespace.flags |= syscall.CLONE_NEWNET
			case specs.MountNamespace:
				c.config.namespace.flags |= syscall.CLONE_NEWNS
			case specs.CgroupNamespace:
				c.config.namespace.flags |= 0x2000000
			}
		}
	}
}

// SetNsPath sets corresponding namespace to be joined
func (c *Config) SetNsPath(nstype specs.LinuxNamespaceType, path string) error {
	cpath := unsafe.Pointer(C.CString(path))
	l := len(path)
	size := C.size_t(l)

	if l > C.MAX_NS_PATH_SIZE-1 {
		return fmt.Errorf("%s namespace path too big", nstype)
	}

	switch nstype {
	case specs.UserNamespace:
		C.memcpy(unsafe.Pointer(&c.config.namespace.user[0]), cpath, size)
	case specs.IPCNamespace:
		C.memcpy(unsafe.Pointer(&c.config.namespace.ipc[0]), cpath, size)
	case specs.UTSNamespace:
		C.memcpy(unsafe.Pointer(&c.config.namespace.uts[0]), cpath, size)
	case specs.PIDNamespace:
		C.memcpy(unsafe.Pointer(&c.config.namespace.pid[0]), cpath, size)
	case specs.NetworkNamespace:
		C.memcpy(unsafe.Pointer(&c.config.namespace.network[0]), cpath, size)
	case specs.MountNamespace:
		C.memcpy(unsafe.Pointer(&c.config.namespace.mount[0]), cpath, size)
	case specs.CgroupNamespace:
		C.memcpy(unsafe.Pointer(&c.config.namespace.cgroup[0]), cpath, size)
	}

	C.free(cpath)

	return nil
}

// SetNsPathFromSpec sets corresponding namespace to be joined from OCI spec
func (c *Config) SetNsPathFromSpec(namespaces []specs.LinuxNamespace) error {
	for _, namespace := range namespaces {
		if namespace.Path != "" {
			cpath := unsafe.Pointer(C.CString(namespace.Path))
			l := len(namespace.Path)
			size := C.size_t(l)

			if l > C.MAX_NS_PATH_SIZE-1 {
				return fmt.Errorf("%s namespace path too big", namespace.Type)
			}

			switch namespace.Type {
			case specs.UserNamespace:
				C.memcpy(unsafe.Pointer(&c.config.namespace.user[0]), cpath, size)
			case specs.IPCNamespace:
				C.memcpy(unsafe.Pointer(&c.config.namespace.ipc[0]), cpath, size)
			case specs.UTSNamespace:
				C.memcpy(unsafe.Pointer(&c.config.namespace.uts[0]), cpath, size)
			case specs.PIDNamespace:
				C.memcpy(unsafe.Pointer(&c.config.namespace.pid[0]), cpath, size)
			case specs.NetworkNamespace:
				C.memcpy(unsafe.Pointer(&c.config.namespace.network[0]), cpath, size)
			case specs.MountNamespace:
				C.memcpy(unsafe.Pointer(&c.config.namespace.mount[0]), cpath, size)
			case specs.CgroupNamespace:
				C.memcpy(unsafe.Pointer(&c.config.namespace.cgroup[0]), cpath, size)
			}

			C.free(cpath)
		}
	}

	return nil
}

// SetCapabilities sets corresponding capability set identified by ctype
// from a capability string list identified by ctype
func (c *Config) SetCapabilities(ctype string, caps []string) {
	switch ctype {
	case capabilities.Permitted:
		c.config.capabilities.permitted = 0
		for _, v := range caps {
			c.config.capabilities.permitted |= C.ulonglong(1 << capabilities.Map[v].Value)
		}
	case capabilities.Effective:
		c.config.capabilities.effective = 0
		for _, v := range caps {
			c.config.capabilities.effective |= C.ulonglong(1 << capabilities.Map[v].Value)
		}
	case capabilities.Inheritable:
		c.config.capabilities.inheritable = 0
		for _, v := range caps {
			c.config.capabilities.inheritable |= C.ulonglong(1 << capabilities.Map[v].Value)
		}
	case capabilities.Bounding:
		c.config.capabilities.bounding = 0
		for _, v := range caps {
			c.config.capabilities.bounding |= C.ulonglong(1 << capabilities.Map[v].Value)
		}
	case capabilities.Ambient:
		c.config.capabilities.ambient = 0
		for _, v := range caps {
			c.config.capabilities.ambient |= C.ulonglong(1 << capabilities.Map[v].Value)
		}
	}
}

// SetTargetUID sets target UID to execute the container process as user ID
func (c *Config) SetTargetUID(uid int) {
	c.config.container.targetUID = C.uid_t(uid)
}

// SetTargetGID sets target GIDs to execute container process as group IDs
func (c *Config) SetTargetGID(gids []int) {
	c.config.container.numGID = C.int(len(gids))

	for i, gid := range gids {
		if i >= C.MAX_GID {
			sylog.Warningf("you can't specify more than %d group IDs", C.MAX_GID)
			c.config.container.numGID = C.MAX_GID
			break
		}
		c.config.container.targetGID[i] = C.gid_t(gid)
	}
}

// Release performs a unmap on starter config and release mapped memory
func (c *Config) Release() error {
	if C.munmap(unsafe.Pointer(c.config), C.sizeof_struct_cConfig) != 0 {
		return fmt.Errorf("failed to release starter memory")
	}
	return nil
}
