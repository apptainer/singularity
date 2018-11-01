// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package config

// Common provides the basis for all engine configs. Anything that can not be
// properly described through the OCI config can be stored as generic JSON []byte
type Common struct {
	EngineName  string `json:"engineName"`
	ContainerID string `json:"containerID"`
	// EngineConfig is the raw JSON representation of the Engine's underlying config
	EngineConfig EngineConfig `json:"engineConfig"`
}

// EngineConfig is a generic interface to represent the implementations of an EngineConfig
type EngineConfig interface{}
