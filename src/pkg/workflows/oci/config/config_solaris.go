// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package config

// Platform represents the current platform
const Platform = "solaris"

// RuntimeOciPlatform is the OCI runtime platform.
type RuntimeOciPlatform struct {
	Linux   interface{}
	Solaris RuntimeOciSolaris
	Windows interface{}
}
