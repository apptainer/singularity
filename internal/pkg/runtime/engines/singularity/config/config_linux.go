// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"github.com/sylabs/singularity/internal/pkg/cgroups"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config/oci"
	"github.com/sylabs/singularity/pkg/network"
)

// EngineConfig stores both the JSONConfig and the FileConfig
type EngineConfig struct {
	JSON      *JSONConfig      `json:"jsonConfig"`
	OciConfig *oci.Config      `json:"ociConfig"`
	File      *FileConfig      `json:"-"`
	Network   *network.Setup   `json:"-"`
	Cgroups   *cgroups.Manager `json:"-"`
}
