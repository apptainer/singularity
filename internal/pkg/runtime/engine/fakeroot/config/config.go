// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package fakeroot

// Name of the engine
const Name = "fakeroot"

// EngineConfig is the config for the fakeroot engine used to execute
// a command in a fakeroot context
type EngineConfig struct {
	Args     []string `json:"args"`
	Envs     []string `json:"envs"`
	Home     string   `json:"home"`
	BuildEnv bool     `json:"buildEnv"`
}
