// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"os"
	"strings"
	"syscall"

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
	if c.User == "" && c.Group == "" && !c.All {
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

	outputCaps := 0

	// if --all specified, take priority over listing specific user/group
	if c.All {
		users, groups := capConfig.ListAllCaps()

		for user, cap := range users {
			if len(cap) > 0 {
				fmt.Printf("%s [user]: %s\n", user, strings.Join(cap, ","))
				outputCaps++
			}
		}

		for group, cap := range groups {
			if len(cap) > 0 {
				fmt.Printf("%s [group]: %s\n", group, strings.Join(cap, ","))
				outputCaps++
			}
		}

		if outputCaps == 0 {
			return fmt.Errorf("no capability set for users or groups")
		}

		return nil
	}

	if c.User != "" {
		caps := capConfig.ListUserCaps(c.User)
		if len(caps) > 0 {
			fmt.Printf("%s [user]: %s\n", c.User, strings.Join(caps, ","))
			outputCaps++
		}
	}

	if c.Group != "" {
		caps := capConfig.ListGroupCaps(c.Group)
		if len(caps) > 0 {
			fmt.Printf("%s [group]: %s\n", c.Group, strings.Join(caps, ","))
			outputCaps++
		}
	}

	if outputCaps == 0 {
		return fmt.Errorf("no capability set for user/group %s", c.User)
	}

	return nil
}
