// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package uri

import "testing"

func Test_NameFromURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{"docker no scope", "docker://ubuntu", "ubuntu"},
		{"docker scoped", "docker://user/image", "image"},
		{"dave's magical lolcow", "docker://godlovedc/lolcow", "lolcow"},
		{"docker w/ tags", "docker://godlovedc/lolcow:latest", "lolcow"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if n := NameFromURI(tt.uri); n != tt.expected {
				t.Errorf("incorrectly parsed name as \"%v\" (expected \"%s\")", n, tt.expected)
			}
		})
	}
}

func Test_SplitURI(t *testing.T) {
	tests := []struct {
		name      string
		uri       string
		transport string
		ref       string
	}{
		{"docker no scope", "docker://ubuntu", "docker", "//ubuntu"},
		{"docker scoped", "docker://user/image", "docker", "//user/image"},
		{"dave's magical lolcow", "docker://godlovedc/lolcow", "docker", "//godlovedc/lolcow"},
		{"docker w/ tags", "docker://godlovedc/lolcow:latest", "docker", "//godlovedc/lolcow:latest"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tr, r := SplitURI(tt.uri); tr != tt.transport || r != tt.ref {
				t.Errorf("incorrectly parsed uri as %s : %s (expected %s : %s)", tr, r, tt.transport, tt.ref)
			}
		})
	}
}
