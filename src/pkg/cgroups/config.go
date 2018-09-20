// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cgroups

import (
	"github.com/pelletier/go-toml"
)

// CgroupsConfig describes the structure of a cgroups controls configuration file
type CgroupsConfig struct {
}

// LoadConfig opens cgroups controls config file and unmarshals it into structures
func LoadConfig(confPath string) (cConf CgroupsConfig, err error) {
	// read in the Cgroups config file
	b, err := ioutil.ReadFile(confPath)
	if err != nil {
		return
	}

	// Unmarshal config file
	err = toml.Unmarshal(b, &cConf)
	return
}

// PutConfig takes the content of a CgroupsConfig struct and Marshals it to file
func PutConfig(cConf CgroupsConfig, confPath string) (err error) {
	data, err := toml.Marshal(cConf)
	if err != nil {
		return
	}

	return ioutil.WriteFile(confPath, data, 0600)
}

// ValidateConfig makes sure elements from configs are sane
func (cConf *CgroupsConfig) ValidateConfig() (err error) {
	return
}
