// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Package goversion performs a build-time version check for the minimum
// required version of the Go compiler.
//
// A blank import in all the code that requires a specific Go version is
// sufficient to trigger a build failure like:
//
//     ...
//     ../internal/pkg/util/goversion/version_check.go:19:9: undefined: __BUILD_REQUIRES_GO_VERSION_1_10_OR_LATER__
//
//
// This is based on the technique presented at
// https://github.com/theckman/goconstraint
package goversion

// keep the variable here in sync with the mininum required version
// specified in goversion.go
var _ = __BUILD_REQUIRES_GO_VERSION_1_10_OR_LATER__
