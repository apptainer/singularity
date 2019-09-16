// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package testhelper

import "testing"

// TestRunner returns a function that when called runs the provided list
// of tests within a specific test context.
func TestRunner(tests map[string]func(*testing.T)) func(*testing.T) {
	return func(t *testing.T) {
		for name, testfunc := range tests {
			t.Run(name, testfunc)
		}
	}
}
