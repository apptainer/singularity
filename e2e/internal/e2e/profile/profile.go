// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Package profile provides an interface to represent properties
// required to run an end-to-end test under a particular user profile,
// that is either with specific user IDs or using singularity flags to
// impersonate or isolate user IDs.
package profile

import (
	"strings"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/fakeroot"
	"github.com/sylabs/singularity/internal/pkg/test/tool/require"
	"github.com/sylabs/singularity/internal/pkg/util/user"
)

// Profile represents various properties required to run an E2E test
// under a particular user profile.
type Profile interface {
	// Privileged returns whether the test should be executed with
	// elevated privileges or not.
	Privileged() bool

	// Requirements calls the different require.* functions
	// necessary for running an E2E test under this profile.
	Requirements(t *testing.T)

	// Args returns the additional arguments, if any, to be passed
	// to the singularity command specified by cmd in order to run a
	// test under this profile.
	Args(cmd []string) []string

	// User returns the user to be used during the execution of the
	// singularity command.
	User(t *testing.T) *user.User

	// In returns true if the specified list of profiles contains
	// this profile.
	In(profiles ...Profile) bool

	// String provides a string representation of this profile.
	String() string
}

// baseProfile almost implements the Profile interface (but it doesn't,
// it's missing the In and String methods). This exists in order to
// reduce code duplication.
type baseProfile struct{}

func (baseProfile) Privileged() bool {
	return false
}

func (baseProfile) Requirements(t *testing.T) {
}

func (baseProfile) Args([]string) []string {
	return nil
}

func (baseProfile) User(t *testing.T) *user.User {
	return getOrigUser(t)
}

// User implements the Profile interface describing a non-privileged
// user.
type User struct {
	baseProfile
}

func (p User) In(profiles ...Profile) bool {
	return containsProfile(profiles, p)
}

func (User) String() string {
	return "User"
}

// Root implements the Profile interface describing a privileged user.
type Root struct {
	baseProfile
}

func (Root) Privileged() bool {
	return true
}

func (Root) User(t *testing.T) *user.User {
	u, err := user.GetPwUID(0)
	if err != nil {
		t.Fatalf("failed to retrieve root user information: %s", err)
	}
	return u
}

func (p Root) In(profiles ...Profile) bool {
	return containsProfile(profiles, p)
}

func (Root) String() string {
	return "Root"
}

// Fakeroot implements the Profile interface describing a "fake" root
// user and makes use of singularity --fakeroot flag where appropriate.
type Fakeroot struct {
	baseProfile
}

func (Fakeroot) Requirements(t *testing.T) {
	require.UserNamespace(t)

	uid := uint32(e2e.OrigUID())

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

func (Fakeroot) Args(cmd []string) []string {
	command := strings.Join(cmd, " ")

	commands := []string{
		"shell",
		"exec",
		"run",
		"test",
		"instance start",
		"build",
	}

	if containsString(commands, command) {
		return []string{"--fakeroot"}
	}

	return nil
}

func (Fakeroot) User(t *testing.T) *user.User {
	return getOrigUser(t)
}

func (p Fakeroot) In(profiles ...Profile) bool {
	return containsProfile(profiles, p)
}

func (Fakeroot) String() string {
	return "Fakeroot"
}

// UserNamespace implemets Profile describing an unprivileged user that
// operates in a separate user namespace.
type UserNamespace struct {
	baseProfile
}

func (UserNamespace) Args(cmd []string) []string {
	command := strings.Join(cmd, " ")

	commands := []string{
		"shell",
		"exec",
		"run",
		"test",
		"instance start",
	}

	if containsString(commands, command) {
		return []string{"--userns"}
	}

	return nil
}

func (UserNamespace) Requirements(t *testing.T) {
	require.UserNamespace(t)
}

func (p UserNamespace) In(profiles ...Profile) bool {
	return containsProfile(profiles, p)
}

func (UserNamespace) String() string {
	return "UserNamespace"
}

// RootUserNamespace implements Profile describing a privileged user that
// operates in a separate user namespace.
type RootUserNamespace struct {
	UserNamespace
	Root
}

func (p RootUserNamespace) In(profiles ...Profile) bool {
	return containsProfile(profiles, p)
}

func (RootUserNamespace) String() string {
	return "RootUserNamespace"
}

// containsProfile searches for needle in haystack.
func containsProfile(haystack []Profile, needle Profile) bool {
	for _, item := range haystack {
		if needle == item {
			return true
		}
	}

	return false
}

// containsString searches for needle in haystack.
func containsString(haystack []string, needle string) bool {
	for _, item := range haystack {
		if needle == item {
			return true
		}
	}

	return false
}

// getOrigUser returns the user.User structure corresponding to the user
// running the test suite.
func getOrigUser(t *testing.T) *user.User {
	u, err := user.GetPwUID(uint32(e2e.OrigUID()))
	if err != nil {
		t.Fatalf("failed to retrieve original user information: %s", err)
	}
	return u
}
