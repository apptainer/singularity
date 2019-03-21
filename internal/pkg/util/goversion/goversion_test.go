// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package goversion_test

import (
	"testing"

	_ "github.com/sylabs/singularity/internal/pkg/util/goversion"
)

// TestVersion tests that a supported Go version is being used. The
// blank import above will trigger the version check and go test will
// report failure in case of an unsupported Go version.
func TestVersion(*testing.T) {}
