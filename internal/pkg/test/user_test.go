// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package test

import (
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

func TestGetCurrentUser(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	tests := []struct {
		name      string
		shallPass bool
	}{
		{
			name:      "default user",
			shallPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			me, err := GetCurrentUser(t)
			if tt.shallPass == true && (me == nil || err != nil) {
				t.Fatal("valid case failed")
			}

			if tt.shallPass == false && me != nil && err == nil {
				t.Fatal("invalid case passed")
			}
		})
	}
}
