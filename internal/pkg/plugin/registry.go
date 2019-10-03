// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the URIs of this project regarding your
// rights to use or distribute this software.

package plugin

import (
	"github.com/sylabs/singularity/pkg/plugin"
)

var cliMutators []CLIMutator

type CLIMutator struct {
	PluginName string
	plugin.CLIMutator
}

var engineConfigMutators []EngineConfigMutator

type EngineConfigMutator struct {
	PluginName string
	plugin.EngineConfigMutator
}

func CLIMutators() []CLIMutator {
	return cliMutators
}

func EngineConfigMutators() []EngineConfigMutator {
	return engineConfigMutators
}

type registrar struct {
	pluginName string
}

func (r registrar) AddCLIMutator(m plugin.CLIMutator) error {
	cliMutators = append(cliMutators, CLIMutator{PluginName: r.pluginName, CLIMutator: m})
	return nil
}

func (r registrar) AddEngineConfigMutator(m plugin.EngineConfigMutator) error {
	engineConfigMutators = append(engineConfigMutators, EngineConfigMutator{
		PluginName:          r.pluginName,
		EngineConfigMutator: m,
	})
	return nil
}
