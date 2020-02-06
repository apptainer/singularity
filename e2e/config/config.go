// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package config

import (
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/e2e/internal/testhelper"
	"github.com/sylabs/singularity/internal/pkg/test/tool/require"
	"github.com/sylabs/singularity/internal/pkg/util/user"
)

type configTests struct {
	env e2e.TestEnv
}

func (c configTests) configGlobal(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	setDirective := func(t *testing.T, directive, value string) {
		c.env.RunSingularity(
			t,
			e2e.WithProfile(e2e.RootProfile),
			e2e.WithCommand("config global"),
			e2e.WithArgs("--set", directive, value),
			e2e.ExpectExit(0),
		)
	}
	resetDirective := func(t *testing.T, directive string) {
		c.env.RunSingularity(
			t,
			e2e.WithProfile(e2e.RootProfile),
			e2e.WithCommand("config global"),
			e2e.WithArgs("--reset", directive),
			e2e.ExpectExit(0),
		)
	}

	u := e2e.UserProfile.HostUser(t)
	g, err := user.GetGrGID(u.GID)
	if err != nil {
		t.Fatalf("could not retrieve user group information: %s", err)
	}

	tests := []struct {
		name              string
		argv              []string
		profile           e2e.Profile
		addRequirementsFn func(*testing.T)
		cwd               string
		directive         string
		directiveValue    string
		exit              int
		resultOp          e2e.SingularityCmdResultOp
	}{
		{
			name: "AllowSetuid",
			argv: []string{c.env.ImagePath, "true"},
			// We are testing if we fall back to user namespace without `--userns`
			// so we need to use the UserProfile, and check separately if userns
			// support is possible.
			profile:           e2e.UserProfile,
			addRequirementsFn: require.UserNamespace,
			directive:         "allow setuid",
			directiveValue:    "no",
			exit:              0,
		},
		{
			name:           "MaxLoopDevices",
			argv:           []string{c.env.ImagePath, "true"},
			profile:        e2e.UserProfile,
			directive:      "max loop devices",
			directiveValue: "0",
			exit:           255,
		},
		{
			name:           "AllowPidNsNo",
			argv:           []string{"--pid", "--no-init", c.env.ImagePath, "/bin/sh", "-c", "echo $$"},
			profile:        e2e.UserProfile,
			directive:      "allow pid ns",
			directiveValue: "no",
			exit:           0,
			resultOp:       e2e.ExpectOutput(e2e.UnwantedMatch, "1"),
		},
		{
			name:           "AllowPidNsYes",
			argv:           []string{"--pid", "--no-init", c.env.ImagePath, "/bin/sh", "-c", "echo $$"},
			profile:        e2e.UserProfile,
			directive:      "allow pid ns",
			directiveValue: "yes",
			exit:           0,
			resultOp:       e2e.ExpectOutput(e2e.ExactMatch, "1"),
		},
		{
			name:           "ConfigPasswdNo",
			argv:           []string{c.env.ImagePath, "grep", "/etc/passwd.*- tmpfs", "/proc/self/mountinfo"},
			profile:        e2e.UserProfile,
			directive:      "config passwd",
			directiveValue: "no",
			exit:           1,
		},
		{
			name:           "ConfigPasswdYes",
			argv:           []string{c.env.ImagePath, "grep", "/etc/passwd.*- tmpfs", "/proc/self/mountinfo"},
			profile:        e2e.UserProfile,
			directive:      "config passwd",
			directiveValue: "yes",
			exit:           0,
		},
		{
			name:           "ConfigGroupNo",
			argv:           []string{c.env.ImagePath, "grep", "/etc/group.*- tmpfs", "/proc/self/mountinfo"},
			profile:        e2e.UserProfile,
			directive:      "config group",
			directiveValue: "no",
			exit:           1,
		},
		{
			name:           "ConfigGroupYes",
			argv:           []string{c.env.ImagePath, "grep", "/etc/group.*- tmpfs", "/proc/self/mountinfo"},
			profile:        e2e.UserProfile,
			directive:      "config group",
			directiveValue: "yes",
			exit:           0,
		},
		{
			name:           "ConfigResolvConfNo",
			argv:           []string{c.env.ImagePath, "grep", "/etc/resolv.conf.*- tmpfs", "/proc/self/mountinfo"},
			profile:        e2e.UserProfile,
			directive:      "config resolv_conf",
			directiveValue: "no",
			exit:           1,
		},
		{
			name:           "ConfigResolvConfYes",
			argv:           []string{c.env.ImagePath, "grep", "/etc/resolv.conf.*- tmpfs", "/proc/self/mountinfo"},
			profile:        e2e.UserProfile,
			directive:      "config resolv_conf",
			directiveValue: "yes",
			exit:           0,
		},
		{
			name:           "MountProcNo",
			argv:           []string{c.env.ImagePath, "test", "-d", "/proc/self"},
			profile:        e2e.UserProfile,
			directive:      "mount proc",
			directiveValue: "no",
			exit:           1,
		},
		{
			name:           "MountProcYes",
			argv:           []string{c.env.ImagePath, "test", "-d", "/proc/self"},
			profile:        e2e.UserProfile,
			directive:      "mount proc",
			directiveValue: "yes",
			exit:           0,
		},
		{
			name:           "MountSysNo",
			argv:           []string{c.env.ImagePath, "test", "-d", "/sys/kernel"},
			profile:        e2e.UserProfile,
			directive:      "mount sys",
			directiveValue: "no",
			exit:           1,
		},
		{
			name:           "MountSysYes",
			argv:           []string{c.env.ImagePath, "test", "-d", "/sys/kernel"},
			profile:        e2e.UserProfile,
			directive:      "mount sys",
			directiveValue: "yes",
			exit:           0,
		},
		{
			name:           "MountDevNo",
			argv:           []string{c.env.ImagePath, "test", "-d", "/dev/pts"},
			profile:        e2e.UserProfile,
			directive:      "mount dev",
			directiveValue: "no",
			exit:           1,
		},
		{
			name:           "MountDevMinimal",
			argv:           []string{c.env.ImagePath, "test", "-b", "/dev/loop0"},
			profile:        e2e.UserProfile,
			directive:      "mount dev",
			directiveValue: "minimal",
			exit:           1,
		},
		{
			name:           "MountDevYes",
			argv:           []string{c.env.ImagePath, "test", "-b", "/dev/loop0"},
			profile:        e2e.UserProfile,
			directive:      "mount dev",
			directiveValue: "yes",
			exit:           0,
		},
		// just test 'mount devpts = no' as yes depends of kernel version
		{
			name:           "MountDevPtsNo",
			argv:           []string{"-C", c.env.ImagePath, "test", "-d", "/dev/pts"},
			profile:        e2e.UserProfile,
			directive:      "mount devpts",
			directiveValue: "no",
			exit:           1,
		},
		{
			name:           "MountHomeNo",
			argv:           []string{c.env.ImagePath, "test", "-d", u.Dir},
			profile:        e2e.UserProfile,
			cwd:            "/",
			directive:      "mount home",
			directiveValue: "no",
			exit:           1,
		},
		{
			name:           "MountHomeYes",
			argv:           []string{c.env.ImagePath, "test", "-d", u.Dir},
			profile:        e2e.UserProfile,
			cwd:            "/",
			directive:      "mount home",
			directiveValue: "yes",
			exit:           0,
		},
		{
			name:           "MountTmpNo",
			argv:           []string{c.env.ImagePath, "test", "-d", c.env.TestDir},
			profile:        e2e.UserProfile,
			directive:      "mount tmp",
			directiveValue: "no",
			exit:           1,
		},
		{
			name:           "MountTmpYes",
			argv:           []string{c.env.ImagePath, "test", "-d", c.env.TestDir},
			profile:        e2e.UserProfile,
			directive:      "mount tmp",
			directiveValue: "yes",
			exit:           0,
		},
		{
			name:           "BindPathPasswd",
			argv:           []string{c.env.ImagePath, "test", "-f", "/passwd"},
			profile:        e2e.UserProfile,
			directive:      "bind path",
			directiveValue: "/etc/passwd:/passwd",
			exit:           0,
		},
		{
			name:           "UserBindControlNo",
			argv:           []string{"--bind", "/etc/passwd:/passwd", c.env.ImagePath, "test", "-f", "/passwd"},
			profile:        e2e.UserProfile,
			directive:      "user bind control",
			directiveValue: "no",
			exit:           1,
		},
		{
			name:           "UserBindControlYes",
			argv:           []string{"--bind", "/etc/passwd:/passwd", c.env.ImagePath, "test", "-f", "/passwd"},
			profile:        e2e.UserProfile,
			directive:      "user bind control",
			directiveValue: "yes",
			exit:           0,
		},
		// overlay may or not be available, just test with no
		{
			name:           "EnableOverlayNo",
			argv:           []string{c.env.ImagePath, "grep", "\\- overlay overlay", "/proc/self/mountinfo"},
			profile:        e2e.UserProfile,
			directive:      "enable overlay",
			directiveValue: "no",
			exit:           1,
		},
		// use user namespace profile to force underlay use
		{
			name:           "EnableUnderlayNo",
			argv:           []string{"--bind", "/etc/passwd:/passwd", c.env.ImagePath, "test", "-f", "/passwd"},
			profile:        e2e.UserNamespaceProfile,
			directive:      "enable underlay",
			directiveValue: "no",
			exit:           255,
		},
		{
			name:           "EnableUnderlayYes",
			argv:           []string{"--bind", "/etc/passwd:/passwd", c.env.ImagePath, "test", "-f", "/passwd"},
			profile:        e2e.UserNamespaceProfile,
			directive:      "enable underlay",
			directiveValue: "yes",
			exit:           0,
		},
		// test image is owned by root:root
		{
			name:           "LimitContainerOwnersUser",
			argv:           []string{c.env.ImagePath, "true"},
			profile:        e2e.UserProfile,
			directive:      "limit container owners",
			directiveValue: u.Name,
			exit:           255,
		},
		{
			name:           "LimitContainerOwnersUserAndRoot",
			argv:           []string{c.env.ImagePath, "true"},
			profile:        e2e.UserProfile,
			directive:      "limit container owners",
			directiveValue: u.Name + ", root",
			exit:           0,
		},
		{
			name:           "LimitContainerGroupsUser",
			argv:           []string{c.env.ImagePath, "true"},
			profile:        e2e.UserProfile,
			directive:      "limit container groups",
			directiveValue: g.Name,
			exit:           255,
		},
		{
			name:           "LimitContainerGroupsUserAndRoot",
			argv:           []string{c.env.ImagePath, "true"},
			profile:        e2e.UserProfile,
			directive:      "limit container groups",
			directiveValue: g.Name + ", root",
			exit:           0,
		},
		{
			name:           "LimitContainerPathsProc",
			argv:           []string{c.env.ImagePath, "true"},
			profile:        e2e.UserProfile,
			directive:      "limit container paths",
			directiveValue: "/proc",
			exit:           255,
		},
		{
			name:           "LimitContainerPathsTestdir",
			argv:           []string{c.env.ImagePath, "true"},
			profile:        e2e.UserProfile,
			directive:      "limit container paths",
			directiveValue: c.env.TestDir,
			exit:           0,
		},
		{
			name:           "AllowContainerSquashfsNo",
			argv:           []string{c.env.ImagePath, "true"},
			profile:        e2e.UserProfile,
			directive:      "allow container squashfs",
			directiveValue: "no",
			exit:           255,
		},
		{
			name:           "AllowContainerSquashfsYes",
			argv:           []string{c.env.ImagePath, "true"},
			profile:        e2e.UserProfile,
			directive:      "allow container squashfs",
			directiveValue: "yes",
			exit:           0,
		},
		{
			name:           "AllowContainerDirNo",
			argv:           []string{c.env.ImagePath, "true"},
			profile:        e2e.UserNamespaceProfile,
			directive:      "allow container dir",
			directiveValue: "no",
			exit:           255,
		},
		{
			name:           "AllowContainerDirYes",
			argv:           []string{c.env.ImagePath, "true"},
			profile:        e2e.UserNamespaceProfile,
			directive:      "allow container dir",
			directiveValue: "yes",
			exit:           0,
		},
	}

	for _, tt := range tests {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(tt.profile),
			e2e.WithDir(tt.cwd),
			e2e.PreRun(func(t *testing.T) {
				if tt.addRequirementsFn != nil {
					tt.addRequirementsFn(t)
				}
				setDirective(t, tt.directive, tt.directiveValue)
			}),
			e2e.PostRun(func(t *testing.T) {
				resetDirective(t, tt.directive)
			}),
			e2e.WithCommand("exec"),
			e2e.WithArgs(tt.argv...),
			e2e.ExpectExit(tt.exit, tt.resultOp),
		)
	}
}

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) func(*testing.T) {
	c := configTests{
		env: env,
	}

	return testhelper.TestRunner(map[string]func(*testing.T){
		"config global": c.configGlobal, // test various global configuration
	})
}
