// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package arrayhelper

import (
	"strings"
	"testing"
)

func Test_IsIn(t *testing.T) {
	sample := []string{"foo", "bar"}
	blank := []string{}

	t.Run("string present", func(t *testing.T) {
		if out := IsIn(sample, "foo"); !out {
			t.Errorf("arrayhelper.IsIn returned false when searching for \"bar\" in array [\"foo\" \"bar\"]")
		}
	})

	t.Run("string absent", func(t *testing.T) {
		if out := IsIn(sample, "blah"); out {
			t.Errorf("arrayhelper.IsIn returned true when searching for \"blah\" in array [\"foo\" \"bar\"]")
		}
	})

	t.Run("blank array", func(t *testing.T) {
		if out := IsIn(blank, "foo"); out {
			t.Errorf("arrayhelper.IsIn returned true when searching for \"foo\" in array []")
		}
	})
}

func Test_Unique(t *testing.T) {
	sampleA := []string{"foo", "bar", "bar"}
	sampleB := []string{"foo", "bar"}
	blank := []string{}

	t.Run("duplicates present", func(t *testing.T) {
		if out := strings.Join(Unique(sampleA), " "); out != "foo bar" {
			t.Errorf("arrayhelper.Unique did not properly remove duplicates from array")
		}
	})

	t.Run("duplicates absent", func(t *testing.T) {
		if out := strings.Join(Unique(sampleB), " "); out != "foo bar" {
			t.Errorf("arrayhelper.Unique did not properly skip removing duplicates when they were absent")
		}
	})

	t.Run("blank array", func(t *testing.T) {
		if out := strings.Join(Unique(blank), " "); out != "" {
			t.Errorf("arrayhelper.Unique did not return a blank array when given a blank array as input")
		}
	})
}
