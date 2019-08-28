// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package lock

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/sylabs/singularity/internal/pkg/test"
)

func TestExclusive(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	if _, err := Exclusive(""); err == nil {
		t.Errorf("unexpected success with empty path")
	}

	ch := make(chan bool, 1)

	fd, err := Exclusive("/dev")
	if err != nil {
		t.Error(err)
	}

	go func() {
		Exclusive("/dev")
		ch <- true
	}()

	select {
	case <-time.After(1 * time.Second):
		Release(fd)
		if err := Release(fd); err == nil {
			t.Errorf("unexpected success during Release second call")
		}
	case <-ch:
		t.Errorf("lock acquired")
	}
}

func TestByteRange(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	// test with a wrong file descriptor
	br := NewByteRange(1111, 0, 0)
	if err := br.Lock(); err == nil {
		t.Fatalf("unexpected success with a wrong file descriptor")
	}

	// create the temporary test file used for locking
	f, err := ioutil.TempFile("", "byterange-")
	if err != nil {
		t.Fatalf("failed to create temporary lock file: %s", err)
	}
	testFile := f.Name()
	defer os.Remove(testFile)

	f.Close()

	// write some content in test file
	if err := ioutil.WriteFile(testFile, []byte("testing\n"), 0644); err != nil {
		t.Fatalf("failed to write content in testfile %s: %s", testFile, err)
	}

	// re-open it and use it for testing
	f, err = os.OpenFile(testFile, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("failed to open %s: %s", testFile, err)
	}
	defer f.Close()
	// create the byte-range lock
	br = NewByteRange(int(f.Fd()), 0, 1)

	// acquire the lock, must succeed
	if err := br.Lock(); err != nil {
		t.Fatalf("unexpected error while locking file %s: %s", testFile, err)
	}

	// at this stage we can't test the condition where
	// the lock is already acquired as we are in the same
	// process where locks are shared, so we just release it
	if err := br.Unlock(); err != nil {
		t.Fatalf("unexpected error while releasing lock: %s", err)
	}

	// open /dev/null read-only
	f, err = os.Open("/dev/null")
	if err != nil {
		t.Fatalf("failed to open /dev/null: %s", err)
	}
	br = NewByteRange(int(f.Fd()), 0, 1)

	// acquire a write lock, must fail
	if err := br.Lock(); err == nil {
		t.Fatalf("unexpected success while locking %s", f.Name())
	}
	// acquire a read lock, must succeed
	if err := br.RLock(); err != nil {
		t.Fatalf("unexpected error while getting lock for %s", f.Name())
	}
}
