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
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/util/capabilities"
)

const (
	searchPath = "/usr/bin:/usr/sbin:/bin:/sbin:/usr/local/bin:/usr/local/sbin"
)

// SConfig is the common type for C.struct_starterConfig
type SConfig *C.struct_starterConfig

// Config represents structure to manipulate C starter configuration
type Config struct {
	config SConfig
	nsPath []byte
}

// NewConfig takes a pointer to C starter configuration and returns a
// pointer to a Config
func NewConfig(config SConfig) *Config {
	return &Config{config: config, nsPath: make([]byte, 1)}
}

// GetIsSUID returns if SUID workflow is enabled or not
func (c *Config) GetIsSUID() bool {
	return c.config.starter.isSuid == C.true
}

// GetContainerPid returns container process ID
func (c *Config) GetContainerPid() int {
	return int(c.config.container.pid)
}

// SetInstance sets if starter should spawn instance or not
func (c *Config) SetInstance(instance bool) {
	if instance {
		c.config.container.isInstance = C.true
	} else {
		c.config.container.isInstance = C.false
	}
}

// GetInstance returns if container run as instance or not
func (c *Config) GetInstance() bool {
	return c.config.container.isInstance == C.true
}

// SetNoNewPrivs sets NO_NEW_PRIVS flag
func (c *Config) SetNoNewPrivs(noprivs bool) {
	if noprivs {
		c.config.container.privileges.noNewPrivs = C.true
	} else {
		c.config.container.privileges.noNewPrivs = C.false
	}
}

// GetNoNewPrivs returns if NO_NEW_PRIVS flag is set or not
func (c *Config) GetNoNewPrivs() bool {
	return c.config.container.privileges.noNewPrivs == C.true
}

// SetMasterPropagateMount sets if master/container shares mount point
func (c *Config) SetMasterPropagateMount(propagate bool) {
	if propagate {
		c.config.starter.masterPropagateMount = C.true
	} else {
		c.config.starter.masterPropagateMount = C.false
	}
}

// GetMasterPropagateMount returns if master/container shares mount point or not
func (c *Config) GetMasterPropagateMount() bool {
	return c.config.starter.masterPropagateMount == 1
}

// SetNamespaceJoinOnly sets if container process join a mount namespace
func (c *Config) SetNamespaceJoinOnly(join bool) {
	if join {
		c.config.container.namespace.joinOnly = C.true
	} else {
		c.config.container.namespace.joinOnly = C.false
	}
}

// GetNamespaceJoinOnly returns if container process join a mount namespace
func (c *Config) GetNamespaceJoinOnly() bool {
	return c.config.container.namespace.joinOnly == C.true
}

// SetBringLoopbackInterface sets if starter bring loopback network interface
func (c *Config) SetBringLoopbackInterface(bring bool) {
	if bring {
		c.config.container.namespace.bringLoopbackInterface = C.true
	} else {
		c.config.container.namespace.bringLoopbackInterface = C.false
	}
}

// GetBringLoopbackInterface returns if starter bring loopback network interface
func (c *Config) GetBringLoopbackInterface() bool {
	return c.config.container.namespace.bringLoopbackInterface == C.true
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
	c.config.container.namespace.mountPropagation = C.ulong(flags)
}

// SetWorkingDirectoryFd sets current working directory pointed by
// file descriptor, starter will use this file descriptor with fchdir
func (c *Config) SetWorkingDirectoryFd(fd int) {
	c.config.starter.workingDirectoryFd = C.int(fd)
}

// KeepFileDescriptor adds a file descriptor to an array of file
// descriptor that starter will kept open. All files opened during
// stage 1 will be shared with starter process, once stage 1 returns
// all file descriptor which are not listed here will be closed
func (c *Config) KeepFileDescriptor(fd int) error {
	if c.config.starter.numfds >= C.MAX_STARTER_FDS {
		return fmt.Errorf("maximum number of kept file descriptors reached")
	}
	c.config.starter.fds[c.config.starter.numfds] = C.int(fd)
	c.config.starter.numfds++
	return nil
}

// SetHybridWorkflow sets the flag to tell starter container setup
// will require an hybrid workflow. Typically used for fakeroot
func (c *Config) SetHybridWorkflow(hybrid bool) {
	if hybrid {
		c.config.starter.hybridWorkflow = C.true
	} else {
		c.config.starter.hybridWorkflow = C.false
	}
}

// SetAllowSetgroups allows use of setgroups syscall from user
// namespace
func (c *Config) SetAllowSetgroups(allow bool) {
	if allow {
		c.config.container.privileges.allowSetgroups = C.true
	} else {
		c.config.container.privileges.allowSetgroups = C.false
	}
}

// GetJSONConfig returns pointer to the engine's JSON configuration.
// A copy of the original bytes allocated on C heap is returned.
func (c *Config) GetJSONConfig() []byte {
	return C.GoBytes(unsafe.Pointer(&c.config.engine.config[0]), C.int(c.config.engine.size))
}

// WriteConfig writes raw C configuration.
func (c *Config) Write(payload interface{}) error {
	jsonConf, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %s", err)
	}
	size := len(jsonConf)
	maxSize := C.MAX_JSON_SIZE - 1
	c.config.engine.size = C.size_t(size)

	if size >= maxSize {
		return fmt.Errorf("json configuration too big %d > %d", size, maxSize)
	}

	json := C.CBytes(jsonConf)

	C.memcpy(unsafe.Pointer(&c.config.engine.config[0]), json, c.config.engine.size)
	C.free(json)

	return nil
}

// AddUIDMappings sets user namespace UID mapping.
func (c *Config) AddUIDMappings(uids []specs.LinuxIDMapping) error {
	uidMap := ""
	for _, uid := range uids {
		uidMap = uidMap + fmt.Sprintf("%d %d %d\n", uid.ContainerID, uid.HostID, uid.Size)
	}

	l := len(uidMap)
	if l >= C.MAX_MAP_SIZE-1 {
		return fmt.Errorf("uid map too big")
	}

	if l > 0 {
		cpath := unsafe.Pointer(C.CString(uidMap))
		size := C.size_t(l)

		C.memcpy(unsafe.Pointer(&c.config.container.privileges.uidMap[0]), cpath, size)
		C.free(cpath)
	}

	return nil
}

// AddGIDMappings sets user namespace GID mapping
func (c *Config) AddGIDMappings(gids []specs.LinuxIDMapping) error {
	gidMap := ""
	for _, gid := range gids {
		gidMap = gidMap + fmt.Sprintf("%d %d %d\n", gid.ContainerID, gid.HostID, gid.Size)
	}

	l := len(gidMap)
	if l >= C.MAX_MAP_SIZE-1 {
		return fmt.Errorf("gid map too big")
	}

	if l > 0 {
		cpath := unsafe.Pointer(C.CString(gidMap))
		size := C.size_t(l)

		C.memcpy(unsafe.Pointer(&c.config.container.privileges.gidMap[0]), cpath, size)
		C.free(cpath)
	}

	return nil
}

func setNewIDMapPath(command string, pathPointer unsafe.Pointer) error {
	os.Setenv("PATH", searchPath)
	defer os.Unsetenv("PATH")

	path, err := exec.LookPath(command)
	if err != nil {
		return fmt.Errorf(
			"%s was not found in PATH (%s), required with fakeroot and unprivileged installation",
			command, searchPath,
		)
	}

	lpath := len(path)
	size := C.size_t(lpath)
	if lpath >= C.MAX_PATH_SIZE-1 {
		return fmt.Errorf("%s path too long", command)
	}

	cpath := unsafe.Pointer(C.CString(path))
	C.memcpy(pathPointer, cpath, size)
	C.free(cpath)

	return nil
}

// SetNewUIDMapPath sets absolute path to newuidmap binary if found.
func (c *Config) SetNewUIDMapPath() error {
	return setNewIDMapPath(
		"newuidmap",
		unsafe.Pointer(&c.config.container.privileges.newuidmapPath[0]),
	)
}

// SetNewGIDMapPath sets absolute path to newgidmap binary if found.
func (c *Config) SetNewGIDMapPath() error {
	return setNewIDMapPath(
		"newgidmap",
		unsafe.Pointer(&c.config.container.privileges.newgidmapPath[0]),
	)
}

// SetNsFlags sets namespaces flag directly from flags argument.
func (c *Config) SetNsFlags(flags int) {
	c.config.container.namespace.flags = C.uint(flags)
}

// SetNsFlagsFromSpec sets namespaces flag from OCI spec.
func (c *Config) SetNsFlagsFromSpec(namespaces []specs.LinuxNamespace) {
	c.config.container.namespace.flags = 0
	for _, namespace := range namespaces {
		if namespace.Path == "" {
			switch namespace.Type {
			case specs.UserNamespace:
				c.config.container.namespace.flags |= syscall.CLONE_NEWUSER
			case specs.IPCNamespace:
				c.config.container.namespace.flags |= syscall.CLONE_NEWIPC
			case specs.UTSNamespace:
				c.config.container.namespace.flags |= syscall.CLONE_NEWUTS
			case specs.PIDNamespace:
				c.config.container.namespace.flags |= syscall.CLONE_NEWPID
			case specs.NetworkNamespace:
				c.config.container.namespace.flags |= syscall.CLONE_NEWNET
			case specs.MountNamespace:
				c.config.container.namespace.flags |= syscall.CLONE_NEWNS
			case specs.CgroupNamespace:
				c.config.container.namespace.flags |= 0x2000000
			}
		}
	}
}

// SetNsPath sets corresponding namespace to be joined.
func (c *Config) SetNsPath(nstype specs.LinuxNamespaceType, path string) error {
	cpath := unsafe.Pointer(C.CString(path))
	l := len(path)
	size := C.size_t(l)

	if l > C.MAX_PATH_SIZE-1 {
		return fmt.Errorf("%s namespace path too big", nstype)
	}

	switch nstype {
	case specs.UserNamespace:
		C.memcpy(unsafe.Pointer(&c.config.container.namespace.user[0]), cpath, size)
	case specs.IPCNamespace:
		C.memcpy(unsafe.Pointer(&c.config.container.namespace.ipc[0]), cpath, size)
	case specs.UTSNamespace:
		C.memcpy(unsafe.Pointer(&c.config.container.namespace.uts[0]), cpath, size)
	case specs.PIDNamespace:
		C.memcpy(unsafe.Pointer(&c.config.container.namespace.pid[0]), cpath, size)
	case specs.NetworkNamespace:
		C.memcpy(unsafe.Pointer(&c.config.container.namespace.network[0]), cpath, size)
	case specs.MountNamespace:
		C.memcpy(unsafe.Pointer(&c.config.container.namespace.mount[0]), cpath, size)
	case specs.CgroupNamespace:
		C.memcpy(unsafe.Pointer(&c.config.container.namespace.cgroup[0]), cpath, size)
	}

	C.free(cpath)

	return nil
}

// SetNsPathFromSpec sets corresponding namespace to be joined from OCI spec.
func (c *Config) SetNsPathFromSpec(namespaces []specs.LinuxNamespace) error {
	for _, namespace := range namespaces {
		if namespace.Path != "" {
			cpath := unsafe.Pointer(C.CString(namespace.Path))
			l := len(namespace.Path)
			size := C.size_t(l)

			if l > C.MAX_PATH_SIZE-1 {
				return fmt.Errorf("%s namespace path too big", namespace.Type)
			}

			switch namespace.Type {
			case specs.UserNamespace:
				C.memcpy(unsafe.Pointer(&c.config.container.namespace.user[0]), cpath, size)
			case specs.IPCNamespace:
				C.memcpy(unsafe.Pointer(&c.config.container.namespace.ipc[0]), cpath, size)
			case specs.UTSNamespace:
				C.memcpy(unsafe.Pointer(&c.config.container.namespace.uts[0]), cpath, size)
			case specs.PIDNamespace:
				C.memcpy(unsafe.Pointer(&c.config.container.namespace.pid[0]), cpath, size)
			case specs.NetworkNamespace:
				C.memcpy(unsafe.Pointer(&c.config.container.namespace.network[0]), cpath, size)
			case specs.MountNamespace:
				C.memcpy(unsafe.Pointer(&c.config.container.namespace.mount[0]), cpath, size)
			case specs.CgroupNamespace:
				C.memcpy(unsafe.Pointer(&c.config.container.namespace.cgroup[0]), cpath, size)
			}

			C.free(cpath)
		}
	}

	return nil
}

// SetCapabilities sets corresponding capability set identified by ctype
// from a capability string list identified by ctype.
func (c *Config) SetCapabilities(ctype string, caps []string) {
	switch ctype {
	case capabilities.Permitted:
		c.config.container.privileges.capabilities.permitted = 0
		for _, v := range caps {
			c.config.container.privileges.capabilities.permitted |= C.ulonglong(1 << capabilities.Map[v].Value)
		}
	case capabilities.Effective:
		c.config.container.privileges.capabilities.effective = 0
		for _, v := range caps {
			c.config.container.privileges.capabilities.effective |= C.ulonglong(1 << capabilities.Map[v].Value)
		}
	case capabilities.Inheritable:
		c.config.container.privileges.capabilities.inheritable = 0
		for _, v := range caps {
			c.config.container.privileges.capabilities.inheritable |= C.ulonglong(1 << capabilities.Map[v].Value)
		}
	case capabilities.Bounding:
		c.config.container.privileges.capabilities.bounding = 0
		for _, v := range caps {
			c.config.container.privileges.capabilities.bounding |= C.ulonglong(1 << capabilities.Map[v].Value)
		}
	case capabilities.Ambient:
		c.config.container.privileges.capabilities.ambient = 0
		for _, v := range caps {
			c.config.container.privileges.capabilities.ambient |= C.ulonglong(1 << capabilities.Map[v].Value)
		}
	}
}

// SetTargetUID sets target UID to execute the container process as user ID.
func (c *Config) SetTargetUID(uid int) {
	c.config.container.privileges.targetUID = C.uid_t(uid)
}

// SetTargetGID sets target GIDs to execute container process as group IDs.
func (c *Config) SetTargetGID(gids []int) {
	c.config.container.privileges.numGID = C.int(len(gids))

	for i, gid := range gids {
		if i >= C.MAX_GID {
			sylog.Warningf("you can't specify more than %d group IDs", C.MAX_GID)
			c.config.container.privileges.numGID = C.MAX_GID
			break
		}
		c.config.container.privileges.targetGID[i] = C.gid_t(gid)
	}
}

// Release performs a unmap on starter config and release mapped memory.
func (c *Config) Release() error {
	if C.munmap(unsafe.Pointer(c.config), C.sizeof_struct_starterConfig) != 0 {
		return fmt.Errorf("failed to release starter memory")
	}
	return nil
}
