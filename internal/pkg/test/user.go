// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package test

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
	"testing"
)

// GetCurrentUser ensures that the correct user structure is retrieved
// when running tests. Because tests are using test.DropPrivilege() and
// test.EnsurePrivilege(t), getting the correct structure needs to be
// done a specific way (user.Current() will not necessarily return the
// expected result).
func GetCurrentUser(t *testing.T) (*user.User, error) {
	uid := os.Getuid()
	me, err := user.LookupId(strconv.Itoa(uid))
	if err != nil {
		return nil, fmt.Errorf("failed to look up ID: %s", err)
	}

	return me, nil
}
