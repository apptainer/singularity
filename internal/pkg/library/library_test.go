// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package library

import (
	"testing"

	"github.com/sylabs/scs-library-client/client"
)

func TestNormalizeLibraryRef(t *testing.T) {
	tests := []struct {
		name        string
		libraryRef  string
		expected    string
		expectedTag string
	}{
		{"with tag", "library://alpine:latest", "alpine:latest", "latest"},
		{"fully qualified with tag", "library://user/collection/container:2.0.0", "user/collection/container:2.0.0", "2.0.0"},
		{"without tag", "library://alpine", "alpine:latest", "latest"},
		{"with tag variation", "library://alpine:1.0.1", "alpine:1.0.1", "1.0.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeLibraryRef(tt.libraryRef)

			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}

			// pass (potentially) reformatted library ref to scs-library-client
			// parser for further validation
			r, err := client.Parse("library:///" + result)
			if err != nil {
				t.Errorf("Error parsing reformatted library ref (%s): %v", result, err)
			}

			if r.Tags[0] != tt.expectedTag {
				t.Errorf("Expected tag %s, got %s", tt.expectedTag, r.Tags[0])
			}
		})
	}
}
