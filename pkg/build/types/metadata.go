// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package types

// MetaData ...
type MetaData struct {
	// DefaultCommand is the process which should be executed by default when calling
	// "singularity run ... "
	DefaultCommand string `json:"defaultCommand"`

	// Overridable sets whether or not the user supplied arguments to "singularity run ..."
	// can override the default Command.
	Overridable bool `json:"overridable"`

	// DefaultArgs are the default arguments passed to the Command to run in the Container. These
	// can *always* be overridden by arguments given to "singularity run ..."
	DefaultArgs string `json:"defaultArgs"`

	// BaseEnv provides the base environment variables of the container.
	BaseEnv []string `json:"baseEnv"`

	BuildHistory *BuildHistory `json:"buildHistory"`

	//OverylayPartition string

	//PartionBinds map[string][]string
}

// BuildHistory ...
type BuildHistory struct {
	DefinitionHash string `json:"definitionHash"`
	Definition     `json:"definition"`
	Parent         *BuildHistory `json:"parent"`
}
