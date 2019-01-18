// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package lock

import (
	"testing"
	"time"

	"github.com/sylabs/singularity/internal/pkg/test"
)

func TestLock(t *testing.T) {
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
