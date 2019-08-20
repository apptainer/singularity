/*
 * Imported from opencontainers/runc/libcontainer/user.
 * Copyright (C) 2014 Docker, Inc.
 * Copyright (C) The Linux Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package user

import (
	"errors"
	"syscall"
)

var (
	// The current operating system does not provide the required data for user lookups.
	ErrUnsupported = errors.New("user lookup: operating system does not provide passwd-formatted data")
	// No matching entries found in file.
	ErrNoPasswdEntries = errors.New("no matching entries in passwd file")
	ErrNoGroupEntries  = errors.New("no matching entries in group file")
)

func lookupUser(filter func(u User) bool) (User, error) {
	// Get operating system-specific passwd reader-closer.
	passwd, err := GetPasswd()
	if err != nil {
		return User{}, err
	}
	defer passwd.Close()

	// Get the users.
	users, err := ParsePasswdFilter(passwd, filter)
	if err != nil {
		return User{}, err
	}

	// No user entries found.
	if len(users) == 0 {
		return User{}, ErrNoPasswdEntries
	}

	// Assume the first entry is the "correct" one.
	return users[0], nil
}

// CurrentUser looks up the current user by their user id in /etc/passwd. If the
// user cannot be found (or there is no /etc/passwd file on the filesystem),
// then CurrentUser returns an error.
func CurrentUser() (User, error) {
	return LookupUid(syscall.Getuid())
}

// LookupUser looks up a user by their username in /etc/passwd. If the user
// cannot be found (or there is no /etc/passwd file on the filesystem), then
// LookupUser returns an error.
func LookupUser(username string) (User, error) {
	return lookupUser(func(u User) bool {
		return u.Name == username
	})
}

// LookupUid looks up a user by their user id in /etc/passwd. If the user cannot
// be found (or there is no /etc/passwd file on the filesystem), then LookupId
// returns an error.
func LookupUid(uid int) (User, error) {
	return lookupUser(func(u User) bool {
		return u.Uid == uid
	})
}

func lookupGroup(filter func(g Group) bool) (Group, error) {
	// Get operating system-specific group reader-closer.
	group, err := GetGroup()
	if err != nil {
		return Group{}, err
	}
	defer group.Close()

	// Get the users.
	groups, err := ParseGroupFilter(group, filter)
	if err != nil {
		return Group{}, err
	}

	// No user entries found.
	if len(groups) == 0 {
		return Group{}, ErrNoGroupEntries
	}

	// Assume the first entry is the "correct" one.
	return groups[0], nil
}

// CurrentGroup looks up the current user's group by their primary group id's
// entry in /etc/passwd. If the group cannot be found (or there is no
// /etc/group file on the filesystem), then CurrentGroup returns an error.
func CurrentGroup() (Group, error) {
	return LookupGid(syscall.Getgid())
}

// LookupGroup looks up a group by its name in /etc/group. If the group cannot
// be found (or there is no /etc/group file on the filesystem), then LookupGroup
// returns an error.
func LookupGroup(groupname string) (Group, error) {
	return lookupGroup(func(g Group) bool {
		return g.Name == groupname
	})
}

// LookupGid looks up a group by its group id in /etc/group. If the group cannot
// be found (or there is no /etc/group file on the filesystem), then LookupGid
// returns an error.
func LookupGid(gid int) (Group, error) {
	return lookupGroup(func(g Group) bool {
		return g.Gid == gid
	})
}
