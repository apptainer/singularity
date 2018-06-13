// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

/*
 * see https://github.com/opencontainers/runtime-spec/blob/master/runtime.md#lifecycle
 * we will run step 8/9 there
 */

// CleanupContainer cleans up the container
func (engine *EngineOperations) CleanupContainer() error {
	return nil
}
