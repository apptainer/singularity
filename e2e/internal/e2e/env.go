// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import "testing"

// TestEnv defines the environment used to execute a single e2e test.
type TestEnv struct {
	RunDisabled   bool
	CmdPath       string
	ImagePath     string
	OrasTestImage string
	TestDir       string
	TestRegistry  string
}

// RunSingularity is a convenient wrapper for the standalone
// RunSingularity function, ensuring that RunSingularity gets called
// with the correct singularity path as specified by the test
// environment.
func (env TestEnv) RunSingularity(t *testing.T, cmdOps ...SingularityCmdOp) {
	RunSingularity(t, env.CmdPath, cmdOps...)
}
