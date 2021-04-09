// Copyright (c) 2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package slice

// ContainsString returns true if string slice s contains match
func ContainsString(s []string, match string) bool {
	for _, a := range s {
		if a == match {
			return true
		}
	}
	return false
}

// ContainsAnyString returns true if string slice s contains any of matches
func ContainsAnyString(s []string, matches []string) bool {
	for _, m := range matches {
		for _, a := range s {
			if a == m {
				return true
			}
		}
	}
	return false
}
