// Copyright (c) 2020, Control Command Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package login

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/e2e/internal/testhelper"
)

type ctx struct {
	env e2e.TestEnv
}

func (c ctx) testBasicLogin(t *testing.T) {
	e2e.PrepRegistry(t, c.env)

	tests := []struct {
		name       string
		command    string
		args       []string
		stdin      io.Reader
		expectExit int
	}{
		{
			name:       "login username and empty password",
			command:    "login",
			args:       []string{"-u", e2e.DefaultUsername, "-p", "", c.env.TestRegistry},
			expectExit: 255,
		},
		{
			name:       "login empty username and empty password",
			command:    "login",
			args:       []string{"-p", "", c.env.TestRegistry},
			expectExit: 255,
		},
		{
			name:       "login empty username and bad password",
			command:    "login",
			args:       []string{"-p", "bad", c.env.TestRegistry},
			expectExit: 255,
		},
		{
			name:       "login KO",
			command:    "login",
			args:       []string{"-u", e2e.DefaultUsername, "-p", "bad", c.env.TestRegistry},
			expectExit: 255,
		},
		{
			name:       "login OK",
			command:    "login",
			args:       []string{"-u", e2e.DefaultUsername, "-p", e2e.DefaultPassword, c.env.TestRegistry},
			expectExit: 0,
		},
		{
			name:       "login password-stdin",
			command:    "login",
			args:       []string{"-u", e2e.DefaultUsername, "--password-stdin", c.env.TestRegistry},
			stdin:      strings.NewReader(e2e.DefaultPassword),
			expectExit: 0,
		},
		{
			name:       "logout KO",
			command:    "logout",
			args:       []string{"bad_registry:5000"},
			expectExit: 255,
		},
		{
			name:       "logout OK",
			command:    "logout",
			args:       []string{c.env.TestRegistry},
			expectExit: 0,
		},
	}

	for _, tt := range tests {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithStdin(tt.stdin),
			e2e.WithCommand(tt.command),
			e2e.WithArgs(tt.args...),
			e2e.ExpectExit(tt.expectExit),
		)
	}
}

func (c ctx) testPrivateWithLogin(t *testing.T) {
	e2e.PrepRegistry(t, c.env)
	e2e.EnsureImage(t, c.env)

	repo := fmt.Sprintf("oras://%s/private/e2e:1.0.0", c.env.TestRegistry)

	tests := []struct {
		name       string
		command    string
		args       []string
		expectExit int
	}{
		{
			name:       "push before login",
			command:    "push",
			args:       []string{c.env.ImagePath, repo},
			expectExit: 255,
		},
		{
			name:       "login",
			command:    "login",
			args:       []string{"-u", e2e.DefaultUsername, "-p", e2e.DefaultPassword, c.env.TestRegistry},
			expectExit: 0,
		},
		{
			name:       "push after login",
			command:    "push",
			args:       []string{c.env.ImagePath, repo},
			expectExit: 0,
		},
		{
			name:       "logout",
			command:    "logout",
			args:       []string{c.env.TestRegistry},
			expectExit: 0,
		},
	}

	for _, tt := range tests {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand(tt.command),
			e2e.WithArgs(tt.args...),
			e2e.ExpectExit(tt.expectExit),
		)
	}
}

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) testhelper.Tests {
	c := ctx{
		env: env,
	}

	np := testhelper.NoParallel

	return testhelper.Tests{
		"basic":   np(c.testBasicLogin),
		"private": np(c.testPrivateWithLogin),
	}
}
