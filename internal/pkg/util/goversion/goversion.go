// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build go1.10

package goversion

// __BUILD_REQUIRES_GO_VERSION_1_10_OR_LATER__ provides a human-readable
// error message when building this package with an unsupported version
// of the Go compiler.
//
// Keep the name of this variable in sync with the minimum required
// version specified in the build tag above.
//
// nolint:golint
const __BUILD_REQUIRES_GO_VERSION_1_10_OR_LATER__ = uint8(0)
