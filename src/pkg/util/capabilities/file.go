// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package capabilities

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/singularityware/singularity/src/pkg/buildcfg"
	"github.com/singularityware/singularity/src/pkg/util/fs/layout"
)

type File struct {
	Users  map[string][]string `json:"users,omitempty"`
	Groups map[string][]string `json:"groups,omitempty"`
}

// Split takes a list of capabilities separated by commas and
// returns a string list with normalized capability name and a
// second list with unrecognized capabitilies
func Split(caps string) ([]string, []string) {
	include := make([]string, 0)
	exclude := make([]string, 0)

	f := func(c rune) bool {
		if c == ',' {
			return true
		}
		return false
	}
	capabilities := strings.FieldsFunc(caps, f)

	for _, capability := range capabilities {
		c := strings.ToUpper(capability)
		if !strings.HasPrefix(c, "CAP_") {
			c = "CAP_" + c
		}
		if _, ok := Map[c]; !ok {
			exclude = append(exclude, capability)
			continue
		}
		include = append(include, c)
	}

	return include, exclude
}

func Read(path string) (*File, error) {
	capabilitiesDir := filepath.Join(buildcfg.SYSCONFDIR, "singularity/capabilities")

	layoutManager := &layout.Manager{}
	if err := layoutManager.SetRootPath(capabilitiesDir); err != nil {
		return nil, err
	}

	return &File{}, nil
}

func (f *File) checkCaps(caps []string) error {
	for _, c := range caps {
		if _, ok := Map[c]; !ok {
			return fmt.Errorf("unknown capability %s", c)
		}
	}
	return nil
}

func (f *File) checkUser(user string) error {
	if _, ok := f.Users[user]; !ok {
		return fmt.Errorf("user '%s' not found", user)
	}
	return nil
}

func (f *File) checkGroup(group string) error {
	if _, ok := f.Groups[group]; !ok {
		return fmt.Errorf("group '%s' not found", group)
	}
	return nil
}

func (f *File) AddUserCaps(user string, caps []string) error {
	if err := f.checkUser(user); err != nil {
		return err
	}
	if err := f.checkCaps(caps); err != nil {
		return err
	}
	if err := f.checkUser(user); err != nil {
		f.Users[user] = make([]string, 0)
	}
	f.Users[user] = append(f.Users[user], caps...)
	return nil
}

func (f *File) AddGroupCaps(group string, caps []string) error {
	if err := f.checkCaps(caps); err != nil {
		return err
	}
	if err := f.checkGroup(group); err != nil {
		f.Groups[group] = make([]string, 0)
	}
	f.Groups[group] = append(f.Groups[group], caps...)
	return nil
}

func (f *File) DropUserCaps(user string, caps []string) error {
	if err := f.checkUser(user); err != nil {
		return err
	}
	if err := f.checkCaps(caps); err != nil {
		return err
	}
	return nil
}

func (f *File) DropGroupCaps(group string, caps []string) error {
	if err := f.checkGroup(group); err != nil {
		return err
	}
	if err := f.checkCaps(caps); err != nil {
		return err
	}
	return nil
}

func (f *File) ListUserCaps(user string) ([]string, error) {
	if err := f.checkUser(user); err != nil {
		return nil, err
	}
	return f.Users[user], nil
}

func (f *File) ListGroupCaps(group string) ([]string, error) {
	if err := f.checkGroup(group); err != nil {
		return nil, err
	}
	return f.Groups[group], nil
}

func (f *File) ListAllCaps() error {
	return nil
}
