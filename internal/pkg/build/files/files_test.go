// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package files

import (
	"testing"
)

func TestJoinSlash(t *testing.T) {
	tests := []struct {
		name    string
		prefix  string
		path    string
		correct string
	}{
		{
			name:    "sanity",
			prefix:  "",
			path:    "/some/path",
			correct: "/some/path",
		},
		{
			name:    "basicPrepend",
			prefix:  "/some",
			path:    "/path",
			correct: "/some/path",
		},
		{
			name:    "basicJoinTrailingSlash",
			prefix:  "/some",
			path:    "/path/",
			correct: "/some/path/",
		},
		{
			name:    "manySlashes",
			prefix:  "/some/",
			path:    "//path/to/dest//",
			correct: "/some/path/to/dest/",
		},
		{
			name:    "root",
			prefix:  "",
			path:    "/",
			correct: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// run it through wildcard function
			path := joinKeepSlash(tt.prefix, tt.path)
			if path != tt.correct {
				t.Errorf("join created incorrect path: %s correct: %s", path, tt.correct)
			}
		})
	}
}

func TestSecureJoinSlash(t *testing.T) {
	tests := []struct {
		name    string
		prefix  string
		path    string
		correct string
	}{
		{
			name:    "sanity",
			prefix:  "",
			path:    "/some/path",
			correct: "/some/path",
		},
		{
			name:    "basicPrepend",
			prefix:  "/some",
			path:    "/path",
			correct: "/some/path",
		},
		{
			name:    "basicJoinTrailingSlash",
			prefix:  "/some",
			path:    "/path/",
			correct: "/some/path/",
		},
		{
			name:    "manySlashes",
			prefix:  "/some/",
			path:    "//path/to/dest//",
			correct: "/some/path/to/dest/",
		},
		{
			name:    "root",
			prefix:  "",
			path:    "/",
			correct: "/",
		},
		{
			name:    "escape dir",
			prefix:  "/some/",
			path:    "/../../../../",
			correct: "/some/",
		},
		{
			name:    "escape root",
			prefix:  "",
			path:    "/../../../../",
			correct: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// run it through wildcard function
			path, err := secureJoinKeepSlash(tt.prefix, tt.path)
			if err != nil {
				t.Errorf("failed with error: %s", err)
			}
			if path != tt.correct {
				t.Errorf("join created incorrect path: %s correct: %s", path, tt.correct)
			}
		})
	}
}
