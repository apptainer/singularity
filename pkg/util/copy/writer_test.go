// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package copy

import (
	"bytes"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

func TestMultiWriter(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	// Create multiwriter instance
	mw := &MultiWriter{}

	// Write some bytes, not duplicated since there is no writer
	n, err := mw.Write([]byte("test"))
	if err != nil {
		t.Error(err)
	}
	if n != 4 {
		t.Errorf("wrong number of bytes written")
	}

	// Create two writers
	buf1 := new(bytes.Buffer)
	buf2 := new(bytes.Buffer)

	// no effect
	mw.Add(nil)

	// Add first writer
	mw.Add(buf1)

	// Write some bytes
	n, err = mw.Write([]byte("test"))
	if err != nil {
		t.Error(err)
	}
	if n != 4 {
		t.Errorf("wrong number of bytes written")
	}

	// Check if first writer get the right content
	if buf1.String() != "test" {
		t.Errorf("wrong data returned")
	}

	// Reset buffer content for later check
	buf1.Reset()

	// Remove it from writer queue
	mw.Del(buf1)

	// Add second writer
	mw.Add(buf2)

	n, err = mw.Write([]byte("test"))
	if err != nil {
		t.Error(err)
	}
	if n != 4 {
		t.Errorf("wrong number of bytes written")
	}

	// Check if second writer get the right content
	if buf2.String() != "test" {
		t.Errorf("wrong data returned")
	}

	// Check that first writer has empty buffer
	if buf1.String() != "" {
		t.Errorf("unexpected data in buf1")
	}

	mw.Del(buf2)
}
