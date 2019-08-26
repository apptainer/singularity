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

const (
	userProfile              = "UserProfile"
	rootProfile              = "RootProfile"
	fakerootProfile          = "FakerootProfile"
	userNamespaceProfile     = "UserNamespaceProfile"
	rootUserNamespaceProfile = "RootUserNamespaceProfile"
)

var (
	// UserProfile is the execution profile for a regular user.
	UserProfile = Profiles[userProfile]
	// RootProfile is the execution profile for root.
	RootProfile = Profiles[rootProfile]
	// FakerootProfile is the execution profile for fakeroot.
	FakerootProfile = Profiles[fakerootProfile]
	// UserNamespaceProfile is the execution profile for a regular user and a user namespace.
	UserNamespaceProfile = Profiles[userNamespaceProfile]
	// RootUserNamespaceProfile is the execution profile for root and a user namespace.
	RootUserNamespaceProfile = Profiles[rootUserNamespaceProfile]
)

// Profile represents various properties required to run an E2E test
// under a particular user profile. A profile can define if `RunSingularity`
// will run with privileges (`privileged`), if an option flag is injected
// (`singularityOption`), the option injection is also controllable for a
// subset of singularity commands with `optionForCommands`. A profile can
// also set a default current working directory via `defaultCwd`, profile
// like "RootUserNamespace" need to run from a directory owned by root. A
// profile can also have two identities (eg: "Fakeroot" profile), a host
// identity corresponding to user ID `hostUID` and a container identity
// corresponding to user ID `containerUID`.
type Profile struct {
	name              string           // name of the profile
	privileged        bool             // is the profile will run with privileges ?
	hostUID           int              // user ID corresponding to the profile outside container
	containerUID      int              // user ID corresponding to the profile inside container
	defaultCwd        string           // the default current working directory if specified
	requirementsFn    func(*testing.T) // function checking requirements for the profile
	singularityOption string           // option added to singularity command for the profile
	optionForCommands []string         // singularity commands concerned by the option to be added
}

// Profiles defines all available profiles.
var Profiles = map[string]Profile{
	userProfile: {
		name:              "User",
		privileged:        false,
		hostUID:           origUID,
		containerUID:      origUID,
		defaultCwd:        "",
		requirementsFn:    nil,
		singularityOption: "",
		optionForCommands: []string{},
	},
	rootProfile: {
		name:              "Root",
		privileged:        true,
		hostUID:           0,
		containerUID:      0,
		defaultCwd:        "",
		requirementsFn:    nil,
		singularityOption: "",
		optionForCommands: []string{},
	},
	fakerootProfile: {
		name:              "Fakeroot",
		privileged:        false,
		hostUID:           origUID,
		containerUID:      0,
		defaultCwd:        "",
		requirementsFn:    fakerootRequirements,
		singularityOption: "--fakeroot",
		optionForCommands: []string{"shell", "exec", "run", "test", "instance start", "build"},
	},
	userNamespaceProfile: {
		name:              "UserNamespace",
		privileged:        false,
		hostUID:           origUID,
		containerUID:      origUID,
		defaultCwd:        "",
		requirementsFn:    require.UserNamespace,
		singularityOption: "--userns",
		optionForCommands: []string{"shell", "exec", "run", "test", "instance start"},
	},
	rootUserNamespaceProfile: {
		name:              "RootUserNamespace",
		privileged:        true,
		hostUID:           0,
		containerUID:      0,
		defaultCwd:        "/root", // need to run in a directory owned by root
		requirementsFn:    require.UserNamespace,
		singularityOption: "--userns",
		optionForCommands: []string{"shell", "exec", "run", "test", "instance start"},
	},
}

// Privileged returns whether the test should be executed with
// elevated privileges or not.
func (p Profile) Privileged() bool {
	return p.privileged
}

// Requirements calls the different require.* functions
// necessary for running an E2E test under this profile.
func (p Profile) Requirements(t *testing.T) {
	if p.requirementsFn != nil {
		p.requirementsFn(t)
	}
}

// Args returns the additional arguments, if any, to be passed
// to the singularity command specified by cmd in order to run a
// test under this profile.
func (p Profile) args(cmd []string) []string {
	if p.singularityOption == "" {
		return nil
	}

	command := strings.Join(cmd, " ")

	for _, c := range p.optionForCommands {
		if c == command {
			return strings.Split(p.singularityOption, " ")
		}
	}

	return nil
}

// ContainerUser returns the container user information.
func (p Profile) ContainerUser(t *testing.T) *user.User {
	u, err := user.GetPwUID(uint32(p.containerUID))
	if err != nil {
		t.Fatalf("failed to retrieve user container information for user ID %d: %s", p.containerUID, err)
	}

	return u
}

// HostUser returns the host user information.
func (p Profile) HostUser(t *testing.T) *user.User {
	u, err := user.GetPwUID(uint32(p.hostUID))
	if err != nil {
		t.Fatalf("failed to retrieve user host information for user ID %d: %s", p.containerUID, err)
	}

	return u
}

// In returns true if the specified list of profiles contains
// this profile.
func (p Profile) In(profiles ...Profile) bool {
	for _, pr := range profiles {
		if p.name == pr.name {
			return true
		}
	}

	return false
}

// String provides a string representation of this profile.
func (p Profile) String() string {
	return p.name
}

// fakerootRequirements ensures requirements are satisfied to
// correctly execute commands with the fakeroot profile.
func fakerootRequirements(t *testing.T) {
	require.UserNamespace(t)

	uid := uint32(origUID)

	// check that current user has valid mappings in /etc/subuid
	if _, err := fakeroot.GetIDRange(fakeroot.SubUIDFile, uid); err != nil {
		t.Fatalf("fakeroot configuration error: %s", err)
	}

	// check that current user has valid mappings in /etc/subgid;
	// since that file contains the group mappings for a given user
	// *name*, it is keyed by user name, not by group name. This
	// means that even if we are requesting the *group* mappings, we
	// need to pass the *user* ID.
	if _, err := fakeroot.GetIDRange(fakeroot.SubGIDFile, uid); err != nil {
		t.Fatalf("fakeroot configuration error: %s", err)
	}
}
