// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package user

import (
	"os"
	osuser "os/user"
	"strconv"

	"github.com/sylabs/singularity/pkg/util/namespaces"
)

// User represents an Unix user account information
type User struct {
	Name  string
	UID   uint32
	GID   uint32
	Gecos string
	Dir   string
	Shell string
}

// Group represents an Unix group information
type Group struct {
	Name string
	GID  uint32
}

func convertUser(user *osuser.User) (*User, error) {
	uid, err := strconv.ParseUint(user.Uid, 10, 32)
	if err != nil {
		return nil, err
	}
	gid, err := strconv.ParseUint(user.Gid, 10, 32)
	if err != nil {
		return nil, err
	}
	u := &User{
		Name:  user.Username,
		UID:   uint32(uid),
		GID:   uint32(gid),
		Dir:   user.HomeDir,
		Gecos: user.Name,
		Shell: "/bin/sh",
	}
	return u, nil
}

func convertGroup(group *osuser.Group) (*Group, error) {
	gid, err := strconv.ParseUint(group.Gid, 10, 32)
	if err != nil {
		return nil, err
	}
	return &Group{Name: group.Name, GID: uint32(gid)}, nil
}

// GetPwUID returns a pointer to User structure associated with user uid
func GetPwUID(uid uint32) (*User, error) {
	u, err := osuser.LookupId(strconv.FormatUint(uint64(uid), 10))
	if err != nil {
		return nil, err
	}
	return convertUser(u)
}

// GetPwNam returns a pointer to User structure associated with user name
func GetPwNam(name string) (*User, error) {
	u, err := osuser.Lookup(name)
	if err != nil {
		return nil, err
	}
	return convertUser(u)
}

// GetGrGID returns a pointer to Group structure associated with groud gid
func GetGrGID(gid uint32) (*Group, error) {
	g, err := osuser.LookupGroupId(strconv.FormatUint(uint64(gid), 10))
	if err != nil {
		return nil, err
	}
	return convertGroup(g)
}

// GetGrNam returns a pointer to Group structure associated with group name
func GetGrNam(name string) (*Group, error) {
	g, err := osuser.LookupGroup(name)
	if err != nil {
		return nil, err
	}
	return convertGroup(g)
}

// Current returns a pointer to User structure associated with current
// user
func Current() (*User, error) {
	return GetPwUID(uint32(os.Getuid()))
}

// CurrentOriginal returns a pointer to User structure associated with the
// original current user, if current user is inside a user namespace with a
// custom user mappings, it will returns information about the original user
// otherwise it returns information about the current user
func CurrentOriginal() (*User, error) {
	uid, err := namespaces.HostUID()
	if err != nil {
		return nil, err
	}
	return GetPwUID(uint32(uid))
}
