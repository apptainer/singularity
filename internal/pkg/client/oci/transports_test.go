// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oci

import (
	"testing"

	"github.com/containers/image/v5/transports"
)

func TestIsSupported(t *testing.T) {
	// We individually check all the known transports. This is a
	// very naive test since mimicking the actual code but still ensures
	// that everything is consistent
	for _, transport := range transports.ListNames() {
		if IsSupported(transport) == "" {
			t.Fatalf("transport %s reported as not supported", transport)
		}
	}

	// Now error cases
	tests := []struct {
		name      string
		transport string
	}{
		{
			name:      "empty",
			transport: "",
		},
		{
			name:      "random",
			transport: "fake",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if IsSupported(tt.transport) != "" {
				t.Fatalf("invalid transport %s reported as supported", tt.transport)
			}
		})
	}
}
