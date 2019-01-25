// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/user"
	"github.com/sylabs/singularity/pkg/util/capabilities"
)

// CapListConfig instructs CapabilityList on what to list
type CapListConfig struct {
	User  string
	Group string
	All   bool
}

// CapabilityList lists the capabilities based on the CapListConfig
func CapabilityList(capFile string, c CapListConfig) error {
	if os.Getuid() != 0 {
		return fmt.Errorf("while listing capabilities: only root user can list capabilities")
	}

	if c.User == "" && c.Group == "" && c.All == false {
		return fmt.Errorf("while listing capabilities: must specify user, group, or listall")
	}

	oldmask := syscall.Umask(0)
	defer syscall.Umask(oldmask)

	file, err := os.OpenFile(capFile, os.O_RDONLY, 0644)
	if err != nil {
		return fmt.Errorf("while opening capability config file: %s", err)
	}
	defer file.Close()

	capConfig, err := capabilities.ReadFrom(file)
	if err != nil {
		return fmt.Errorf("while parsing capability config data: %s", err)
	}

	// if --all specified, take priority over listing specific user/group
	if c.All {
		users, groups := capConfig.ListAllCaps()

		for user, cap := range users {
			if len(cap) > 0 {
				fmt.Printf("%s [user]: %s\n", user, strings.Join(cap, ","))
			}
		}

		for group, cap := range groups {
			if len(cap) > 0 {
				fmt.Printf("%s [group]: %s\n", group, strings.Join(cap, ","))
			}
		}

		return nil
	}

	if c.User != "" {
		if !userExists(c.User) {
			return fmt.Errorf("while listing user capabilities: user does not exist")
		}

		caps := capConfig.ListUserCaps(c.User)
		if len(caps) > 0 {
			fmt.Printf("%s [user]: %s\n", c.User, strings.Join(caps, ","))
		}
	}

	if c.Group != "" {
		if !groupExists(c.Group) {
			return fmt.Errorf("while listing group capabilities: group does not exist")
		}

		caps := capConfig.ListGroupCaps(c.Group)
		if len(caps) > 0 {
			fmt.Printf("%s [group]: %s\n", c.Group, strings.Join(caps, ","))
		}

	}

	return nil
}

// CapManageConfig specifies what capability set to edit in the capability file
type CapManageConfig struct {
	Caps  string
	User  string
	Group string
	Desc  bool
}

type manageType struct {
	UserFn  func(*capabilities.Config, string, []string) error
	GroupFn func(*capabilities.Config, string, []string) error
}

// CapabilityAdd adds the specified capability set to the capability file
func CapabilityAdd(capFile string, c CapManageConfig) error {
	addType := manageType{
		UserFn: func(c *capabilities.Config, a string, b []string) error {
			return c.AddUserCaps(a, b)
		},
		GroupFn: func(c *capabilities.Config, a string, b []string) error {
			return c.AddGroupCaps(a, b)
		},
	}

	return manageCaps(capFile, c, addType)
}

// CapabilityDrop drops the specified capability set from the capability file
func CapabilityDrop(capFile string, c CapManageConfig) error {
	dropType := manageType{
		UserFn: func(c *capabilities.Config, a string, b []string) error {
			return c.DropUserCaps(a, b)
		},
		GroupFn: func(c *capabilities.Config, a string, b []string) error {
			return c.DropGroupCaps(a, b)
		},
	}

	return manageCaps(capFile, c, dropType)
}

func manageCaps(capFile string, c CapManageConfig, t manageType) error {
	if os.Getuid() != 0 {
		return fmt.Errorf("while managing capability file: only root user can manage capabilities")
	}

	oldmask := syscall.Umask(0)
	defer syscall.Umask(oldmask)

	file, err := os.OpenFile(capFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("while opening capability config file: %s", err)
	}
	defer file.Close()

	capConfig, err := capabilities.ReadFrom(file)
	if err != nil {
		return fmt.Errorf("while parsing capability config data: %s", err)
	}

	caps, ign := capabilities.Split(c.Caps)
	if len(ign) > 0 {
		sylog.Warningf("Ignoring unkown capabilities: %s", ign)
	}

	if c.Desc {
		for _, cap := range caps {
			fmt.Printf("%-22s %s\n\n", cap+":", capabilities.Map[cap].Description)
		}
	}

	if c.User != "" {
		if !userExists(c.User) {
			return fmt.Errorf("while setting capabilities for user %s: user does not exist", c.User)
		}

		if err := t.UserFn(capConfig, c.User, caps); err != nil {
			return fmt.Errorf("while setting capabilities for user %s: %s", c.User, err)
		}
	}

	if c.Group != "" {
		if !groupExists(c.Group) {
			return fmt.Errorf("while setting capabilities for group %s: group does not exist", c.Group)
		}

		if err := t.GroupFn(capConfig, c.Group, caps); err != nil {
			return fmt.Errorf("while setting capabilities for group %s: %s", c.Group, err)
		}
	}

	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("while truncating capability config file: %s", err)
	}

	if n, err := file.Seek(0, os.SEEK_SET); err != nil || n != 0 {
		return fmt.Errorf("failed to reset %s cursor: %s", file.Name(), err)
	}

	if _, err := capConfig.WriteTo(file); err != nil {
		return fmt.Errorf("while writing capability data to file: %s", err)
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to flush capability config file %s: %s", file.Name(), err)
	}

	return nil
}

func userExists(usr string) bool {
	if _, err := user.GetPwNam(usr); err != nil {
		return false
	}
	return true
}

func groupExists(group string) bool {
	if _, err := user.GetGrNam(group); err != nil {
		return false
	}
	return true
}
