// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package arrayhelper

// IsIn returns true if a string is found within an array and false otherwise.
// Always returns false if the input array is empty.
func IsIn(a []string, s string) bool {
	found := false
	for _, v := range a {
		if v == s {
			found = true
		}
	}
	return found
}

// Unique returns the input string array with any duplicates removed
func Unique(a []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, s := range a {
		if _, value := keys[s]; !value {
			keys[s] = true
			list = append(list, s)
		}
	}
	return list
}
