// Copyright (c) 2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package user

import (
	"github.com/sylabs/singularity/pkg/sylog"
	"github.com/sylabs/singularity/pkg/util/slice"
	"os/user"
)

// UserInList returns true if the user with supplied uid is in list
// List is a string slice that may contain UIDs, usernames, or both.
func UserInList(uid string, list []string) (bool, error) {
	eUser, err := user.LookupId(uid)
	if err != nil {
		return false, err
	}
	return slice.ContainsAnyString(list, []string{ eUser.Uid, eUser.Name }), nil
}

// UserInGroup returns true if the user with supplied uid is a member of a group in list
// List is a string slice that may contain GIDs, groupnames, or both.
func UserInGroup(uid string, list []string) (bool, error) {
	u, err := user.LookupId(uid)
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