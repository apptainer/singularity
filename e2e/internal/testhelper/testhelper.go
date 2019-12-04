// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package testhelper

import (
	"reflect"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
)

// TestRunner returns a function that when called runs the provided list
// of tests within a specific test context.
func TestRunner(tests map[string]func(*testing.T)) func(*testing.T) {
	return func(t *testing.T) {
		for name, testfunc := range tests {
			t.Run(name, testfunc)
		}
	}
}

type Tests map[string]func(*testing.T)

type Group func(e2e.TestEnv) Tests

type Suite struct {
	t      *testing.T
	env    e2e.TestEnv
	groups map[string]Group
}

func NewSuite(t *testing.T, env e2e.TestEnv) *Suite {
	suite := &Suite{
		t:      t,
		env:    env,
		groups: make(map[string]Group),
	}
	return suite
}

func (s *Suite) AddGroup(name string, group Group) {
	s.groups[name] = group
}

func (s *Suite) Run() {
	var tests = make(map[string]Tests)

	for name, gr := range s.groups {
		env := s.env
		env.TestDir, _ = e2e.MakeTempDir(s.t, s.env.TestDir, "group-", "")
		tests[name] = gr(s.env)
	}

	// Run parallel test first
	s.t.Run("PAR", func(t *testing.T) {
		for name := range s.groups {
			name := name

			t.Run(name, func(t *testing.T) {
				t.Parallel()

				for testName, fn := range tests[name] {
					fn := fn
					testName := testName

					pc := reflect.ValueOf(fn).Pointer()
					if _, ok := npTests[pc]; ok {
						continue
					}

					t.Run(testName, func(t *testing.T) {
						t.Parallel()
						fn(t)
					})
				}
			})
		}
	})

	s.t.Run("SEQ", func(t *testing.T) {
		for name := range s.groups {
			name := name

			t.Run(name, func(t *testing.T) {
				for testName, fn := range tests[name] {
					pc := reflect.ValueOf(fn).Pointer()
					if _, ok := npTests[pc]; !ok {
						continue
					}
					t.Run(testName, fn)
				}
			})
		}
	})
}

var npTests = make(map[uintptr]struct{})

func NoParallel(fn func(*testing.T)) func(*testing.T) {
	npTests[reflect.ValueOf(fn).Pointer()] = struct{}{}
	return fn
}
