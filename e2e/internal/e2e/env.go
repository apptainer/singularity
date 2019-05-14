// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.
package e2e

import (
	"os"
	"testing"

	"github.com/kelseyhightower/envconfig"
)

func LoadEnv(t *testing.T, env interface{}) {
	if err := envconfig.Process("E2E", env); err != nil {
		t.Fatalf("Failed to load environment: %+v\n", err)
	}
}

// HomeDir will return the users home directory.
func HomeDir() string {
	return os.Getenv("HOME")
}
