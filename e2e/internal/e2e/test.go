// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"testing"
)

// TestEnv holds the test environment variables.
type TestEnv struct {
	RunDisabled   bool
	CmdPath       string
	ImagePath     string
	OrasTestImage string
	TestDir       string
	TestRegistry  string
	KeyringDir    string
}

// RunSingularity is a convinience wrapper for the standalone
// RunSingularity function, ensuring that RunSingularity gets called
// with the correct singularity path as specified by the test
// environment.
func (env TestEnv) RunSingularity(t *testing.T, cmdOps ...SingularityCmdOp) {
	RunSingularity(t, env.CmdPath, cmdOps...)
}

// TestContext defines a test execution context holding
// the current test instance, the test environment and
// the execution profile.
type TestContext struct {
	t       *testing.T
	env     TestEnv
	profile SingularityProfile
}

// NewTestContext creates a new test execution context.
func NewTestContext(t *testing.T, e TestEnv, p SingularityProfile) *TestContext {
	return &TestContext{
		t:       t,
		env:     e,
		profile: p,
	}
}

// Get returns information embedded by the test execution context.
func (c *TestContext) Get() (*testing.T, TestEnv, SingularityProfile) {
	return c.t, c.env, c.profile
}
