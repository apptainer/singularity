// Copyright (c) 2018-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cgroups

import (
	"io/ioutil"
	"path/filepath"

	"github.com/pelletier/go-toml"
)

func Int64ptr(i int) *int64 {
	t := int64(i)
	return &t
}

var wildcard = Int64ptr(-1)

// LinuxHugepageLimit structure corresponds to limiting kernel hugepages
type LinuxHugepageLimit struct {
	// Pagesize is the hugepage size
	Pagesize string `toml:"pageSize" json:"pageSize"`
	// Limit is the limit of "hugepagesize" hugetlb usage
	Limit uint64 `toml:"limit" json:"limit"`
}

// LinuxInterfacePriority for network interfaces
type LinuxInterfacePriority struct {
	// Name is the name of the network interface
	Name string `toml:"name" json:"name"`
	// Priority for the interface
	Priority uint32 `toml:"priority" json:"priority"`
}

// LinuxWeightDevice struct holds a `major:minor weight` pair for weightDevice
type LinuxWeightDevice struct {
	// Major is the device's major number.
	Major int64 `toml:"major" json:"major"`
	// Minor is the device's minor number.
	Minor int64 `toml:"minor" json:"minor"`
	// Weight is the bandwidth rate for the device.
	Weight *uint16 `toml:"weight" json:"weight,omitempty"`
	// LeafWeight is the bandwidth rate for the device while competing with the cgroup's child cgroups, CFQ scheduler only
	LeafWeight *uint16 `toml:"leafWeight" json:"leafWeight,omitempty"`
}

// LinuxThrottleDevice struct holds a `major:minor rate_per_second` pair
type LinuxThrottleDevice struct {
	// Major is the device's major number.
	Major int64 `toml:"major" json:"major"`
	// Minor is the device's minor number.
	Minor int64 `toml:"minor" json:"minor"`
	// Rate is the IO rate limit per cgroup per device
	Rate uint64 `toml:"rate" json:"rate"`
}

// LinuxBlockIO for Linux cgroup 'blkio' resource management
type LinuxBlockIO struct {
	// Specifies per cgroup weight
	Weight *uint16 `toml:"weight" json:"weight,omitempty"`
	// Specifies tasks' weight in the given cgroup while competing with the cgroup's child cgroups, CFQ scheduler only
	LeafWeight *uint16 `toml:"leafWeight" json:"leafWeight,omitempty"`
	// Weight per cgroup per device, can override BlkioWeight
	WeightDevice []LinuxWeightDevice `toml:"weightDevice" json:"weightDevice,omitempty"`
	// IO read rate limit per cgroup per device, bytes per second
	ThrottleReadBpsDevice []LinuxThrottleDevice `toml:"throttleReadBpsDevice" json:"throttleReadBpsDevice,omitempty"`
	// IO write rate limit per cgroup per device, bytes per second
	ThrottleWriteBpsDevice []LinuxThrottleDevice `toml:"throttleWriteBpsDevice" json:"throttleWriteBpsDevice,omitempty"`
	// IO read rate limit per cgroup per device, IO per second
	ThrottleReadIOPSDevice []LinuxThrottleDevice `toml:"throttleReadIOPSDevice" json:"throttleReadIOPSDevice,omitempty"`
	// IO write rate limit per cgroup per device, IO per second
	ThrottleWriteIOPSDevice []LinuxThrottleDevice `toml:"throttleWriteIOPSDevice" json:"throttleWriteIOPSDevice,omitempty"`
}

// LinuxMemory for Linux cgroup 'memory' resource management
type LinuxMemory struct {
	// Memory limit (in bytes).
	Limit *int64 `toml:"limit" json:"limit,omitempty"`
	// Memory reservation or soft_limit (in bytes).
	Reservation *int64 `toml:"reservation" json:"reservation,omitempty"`
	// Total memory limit (memory + swap).
	Swap *int64 `toml:"swap" json:"swap,omitempty"`
	// Kernel memory limit (in bytes).
	Kernel *int64 `toml:"kernel" json:"kernel,omitempty"`
	// Kernel memory limit for tcp (in bytes)
	KernelTCP *int64 `toml:"kernelTCP" json:"kernelTCP,omitempty"`
	// How aggressive the kernel will swap memory pages.
	Swappiness *uint64 `toml:"swappiness" json:"swappiness,omitempty"`
	// DisableOOMKiller disables the OOM killer for out of memory conditions
	DisableOOMKiller *bool `toml:"disableOOMKiller" json:"disableOOMKiller,omitempty"`
}

// LinuxCPU for Linux cgroup 'cpu' resource management
type LinuxCPU struct {
	// CPU shares (relative weight (ratio) vs. other cgroups with cpu shares).
	Shares *uint64 `toml:"shares" json:"shares,omitempty"`
	// CPU hardcap limit (in usecs). Allowed cpu time in a given period.
	Quota *int64 `toml:"quota" json:"quota,omitempty"`
	// CPU period to be used for hardcapping (in usecs).
	Period *uint64 `toml:"period" json:"period,omitempty"`
	// How much time realtime scheduling may use (in usecs).
	RealtimeRuntime *int64 `toml:"realtimeRuntime" json:"realtimeRuntime,omitempty"`
	// CPU period to be used for realtime scheduling (in usecs).
	RealtimePeriod *uint64 `toml:"realtimePeriod" json:"realtimePeriod,omitempty"`
	// CPUs to use within the cpuset. Default is to use any CPU available.
	Cpus string `toml:"cpus" json:"cpus,omitempty"`
	// List of memory nodes in the cpuset. Default is to use any available memory node.
	Mems string `toml:"mems" json:"mems,omitempty"`
}

// LinuxPids for Linux cgroup 'pids' resource management (Linux 4.3)
type LinuxPids struct {
	// Maximum number of PIDs. Default is "no limit".
	Limit int64 `toml:"limit" json:"limit"`
}

// LinuxNetwork identification and priority configuration
type LinuxNetwork struct {
	// Set class identifier for container's network packets
	ClassID *uint32 `toml:"classID" json:"classID,omitempty"`
	// Set priority of network traffic for container
	Priorities []LinuxInterfacePriority `toml:"priorities" json:"priorities,omitempty"`
}

// LinuxRdma for Linux cgroup 'rdma' resource management (Linux 4.11)
type LinuxRdma struct {
	// Maximum number of HCA handles that can be opened. Default is "no limit".
	HcaHandles *uint32 `toml:"hcaHandles" json:"hcaHandles,omitempty"`
	// Maximum number of HCA objects that can be created. Default is "no limit".
	HcaObjects *uint32 `toml:"hcaObjects" json:"hcaObjects,omitempty"`
}

// LinuxDeviceCgroup represents a device rule for the whitelist controller
type LinuxDeviceCgroup struct {
	// Allow or deny
	Allow bool `toml:"allow" json:"allow" comment:"test"`
	// Device type, block, char, etc.
	Type string `toml:"type" json:"type,omitempty"`
	// Major is the device's major number.
	Major *int64 `toml:"major" json:"major,omitempty"`
	// Minor is the device's minor number.
	Minor *int64 `toml:"minor" json:"minor,omitempty"`
	// Cgroup access permissions format, rwm.
	Access string `toml:"access" json:"access,omitempty"`
}

// Config has container runtime resource constraints
type Config struct {
	// Devices configures the device whitelist.
	Devices []LinuxDeviceCgroup `toml:"devices" json:"devices,omitempty"`
	// Memory restriction configuration
	Memory *LinuxMemory `toml:"memory" json:"memory,omitempty"`
	// CPU resource restriction configuration
	CPU *LinuxCPU `toml:"cpu" json:"cpu,omitempty"`
	// Task resource restriction configuration.
	Pids *LinuxPids `toml:"pids" json:"pids,omitempty"`
	// BlockIO restriction configuration
	BlockIO *LinuxBlockIO `toml:"blockIO" json:"blockIO,omitempty"`
	// Hugetlb limit (in bytes)
	HugepageLimits []LinuxHugepageLimit `toml:"hugepageLimits" json:"hugepageLimits,omitempty"`
	// Network restriction configuration
	Network *LinuxNetwork `toml:"network" json:"network,omitempty"`
	// Rdma resource restriction configuration.
	// Limits are a set of key value pairs that define RDMA resource limits,
	// where the key is device name and value is resource limits.
	Rdma map[string]LinuxRdma `toml:"rdma" json:"rdma,omitempty"`
	// TODO: Enable support for native cgroup v2 resource specifications
	// Unified map[string]string `toml:"unified" json:"unified,omitempty"`
}

// LoadConfig opens cgroups controls config file and unmarshals it into structures
func LoadConfig(confPath string) (config Config, err error) {
	path, err := filepath.Abs(confPath)
	if err != nil {
		return
	}

	// read in the Cgroups config file
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}

	// Unmarshal config file
	err = toml.Unmarshal(b, &config)
	return
}

// PutConfig takes the content of a CgroupsConfig struct and Marshals it to file
func PutConfig(config Config, confPath string) (err error) {
	data, err := toml.Marshal(config)
	if err != nil {
		return
	}

	return ioutil.WriteFile(confPath, data, 0o600)
}
