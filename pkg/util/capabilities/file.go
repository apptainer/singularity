// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package capabilities

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

// Caplist defines a map of users/groups with associated list of capabilities
type Caplist map[string][]string

type data struct {
	Users  Caplist `json:"users,omitempty"`
	Groups Caplist `json:"groups,omitempty"`
}

// File represents a file containing a list of users/groups
// associated with authorized capabilities
type File struct {
	file *os.File
	data *data
}

// Open reads a capability file provided in path and returns a capability
// file with users/groups authorized capabilities
func Open(path string, readonly bool) (*File, error) {
	oldmask := syscall.Umask(0)
	defer syscall.Umask(oldmask)

	flag := os.O_RDWR | os.O_CREATE
	if readonly {
		flag = os.O_RDONLY
	}

	// check for ownership of capability file before reading
	if !fs.IsOwner(path, 0) {
		return nil, fmt.Errorf("%s must be owned by root", path)
	}

	f, err := os.OpenFile(path, flag, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s capabilities: %s", path, err)
	}

	file := &File{file: f, data: &data{
		Users:  make(Caplist, 0),
		Groups: make(Caplist, 0),
	}}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to read %s: %s", path, err)
	}
	if len(b) > 0 {
		if err := json.Unmarshal(b, file.data); err != nil {
			f.Close()
			return nil, fmt.Errorf("failed to decode JSON data in %s: %s", path, err)
		}
	} else {
		data, err := json.Marshal(file.data)
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("failed to initialize data")
		}
		json.Unmarshal(data, file.data)
	}
	return file, nil
}

// Write writes capability modification into opened file
func (f *File) Write() error {
	json, err := json.MarshalIndent(f.data, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to save capabilities in file %s: %s", f.file.Name(), err)
	}
	if err := f.file.Truncate(0); err != nil {
		return fmt.Errorf("failed to truncate file %s: %s", f.file.Name(), err)
	}
	if n, err := f.file.Seek(0, os.SEEK_SET); err != nil || n != 0 {
		return fmt.Errorf("failed to reset %s cursor: %s", f.file.Name(), err)
	}
	if f.file.Write(json); err != nil {
		return fmt.Errorf("failed to save capabilities in file %s: %s", f.file.Name(), err)
	}
	if f.file.Sync(); err != nil {
		return fmt.Errorf("failed to flush capabilities in file %s: %s", f.file.Name(), err)
	}
	return nil
}

// Close closes capability file
func (f *File) Close() error {
	return f.file.Close()
}

func (f *File) checkCaps(caps []string) error {
	for _, c := range caps {
		if _, ok := Map[c]; !ok {
			return fmt.Errorf("unknown capability %s", c)
		}
	}
	return nil
}

// AddUserCaps adds an authorized capability set to user
func (f *File) AddUserCaps(user string, caps []string) error {
	if err := f.checkCaps(caps); err != nil {
		return err
	}
	for _, cap := range caps {
		present := false
		for _, c := range f.data.Users[user] {
			if c == cap {
				present = true
			}
		}
		if !present {
			f.data.Users[user] = append(f.data.Users[user], cap)
		}
	}
	return nil
}

// AddGroupCaps adds an authorized capability set to group
func (f *File) AddGroupCaps(group string, caps []string) error {
	if err := f.checkCaps(caps); err != nil {
		return err
	}
	for _, cap := range caps {
		present := false
		for _, c := range f.data.Groups[group] {
			if c == cap {
				present = true
			}
		}
		if !present {
			f.data.Groups[group] = append(f.data.Groups[group], cap)
		}
	}
	return nil
}

// DropUserCaps drops a set of capabilities for user
func (f *File) DropUserCaps(user string, caps []string) error {
	if err := f.checkCaps(caps); err != nil {
		return err
	}
	if _, ok := f.data.Users[user]; !ok {
		return fmt.Errorf("user '%s' not found", user)
	}
	for _, cap := range caps {
		for i := len(f.data.Users[user]) - 1; i >= 0; i-- {
			if f.data.Users[user][i] == cap {
				f.data.Users[user] = append(f.data.Users[user][:i], f.data.Users[user][i+1:]...)
			}
		}
	}
	return nil
}

// DropGroupCaps drops a set of capabilities for group
func (f *File) DropGroupCaps(group string, caps []string) error {
	if err := f.checkCaps(caps); err != nil {
		return err
	}
	if _, ok := f.data.Groups[group]; !ok {
		return fmt.Errorf("group '%s' not found", group)
	}
	for _, cap := range caps {
		for i := len(f.data.Groups[group]) - 1; i >= 0; i-- {
			if f.data.Groups[group][i] == cap {
				f.data.Groups[group] = append(f.data.Groups[group][:i], f.data.Groups[group][i+1:]...)
			}
		}
	}
	return nil
}

// ListUserCaps returns a capability list authorized for user
func (f *File) ListUserCaps(user string) []string {
	return f.data.Users[user]
}

// ListGroupCaps returns a capability list authorized for group
func (f *File) ListGroupCaps(group string) []string {
	return f.data.Groups[group]
}

// ListAllCaps returns capability list for both authorized users and groups
func (f *File) ListAllCaps() (Caplist, Caplist) {
	return f.data.Users, f.data.Groups
}

// CheckUserCaps checks if provided capability list for user are whether
// or not authorized by returning two lists, the first one containing
// authorized capabilities and the second one containing unauthorized
// capabilities
func (f *File) CheckUserCaps(user string, caps []string) (authorized []string, unauthorized []string) {
	for _, ca := range caps {
		present := false
		for _, userCap := range f.ListUserCaps(user) {
			if userCap == ca {
				authorized = append(authorized, ca)
				present = true
				break
			}
		}
		if !present {
			unauthorized = append(unauthorized, ca)
		}
	}
	return authorized, unauthorized
}

// CheckGroupCaps checks if provided capability list for group are whether
// or not authorized by returning two lists, the first one containing
// authorized capabilities and the second one containing unauthorized
// capabilities
func (f *File) CheckGroupCaps(group string, caps []string) (authorized []string, unauthorized []string) {
	for _, ca := range caps {
		present := false
		for _, groupCap := range f.ListGroupCaps(group) {
			if groupCap == ca {
				authorized = append(authorized, ca)
				present = true
				break
			}
		}
		if !present {
			unauthorized = append(unauthorized, ca)
		}
	}
	return authorized, unauthorized
}
