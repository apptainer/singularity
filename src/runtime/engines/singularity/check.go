// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"

	"github.com/singularityware/singularity/src/runtime/engines/singularity/config"
)

// CheckConfig checks the runtime engine config
func (engine *EngineOperations) CheckConfig() error {
	if engine.CommonConfig.EngineName != config.Name {
		return fmt.Errorf("incorrect engine")
	}

	return nil
}
