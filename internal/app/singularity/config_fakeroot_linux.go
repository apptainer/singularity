// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"

	"github.com/sylabs/singularity/internal/pkg/fakeroot"
)

// FakerootConfigOp defines a type for a fakeroot
// configuration operation.
type FakerootConfigOp uint8

const (
	// FakerootAddUser is the operation to add a user fakeroot mapping.
	FakerootAddUser FakerootConfigOp = iota
	// FakerootRemoveUser is the operation to remove a user fakeroot mapping.
	FakerootRemoveUser
	// FakerootEnableUser is the operation to enable a user fakeroot mapping.
	FakerootEnableUser
	// FakerootDisableUser is the operation to disable a user fakeroot mapping.
	FakerootDisableUser
)

// FakerootConfig allows to add/remove/enable/disable a user fakeroot
// mapping entry in /etc/subuid and /etc/subgid files.
func FakerootConfig(username string, op FakerootConfigOp) error {
	subUIDConfig, err := fakeroot.GetConfig(fakeroot.SubUIDFile, true, nil)
	if err != nil {
		return fmt.Errorf("while opening %s: %s", fakeroot.SubUIDFile, err)
	}
	subGIDConfig, err := fakeroot.GetConfig(fakeroot.SubGIDFile, true, nil)
	if err != nil {
		return fmt.Errorf("while opening %s: %s", fakeroot.SubGIDFile, err)
	}

	switch op {
	case FakerootAddUser:
		if err := subUIDConfig.AddUser(username); err != nil {
			return fmt.Errorf("while adding %s: %s", username, err)
		}
		if err := subGIDConfig.AddUser(username); err != nil {
			return fmt.Errorf("while adding %s: %s", username, err)
		}
	case FakerootRemoveUser:
		if err := subUIDConfig.RemoveUser(username); err != nil {
			return fmt.Errorf("while removing %s: %s", username, err)
		}
		if err := subGIDConfig.RemoveUser(username); err != nil {
			return fmt.Errorf("while removing %s: %s", username, err)
		}
	case FakerootEnableUser:
		if err := subUIDConfig.EnableUser(username); err != nil {
			return fmt.Errorf("while enabling %s: %s", username, err)
		}
		if err := subGIDConfig.EnableUser(username); err != nil {
			return fmt.Errorf("while enabling %s: %s", username, err)
		}
	case FakerootDisableUser:
		if err := subUIDConfig.DisableUser(username); err != nil {
			return fmt.Errorf("while disabling %s: %s", username, err)
		}
		if err := subGIDConfig.DisableUser(username); err != nil {
			return fmt.Errorf("while disabling %s: %s", username, err)
		}
	default:
		return fmt.Errorf("unknown configuration operation")
	}

	if err := subUIDConfig.Close(); err != nil {
		return fmt.Errorf("while writing configuration: %s", err)
	}
	if err := subGIDConfig.Close(); err != nil {
		return fmt.Errorf("while writing configuration: %s", err)
	}

	return nil
}
