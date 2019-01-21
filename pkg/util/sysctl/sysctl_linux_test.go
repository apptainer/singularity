// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sysctl

import (
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

func TestGetSet(t *testing.T) {
	test.EnsurePrivilege(t)

	value, err := Get("net.ipv4.ip_forward")
	if err != nil {
		t.Error(err)
	}

	if value != "0" && value != "1" {
		t.Fatalf("non expected value 0 or 1: %s", value)
	}

	if err := Set("net.ipv4.ip_forward", value); err != nil {
		t.Error(err)
	}

	value, err = Get("net.ipv4.ip_forward2")
	if err == nil {
		t.Errorf("shoud have failed, key doesn't exists")
	}

	if err := Set("net.ipv4.ip_forward2", value); err == nil {
		t.Errorf("shoud have failed, key doesn't exists")
	}
}
