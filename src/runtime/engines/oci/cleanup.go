// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oci

import (
	"encoding/json"
	"net"

	"github.com/sylabs/singularity/src/pkg/sylog"
)

// CleanupContainer cleans up the container
func (engine *EngineOperations) CleanupContainer() error {
	if engine.EngineConfig.Cgroups != nil {
		engine.EngineConfig.Cgroups.Remove()
	}

	if err := engine.updateState("stopped"); err != nil {
		return err
	}

	socketKey := "io.sylabs.oci.runtime.cri-sync-socket"

	if socketPath, ok := engine.EngineConfig.OciConfig.Annotations[socketKey]; ok {
		c, err := net.Dial("unix", socketPath)
		if err != nil {
			sylog.Warningf("failed to connect to cri sync socket: %s", err)
		} else {
			defer c.Close()

			data, err := json.Marshal(engine.EngineConfig.State)
			if err != nil {
				sylog.Warningf("failed to marshal state data: %s", err)
			} else if _, err := c.Write(data); err != nil {
				sylog.Warningf("failed to send state over socket: %s", err)
			}
		}
	}

	return nil
}
