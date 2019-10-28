// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

func TestGenConf(t *testing.T) {
	tmpl := "testdata/test_default.tmpl"
	tests := []struct {
		name            string
		confInPath      string
		confOutPath     string
		confCorrectPath string
	}{
		{"gen_new", "", "testdata/test_1.out", "testdata/test_1.out.correct"},
		{"gen_update", "testdata/test_2.in", "testdata/test_2.out", "testdata/test_2.out.correct"},
		{"gen_update_newvals", "testdata/test_3.in", "testdata/test_3.out", "testdata/test_3.out.correct"},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			defer os.Remove(tt.confOutPath)

			genConf(tmpl, tt.confInPath, tt.confOutPath)

			if eq, err := compareFile(tt.confOutPath, tt.confCorrectPath); err != nil {
				t.Fatalf("Unable to compare files: %v\n", err)
			} else if !eq {
				t.Fatalf("Output file %v does not match correct output %v\n", tt.confOutPath, tt.confCorrectPath)
			}
		}))
	}
}

func compareFile(p1, p2 string) (bool, error) {
	f1, err := ioutil.ReadFile(p1)
	if err != nil {
		return false, err
	}

	f2, err := ioutil.ReadFile(p2)
	if err != nil {
		return false, err
	}

	return bytes.Equal(f1, f2), nil
}
