// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package config

import (
	"fmt"
	"os"
	"syscall"
)

// CheckUid for ownership of config file before reading it
func CheckUid(filepath string) error {
	fmt.Println(filepath)
	configFile, err := os.Stat(filepath)
	if err != nil {
		return fmt.Errorf("Error with Stat() on %s: %v", filepath, err)
	}
	if int(configFile.Sys().(*syscall.Stat_t).Uid) != 0 {
		return fmt.Errorf("%s must be owned by root", filepath)
	}
	return nil
}
