// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"os"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/util/user"
)

// CurrentUser returns the current user account information. Use of user.Current is
// not safe with e2e tests as the user information is cached after the first call,
// so it will always return the same user information which could be wrong if
// user.Current was first called in unprivileged context and called after in a
// privileged context as it will return information of unprivileged user.
func CurrentUser(t *testing.T) *user.User {
	u, err := user.GetPwUID(uint32(os.Getuid()))
	if err != nil {
		t.Fatalf("failed to retrieve user information")
	}
	return u
}
