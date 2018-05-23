// +build linux

// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package config

import (
	"os"

	"github.com/opencontainers/runtime-spec/specs-go"
)

// RuntimeOciLinux describes the linux OCI runtime interface.
type RuntimeOciLinux interface {
	GetSpec() *specs.Linux

	GetUIDMappings() []specs.LinuxIDMapping
	SetUIDMappings(uidmap []specs.LinuxIDMapping) error
	AddUIDMapping(hostid uint32, containerid uint32, size uint32) error
	DelUIDMapping(hostid uint32) error

	GetGIDMappings() []specs.LinuxIDMapping
	SetGIDMappings(gidmap []specs.LinuxIDMapping) error
	AddGIDMapping(hostid uint32, containerid uint32, size uint32) error
	DelGIDMapping(hostid uint32) error

	GetSysctl() map[string]string
	SetSysctl(sys map[string]string) error
	AddSysctl(key string, value string) error
	DelSysctl(key string) error

	LinuxResources

	GetCgroupsPath() string
	SetCgroupsPath(path string)

	GetNamespaces() []specs.LinuxNamespace
	SetNamespaces(namespaces []specs.LinuxNamespace) error
	AddPIDNamespace(path string) error
	DelPIDNamespace() error
	GetPIDNamespacePid() (int, bool)
	AddNetworkNamespace(path string) error
	DelNetworkNamespace() error
	GetNetworkNamespacePid() (int, bool)
	AddMountNamespace(path string) error
	DelMountNamespace() error
	GetMountNamespacePid() (int, bool)
	AddIPCNamespace(path string) error
	DelIPCNamespace() error
	GetIPCNamespacePid() (int, bool)
	AddUTSNamespace(path string) error
	DelUTSNamespace() error
	GetUTSNamespacePid() (int, bool)
	AddUserNamespace(path string) error
	DelUserNamespace() error
	GetUserNamespacePid() (int, bool)
	AddCgroupNamespace(path string) error
	DelCgroupNamespace() error
	GetCgroupNamespacePid() (int, bool)

	GetDevices() []specs.LinuxDevice
	SetDevices(devices []specs.LinuxDevice) error
	AddDevice(path string, dtype string, major int64, minor int64, filemode *os.FileMode, uid *uint32, gid *uint32) error
	DelDevice(path string) error

	LinuxSeccomp

	GetRootfsPropagation() string
	SetRootfsPropagation(path string)

	GetMaskedPaths() []string
	SetMaskedPaths(paths []string) error
	AddMaskedPath(path string) error
	DelMaskedPath(path string) error

	GetReadonlyPaths() []string
	SetReadonlyPaths(path []string) error
	AddReadonlyPath(path string) error
	DelReadonlyPath(path string) error

	GetMountLabel() string
	SetMountLabel(label string)

	GetIntelRdt() *specs.LinuxIntelRdt
	SetIntelRdt(rdt *specs.LinuxIntelRdt) error
	AddIntelRdt(l3cacheschema string) error
	DelIntelRdt() error
}

// LinuxResources describes the linux resources interface.
type LinuxResources interface {
	GetResourcesDevices() []specs.LinuxDeviceCgroup
	SetResourcesDevices(devices []specs.LinuxDeviceCgroup) error
	AddResourcesDevice(allow bool, dtype string, major *int64, minor *int64) error
	DelResourcesDevice(major *int64, minor *int64) error

	GetResourcesMemory() *specs.LinuxMemory
	SetResourcesMemory(memory *specs.LinuxMemory) error
	AddResourcesMemory(limit *int64, reservation *int64, swap *int64, kernel *int64, kerneltcp *int64, swappiness *uint64, disableoomkiller bool) error
	DelResourcesMemory() error

	GetResourcesCPU() *specs.LinuxCPU
	SetResourcesCPU(cpu *specs.LinuxCPU) error
	AddResourcesCPU(shares *uint64, quota *int64, period *uint64, realtimeRuntime *int64, realtimePeriod *uint64, cpus string, mems string) error
	DelResourcesCPU() error

	GetResourcesPids() *specs.LinuxPids
	SetResourcesPids(pids *specs.LinuxPids) error
	AddResourcesPids(limit int64) error
	DelResourcesPids() error

	GetResourcesBlockIO() *specs.LinuxBlockIO
	SetResourcesBlockIO(blockio *specs.LinuxBlockIO) error
	AddResourcesBlockIO(weight *uint16, leafweight *uint16, weightdevice []specs.LinuxWeightDevice, ThrottleReadBpsDevice []specs.LinuxThrottleDevice, ThrottleWriteBpsDevice []specs.LinuxThrottleDevice, ThrottleReadIOPSDevice []specs.LinuxThrottleDevice, ThrottleWriteIOPSDevice []specs.LinuxThrottleDevice) error
	DelResourcesBlockIO() error

	GetResourcesHugepageLimit() []specs.LinuxHugepageLimit
	SetResourcesHugepageLimit(limits []specs.LinuxHugepageLimit) error
	AddResourcesHugepageLimit(pagesize string, limit uint64) error
	DelResroucesHugepageLimit(pagesize string) error

	GetResourcesNetwork() *specs.LinuxNetwork
	SetResourcesNetwork(network *specs.LinuxNetwork) error
	AddResourcesNetwork(classid *uint32, priorities []specs.LinuxInterfacePriority) error
	DelResourcesNetwork() error
	/*
		GetResourcesRdma() map[string]specs.LinuxRdma
		SetResourcesRdma(rdma map[string]specs.LinuxRdma) error
		AddResourcesRdma(name string, hcahandles *uint32, hcaobjects *uint32) error
		DelResourcesRdma(name string) error*/
}

// LinuxSeccomp describes the linux seccomp interface.
type LinuxSeccomp interface {
	GetSeccomp() *specs.LinuxSeccomp
	SetSeccomp(seccomp *specs.LinuxSeccomp) error

	SetSeccompDefaultAction(action specs.LinuxSeccompAction) error
	AddSeccompArchitecture(arch specs.Arch) error
	DelSeccompArchitecture(arch specs.Arch) error

	SetSeccompSyscalls(syscalls []specs.LinuxSyscall) error
}

// LinuxSyscall describes the linux system call interface.
type LinuxSyscall interface {
	Get() []specs.LinuxSyscall
	Set(syscalls []specs.LinuxSyscall) error
	Add(names []string, action specs.LinuxSeccompAction, args []specs.LinuxSeccompArg) error
}

// LinuxSeccompArg describes the linux seccomp argument interface.
type LinuxSeccompArg interface {
	Get() []specs.LinuxSeccompArg
	Set(args []specs.LinuxSeccompArg) error
	Add(index uint, value uint64, valuetwo uint64, op specs.LinuxSeccompOperator) error
}

/*
type DefaultRuntimeOciLinux struct {
    RuntimeOciSpec *RuntimeOciSpec
}

func (c *DefaultRuntimeOciLinux) init() {
    if c.RuntimeOciSpec.Linux == nil {
        c.RuntimeOciSpec.Linux = &specs.Linux{}
    }
}

func (c *DefaultRuntimeOciLinux) GetUIDMappings() []specs.LinuxIDMapping {
    c.init()
}

func (c *DefaultRuntimeOciLinux) SetUIDMappings(uidmap []specs.LinuxIDMapping) error {
    c.init()
}

func (c *DefaultRuntimeOciLinux) AddUIDMapping(hostid uint32, containerid uint32, size uint32) error {
    c.init()
}

func (c *DefaultRuntimeOciLinux) DelUIDMapping(hostid uint32) error {
    c.init()
}

func (c *DefaultRuntimeOciLinux) GetGIDMappings() []specs.LinuxIDMapping {
    c.init()
}

func (c *DefaultRuntimeOciLinux) SetGIDMappings(uidmap []specs.LinuxIDMapping) error {
    c.init()
}

func (c *DefaultRuntimeOciLinux) AddGIDMapping(hostid uint32, containerid uint32, size uint32) error {
    c.init()
}

func (c *DefaultRuntimeOciLinux) DelGIDMapping(hostid uint32) error {
    c.init()
}

func (c *DefaultRuntimeOciLinux) GetSysctl() map[string]string {
    c.init()
}

func (c *DefaultRuntimeOciLinux) SetSysctl(sys map[string]string) error {
    c.init()
}

func (c *DefaultRuntimeOciLinux) AddSysctl(key string, value string) error {
    c.init()
}

func (c *DefaultRuntimeOciLinux) DelSysctl(key string) error {
    c.init()
}
*/
/*
   GetSpec() *specs.Linux

   GetUIDMappings() []specs.LinuxIDMapping
   SetUIDMappings(uidmap []specs.LinuxIDMapping) error
   AddUIDMapping(hostid uint32, containerid uint32, size uint32) error
   DelUIDMapping(hostid uint32) error

   GetGIDMappings() []specs.LinuxIDMapping
   SetGIDMappings(gidmap []specs.LinuxIDMapping) error
   AddGIDMapping(hostid uint32, containerid uint32, size uint32) error
   DelGIDMapping(hostid uint32) error

   GetSysctl() map[string]string
   SetSysctl(sys map[string]string) error
   AddSysctl(key string, value string) error
   DelSysctl(key string) error

   LinuxResources

   GetCgroupsPath() string
   SetCgroupsPath(path string)

   GetNamespaces() []specs.LinuxNamespace
   SetNamespaces(namespaces []specs.LinuxNamespace) error
   AddPIDNamespace(path string) error
   DelPIDNamespace() error
   GetPIDNamespacePid() (int, bool)
   AddNetworkNamespace(path string) error
   DelNetworkNamespace() error
   GetNetworkNamespacePid() (int, bool)
   AddMountNamespace(path string) error
   DelMountNamespace() error
   GetMountNamespacePid() (int, bool)
   AddIPCNamespace(path string) error
   DelIPCNamespace() error
   GetIPCNamespacePid() (int, bool)
   AddUTSNamespace(path string) error
   DelUTSNamespace() error
   GetUTSNamespacePid() (int, bool)
   AddUserNamespace(path string) error
   DelUserNamespace() error
   GetUserNamespacePid() (int, bool)
   AddCgroupNamespace(path string) error
   DelCgroupNamespace() error
   GetCgroupNamespacePid() (int, bool)

   GetDevices() []specs.LinuxDevice
   SetDevices(devices []specs.LinuxDevice) error
   AddDevice(path string, dtype string, major int64, minor int64, filemode *os.FileMode, uid *uint32, gid *uint32) error
   DelDevice(path string) error

   LinuxSeccomp

   GetRootfsPropagation() string
   SetRootfsPropagation(path string)

   GetMaskedPaths() []string
   SetMaskedPaths(paths []string) error
   AddMaskedPath(path string) error
   DelMaskedPath(path string) error

   GetReadonlyPaths() []string
   SetReadonlyPaths(path []string) error
   AddReadonlyPath(path string) error
   DelReadonlyPath(path string) error

   GetMountLabel() string
   SetMountLabel(label string)

   GetIntelRdt() *specs.LinuxIntelRdt
   SetIntelRdt(rdt *specs.LinuxIntelRdt) error
   AddIntelRdt(l3cacheschema string) error
   DelIntelRdt() error
*/
