package config

import (
	"encoding/json"

	oci "github.com/singularityware/singularity/pkg/runtime/oci/config"
)

// Runtime template specification
type RuntimeSpec struct {
	RuntimeName       string              `json:"runtimeName"`
	ID                string              `json:"containerID"`
	RuntimeOciSpec    *oci.RuntimeOciSpec `json:"ociConfig"`
	RuntimeEngineSpec RuntimeEngineSpec   `json:"runtimeConfig"`
}

// Runtime engine specification
type RuntimeEngineSpec interface{}

// Runtime engine configuration
type RuntimeEngineConfig struct {
	RuntimeEngineSpec
}

// Generic runtime configuration
type RuntimeConfig struct {
	RuntimeSpec
	OciConfig    oci.RuntimeOciConfig
	EngineConfig RuntimeEngineConfig
}

// Return runtime configuration in JSON format
func (r *RuntimeConfig) GetConfig() ([]byte, error) {
	b, err := json.Marshal(r.RuntimeSpec)
	if err != nil {
		return []byte(""), err
	}
	return b, nil
}

// Set runtime configuration based on JSON input
func (r *RuntimeConfig) SetConfig(jsonConfig []byte) error {
	if r.RuntimeSpec.RuntimeOciSpec == nil {
		r.RuntimeSpec.RuntimeOciSpec = &r.OciConfig.RuntimeOciSpec
	}
	if r.RuntimeSpec.RuntimeEngineSpec == nil {
		r.RuntimeSpec.RuntimeEngineSpec = &r.EngineConfig.RuntimeEngineSpec
	}
	return json.Unmarshal(jsonConfig, &r.RuntimeSpec)
}
