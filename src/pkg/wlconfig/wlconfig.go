// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Package wlconfig implements the loading and management of container white
// listing feature. This code uses the TOML config file standard to extract
// the structured configuration activating or disabling and controlling of
// the container execution white listing feature.
package wlconfig

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/pelletier/go-toml"
	"github.com/singularityware/singularity/src/pkg/signing"
)

// WlConfig describes the structure of a whitelist configuration file
type WlConfig struct {
	Activated bool         `toml:"activated"`
	AuthList  []authorized `toml:"authorized"`
}

type authorized struct {
	Path     string   `toml:"path"`
	Entities []string `toml:"entities"`
}

// LoadConfig opens a white list config file and unmarshals it into structures
func LoadConfig(confPath string) (wlcfg WlConfig, err error) {
	// read in the whitelist config file
	b, err := ioutil.ReadFile(confPath)
	if err != nil {
		return
	}

	// Unmarshal config file
	err = toml.Unmarshal(b, &wlcfg)
	return
}

// PutConfig takes the content of a wlConfig struct and Marshals it to file
func PutConfig(wlcfg WlConfig, confPath string) (err error) {
	data, err := toml.Marshal(wlcfg)
	if err != nil {
		return
	}

	return ioutil.WriteFile(confPath, data, 0600)
}

// ValidateConfig makes sure paths from configs are fully resolved
func (wlcfg *WlConfig) ValidateConfig() (err error) {
	for _, v := range wlcfg.AuthList {
		path, err := filepath.EvalSymlinks(v.Path)
		if err != nil {
			return err
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		if v.Path != abs {
			return fmt.Errorf("all paths should be fully cleaned with symlinks resolved")
		}
	}
	return
}

// ShouldRun determines if a container should run according to the white list state
func (wlcfg *WlConfig) ShouldRun(cpath string) (run bool, err error) {
	var auth *authorized

	// look if whitelisting is activated
	if wlcfg.Activated == false {
		return true, nil
	}

	// look if container is part of a defined domain
	for _, v := range wlcfg.AuthList {
		if filepath.Dir(cpath) == v.Path {
			auth = &v
			break
		}
	}
	if auth == nil {
		return false, fmt.Errorf("%s not part of any authorized domains", cpath)
	}

	// get all signing entities on the primary partition
	entities, err := signing.GetSignEntities(cpath)
	if err != nil {
		return
	}

	// was the primary partition signed by an authorized entity?
	for _, v := range auth.Entities {
		for _, u := range entities {
			if v == u {
				run = true
			}
		}
	}
	if run == false {
		return false, fmt.Errorf("%s is not signed by required entities", cpath)
	}

	return
}
