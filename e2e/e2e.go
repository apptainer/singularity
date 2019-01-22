// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"os"
	"testing"

	"github.com/sylabs/singularity/e2e/suite"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
)

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(t *testing.T) {
	if os.Getenv("SINGULARITY_E2E") == "" {
		t.Skip("Skipping e2e tests, SINGULARITY_E2E not set")
	} else {
		run(t)
	}
}

func run(t *testing.T) {
	suite.Run(t)
}

func init() {
	useragent.InitValue(buildcfg.PACKAGE_NAME, buildcfg.PACKAGE_VERSION)
}
