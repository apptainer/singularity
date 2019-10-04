// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package config

import (
	"encoding/json"
)

// Common provides the basis for all engine configs. Anything that can not be
// properly described through the OCI config can be stored as a generic JSON []byte.
type Common struct {
	EngineName  string `json:"engineName"`
	ContainerID string `json:"containerID"`
	// EngineConfig is the raw JSON representation of the Engine's underlying config.
	EngineConfig EngineConfig `json:"engineConfig"`
	// Plugin is the raw JSON representation of the plugins configurations.
	Plugin map[string]json.RawMessage `json:"plugin"`
}

// EngineConfig is a generic interface to represent the implementations of an EngineConfig.
type EngineConfig interface{}

// GetPluginConfig retrieves the configuration for the named plugin.
func (c *Common) GetPluginConfig(plugin string, cfg interface{}) error {
	if tmp, found := c.Plugin[plugin]; found {
		return json.Unmarshal(tmp, cfg)
	}

	return nil
}

// SetPluginConfig sets the configuration for the named plugin.
func (c *Common) SetPluginConfig(plugin string, cfg interface{}) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	c.Plugin[plugin] = data
	return nil
}
