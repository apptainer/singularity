// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build !linux

package singularity

import (
	"github.com/sylabs/singularity/internal/pkg/runtime/engine/config/oci"
	"github.com/sylabs/singularity/pkg/runtime/engine/config"
)

// EngineConfig stores both the JSONConfig and the FileConfig
type EngineConfig struct {
	JSON      *JSONConfig        `json:"jsonConfig"`
	OciConfig *oci.Config        `json:"ociConfig"`
	File      *config.FileConfig `json:"-"`
}

// NewConfig returns singularity.EngineConfig with a parsed FileConfig
func NewConfig() *EngineConfig {
	ret := &EngineConfig{
		JSON:      new(JSONConfig),
		OciConfig: new(oci.Config),
		File:      new(config.FileConfig),
	}

	return ret
}
