// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"net"
)

// PrepareConfig checks and prepares the runtime engine config
func (engine *EngineOperations) PrepareConfig(masterConn net.Conn) error {
	if engine.CommonConfig.EngineName != Name {
		return fmt.Errorf("incorrect engine")
	}

	engine.CommonConfig.OciConfig.SetProcessNoNewPrivileges(true)
	return nil
}
