// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package imgbuild

import (
	"github.com/sylabs/singularity/internal/pkg/build/types"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config/oci"
)

// Name of the engine
const Name = "imgbuild"

// EngineConfig is the config for the Singularity engine used to run a minimal image
// during image build process
type EngineConfig struct {
	types.Bundle `json:"bundle"`
	OciConfig    *oci.Config `json:"ociConfig"`
}
