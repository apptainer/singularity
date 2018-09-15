// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oci

import (
	"encoding/json"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/opencontainers/runtime-tools/generate"
)

// Config is the OCI runtime configuration.
type Config struct {
	generate.Generator
	specs.Spec
}

// MarshalJSON is for json.Marshaler
func (c *Config) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(&c.Spec)

	return b, err
}

// UnmarshalJSON is for json.Unmarshaler
func (c *Config) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &c.Spec); err != nil {
		return err
	}
	c.Generator = generate.Generator{Config: &c.Spec}
	return nil
}
