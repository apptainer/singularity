// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package remotebuilder

import (
	"os"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
	types "github.com/sylabs/singularity/pkg/build/legacy"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
)

func TestMain(m *testing.M) {
	useragent.InitValue("singularity", "3.0.0-alpha.1-303-gaed8d30-dirty")

	os.Exit(m.Run())
}

func TestBuild(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	tests := []struct {
		description   string
		expectSuccess bool
		builderAddr   string
	}{
		{"BadBuilderURI", false, "ftp:?abc//foo.bar:abc"},
		{"BadBuilderScheme", false, "ftp://build.sylabs.io"},
		{"SuccessBuilderAddr", true, "http://build.sylabs.io"},
		{"SuccessBuilderAddrSecure", true, "https://build.sylabs.io"},
	}

	// Loop over test cases
	for _, tt := range tests {
		t.Run(tt.description, test.WithoutPrivilege(func(t *testing.T) {
			_, err := New("", "", types.Definition{}, false, false, tt.builderAddr, "")
			if tt.expectSuccess {
				// Ensure the handler returned no error, and the response is as expected
				if err != nil {
					t.Fatalf("unexpected failure: %v", err)
				}
			} else {
				// Ensure the handler returned an error
				if err == nil {
					t.Fatalf("unexpected success")
				}
			}
		}))
	}
}
