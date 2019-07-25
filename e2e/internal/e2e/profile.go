// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"strings"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/fakeroot"
	"github.com/sylabs/singularity/internal/pkg/test/tool/require"
	"github.com/sylabs/singularity/internal/pkg/util/user"
)

// SingularityProfile represents a Singularity execution profile
// provided to RunSingularity in order to setup the corresponding
// execution environment like using privileges or not and injecting
// command arguments if required.
type SingularityProfile uint8

const (
	// noProfile is the equivalent of no profile and for internal use only.
	noProfile SingularityProfile = iota
	// UserProfile is the execution profile with a regular user.
	UserProfile
	// RootProfile is the execution profile with root.
	RootProfile
	// FakerootProfile is the execution profile with fakeroot.
	FakerootProfile
	// UserNamespaceProfile is the execution profile with a user namespace.
	UserNamespaceProfile
	// RootUserNamespaceProfile is the execution profile with root and a user namespace.
	RootUserNamespaceProfile
)

// Profiles groups all Singularity execution profiles.
var Profiles = []SingularityProfile{
	UserProfile,
	RootProfile,
	FakerootProfile,
	UserNamespaceProfile,
	// NEED FIX: disabled until tests are fixed to run it correctly
	// RootUserNamespaceProfile,
}

// privileged returns if the profile requires to run with
// privileges or not.
func (p SingularityProfile) privileged() bool {
	switch p {
	case RootProfile, RootUserNamespaceProfile:
		return true
	}
	return false
}

// withArgs returns Singularity arguments for the corresponding
// profile if any.
func (p SingularityProfile) withArgs(s *singularityCmd) []string {
	switch p {
	case FakerootProfile:
		// fakeroot is available for the following commands
		commands := []string{
			"shell",
			"exec",
			"run",
			"test",
			"instance start",
			"build",
		}

		scmd := strings.Join(s.cmd, " ")
		for _, c := range commands {
			if scmd == c {
				return append([]string{"--fakeroot"}, s.args...)
			}
		}
	case UserNamespaceProfile, RootUserNamespaceProfile:
		// user namespace is available for the following commands
		commands := []string{
			"shell",
			"exec",
			"run",
			"test",
			"instance start",
		}

		scmd := strings.Join(s.cmd, " ")
		for _, c := range commands {
			if scmd == c {
				return append([]string{"--userns"}, s.args...)
			}
		}
	}
	return s.args
}

// Require checks and ensures that the corresponding execution
// profile has all the requirements, the current test is skipped
// if not.
func (p SingularityProfile) Require(t *testing.T) {
	switch p {
	case UserNamespaceProfile, RootUserNamespaceProfile:
		require.UserNamespace(t)
	case FakerootProfile:
		require.UserNamespace(t)
		// now check that current user has valid mappings
		// in /etc/subuid and /etc/subgid
		if _, err := fakeroot.GetIDRange(fakeroot.SubUIDFile, uint32(origUID)); err != nil {
			t.Fatalf("fakeroot configuration error: %s", err)
		}
		if _, err := fakeroot.GetIDRange(fakeroot.SubGIDFile, uint32(origUID)); err != nil {
			t.Fatalf("fakeroot configuration error: %s", err)
		}
	}
}

// In returns if one of the provided profiles corresponds to the current one.
// Practically, this function is part of the utility functions that are
// available to developers to handle tests that are relevant only to a
// sub-set of the existing profiles. Tests are assumed by default to
// be relevant in all existing profiles. If not, it is the responsibility
// of the developer to explicitly exclude profiles that are not applicable
// through a `PreRun()` function. This function is part of the capabilities
// provided by the E2E framework to ease such a task.
func (p SingularityProfile) In(profiles ...SingularityProfile) bool {
	for _, profile := range profiles {
		if profile == p {
			return true
		}
	}
	return false
}

// User returns user information for the corresponding profile.
func (p SingularityProfile) User(t *testing.T) *user.User {
	switch p {
	case FakerootProfile, RootProfile, RootUserNamespaceProfile:
		u, err := user.GetPwUID(0)
		if err != nil {
			t.Fatalf("failed to retrieve root user information: %s", err)
		}
		return u
	}
	u, err := user.GetPwUID(uint32(origUID))
	if err != nil {
		t.Fatalf("failed to retrieve original user information: %s", err)
	}
	return u
}

// Name returns the profile name.
func (p SingularityProfile) Name() string {
	return strings.TrimSuffix(p.String(), "Profile")
}

// String returns the string representation of the profile.
func (p SingularityProfile) String() string {
	switch p {
	case UserProfile:
		return "UserProfile"
	case RootProfile:
		return "RootProfile"
	case FakerootProfile:
		return "FakerootProfile"
	case UserNamespaceProfile:
		return "UserNamespaceProfile"
	case RootUserNamespaceProfile:
		return "RootUserNamespaceProfile"
	}
	return ""
}
