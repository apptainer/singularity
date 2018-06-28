// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import "github.com/singularityware/singularity/src/pkg/util/priv"

/*
 * see https://github.com/opencontainers/runtime-spec/blob/master/runtime.md#lifecycle
 * we will run step 8/9 there
 */

// CleanupContainer cleans up the container
func (engine *EngineOperations) CleanupContainer() error {
	if engine.EngineConfig.Network != nil {
		if err := priv.Escalate(); err != nil {
			return err
		}
		if err := engine.EngineConfig.Network.DelNetworks(); err != nil {
			priv.Drop()
			return err
		}
		priv.Drop()
	}
	return nil
}
