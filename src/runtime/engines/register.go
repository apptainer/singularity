// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package engines

import (
	"fmt"

	"github.com/singularityware/singularity/src/pkg/sylog"
)

// registeredEngines contains a map relating an Engine name to a ContainerLauncher
// created with a default config. A new ContainerLauncher can later override the config
// to enable different container process launchings
var registeredEngines map[string]*ContainerLauncher

// NewContainerLauncher will return a ContainerLauncher that uses the Engine named "name"
// and the config contained in "jsonConfig"
func NewContainerLauncher(name string, jsonConfig []byte) (launcher *ContainerLauncher, err error) {
	sylog.Debugf("Attempting to create ContainerLauncher using %s Engine\n", name)
	launcher, ok := registeredEngines[name]

	if !ok {
		sylog.Errorf("Runtime engine %s does not exist", name)
		return nil, fmt.Errorf("runtime engine %s does not exist", name)
	}

	if err := launcher.SetConfig(jsonConfig); err != nil {
		sylog.Errorf("Unable to set %s runtime config: %v\n", name, err)
		return nil, err
	}

	return launcher, nil
}

// Register is used to register a specific engine in the registeredEngines
// map. This should be called from the init() function of a package implementing
// a data type satisfying the Engine interface
func Register(e Engine, name string) {
	l := &ContainerLauncher{
		Engine:        e,
		RuntimeConfig: e.InitConfig(),
	}

	registeredEngines[name] = l
	if l.RuntimeConfig == nil {
		sylog.Fatalf("failed to initialize %s engine\n", name)
	}
}

func init() {
	registeredEngines = make(map[string]*ContainerLauncher)
}
