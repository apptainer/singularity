/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package config

import (
	"github.com/opencontainers/runtime-spec/specs-go"
)

type ProcessPlatform interface {
	GetRlimits() []specs.POSIXRlimit
	SetRlimits(limits []specs.POSIXRlimit) error
	AddRlimit(rtype string, hard uint64, soft uint64) error
	DelRlimit(rtype string) error
}
