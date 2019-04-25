// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cmdline

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/internal/pkg/test"
)

var cmd cobra.Command

func TestEnvAppendValue(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	cmd.Flags().BoolSlice("boolFlag", []bool{}, "")
	if cmd.Flag("boolFlag").Value.String() != "[]" {
		t.Errorf("The flag should be empty.")
	}

	EnvAppendValue(cmd.Flag("boolFlag"), "1")
	if cmd.Flag("boolFlag").Value.String() != "[true]" {
		t.Errorf("The flag should be set to the value provided.")
	}

	EnvAppendValue(cmd.Flag("boolFlag"), "false")
	if cmd.Flag("boolFlag").Value.String() != "[true,false]" {
		t.Errorf("The flag should be appended with the value provided.")
	}

	EnvAppendValue(cmd.Flag("boolFlag"), "bad")
	if cmd.Flag("boolFlag").Value.String() != "[true,false]" {
		t.Errorf("The flag should return previous value.")
	}
}

func TestEnvSetValue(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	cmd.Flags().Int("intFlag", 0, "")
	if cmd.Flag("intFlag").Value.String() != "0" {
		t.Errorf("The flag should be empty.")
	}

	EnvSetValue(cmd.Flag("intFlag"), "any string")
	if cmd.Flag("intFlag").Value.String() != "0" {
		t.Errorf("The flag should be set to the default value.")
	}

	EnvSetValue(cmd.Flag("intFlag"), "-1")
	if cmd.Flag("intFlag").Value.String() != "-1" {
		t.Errorf("The flag should be set to the value provided.")
	}

	cmd.Flags().StringSlice("stringSlice", []string{""}, "")
	if cmd.Flag("stringSlice").Value.String() != "[]" {
		t.Errorf("The flag should be empty.")
	}

	EnvSetValue(cmd.Flag("stringSlice"), "sliceval1,sliceval2")
	if cmd.Flag("stringSlice").Value.String() != "[sliceval1,sliceval2]" {
		t.Errorf("The flag should be set to value provided.")
	}

	EnvSetValue(cmd.Flag("stringSlice"), "newsliceval")
	if cmd.Flag("stringSlice").Value.String() != "[sliceval1,sliceval2]" {
		t.Errorf("Once set, the flag value should not change.")
	}
}
