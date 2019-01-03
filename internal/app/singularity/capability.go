// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build linux

package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/user"
	"github.com/sylabs/singularity/pkg/util/capabilities"
)

// contains flag variables for capability commands
var (
	CapUser    string
	CapGroup   string
	CapDesc    bool
	CapListAll bool
)

const (
	capAdd = iota
	capDrop
	capList
)

func init() {
	SingularityCmd.AddCommand(CapabilityCmd)
	CapabilityCmd.AddCommand(CapabilityAddCmd)
	CapabilityCmd.AddCommand(CapabilityDropCmd)
	CapabilityCmd.AddCommand(CapabilityListCmd)
}

// CapabilityCmd is the capability command
var CapabilityCmd = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("Invalid command")
	},
	DisableFlagsInUseLine: true,

	Use:           docs.CapabilityUse,
	Short:         docs.CapabilityShort,
	Long:          docs.CapabilityLong,
	Example:       docs.CapabilityExample,
	SilenceErrors: true,
}

func manageCap(capStr string, cmd int) {
	if os.Getuid() != 0 {
		sylog.Fatalf("only root user can manage capabilities")
	}

	if CapDesc {
		caps, _ := capabilities.Split(capStr)
		if len(caps) > 0 {
			fmt.Printf("\n")
		} else {
			sylog.Fatalf("unknown %s capabilities", capStr)
		}
		for _, cap := range caps {
			fmt.Printf("%s\n\n", capabilities.Map[cap].Description)
		}
		return
	}

	if CapUser != "" {
		if _, err := user.GetPwNam(CapUser); err != nil {
			sylog.Fatalf("failed to drop user capabilities: %s", err)
		}
	}
	if CapGroup != "" {
		if _, err := user.GetGrNam(CapGroup); err != nil {
			sylog.Fatalf("failed to drop group capabilities: %s", err)
		}
	}

	file, err := capabilities.Open(buildcfg.CAPABILITY_FILE, false)
	if err != nil {
		sylog.Fatalf("%s", err)
	}
	defer file.Close()

	if cmd == capList {
		if CapListAll {
			users, groups := file.ListAllCaps()
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
			return
		}
		if CapUser != "" {
			caps := file.ListUserCaps(CapUser)
			if len(caps) > 0 {
				fmt.Println(strings.Join(caps, ","))
			}
			return
		}
		if CapGroup != "" {
			caps := file.ListGroupCaps(CapGroup)
			if len(caps) > 0 {
				fmt.Println(strings.Join(caps, ","))
			}
			return
		}
		return
	}

	caps, ignored := capabilities.Split(capStr)
	if len(ignored) > 0 {
		sylog.Warningf("unknown capabilities %s were ignored", strings.Join(ignored, ","))
	}

	var userFunc func(string, []string) error
	var groupFunc func(string, []string) error
	action := ""

	switch cmd {
	case capAdd:
		userFunc = file.AddUserCaps
		groupFunc = file.AddGroupCaps
		action = "add"
	case capDrop:
		userFunc = file.DropUserCaps
		groupFunc = file.DropGroupCaps
		action = "drop"
	}

	if CapUser != "" {
		if err := userFunc(CapUser, caps); err != nil {
			sylog.Fatalf("failed to %s %s capabilities for user %s: %s", action, capStr, CapUser, err)
		}
	}
	if CapGroup != "" {
		if err := groupFunc(CapGroup, caps); err != nil {
			sylog.Fatalf("failed to %s %s capabilities for group %s: %s", action, capStr, CapGroup, err)
		}
	}
	if err := file.Write(); err != nil {
		sylog.Fatalf("failed to save changes in capabilities configuration file: %s", err)
	}
}
