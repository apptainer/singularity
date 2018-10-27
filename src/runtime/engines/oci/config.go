// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oci

import (
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/src/runtime/engines/config/oci"
)

// Name of the engine
const Name = "oci"

// EngineConfig is the config for the OCI engine.
type EngineConfig struct {
	BundlePath string      `json:"bundlePath"`
	OciConfig  *oci.Config `json:"ociConfig"`
	State      specs.State `json:"state"`
}

// NewConfig returns singularity.EngineConfig with a parsed FileConfig
func NewConfig() *EngineConfig {
	ret := &EngineConfig{
		OciConfig: &oci.Config{},
	}

	return ret
}

// SetBundlePath sets the container bundle path.
func (e *EngineConfig) SetBundlePath(path string) {
	e.BundlePath = path
}

// GetBundlePath returns the container bundle path.
func (e *EngineConfig) GetBundlePath() string {
	return e.BundlePath
}

// SetState sets the container state as defined by OCI state
// specification
func (e *EngineConfig) SetState(state *specs.State) {
	e.State = *state
}

// GetState returns the container state as defined by OCI state
// specification
func (e *EngineConfig) GetState() *specs.State {
	return &e.State
}
