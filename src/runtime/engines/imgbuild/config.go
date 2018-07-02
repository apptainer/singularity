// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package imgbuild

import (
	"encoding/json"

	"github.com/singularityware/singularity/src/pkg/build"
)

// Name of the engine
const Name = "imgbuild"

// EngineConfig is the config for the Singularity engine used to run a minimal image
// during image build process
type EngineConfig struct {
	build.Bundle
}

// MarshalJSON implements json.Marshaler interface
func (c *EngineConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Bundle)
}

// UnmarshalJSON implements json.Unmarshaler interface
func (c *EngineConfig) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &c.Bundle)
}
