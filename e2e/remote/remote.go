// Copyright (c) 2020, Control Command Inc. All rights reserved.
// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package remote

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/e2e/internal/testhelper"
)

type ctx struct {
	env e2e.TestEnv
}

// remoteAdd checks the functionality of "singularity remote add" command.
// It Verifies that adding valid endpoints results in success and invalid
// one's results in failure.
func (c ctx) remoteAdd(t *testing.T) {
	config, err := ioutil.TempFile(c.env.TestDir, "testConfig-")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(config.Name()) // clean up

	testPass := []struct {
		name   string
		remote string
		uri    string
	}{
		{"AddCloud", "cloud", "cloud.sylabs.io"},
		{"AddOtherCloud", "other", "cloud.sylabs.io"},
	}

	for _, tt := range testPass {
		argv := []string{"--config", config.Name(), "add", "--no-login", tt.remote, tt.uri}
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("remote"),
			e2e.WithArgs(argv...),
			e2e.ExpectExit(0),
		)
	}

	testFail := []struct {
		name   string
		remote string
		uri    string
	}{
		{"AddExistingRemote", "cloud", "cloud.sylabs.io"},
		{"AddExistingRemoteInvalidURI", "other", "anythingcangohere"},
	}

	for _, tt := range testFail {
		argv := []string{"--config", config.Name(), "add", "--no-login", tt.remote, tt.uri}
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("remote"),
			e2e.WithArgs(argv...),
			e2e.ExpectExit(255),
		)
	}
}

// remoteRemove tests the functionality of "singularity remote remove" command.
// 1. Adds remote endpoints
// 2. Deletes the already added entries
// 3. Verfies that removing an invalid entry results in a failure
func (c ctx) remoteRemove(t *testing.T) {
	config, err := ioutil.TempFile(c.env.TestDir, "testConfig-")
	if err != nil {
		log.Fatal(err)
	}

	defer os.Remove(config.Name()) // clean up

	// Prep config by adding multiple remotes
	add := []struct {
		name   string
		remote string
		uri    string
	}{
		{"addCloud", "cloud", "cloud.sylabs.io"},
		{"addOther", "other", "cloud.sylabs.io"},
	}

	for _, tt := range add {
		argv := []string{"--config", config.Name(), "add", "--no-login", tt.remote, tt.uri}
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("remote"),
			e2e.WithArgs(argv...),
			e2e.ExpectExit(0),
		)
	}

	testPass := []struct {
		name   string
		remote string
	}{
		{"RemoveCloud", "cloud"},
		{"RemoveOther", "other"},
	}

	for _, tt := range testPass {
		argv := []string{"--config", config.Name(), "remove", tt.remote}
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("remote"),
			e2e.WithArgs(argv...),
			e2e.ExpectExit(0),
		)
	}

	testFail := []struct {
		name   string
		remote string
	}{
		{"RemoveNonExistingRemote", "cloud"},
	}

	for _, tt := range testFail {
		argv := []string{"--config", config.Name(), "remove", tt.remote}
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("remote"),
			e2e.WithArgs(argv...),
			e2e.ExpectExit(255),
		)
	}
}

// remoteUse tests the functionality of "singularity remote use" command.
// 1. Tries to use non-existing remote entry
// 2. Adds remote entries and tries to use those
func (c ctx) remoteUse(t *testing.T) {
	config, err := ioutil.TempFile(c.env.TestDir, "testConfig-")
	if err != nil {
		log.Fatal(err)
	}

	defer os.Remove(config.Name()) // clean up

	testFail := []struct {
		name   string
		remote string
	}{
		{"UseNonExistingRemote", "cloud"},
	}

	for _, tt := range testFail {
		argv := []string{"--config", config.Name(), "use", tt.remote}
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("remote"),
			e2e.WithArgs(argv...),
			e2e.ExpectExit(255),
		)
	}

	// Prep config by adding multiple remotes
	add := []struct {
		name   string
		remote string
		uri    string
	}{
		{"addCloud", "cloud", "cloud.sylabs.io"},
		{"addOther", "other", "cloud.sylabs.io"},
	}

	for _, tt := range add {
		argv := []string{"--config", config.Name(), "add", "--no-login", tt.remote, tt.uri}
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("remote"),
			e2e.WithArgs(argv...),
			e2e.ExpectExit(0),
		)
	}

	testPass := []struct {
		name   string
		remote string
	}{
		{"UseFromNothingToRemote", "cloud"},
		{"UseFromRemoteToRemote", "other"},
	}

	for _, tt := range testPass {
		argv := []string{"--config", config.Name(), "use", tt.remote}
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("remote"),
			e2e.WithArgs(argv...),
			e2e.ExpectExit(0),
		)
	}
}

// remoteStatus tests the functionality of "singularity remote status" command.
// 1. Adds remote endpoints
// 2. Verifies that remote status command succeeds on existing endpoints
// 3. Verifies that remote status command fails on non-existing endpoints
func (c ctx) remoteStatus(t *testing.T) {
	config, err := ioutil.TempFile(c.env.TestDir, "testConfig-")
	if err != nil {
		log.Fatal(err)
	}

	defer os.Remove(config.Name()) // clean up

	// Prep config by adding multiple remotes
	add := []struct {
		name   string
		remote string
		uri    string
	}{
		{"addCloud", "cloud", "cloud.sylabs.io"},
		{"addInvalidRemote", "invalid", "notarealendpoint"},
	}

	for _, tt := range add {
		argv := []string{"--config", config.Name(), "add", "--no-login", tt.remote, tt.uri}
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("remote"),
			e2e.WithArgs(argv...),
			e2e.ExpectExit(0),
		)
	}

	testPass := []struct {
		name   string
		remote string
	}{
		{"ValidRemote", "cloud"},
	}

	for _, tt := range testPass {
		argv := []string{"--config", config.Name(), "status", tt.remote}
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("remote"),
			e2e.WithArgs(argv...),
			e2e.ExpectExit(0),
		)
	}

	testFail := []struct {
		name   string
		remote string
	}{
		{"NonExistingRemote", "notaremote"},
		{"NonExistingEndpoint", "invalid"},
	}

	for _, tt := range testFail {
		argv := []string{"--config", config.Name(), "status", tt.remote}
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("remote"),
			e2e.WithArgs(argv...),
			e2e.ExpectExit(255),
		)
	}
}

// remoteList tests the functionality of "singularity remote list" command
func (c ctx) remoteList(t *testing.T) {
	config, err := ioutil.TempFile(c.env.TestDir, "testConfig-")
	if err != nil {
		log.Fatal(err)
	}

	defer os.Remove(config.Name()) // clean up

	testPass := []struct {
		name string
	}{
		{"EmptyConfig"},
	}

	for _, tt := range testPass {
		argv := []string{"--config", config.Name(), "list"}
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("remote"),
			e2e.WithArgs(argv...),
			e2e.ExpectExit(0),
		)
	}

	// Prep config by adding multiple remotes
	add := []struct {
		name   string
		remote string
		uri    string
	}{
		{"addCloud", "cloud", "cloud.sylabs.io"},
		{"addRemote", "remote", "cloud.sylabs.io"},
	}

	for _, tt := range add {
		argv := []string{"--config", config.Name(), "add", "--no-login", tt.remote, tt.uri}
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("remote"),
			e2e.WithArgs(argv...),
			e2e.ExpectExit(0),
		)
	}

	testPass = []struct {
		name string
	}{
		{"PopulatedConfig"},
	}

	for _, tt := range testPass {
		argv := []string{"--config", config.Name(), "list"}
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("remote"),
			e2e.WithArgs(argv...),
			e2e.ExpectExit(0),
		)
	}

	// Prep config by selecting a remote to default to
	use := []struct {
		name   string
		remote string
	}{
		{"useCloud", "cloud"},
	}

	for _, tt := range use {
		argv := []string{"--config", config.Name(), "use", tt.remote}
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("remote"),
			e2e.WithArgs(argv...),
			e2e.ExpectExit(0),
		)
	}

	testPass = []struct {
		name string
	}{
		{"PopulatedConfigWithDefault"},
	}

	for _, tt := range testPass {
		argv := []string{"--config", config.Name(), "list"}
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("remote"),
			e2e.WithArgs(argv...),
			e2e.ExpectExit(0),
		)
	}
}

func (c ctx) remoteTestFlag(t *testing.T) {
	tests := []struct {
		name           string
		cmdArgs        []string
		expectedOutput string
	}{
		{
			name:           "add help",
			cmdArgs:        []string{"add", "--help"},
			expectedOutput: "Create a new singularity remote endpoint",
		},
		{
			name:           "list help",
			cmdArgs:        []string{"list", "--help"},
			expectedOutput: "List all singularity remote endpoints and services that are configured",
		},
		{
			name:           "login help",
			cmdArgs:        []string{"login", "--help"},
			expectedOutput: "Log into a singularity remote endpoint, an OCI/Docker registry or a keyserver using credentials",
		},
		{
			name:           "remove help",
			cmdArgs:        []string{"remove", "--help"},
			expectedOutput: "Remove an existing singularity remote endpoint",
		},
		{
			name:           "status help",
			cmdArgs:        []string{"status", "--help"},
			expectedOutput: "Check the status of the singularity services at an endpoint",
		},
		{
			name:           "use help",
			cmdArgs:        []string{"use", "--help"},
			expectedOutput: "Set a singularity remote endpoint to be actively used",
		},
	}

	for _, tt := range tests {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("remote"),
			e2e.WithArgs(tt.cmdArgs...),
			e2e.ExpectExit(
				0,
				e2e.ExpectOutput(e2e.RegexMatch, `^`+tt.expectedOutput),
			),
		)

	}
}

func (c ctx) remoteBasicLogin(t *testing.T) {
	e2e.EnsureRegistry(t)

	var (
		registry    = fmt.Sprintf("oras://%s", c.env.TestRegistry)
		badRegistry = "oras://bad_registry:5000"
	)

	tests := []struct {
		name       string
		command    string
		args       []string
		stdin      io.Reader
		expectExit int
	}{
		{
			name:       "login username and empty password",
			command:    "remote login",
			args:       []string{"-u", e2e.DefaultUsername, "-p", "", registry},
			expectExit: 255,
		},
		{
			name:       "login empty username and empty password",
			command:    "remote login",
			args:       []string{"-p", "", registry},
			expectExit: 255,
		},
		{
			name:       "login empty username and bad password",
			command:    "remote login",
			args:       []string{"-p", "bad", registry},
			expectExit: 255,
		},
		{
			name:       "login KO",
			command:    "remote login",
			args:       []string{"-u", e2e.DefaultUsername, "-p", "bad", registry},
			expectExit: 255,
		},
		{
			name:       "login without scheme KO",
			command:    "remote login",
			args:       []string{"-u", e2e.DefaultUsername, "-p", e2e.DefaultPassword, c.env.TestRegistry},
			expectExit: 255,
		},
		{
			name:       "login into non-existing keyserver",
			command:    "remote login",
			args:       []string{"http://localhost:11371"},
			expectExit: 255,
		},
		{
			name:       "login OK",
			command:    "remote login",
			args:       []string{"-u", e2e.DefaultUsername, "-p", e2e.DefaultPassword, registry},
			expectExit: 0,
		},
		{
			name:       "login password-stdin",
			command:    "remote login",
			args:       []string{"-u", e2e.DefaultUsername, "--password-stdin", registry},
			stdin:      strings.NewReader(e2e.DefaultPassword),
			expectExit: 0,
		},
		{
			name:       "logout KO",
			command:    "remote logout",
			args:       []string{badRegistry},
			expectExit: 255,
		},
		{
			name:       "logout OK",
			command:    "remote logout",
			args:       []string{registry},
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

func (c ctx) remoteLoginPushPrivate(t *testing.T) {
	e2e.EnsureRegistry(t)
	e2e.EnsureImage(t, c.env)

	var (
		registry = fmt.Sprintf("oras://%s", c.env.TestRegistry)
		repo     = fmt.Sprintf("oras://%s/private/e2e:1.0.0", c.env.TestRegistry)
	)

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
			command:    "remote login",
			args:       []string{"-u", e2e.DefaultUsername, "-p", e2e.DefaultPassword, registry},
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
			command:    "remote logout",
			args:       []string{registry},
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

func (c ctx) remoteKeyserver(t *testing.T) {
	var (
		sylabsKeyserver = "https://keys.sylabs.io"
		testKeyserver   = "http://localhost:11371"
		addKeyserver    = "remote add-keyserver"
		removeKeyserver = "remote remove-keyserver"
	)

	tests := []struct {
		name       string
		command    string
		args       []string
		listLines  []string
		expectExit int
		profile    e2e.Profile
	}{
		{
			name:       "add-keyserver non privileged",
			command:    addKeyserver,
			args:       []string{testKeyserver},
			expectExit: 255,
			profile:    e2e.UserProfile,
		},
		{
			name:    "add-keyserver without order",
			command: addKeyserver,
			args:    []string{"--insecure", testKeyserver},
			listLines: []string{
				"URI                     GLOBAL  INSECURE  ORDER",
				sylabsKeyserver + "  YES     NO        1*",
				testKeyserver + "  YES     YES       2",
			},
			expectExit: 0,
			profile:    e2e.RootProfile,
		},
		{
			name:       "remove-keyserver previous",
			command:    removeKeyserver,
			args:       []string{testKeyserver},
			expectExit: 0,
			profile:    e2e.RootProfile,
		},
		{
			name:       "remove-keyserver non-existent",
			command:    removeKeyserver,
			args:       []string{testKeyserver},
			expectExit: 255,
			profile:    e2e.RootProfile,
		},
		{
			name:       "add-keyserver with order 0",
			command:    addKeyserver,
			args:       []string{"--order", "0", testKeyserver},
			expectExit: 255,
			profile:    e2e.RootProfile,
		},
		{
			name:    "add-keyserver with order 1",
			command: addKeyserver,
			args:    []string{"--order", "1", testKeyserver},
			listLines: []string{
				"URI                     GLOBAL  INSECURE  ORDER",
				testKeyserver + "  YES     NO        1",
				sylabsKeyserver + "  YES     NO        2*",
			},
			expectExit: 0,
			profile:    e2e.RootProfile,
		},
		{
			name:       "add-keyserver duplicate",
			command:    addKeyserver,
			args:       []string{testKeyserver},
			expectExit: 255,
			profile:    e2e.RootProfile,
		},
		{
			name:    "remove-keyserver sylabs",
			command: removeKeyserver,
			args:    []string{sylabsKeyserver},
			listLines: []string{
				"URI                     GLOBAL  INSECURE  ORDER",
				testKeyserver + "  YES     NO        1",
			},
			expectExit: 0,
			profile:    e2e.RootProfile,
		},
		{
			name:       "remove-keyserver primary KO",
			command:    removeKeyserver,
			args:       []string{testKeyserver},
			expectExit: 255,
			profile:    e2e.RootProfile,
		},
		{
			name:    "add-keyserver restore sylabs",
			command: addKeyserver,
			args:    []string{sylabsKeyserver},
			listLines: []string{
				"URI                     GLOBAL  INSECURE  ORDER",
				testKeyserver + "  YES     NO        1",
				sylabsKeyserver + "  YES     NO        2*",
			},
			expectExit: 0,
			profile:    e2e.RootProfile,
		},
		{
			name:    "remove-keyserver primary OK",
			command: removeKeyserver,
			args:    []string{testKeyserver},
			listLines: []string{
				"URI                     GLOBAL  INSECURE  ORDER",
				sylabsKeyserver + "  YES     NO        1*",
			},
			expectExit: 0,
			profile:    e2e.RootProfile,
		},
		{
			name:       "add-keyserver out of order",
			command:    addKeyserver,
			args:       []string{"--order", "100", testKeyserver},
			expectExit: 255,
			profile:    e2e.RootProfile,
		},
	}

	for _, tt := range tests {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(tt.profile),
			e2e.WithCommand(tt.command),
			e2e.WithArgs(tt.args...),
			e2e.PostRun(func(t *testing.T) {
				if t.Failed() || len(tt.listLines) == 0 {
					return
				}
				c.env.RunSingularity(
					t,
					e2e.WithProfile(e2e.UserProfile),
					e2e.WithCommand("remote list"),
					e2e.ExpectExit(
						0,
						e2e.ExpectOutput(
							e2e.ContainMatch,
							strings.Join(tt.listLines, "\n"),
						),
					),
				)
			}),
			e2e.ExpectExit(tt.expectExit),
		)
	}
}

func (c ctx) remoteUseExclusive(t *testing.T) {
	var (
		sylabsRemote = "SylabsCloud"
		testRemote   = "e2e"
	)

	tests := []struct {
		name       string
		command    string
		args       []string
		expectExit int
		profile    e2e.Profile
	}{
		{
			name:       "use exclusive as user",
			command:    "remote use",
			args:       []string{"--exclusive", "--global", testRemote},
			expectExit: 255,
			profile:    e2e.UserProfile,
		},
		{
			name:       "add remote",
			command:    "remote add",
			args:       []string{"--global", testRemote, "cloud.test.com"},
			expectExit: 0,
			profile:    e2e.RootProfile,
		},
		{
			name:       "use remote exclusive with global as root",
			command:    "remote use",
			args:       []string{"--exclusive", "--global", testRemote},
			expectExit: 0,
			profile:    e2e.RootProfile,
		},
		{
			name:       "use remote SylabsCloud as user KO",
			command:    "remote use",
			args:       []string{sylabsRemote},
			expectExit: 255,
			profile:    e2e.UserProfile,
		},
		{
			name:       "remove e2e remote",
			command:    "remote remove",
			args:       []string{"--global", testRemote},
			expectExit: 0,
			profile:    e2e.RootProfile,
		},
		{
			name:       "use remote SylabsCloud as user OK",
			command:    "remote use",
			args:       []string{sylabsRemote},
			expectExit: 0,
			profile:    e2e.UserProfile,
		},
		{
			name:       "add remote",
			command:    "remote add",
			args:       []string{"--global", testRemote, "cloud.test.com"},
			expectExit: 0,
			profile:    e2e.RootProfile,
		},
		{
			name:       "use remote exclusive without global as root",
			command:    "remote use",
			args:       []string{"--exclusive", testRemote},
			expectExit: 0,
			profile:    e2e.RootProfile,
		},
		{
			name:       "use remote SylabsCloud as exclusive",
			command:    "remote use",
			args:       []string{"--exclusive", sylabsRemote},
			expectExit: 0,
			profile:    e2e.RootProfile,
		},
		{
			name:       "use remote e2e as exclusive",
			command:    "remote use",
			args:       []string{"--exclusive", testRemote},
			expectExit: 0,
			profile:    e2e.RootProfile,
		},
		{
			name:       "use remote SylabsCloud as user KO",
			command:    "remote use",
			args:       []string{sylabsRemote},
			expectExit: 255,
			profile:    e2e.UserProfile,
		},
		{
			name:       "remove e2e remote",
			command:    "remote remove",
			args:       []string{"--global", testRemote},
			expectExit: 0,
			profile:    e2e.RootProfile,
		},
		{
			name:       "no default remote set",
			command:    "key search",
			args:       []string{"@"},
			expectExit: 255,
			profile:    e2e.RootProfile,
		},
		{
			name:       "use remote SylabsCloud global",
			command:    "remote use",
			args:       []string{"--global", sylabsRemote},
			expectExit: 0,
			profile:    e2e.RootProfile,
		},
	}

	for _, tt := range tests {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(tt.profile),
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
		"add":                    c.remoteAdd,
		"list":                   c.remoteList,
		"remove":                 c.remoteRemove,
		"status":                 c.remoteStatus,
		"test flag":              c.remoteTestFlag,
		"use":                    c.remoteUse,
		"oci login basic":        np(c.remoteBasicLogin),
		"oci login push private": np(c.remoteLoginPushPrivate),
		"keyserver":              np(c.remoteKeyserver),
		"use exclusive":          np(c.remoteUseExclusive),
	}
}
