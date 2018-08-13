// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package imgbuild

import (
	"github.com/singularityware/singularity/src/pkg/util/engine/security"
)

// NamespaceFlags uses the default OciNamespaceFlags function
func (e *EngineOperations) NamespaceFlags() uint {
	return security.OciNamespaceFlags(e.CommonConfig.OciConfig.Linux)
}
