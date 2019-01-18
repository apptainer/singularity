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

	mw := &MultiWriter{}

	n, err := mw.Write([]byte("test"))
	if err != nil {
		t.Error(err)
	}
	if n != 4 {
		t.Errorf("wrong number of bytes written")
	}

	buf1 := new(bytes.Buffer)
	buf2 := new(bytes.Buffer)

	mw.Add(nil)
	mw.Add(buf1)

	n, err = mw.Write([]byte("test"))
	if err != nil {
		t.Error(err)
	}
	if n != 4 {
		t.Errorf("wrong number of bytes written")
	}

	if buf1.String() != "test" {
		t.Errorf("wrong data returned")
	}

	buf1.Reset()

	mw.Del(buf1)
	mw.Add(buf2)

	n, err = mw.Write([]byte("test"))
	if err != nil {
		t.Error(err)
	}
	if n != 4 {
		t.Errorf("wrong number of bytes written")
	}

	if buf2.String() != "test" {
		t.Errorf("wrong data returned")
	}

	if buf1.String() != "" {
		t.Errorf("unexpected data in buf1")
	}

	mw.Del(buf2)
}
