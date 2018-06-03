// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package config

import (
	"fmt"

	"github.com/singularityware/singularity/src/runtime/engines/config"
	oci "github.com/singularityware/singularity/src/runtime/engines/oci/config"
)

// Name is the name of the runtime.
const Name = "singularity"

// Configuration describes the runtime configuration.
type Configuration struct {
	AllowSetuid             bool     `default:"yes" authorized:"yes,no" directive:"allow setuid"`
	MaxLoopDevices          uint     `default:"256" directive:"max loop devices"`
	AllowPidNs              bool     `default:"yes" authorized:"yes,no" directive:"allow pid ns"`
	ConfigPasswd            bool     `default:"yes" authorized:"yes,no" directive:"config passwd"`
	ConfigGroup             bool     `default:"yes" authorized:"yes,no" directive:"config group"`
	ConfigResolvConf        bool     `default:"yes" authorized:"yes,no" directive:"config resolv_conf"`
	MountProc               bool     `default:"yes" authorized:"yes,no" directive:"mount proc"`
	MountSys                bool     `default:"yes" authorized:"yes,no" directive:"mount sys"`
	MountDev                string   `default:"yes" authorized:"yes,no,minimal" directive:"mount dev"`
	MountDevPts             bool     `default:"yes" authorized:"yes,no" directive:"mount devpts"`
	MountHome               bool     `default:"yes" authorized:"yes,no" directive:"mount home"`
	MountTmp                bool     `default:"yes" authorized:"yes,no" directive:"mount tmp"`
	MountHostfs             bool     `default:"no" authorized:"yes,no" directive:"mount hostfs"`
	BindPath                []string `default:"/etc/localtime,/etc/hosts" directive:"bind path"`
	UserBindControl         bool     `default:"yes" authorized:"yes,no" directive:"user bind control"`
	EnableOverlay           string   `default:"try" authorized:"yes,no,try" directive:"enable overlay"`
	MountSlave              bool     `default:"yes" authorized:"yes,no" directive:"mount slave"`
	SessiondirMaxSize       uint     `default:"16" directive:"sessiondir max size"`
	LimitContainerOwners    []string `directive:"limit container owners"`
	LimitContainerPaths     []string `directive:"limit container paths"`
	AllowContainerSquashfs  bool     `default:"yes" authorized:"yes,no" directive:"allow container squashfs"`
	AllowContainerExtfs     bool     `default:"yes" authorized:"yes,no" directive:"allow container extfs"`
	AllowContainerDir       bool     `default:"yes" authorized:"yes,no" directive:"allow container dir"`
	AutofsBugPath           []string `directive:"autofs bug path"`
	AlwaysUseNv             bool     `default:"no" authorized:"yes,no" directive:"always use nv"`
	RootDefaultCapabilities string   `default:"full" authorized:"full,file,no" directive:"root default capabilities"`
	AllowRootCapabilities   bool     `default:"yes" authorized:"yes,no" directive:"allow root capabilities"`
	AllowUserCapabilities   bool     `default:"no" authorized:"yes,no" directive:"allow user capabilities"`
}

// RuntimeEngineSpec is the specification of the runtime engine.
type RuntimeEngineSpec struct {
	TestField string `json:"testfield"`
}

// RuntimeEngineConfig is the configuration of the runtime engine.
type RuntimeEngineConfig struct {
	config.RuntimeConfig
	RuntimeEngineSpec RuntimeEngineSpec `json:"runtimeConfig"`
	FileConfig        *Configuration
}

// NewSingularityConfig returns a new Singularity configuration.
func NewSingularityConfig(containerID string) (*oci.RuntimeOciConfig, *RuntimeEngineConfig) {
	c := &Configuration{}
	if err := config.Parser("/usr/local/etc/singularity/singularity.conf", c); err != nil {
		fmt.Println(err)
	}
	runtimecfg := &RuntimeEngineConfig{FileConfig: c}
	cfg := &runtimecfg.RuntimeConfig
	runtimecfg.RuntimeSpec.ID = containerID
	runtimecfg.RuntimeSpec.RuntimeName = Name
	runtimecfg.RuntimeSpec.RuntimeOciSpec = &cfg.OciConfig.RuntimeOciSpec
	runtimecfg.RuntimeSpec.RuntimeEngineSpec = &runtimecfg.RuntimeEngineSpec
	oci.DefaultRuntimeOciConfig(&cfg.OciConfig)
	return &cfg.OciConfig, runtimecfg
}
