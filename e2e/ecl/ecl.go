// Copyright (c) 2020, Control Command Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package ecl

import (
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/e2e/internal/testhelper"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/syecl"
)

var KeyMap = map[string]string{
	"key1": "0C5B8C9A5FFC44E2A0AC79851CD6FA281D476DD1",
	"key2": "78F8AD36B0DCB84B707F23853D608DAE21C8CA10",
}

type ctx struct {
	env e2e.TestEnv
}

func (c *ctx) eclConfig(t *testing.T) {
	tmpDir, remove := e2e.MakeTempDir(t, "", "ecl-", "ECL")
	pgpDir, _ := e2e.MakeSyPGPDir(t, tmpDir)
	c.env.KeyringDir = pgpDir

	signed := filepath.Join(tmpDir, "signed.sif")
	signedOne := filepath.Join(tmpDir, "signed_one.sif")
	unsigned := filepath.Join(tmpDir, "unsigned.sif")

	defer func() {
		c.env.KeyringDir = ""
		remove(t)
	}()

	tests := []struct {
		name       string
		command    string
		args       []string
		profile    e2e.Profile
		consoleOps []e2e.SingularityConsoleOp
		config     *syecl.EclConfig
		exit       int
	}{
		{
			name:    "import key1 local",
			command: "key import",
			profile: e2e.UserProfile,
			args:    []string{"testdata/ecl-pgpkeys/key1.asc"},
			consoleOps: []e2e.SingularityConsoleOp{
				e2e.ConsoleSendLine("e2e"),
			},
			exit: 0,
		},
		{
			name:    "import key2 local",
			command: "key import",
			profile: e2e.UserProfile,
			args:    []string{"testdata/ecl-pgpkeys/key2.asc"},
			consoleOps: []e2e.SingularityConsoleOp{
				e2e.ConsoleSendLine("e2e"),
			},
			exit: 0,
		},
		{
			name:    "import key1 global",
			command: "key import",
			profile: e2e.RootProfile,
			args:    []string{"--global", "testdata/ecl-pgpkeys/pubkey1.asc"},
			exit:    0,
		},
		{
			name:    "import key2 global",
			command: "key import",
			profile: e2e.RootProfile,
			args:    []string{"--global", "testdata/ecl-pgpkeys/pubkey2.asc"},
			exit:    0,
		},
		{
			name:    "build signed image",
			command: "build",
			profile: e2e.UserProfile,
			args:    []string{signed, "library://busybox"},
			exit:    0,
		},
		{
			name:    "build unsigned image",
			command: "build",
			profile: e2e.UserProfile,
			args:    []string{unsigned, signed},
			exit:    0,
		},
		{
			name:    "build single signed image",
			command: "build",
			profile: e2e.UserProfile,
			args:    []string{signedOne, signed},
			exit:    0,
		},
		{
			name:    "sign image with key1",
			command: "sign",
			profile: e2e.UserProfile,
			args:    []string{"-k", "0", signed},
			consoleOps: []e2e.SingularityConsoleOp{
				e2e.ConsoleSendLine("e2e"),
			},
			exit: 0,
		},
		{
			name:    "sign image with key2",
			command: "sign",
			profile: e2e.UserProfile,
			args:    []string{"-k", "1", signed},
			consoleOps: []e2e.SingularityConsoleOp{
				e2e.ConsoleSendLine("e2e"),
			},
			exit: 0,
		},
		{
			name:    "single image signature with key1",
			command: "sign",
			profile: e2e.UserProfile,
			args:    []string{"-k", "0", signedOne},
			consoleOps: []e2e.SingularityConsoleOp{
				e2e.ConsoleSendLine("e2e"),
			},
			exit: 0,
		},
		{
			name:    "run with ECL without execgroup",
			command: "exec",
			profile: e2e.UserProfile,
			config: &syecl.EclConfig{
				Activated: true,
			},
			args: []string{signed, "true"},
			exit: 255,
		},
		{
			name:    "run with whitelist key1 and signed image",
			command: "exec",
			profile: e2e.UserProfile,
			config: &syecl.EclConfig{
				Activated: true,
				ExecGroups: []syecl.Execgroup{
					{
						TagName:  "group1",
						ListMode: "whitelist",
						DirPath:  tmpDir,
						KeyFPs:   []string{KeyMap["key1"]},
					},
				},
			},
			args: []string{signed, "true"},
			exit: 0,
		},
		{
			name:    "run with whitelist key2 and signed image",
			command: "exec",
			profile: e2e.UserProfile,
			config: &syecl.EclConfig{
				Activated: true,
				ExecGroups: []syecl.Execgroup{
					{
						TagName:  "group1",
						ListMode: "whitelist",
						DirPath:  tmpDir,
						KeyFPs:   []string{KeyMap["key2"]},
					},
				},
			},
			args: []string{signed, "true"},
			exit: 0,
		},
		{
			name:    "run with whitelist key1 and unsigned image",
			command: "exec",
			profile: e2e.UserProfile,
			config: &syecl.EclConfig{
				Activated: true,
				ExecGroups: []syecl.Execgroup{
					{
						TagName:  "group1",
						ListMode: "whitelist",
						DirPath:  tmpDir,
						KeyFPs:   []string{KeyMap["key1"]},
					},
				},
			},
			args: []string{unsigned, "true"},
			exit: 255,
		},
		{
			name:    "run with whitelist no key and unsigned image",
			command: "exec",
			profile: e2e.UserProfile,
			config: &syecl.EclConfig{
				Activated: true,
				ExecGroups: []syecl.Execgroup{
					{
						TagName:  "group1",
						ListMode: "whitelist",
						DirPath:  tmpDir,
					},
				},
			},
			args: []string{unsigned, "true"},
			exit: 255,
		},
		{
			name:    "run with whitelist fake directory and signed image",
			command: "exec",
			profile: e2e.UserProfile,
			config: &syecl.EclConfig{
				Activated: true,
				ExecGroups: []syecl.Execgroup{
					{
						TagName:  "group1",
						ListMode: "whitelist",
						DirPath:  "/",
						KeyFPs:   []string{KeyMap["key1"], KeyMap["key2"]},
					},
				},
			},
			args: []string{unsigned, "true"},
			exit: 255,
		},
		{
			name:    "run with whitestrict and signed image",
			command: "exec",
			profile: e2e.UserProfile,
			config: &syecl.EclConfig{
				Activated: true,
				ExecGroups: []syecl.Execgroup{
					{
						TagName:  "group1",
						ListMode: "whitestrict",
						DirPath:  tmpDir,
						KeyFPs:   []string{KeyMap["key1"], KeyMap["key2"]},
					},
				},
			},
			args: []string{signed, "true"},
			exit: 0,
		},
		{
			name:    "run with whitestrict and single signed image",
			command: "exec",
			profile: e2e.UserProfile,
			config: &syecl.EclConfig{
				Activated: true,
				ExecGroups: []syecl.Execgroup{
					{
						TagName:  "group1",
						ListMode: "whitestrict",
						DirPath:  tmpDir,
						KeyFPs:   []string{KeyMap["key1"], KeyMap["key2"]},
					},
				},
			},
			args: []string{signedOne, "true"},
			exit: 255,
		},
		{
			name:    "run with whitestrict and unsigned image",
			command: "exec",
			profile: e2e.UserProfile,
			config: &syecl.EclConfig{
				Activated: true,
				ExecGroups: []syecl.Execgroup{
					{
						TagName:  "group1",
						ListMode: "whitestrict",
						DirPath:  tmpDir,
						KeyFPs:   []string{KeyMap["key1"], KeyMap["key2"]},
					},
				},
			},
			args: []string{unsigned, "true"},
			exit: 255,
		},
		{
			name:    "run with blacklist (key1) and signed image",
			command: "exec",
			profile: e2e.UserProfile,
			config: &syecl.EclConfig{
				Activated: true,
				ExecGroups: []syecl.Execgroup{
					{
						TagName:  "group1",
						ListMode: "blacklist",
						DirPath:  tmpDir,
						KeyFPs:   []string{KeyMap["key1"]},
					},
				},
			},
			args: []string{signed, "true"},
			exit: 255,
		},
		{
			name:    "run with blacklist (key2) and single signed image",
			command: "exec",
			profile: e2e.UserProfile,
			config: &syecl.EclConfig{
				Activated: true,
				ExecGroups: []syecl.Execgroup{
					{
						TagName:  "group1",
						ListMode: "blacklist",
						DirPath:  tmpDir,
						KeyFPs:   []string{KeyMap["key2"]},
					},
				},
			},
			args: []string{signedOne, "true"},
			exit: 0,
		},
		{
			name:    "remove key1 from global",
			command: "key remove",
			profile: e2e.RootProfile,
			args:    []string{"--global", KeyMap["key1"]},
			exit:    0,
		},
		{
			name:    "remove key2 from global",
			command: "key remove",
			profile: e2e.RootProfile,
			args:    []string{"--global", KeyMap["key2"]},
			exit:    0,
		},
		{
			name:    "run unsigned with ecl disabled",
			command: "exec",
			profile: e2e.UserProfile,
			args:    []string{unsigned, "true"},
			config:  &syecl.EclConfig{}, // disable ECL
			exit:    0,
		},
	}

	for _, tt := range tests {
		cmdOps := []e2e.SingularityCmdOp{
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(tt.profile),
			e2e.WithCommand(tt.command),
			e2e.WithArgs(tt.args...),
			e2e.PreRun(func(t *testing.T) {
				if tt.config == nil {
					return
				}
				fn := func(t *testing.T) {
					if err := tt.config.ValidateConfig(); err != nil {
						t.Errorf("while validating ecl config: %s", err)
					}
					err := syecl.PutConfig(*tt.config, buildcfg.ECL_FILE)
					if err != nil {
						t.Errorf("while creating ecl config: %s", err)
					}
				}
				e2e.Privileged(fn)(t)
			}),
			e2e.ExpectExit(tt.exit),
		}

		if tt.consoleOps != nil {
			cmdOps = append(cmdOps, e2e.ConsoleRun(tt.consoleOps...))
		}

		c.env.RunSingularity(
			t,
			cmdOps...,
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
		"config": np(c.eclConfig),
	}
}
