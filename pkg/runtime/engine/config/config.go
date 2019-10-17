// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package config

// Common provides the basis for all engine configs. Anything that can not be
// properly described through the OCI config can be stored as a generic JSON []byte.
type Common struct {
	EngineName  string `json:"engineName"`
	ContainerID string `json:"containerID"`
	// EngineConfig is the raw JSON representation of the Engine's underlying config.
	EngineConfig EngineConfig `json:"engineConfig"`
}

// EngineConfig is a generic interface to represent the implementations of an EngineConfig.
type EngineConfig interface{}

// FileConfig describes the singularity.conf file options
type FileConfig struct {
	AllowSetuid             bool     `default:"yes" authorized:"yes,no" directive:"allow setuid"`
	AllowPidNs              bool     `default:"yes" authorized:"yes,no" directive:"allow pid ns"`
	ConfigPasswd            bool     `default:"yes" authorized:"yes,no" directive:"config passwd"`
	ConfigGroup             bool     `default:"yes" authorized:"yes,no" directive:"config group"`
	ConfigResolvConf        bool     `default:"yes" authorized:"yes,no" directive:"config resolv_conf"`
	MountProc               bool     `default:"yes" authorized:"yes,no" directive:"mount proc"`
	MountSys                bool     `default:"yes" authorized:"yes,no" directive:"mount sys"`
	MountDevPts             bool     `default:"yes" authorized:"yes,no" directive:"mount devpts"`
	MountHome               bool     `default:"yes" authorized:"yes,no" directive:"mount home"`
	MountTmp                bool     `default:"yes" authorized:"yes,no" directive:"mount tmp"`
	MountHostfs             bool     `default:"no" authorized:"yes,no" directive:"mount hostfs"`
	UserBindControl         bool     `default:"yes" authorized:"yes,no" directive:"user bind control"`
	EnableFusemount         bool     `default:"yes" authorized:"yes,no" directive:"enable fusemount"`
	EnableUnderlay          bool     `default:"yes" authorized:"yes,no" directive:"enable underlay"`
	MountSlave              bool     `default:"yes" authorized:"yes,no" directive:"mount slave"`
	AllowContainerSquashfs  bool     `default:"yes" authorized:"yes,no" directive:"allow container squashfs"`
	AllowContainerExtfs     bool     `default:"yes" authorized:"yes,no" directive:"allow container extfs"`
	AllowContainerDir       bool     `default:"yes" authorized:"yes,no" directive:"allow container dir"`
	AlwaysUseNv             bool     `default:"no" authorized:"yes,no" directive:"always use nv"`
	SharedLoopDevices       bool     `default:"no" authorized:"yes,no" directive:"shared loop devices"`
	MaxLoopDevices          uint     `default:"256" directive:"max loop devices"`
	SessiondirMaxSize       uint     `default:"16" directive:"sessiondir max size"`
	MountDev                string   `default:"yes" authorized:"yes,no,minimal" directive:"mount dev"`
	EnableOverlay           string   `default:"try" authorized:"yes,no,try" directive:"enable overlay"`
	BindPath                []string `default:"/etc/localtime,/etc/hosts" directive:"bind path"`
	LimitContainerOwners    []string `directive:"limit container owners"`
	LimitContainerGroups    []string `directive:"limit container groups"`
	LimitContainerPaths     []string `directive:"limit container paths"`
	RootDefaultCapabilities string   `default:"full" authorized:"full,file,no" directive:"root default capabilities"`
	MemoryFSType            string   `default:"tmpfs" authorized:"tmpfs,ramfs" directive:"memory fs type"`
	CniConfPath             string   `directive:"cni configuration path"`
	CniPluginPath           string   `directive:"cni plugin path"`
	MksquashfsPath          string   `directive:"mksquashfs path"`
	CryptsetupPath          string   `directive:"cryptsetup path"`
}
