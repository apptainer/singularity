// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build !sylog

package sylog

import (
	"io/ioutil"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

const envStr = "SINGULARITY_MESSAGELEVEL=-1"

func TestGetLevel(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	l := GetLevel()
	if l != -1 {
		t.Fatalf("%d was returned instead of -1", l)
	}
}

func TestGetEnvVar(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	str := GetEnvVar()
	if str != envStr {
		t.Fatalf("%s was returned instead of %s", str, envStr)
	}
}

func TestWriter(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	w := Writer()
	if w != ioutil.Discard {
		t.Fatalf("Writer() did not return ioutil.Discard as expected")
	}
}

func TestNoOps(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	tests := []struct {
		name string
		str  string
	}{
		{
			name: "empty",
			str:  "",
		},
		{
			name: "string",
			str:  "dummy",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Errorf(tt.str)
			Warningf(tt.str)
			Infof(tt.str)
			Verbosef(tt.str)
			Debugf(tt.str)
		})
	}

	SetLevel(0)
	DisableColor()
}
