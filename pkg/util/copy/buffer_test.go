// Copyright (c) 2019, Sylabs Inc. All rights reserved.
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

	// Create and write some byte to terminal buffer
	ntb := NewTerminalBuffer()

	n, err := ntb.Write([]byte("te"))
	if err != nil {
		t.Error(err)
	}
	if n != 2 {
		t.Errorf("wrong number of bytes written")
	}

	// Get the buffer content
	line := ntb.Line()
	if string(line) != "te" {
		t.Errorf("wrong line returned: %s", line)
	}

	// Write content and end with a newline to clear buffer
	_, err = ntb.Write([]byte("st\n"))
	if err != nil {
		t.Error(err)
	}

	// Test if buffer string is empty
	if string(ntb.Line()) != "" {
		t.Errorf("unexpected line returned")
	}
}
