// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package capabilities

import (
	"sort"
	"testing"
)

func TestSplit(t *testing.T) {
	testCaps := []struct {
		caps   string
		length int
	}{
		{
			caps:   "chown, sys_admin",
			length: 2,
		},
		{
			caps:   "CAP_,     sys_admin        ",
			length: 1,
		},
		{
			caps:   "cap_sys_admin, cap_chown",
			length: 2,
		},
		{
			caps:   "CAP_sys_admin,CHOWN",
			length: 2,
		},
		{
			caps:   "chown, CAP_ALL",
			length: len(Map),
		},
		{
			caps:   "cap_all",
			length: len(Map),
		},
		{
			caps:   "",
			length: 0,
		},
	}
	for _, tc := range testCaps {
		caps, _ := Split(tc.caps)
		if len(caps) != tc.length {
			t.Errorf("should have returned %d as capability len instead of %d", tc.length, len(caps))
		}
	}
}

func TestRemoveDuplicated(t *testing.T) {
	tt := []struct {
		name   string
		in     []string
		expect []string
	}{
		{
			name: "no duplicates",
			in: []string{
				"CAP_CHOWN",
				"CAP_DAC_OVERRIDE",
				"CAP_DAC_READ_SEARCH",
				"CAP_FOWNER",
				"CAP_FSETID",
				"CAP_KILL",
				"CAP_SETGID",
			},
			expect: []string{
				"CAP_CHOWN",
				"CAP_DAC_OVERRIDE",
				"CAP_DAC_READ_SEARCH",
				"CAP_FOWNER",
				"CAP_FSETID",
				"CAP_KILL",
				"CAP_SETGID",
			},
		},
		{
			name: "single duplicate",
			in: []string{
				"CAP_CHOWN",
				"CAP_DAC_OVERRIDE",
				"CAP_DAC_READ_SEARCH",
				"CAP_FOWNER",
				"CAP_DAC_OVERRIDE",
				"CAP_FSETID",
				"CAP_KILL",
				"CAP_SETGID",
			},
			expect: []string{
				"CAP_CHOWN",
				"CAP_DAC_OVERRIDE",
				"CAP_DAC_READ_SEARCH",
				"CAP_FOWNER",
				"CAP_FSETID",
				"CAP_KILL",
				"CAP_SETGID",
			},
		},
		{
			name: "two duplicates",
			in: []string{
				"CAP_KILL",
				"CAP_CHOWN",
				"CAP_DAC_OVERRIDE",
				"CAP_DAC_READ_SEARCH",
				"CAP_FOWNER",
				"CAP_DAC_OVERRIDE",
				"CAP_FSETID",
				"CAP_KILL",
				"CAP_SETGID",
			},
			expect: []string{
				"CAP_CHOWN",
				"CAP_DAC_OVERRIDE",
				"CAP_DAC_READ_SEARCH",
				"CAP_FOWNER",
				"CAP_FSETID",
				"CAP_KILL",
				"CAP_SETGID",
			},
		},
		{
			name: "not once duplicated",
			in: []string{
				"CAP_DAC_OVERRIDE",
				"CAP_CHOWN",
				"CAP_DAC_OVERRIDE",
				"CAP_DAC_READ_SEARCH",
				"CAP_FOWNER",
				"CAP_DAC_OVERRIDE",
				"CAP_DAC_OVERRIDE",
				"CAP_FSETID",
				"CAP_KILL",
				"CAP_SETGID",
				"CAP_DAC_OVERRIDE",
			},
			expect: []string{
				"CAP_CHOWN",
				"CAP_DAC_OVERRIDE",
				"CAP_DAC_READ_SEARCH",
				"CAP_FOWNER",
				"CAP_FSETID",
				"CAP_KILL",
				"CAP_SETGID",
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			actual := RemoveDuplicated(tc.in)
			sort.Strings(tc.expect)
			sort.Strings(actual)
			if len(tc.expect) != len(actual) {
				t.Fatalf("expectected slice of len=%d, got len=%d", len(tc.expect), len(actual))
			}
			for i := range tc.expect {
				if tc.expect[i] != actual[i] {
					t.Fatalf("expected %s at position %d, but got %s", tc.expect[i], i, actual[i])
				}
			}
		})
	}
}
