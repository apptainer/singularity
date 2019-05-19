// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package library

import (
	"regexp"
	"testing"

	"github.com/sylabs/scs-library-client/client"
)

// TestParseLegacyLibraryRef ensures only one legacy formatted library ref is
// reformatted for further parsing
func TestParseLegacyLibraryRef(t *testing.T) {
	tests := []struct {
		name       string
		libraryRef string
		expected   string
	}{
		{"legacy", "library://alpine:latest", "library:///alpine:latest"},
		{"passthrough #1", "library:///alpine:latest", "library:///alpine:latest"},
		{"passthrough #2", "library:path:tags", "library:path:tags"},
		{"passthrough #3", "library:/path:tags", "library:/path:tags"},
		{"passthrough #4", "library:///path:tags", "library:///path:tags"},
		{"passthrough #5", "library://host/path:tags", "library://host/path:tags"},
		{"passthrough #6", "library://host:port/path:tags", "library://host:port/path:tags"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseLegacyLibraryRef(tt.libraryRef)

			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}

			// pass (potentially) reformatted library ref to scs-library-client
			// parser for further validation
			_, err := client.Parse(result)
			if err != nil {
				t.Errorf("Error parsing reformatted library ref (%s): %v", result, err)
			}
		})
	}
}

func parsedTag(s string) string {
	re := regexp.MustCompile(`(?m)(?P<key>.*):(?P<value>.*)$`)
	template := "$value"

	result := []byte{}

	// For each match of the regex in the s.
	for _, submatches := range re.FindAllStringSubmatchIndex(s, -1) {
		// Apply the captured submatches to the template and append the output
		// to the result.
		result = re.ExpandString(result, template, s, submatches)
	}
	return string(result)
}

func TestEnsureTag(t *testing.T) {
	tests := []struct {
		name        string
		libraryRef  string
		expectedTag string
	}{
		{"without tag", "library://alpine", "latest"},
		{"with tag", "library://alpine:1.0.1", "1.0.1"},
		{"without prefix/tag", "alpine", "latest"},
		{"without prefix", "alpine:1.0.2", "1.0.2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EnsureTag(tt.libraryRef)

			parsedTag := parsedTag(result)
			if tt.expectedTag != parsedTag {
				t.Errorf("got tag %s, expected %s", parsedTag, tt.expectedTag)
			}
		})
	}
}
