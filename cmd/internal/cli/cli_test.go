// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/internal/pkg/test"
)

var cmd cobra.Command

func TestEnvAppend(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)
	var appendFlag []string

	cmd.Flags().StringSliceVarP(&appendFlag, "appendFlag", "", []string{""}, "")
	v := cmd.Flag("appendFlag").Value.String()
	assert.Equal(t, v, "[]", "The flag should be unset.")

	envAppend(cmd.Flag("appendFlag"), "appendval")
	v = cmd.Flag("appendFlag").Value.String()
	assert.Equal(t, v, "[appendval]", "The flag should be set to the value provided.")

	envAppend(cmd.Flag("appendFlag"), "appendval")
	v = cmd.Flag("appendFlag").Value.String()
	assert.Equal(t, v, "[appendval,appendval]", "The flag should appended with the value provided.")
}

func TestEnvBool(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)
	var boolFlag bool

	cmd.Flags().BoolVar(&boolFlag, "boolFlag", false, "")
	v := cmd.Flag("boolFlag").Value.String()
	assert.Equal(t, v, "false", "The flag should be unset.")

	envBool(cmd.Flag("boolFlag"), "any string")
	v = cmd.Flag("boolFlag").Value.String()
	assert.Equal(t, v, "true", "The flag should be set to true.")
}

func TestEnvStringNSlice(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)
	var stringFlag string

	cmd.Flags().StringVarP(&stringFlag, "stringFlag", "", "", "")
	v := cmd.Flag("stringFlag").Value.String()
	assert.Equal(t, v, "", "The flag should be unset.")

	envStringNSlice(cmd.Flag("stringFlag"), "stringval")
	v = cmd.Flag("stringFlag").Value.String()
	assert.Equal(t, v, "stringval", "The flag should be set to value provided.")

	envStringNSlice(cmd.Flag("stringFlag"), "newstringval")
	v = cmd.Flag("stringFlag").Value.String()
	assert.Equal(t, v, "stringval", "Once set, the flag should not be overwritten.")

	var stringSlice []string

	cmd.Flags().StringSliceVarP(&stringSlice, "stringSlice", "", []string{""}, "")
	v = cmd.Flag("stringSlice").Value.String()
	assert.Equal(t, v, "[]", "The flag should be unset.")

	envStringNSlice(cmd.Flag("stringSlice"), "sliceval")
	v = cmd.Flag("stringSlice").Value.String()
	assert.Equal(t, v, "[sliceval]", "The flag should be set to value provided.")

	envStringNSlice(cmd.Flag("stringSlice"), "newsliceval")
	v = cmd.Flag("stringSlice").Value.String()
	assert.Equal(t, v, "[sliceval]", "Once set, the flag should not be appended or overwritten.")
}
