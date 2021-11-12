// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package capabilities

import (
	"bytes"
	"reflect"
	"testing"
)

type readWriteTest struct {
	name string
	c    Config
}

func TestReadFromWriteTo(t *testing.T) {
	testsPass := []readWriteTest{
		{
			name: "empty config",
			c: Config{
				Users:  map[string][]string{},
				Groups: map[string][]string{},
			},
		},
		{
			name: "config with stuff",
			c: Config{
				Users: map[string][]string{
					"user1": {"CAP_SYS_ADMIN"},
					"user2": {"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE"},
				},
				Groups: map[string][]string{
					"user1": {"CAP_SYS_ADMIN"},
					"user2": {"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE"},
				},
			},
		},
	}

	for _, test := range testsPass {
		t.Run(test.name, func(t *testing.T) {
			var r bytes.Buffer

			test.c.WriteTo(&r)

			new, err := ReadFrom(&r)
			if err != nil {
				t.Errorf("unexpected failure running %s test: %s", test.name, err)
			}

			if !reflect.DeepEqual(test.c, *new) {
				t.Errorf("failed to read/write config:\n\thave: %v\n\twant: %v", test.c, *new)
			}
		})
	}

	t.Run("empty config no data", func(t *testing.T) {
		var r bytes.Buffer

		_, err := ReadFrom(&r)
		if err != nil {
			t.Errorf("unexpected failure running %s test: %s", t.Name(), err)
		}
	})
}

type capTest struct {
	name string
	old  Config
	new  Config
	id   string
	caps []string
}

//nolint:dupl
func TestAddUserCaps(t *testing.T) {
	testsPass := []capTest{
		{
			name: "add existing user single cap",
			old: Config{
				Users: map[string][]string{
					"root": {"CAP_SYS_ADMIN"},
				},
			},
			new: Config{
				Users: map[string][]string{
					"root": {"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE"},
				},
			},
			id:   "root",
			caps: []string{"CAP_DAC_OVERRIDE"},
		},
		{
			name: "add existing user multiple caps",
			old: Config{
				Users: map[string][]string{
					"user1": {"CAP_SYS_ADMIN"},
				},
			},
			new: Config{
				Users: map[string][]string{
					"user1": {"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE", "CAP_CHOWN"},
				},
			},
			id:   "user1",
			caps: []string{"CAP_DAC_OVERRIDE", "CAP_CHOWN"},
		},
		{
			name: "add new user",
			old: Config{
				Users: map[string][]string{
					"root": {"CAP_SYS_ADMIN"},
				},
			},
			new: Config{
				Users: map[string][]string{
					"root":  {"CAP_SYS_ADMIN"},
					"user1": {"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE", "CAP_CHOWN"},
				},
			},
			id:   "user1",
			caps: []string{"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE", "CAP_CHOWN"},
		},
		{
			name: "add duplicate cap",
			old: Config{
				Users: map[string][]string{
					"root": {"CAP_SYS_ADMIN"},
				},
			},
			new: Config{
				Users: map[string][]string{
					"root": {"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE", "CAP_CHOWN"},
				},
			},
			id:   "root",
			caps: []string{"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE", "CAP_CHOWN"},
		},
	}

	for _, test := range testsPass {
		t.Run(test.name, func(t *testing.T) {
			if err := test.old.AddUserCaps(test.id, test.caps); err != nil {
				t.Error("failed to add capability to config")
			}

			if !reflect.DeepEqual(test.old, test.new) {
				t.Errorf("AddUserCaps failed to set config:\n\thave: %v\n\twant: %v", test.old, test.new)
			}
		})
	}

	testFail := capTest{
		name: "add bad cap fail",
		old: Config{
			Users: map[string][]string{
				"root": {"CAP_SYS_ADMIN"},
			},
		},
		new: Config{
			Users: map[string][]string{
				"root": {"CAP_SYS_ADMIN"},
			},
		},
		id:   "root",
		caps: []string{"CAP_BAD_WRONG_INCORRECT_BAD"},
	}

	t.Run(testFail.name, func(t *testing.T) {
		if err := testFail.old.AddUserCaps(testFail.id, testFail.caps); err == nil {
			t.Error("unexpected success adding non-existent capability")
		}
	})
}

//nolint:dupl
func TestAddGroupCaps(t *testing.T) {
	testsPass := []capTest{
		{
			name: "add existing group single cap",
			old: Config{
				Groups: map[string][]string{
					"root": {"CAP_SYS_ADMIN"},
				},
			},
			new: Config{
				Groups: map[string][]string{
					"root": {"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE"},
				},
			},
			id:   "root",
			caps: []string{"CAP_DAC_OVERRIDE"},
		},
		{
			name: "add existing group multiple caps",
			old: Config{
				Groups: map[string][]string{
					"group1": {"CAP_SYS_ADMIN"},
				},
			},
			new: Config{
				Groups: map[string][]string{
					"group1": {"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE", "CAP_CHOWN"},
				},
			},
			id:   "group1",
			caps: []string{"CAP_DAC_OVERRIDE", "CAP_CHOWN"},
		},
		{
			name: "add new group",
			old: Config{
				Groups: map[string][]string{
					"root": {"CAP_SYS_ADMIN"},
				},
			},
			new: Config{
				Groups: map[string][]string{
					"root":   {"CAP_SYS_ADMIN"},
					"group1": {"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE", "CAP_CHOWN"},
				},
			},
			id:   "group1",
			caps: []string{"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE", "CAP_CHOWN"},
		},
		{
			name: "add duplicate cap",
			old: Config{
				Groups: map[string][]string{
					"root": {"CAP_SYS_ADMIN"},
				},
			},
			new: Config{
				Groups: map[string][]string{
					"root": {"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE", "CAP_CHOWN"},
				},
			},
			id:   "root",
			caps: []string{"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE", "CAP_CHOWN"},
		},
	}

	for _, test := range testsPass {
		t.Run(test.name, func(t *testing.T) {
			if err := test.old.AddGroupCaps(test.id, test.caps); err != nil {
				t.Error("failed to add capability to config")
			}

			if !reflect.DeepEqual(test.old, test.new) {
				t.Errorf("AddGroupCaps failed to set config:\n\thave: %v\n\twant: %v", test.old, test.new)
			}
		})
	}

	testFail := capTest{
		name: "add bad cap fail",
		old: Config{
			Groups: map[string][]string{
				"root": {"CAP_SYS_ADMIN"},
			},
		},
		new: Config{
			Groups: map[string][]string{
				"root": {"CAP_SYS_ADMIN"},
			},
		},
		id:   "root",
		caps: []string{"CAP_BAD_WRONG_INCORRECT_BAD"},
	}

	t.Run(testFail.name, func(t *testing.T) {
		if err := testFail.old.AddGroupCaps(testFail.id, testFail.caps); err == nil {
			t.Error("unexpected success adding non-existent capability")
		}
	})
}

//nolint:dupl
func TestDropUserCaps(t *testing.T) {
	testsPass := []capTest{
		{
			name: "drop existing user single cap",
			old: Config{
				Users: map[string][]string{
					"root": {"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE"},
				},
			},
			new: Config{
				Users: map[string][]string{
					"root": {"CAP_SYS_ADMIN"},
				},
			},
			id:   "root",
			caps: []string{"CAP_DAC_OVERRIDE"},
		},
		{
			name: "drop non-existent capability from user",
			old: Config{
				Users: map[string][]string{
					"root": {"CAP_SYS_ADMIN"},
				},
			},
			new: Config{
				Users: map[string][]string{
					"root": {"CAP_SYS_ADMIN"},
				},
			},
			id:   "root",
			caps: []string{"CAP_DAC_OVERRIDE"},
		},
		{
			name: "drop existing user multiple caps",
			old: Config{
				Users: map[string][]string{
					"user1": {"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE", "CAP_CHOWN"},
				},
			},
			new: Config{
				Users: map[string][]string{
					"user1": {"CAP_SYS_ADMIN"},
				},
			},
			id:   "user1",
			caps: []string{"CAP_DAC_OVERRIDE", "CAP_CHOWN"},
		},
		{
			name: "drop duplicate cap",
			old: Config{
				Users: map[string][]string{
					"root": {"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE", "CAP_CHOWN"},
				},
			},
			new: Config{
				Users: map[string][]string{},
			},
			id:   "root",
			caps: []string{"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE", "CAP_CHOWN"},
		},
	}

	for _, test := range testsPass {
		t.Run(test.name, func(t *testing.T) {
			if err := test.old.DropUserCaps(test.id, test.caps); err != nil {
				t.Error("failed to drop capability to config")
			}

			if !reflect.DeepEqual(test.old, test.new) {
				t.Errorf("DropUserCaps failed to set config:\n\thave: %v\n\twant: %v", test.old, test.new)
			}
		})
	}

	testsFail := []capTest{
		{
			name: "drop bad cap fail",
			old: Config{
				Users: map[string][]string{
					"root": {"CAP_SYS_ADMIN"},
				},
			},
			new: Config{
				Users: map[string][]string{
					"root": {"CAP_SYS_ADMIN"},
				},
			},
			id:   "root",
			caps: []string{"CAP_BAD_WRONG_INCORRECT_BAD"},
		},
		{
			name: "drop bad user fail",
			old: Config{
				Users: map[string][]string{
					"root": {"CAP_SYS_ADMIN"},
				},
			},
			new: Config{
				Users: map[string][]string{
					"root": {"CAP_SYS_ADMIN"},
				},
			},
			id:   "non_existent_user",
			caps: []string{"CAP_SYS_ADMIN"},
		},
	}

	for _, test := range testsFail {
		t.Run(test.name, func(t *testing.T) {
			if err := test.old.DropUserCaps(test.id, test.caps); err == nil {
				t.Error("unexpected success dropping non-existent capability")
			}
		})
	}
}

//nolint:dupl
func TestDropGroupCaps(t *testing.T) {
	testsPass := []capTest{
		{
			name: "drop existing group single cap",
			old: Config{
				Groups: map[string][]string{
					"root": {"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE"},
				},
			},
			new: Config{
				Groups: map[string][]string{
					"root": {"CAP_SYS_ADMIN"},
				},
			},
			id:   "root",
			caps: []string{"CAP_DAC_OVERRIDE"},
		},
		{
			name: "drop non-existent capability from group",
			old: Config{
				Groups: map[string][]string{
					"root": {"CAP_SYS_ADMIN"},
				},
			},
			new: Config{
				Groups: map[string][]string{
					"root": {"CAP_SYS_ADMIN"},
				},
			},
			id:   "root",
			caps: []string{"CAP_DAC_OVERRIDE"},
		},
		{
			name: "drop existing group multiple caps",
			old: Config{
				Groups: map[string][]string{
					"group1": {"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE", "CAP_CHOWN"},
				},
			},
			new: Config{
				Groups: map[string][]string{
					"group1": {"CAP_SYS_ADMIN"},
				},
			},
			id:   "group1",
			caps: []string{"CAP_DAC_OVERRIDE", "CAP_CHOWN"},
		},
		{
			name: "drop duplicate cap",
			old: Config{
				Groups: map[string][]string{
					"root": {"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE", "CAP_CHOWN"},
				},
			},
			new: Config{
				Groups: map[string][]string{},
			},
			id:   "root",
			caps: []string{"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE", "CAP_CHOWN"},
		},
	}

	for _, test := range testsPass {
		t.Run(test.name, func(t *testing.T) {
			if err := test.old.DropGroupCaps(test.id, test.caps); err != nil {
				t.Error("failed to drop capability to config")
			}

			if !reflect.DeepEqual(test.old, test.new) {
				t.Errorf("DropGroupCaps failed to set config:\n\thave: %v\n\twant: %v", test.old, test.new)
			}
		})
	}

	testsFail := []capTest{
		{
			name: "drop bad cap fail",
			old: Config{
				Groups: map[string][]string{
					"root": {"CAP_SYS_ADMIN"},
				},
			},
			new: Config{
				Groups: map[string][]string{
					"root": {"CAP_SYS_ADMIN"},
				},
			},
			id:   "root",
			caps: []string{"CAP_BAD_WRONG_INCORRECT_BAD"},
		},
		{
			name: "drop bad group fail",
			old: Config{
				Groups: map[string][]string{
					"root": {"CAP_SYS_ADMIN"},
				},
			},
			new: Config{
				Groups: map[string][]string{
					"root": {"CAP_SYS_ADMIN"},
				},
			},
			id:   "non_existent_group",
			caps: []string{"CAP_SYS_ADMIN"},
		},
	}

	for _, test := range testsFail {
		t.Run(test.name, func(t *testing.T) {
			if err := test.old.DropGroupCaps(test.id, test.caps); err == nil {
				t.Error("unexpected success dropping non-existent capability")
			}
		})
	}
}

func TestListCaps(t *testing.T) {
	conf := Config{
		Users: map[string][]string{
			"root":  {"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE"},
			"user1": {"CAP_CHOWN"},
		},
		Groups: map[string][]string{
			"root":  {"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE"},
			"user2": {"CAP_CHOWN"},
		},
	}

	if !reflect.DeepEqual(conf.ListUserCaps("root"), conf.Users["root"]) {
		t.Error("user cap lookup failed")
	}

	if !reflect.DeepEqual(conf.ListGroupCaps("user2"), conf.Groups["user2"]) {
		t.Error("group cap lookup failed")
	}

	u, g := conf.ListAllCaps()

	if !reflect.DeepEqual(conf.Users, u) || !reflect.DeepEqual(conf.Groups, g) {
		t.Error("all caps lookup failed")
	}
}

type capCheckTest struct {
	name         string
	id           string
	caps         []string
	authorized   []string
	unauthorized []string
}

func TestCheckCaps(t *testing.T) {
	conf := Config{
		Users: map[string][]string{
			"root":  {"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE"},
			"user1": {"CAP_CHOWN", "CAP_SYS_ADMIN"},
			"user2": {},
		},
		Groups: map[string][]string{
			"root":  {"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE"},
			"user1": {"CAP_CHOWN", "CAP_SYS_ADMIN"},
			"user2": {},
		},
	}

	testsPass := []capCheckTest{
		{
			name:       "check multiple authorized",
			id:         "root",
			caps:       []string{"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE"},
			authorized: []string{"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE"},
		},
		{
			name:         "check multiple unauthorized",
			id:           "user2",
			caps:         []string{"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE"},
			unauthorized: []string{"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE"},
		},
		{
			name:         "check multiple authorized & unauthorized",
			id:           "user1",
			caps:         []string{"CAP_SYS_ADMIN", "CAP_DAC_OVERRIDE"},
			authorized:   []string{"CAP_SYS_ADMIN"},
			unauthorized: []string{"CAP_DAC_OVERRIDE"},
		},
	}

	for _, test := range testsPass {
		t.Run(test.name, func(t *testing.T) {
			aUser, uUser := conf.CheckUserCaps(test.id, test.caps)
			if !reflect.DeepEqual(aUser, test.authorized) {
				t.Errorf("returned incorrect authorized user caps:\n\thave: %v\n\twant: %v", test.authorized, aUser)
			}

			if !reflect.DeepEqual(uUser, test.unauthorized) {
				t.Errorf("returned incorrect unauthorized user caps:\n\thave: %v\n\twant: %v", test.unauthorized, uUser)
			}

			aGroup, uGroup := conf.CheckGroupCaps(test.id, test.caps)
			if !reflect.DeepEqual(aGroup, test.authorized) {
				t.Errorf("returned incorrect authorized group caps:\n\thave: %v\n\twant: %v", test.authorized, aGroup)
			}

			if !reflect.DeepEqual(uGroup, test.unauthorized) {
				t.Errorf("returned incorrect unauthorized group caps:\n\thave: %v\n\twant: %v", test.unauthorized, uGroup)
			}
		})
	}
}
