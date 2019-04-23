// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"os"
	"testing"
)

// Note that the valid use cases are in remote_add_test.go. We still have tests
// here for all the corner cases of RemoteRemove()
func TestRemoteRemove(t *testing.T) {
	validCfgFile := createValidCfgFile(t) // from remote_add_test.go
	defer os.Remove(validCfgFile)

	tests := []struct {
		name       string
		cfgFile    string
		remoteName string
		shallPass  bool
	}{
		{
			name:       "empty config file; empty remote name",
			cfgFile:    "",
			remoteName: "",
			shallPass:  false,
		},
		{
			name:       "valid config file; empty remote name",
			cfgFile:    validCfgFile,
			remoteName: "",
			shallPass:  false,
		},
		{
			name:       "valid config file; valid remote name",
			cfgFile:    validCfgFile,
			remoteName: "cloud_testing",
			shallPass:  true,
		},
	}

	// Add remotes based on our config file
	err := RemoteAdd(validCfgFile, "cloud_testing", "cloud.random.io", false)
	if err != nil {
		t.Fatalf("cannot add remote \"cloud\" for testing: %s\n", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RemoteRemove(tt.cfgFile, tt.remoteName)
			if tt.shallPass == true && err != nil {
				t.Fatalf("valid case failed: %s\n", err)
			}
			if tt.shallPass == false && err == nil {
				t.Fatal("invalid case succeeded")
			}
		})
	}
}
