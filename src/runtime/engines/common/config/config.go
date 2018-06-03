// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package config

import (
	"encoding/json"

	oci "github.com/singularityware/singularity/src/runtime/engines/common/oci/config"
)

// RuntimeSpec is the runtime template specification.
type RuntimeSpec struct {
	RuntimeName       string              `json:"runtimeName"`
	ID                string              `json:"containerID"`
	RuntimeOciSpec    *oci.RuntimeOciSpec `json:"ociConfig"`
	RuntimeEngineSpec RuntimeEngineSpec   `json:"runtimeConfig"`
}

// RuntimeEngineSpec is the runtime engine specification.
type RuntimeEngineSpec interface{}

// RuntimeEngineConfig is the runtime engine configuration.
type RuntimeEngineConfig struct {
	RuntimeEngineSpec
}

// RuntimeConfig is the generic runtime configuration.
type RuntimeConfig struct {
	RuntimeSpec
	OciConfig    oci.RuntimeOciConfig
	EngineConfig RuntimeEngineConfig
}

// GetConfig returns the runtime configuration in JSON format.
func (r *RuntimeConfig) GetConfig() ([]byte, error) {
	b, err := json.Marshal(r.RuntimeSpec)
	if err != nil {
		return []byte(""), err
	}
	return b, nil
}

// SetConfig sets the runtime configuration based on JSON input.
func (r *RuntimeConfig) SetConfig(jsonConfig []byte) error {
	if r.RuntimeSpec.RuntimeOciSpec == nil {
		r.RuntimeSpec.RuntimeOciSpec = &r.OciConfig.RuntimeOciSpec
	}
	if r.RuntimeSpec.RuntimeEngineSpec == nil {
		r.RuntimeSpec.RuntimeEngineSpec = &r.EngineConfig.RuntimeEngineSpec
	}
	return json.Unmarshal(jsonConfig, &r.RuntimeSpec)
}
