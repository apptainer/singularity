// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package copy

import (
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

func TestNewTerminalBuffer(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	ntb := NewTerminalBuffer()

	n, err := ntb.Write([]byte("te"))
	if err != nil {
		t.Error(err)
	}
	if n != 2 {
		t.Errorf("wrong number of bytes written")
	}
	line := ntb.Line()
	if string(line) != "te" {
		t.Errorf("wrong line returned: %s", line)
	}

	n, err = ntb.Write([]byte("st\n"))
	if err != nil {
		t.Error(err)
	}
	if ntb.Line() != nil {
		t.Errorf("unexpected line returned")
	}
}
