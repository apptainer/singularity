// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package profile

import (
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/util/user"
	"gotest.tools/assert"
)

var (
	profiles = []Profile{
		Fakeroot{},
		Root{},
		RootUserNamespace{},
		User{},
		UserNamespace{},
	}

	origUser = userByUID(e2e.OrigUID())
	rootUser = userByUID(0)
)

type dummyProfile struct{}

func (dummyProfile) Privileged() bool {
	return false
}

func (dummyProfile) Requirements(t *testing.T) {
}

func (dummyProfile) Args(cmd []string) []string {
	return nil
}

func (dummyProfile) User(t *testing.T) *user.User {
	return nil
}

func (dummyProfile) In(profiles ...Profile) bool {
	return false
}

func (dummyProfile) String() string {
	return "crash test dummy"
}

func TestUserProfile(t *testing.T) {
	p := User{}

	assert.Equal(t, p.Privileged(), false)

	for _, cmd := range [][]string{{"shell"}, {"exec"}, {"run"}, {"test"}, {"instance", "start"}, {"build"}} {
		assert.DeepEqual(t, p.Args(cmd), []string(nil))
	}

	for _, cmd := range [][]string{{"help"}, {"version"}, {"instance", "stop"}, {"foo"}, {}} {
		assert.DeepEqual(t, p.Args(cmd), []string(nil))
	}

	assert.Equal(t, *p.User(t), origUser)

	assert.Equal(t, p.In(profiles...), true)

	assert.Equal(t, p.In(dummyProfile{}), false)

	assert.Equal(t, p.String(), "User")
}

func TestRootProfile(t *testing.T) {
	p := Root{}

	assert.Equal(t, p.Privileged(), true)

	for _, cmd := range [][]string{{"shell"}, {"exec"}, {"run"}, {"test"}, {"instance", "start"}, {"build"}} {
		var expected []string
		assert.DeepEqual(t, p.Args(cmd), expected)
	}

	for _, cmd := range [][]string{{"help"}, {"version"}, {"instance", "stop"}, {"foo"}, {}} {
		var expected []string
		assert.DeepEqual(t, p.Args(cmd), expected)
	}

	assert.Equal(t, *p.User(t), rootUser)

	assert.Equal(t, p.In(profiles...), true)

	assert.Equal(t, p.In(dummyProfile{}), false)

	assert.Equal(t, p.String(), "Root")
}

func TestFakeroot(t *testing.T) {
	p := Fakeroot{}

	assert.Equal(t, p.Privileged(), false)

	for _, cmd := range [][]string{{"shell"}, {"exec"}, {"run"}, {"test"}, {"instance", "start"}, {"build"}} {
		expected := []string{"--fakeroot"}
		assert.DeepEqual(t, p.Args(cmd), expected)
	}

	for _, cmd := range [][]string{{"help"}, {"version"}, {"instance", "stop"}, {"foo"}, {}} {
		var expected []string
		assert.DeepEqual(t, p.Args(cmd), expected)
	}

	assert.Equal(t, *p.User(t), origUser)

	assert.Equal(t, p.In(profiles...), true)

	assert.Equal(t, p.In(dummyProfile{}), false)

	assert.Equal(t, p.String(), "Fakeroot")
}

func TestUserNamespace(t *testing.T) {
	p := UserNamespace{}

	assert.Equal(t, p.Privileged(), false)

	for _, cmd := range [][]string{{"shell"}, {"exec"}, {"run"}, {"test"}, {"instance", "start"}} {
		expected := []string{"--userns"}
		assert.DeepEqual(t, p.Args(cmd), expected)
	}

	for _, cmd := range [][]string{{"help"}, {"version"}, {"instance", "stop"}, {"foo"}, {}} {
		var expected []string
		assert.DeepEqual(t, p.Args(cmd), expected)
	}

	assert.Equal(t, *p.User(t), origUser)

	assert.Equal(t, p.In(profiles...), true)

	assert.Equal(t, p.In(dummyProfile{}), false)

	assert.Equal(t, p.String(), "UserNamespace")
}

func TestRootUserNamespace(t *testing.T) {
	p := RootUserNamespace{}

	assert.Equal(t, p.Privileged(), true)

	for _, cmd := range [][]string{{"shell"}, {"exec"}, {"run"}, {"test"}, {"instance", "start"}} {
		expected := []string{"--userns"}
		assert.DeepEqual(t, p.Args(cmd), expected)
	}

	for _, cmd := range [][]string{{"help"}, {"version"}, {"instance", "stop"}, {"foo"}, {}} {
		var expected []string
		assert.DeepEqual(t, p.Args(cmd), expected)
	}

	assert.Equal(t, *p.User(t), rootUser)

	assert.Equal(t, p.In(profiles...), true)

	assert.Equal(t, p.In(dummyProfile{}), false)

	assert.Equal(t, p.String(), "RootUserNamespace")
}

func userByUID(uid int) user.User {
	u, err := user.GetPwUID(uint32(uid))
	if err != nil {
		panic("cannot obtain user info")
	}
	return *u
}
