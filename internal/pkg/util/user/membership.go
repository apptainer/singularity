// Copyright (c) 2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package user

import (
	"os/user"
	"strconv"

	"github.com/sylabs/singularity/pkg/sylog"
	"github.com/sylabs/singularity/pkg/util/slice"
)

// UIDInList returns true if the user with supplied uid is in list (match by uid or username).
// List is a string slice that may contain UIDs, usernames, or both.
func UIDInList(uid int, list []string) (bool, error) {
	uidStr := strconv.Itoa(uid)
	u, err := lookupUnixUid(uid)
	if err != nil {
		return false, err
	}
	return slice.ContainsAnyString(list, []string{uidStr, u.Name}), nil
}

// UIDInAnyGroup returns true if the user with supplied uid is a member of any group in list.
// List is a string slice that may contain GIDs, groupnames, or both.
func UIDInAnyGroup(uid int, list []string) (bool, error) {
	uidStr := strconv.Itoa(uid)
	u, err := user.LookupId(uidStr)
	if err != nil {
		return false, err
	}
	// Get the numeric GIDs
	userGroups, err := u.GroupIds()
	if err != nil {
		return false, err
	}
	// Append the group names
	for _, g := range userGroups {
		gname, err := user.LookupGroupId(g)
		if err != nil {
			sylog.Warningf("while looking up gid %s: %v", g, err)
			continue
		}
		userGroups = append(userGroups, gname.Name)
	}
	// Match on the GIDs or group names
	return slice.ContainsAnyString(list, userGroups), nil
}
