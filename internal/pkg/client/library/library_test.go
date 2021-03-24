// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package library

import (
	"reflect"
	"testing"
)

func TestNormalizeLibraryRef(t *testing.T) {
	tests := []struct {
		name         string
		libraryRef   string
		expected     string
		expectedTags []string
		expectedHost string
	}{
		{"with tag", "library://alpine:latest", "alpine", []string{"latest"}, ""},
		{"with multiple tags", "library://alpine:tag1,tag2", "alpine", []string{"tag1", "tag2"}, ""},
		{"fully qualified with tag", "library://user/collection/container:2.0.0", "user/collection/container", []string{"2.0.0"}, ""},
		{"fully qualified with multiple tags", "library://user/collection/container:2.0.0,3.0.0", "user/collection/container", []string{"2.0.0", "3.0.0"}, ""},
		{"without tag", "library://alpine", "alpine", []string{"latest"}, ""},
		{"with tag variation", "library://alpine:1.0.1", "alpine", []string{"1.0.1"}, ""},
		{"with hostname", "library://hostname/collection/container/image:tag1", "collection/container/image", []string{"tag1"}, "hostname"},
		{"with hostname with multiple tags", "library://hostname/collection/container/image:tag1,tag2", "collection/container/image", []string{"tag1", "tag2"}, "hostname"},
		{"with hostname without tag", "library://hostname/collection/container/image", "collection/container/image", []string{"latest"}, "hostname"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizeLibraryRef(tt.libraryRef)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if result.Path != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}

			if !reflect.DeepEqual(result.Tags, tt.expectedTags) {
				t.Errorf("Expected tag %v, got %v", tt.expectedTags, result.Tags)
			}

			if result.Host != tt.expectedHost {
				t.Errorf("Expected host %s, got %s", tt.expectedHost, result.Host)
			}
		})
	}
}
