// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sypgp

import (
	"log"
	"os"
	"testing"

	"golang.org/x/crypto/openpgp"
)

const (
	testName    = "Test Name"
	testComment = "blah"
	testEmail   = "test@test.com"
)

var (
	testEntity *openpgp.Entity
)

func TestMain(m *testing.M) {
	e, err := openpgp.NewEntity(testName, testComment, testEmail, nil)
	if err != nil {
		log.Fatalf("failed to create entity: %v", err)
	}
	testEntity = e

	os.Exit(m.Run())
}
