// Copyright (c) 2018-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

//go:build apparmor
// +build apparmor

package apparmor

import (
	"fmt"
	"io/ioutil"
	"os"
)

// Enabled returns whether AppArmor is enabled.
func Enabled() bool {
	data, err := ioutil.ReadFile("/sys/module/apparmor/parameters/enabled")
	if err == nil && len(data) > 0 && data[0] == 'Y' {
		return true
	}
	return false
}

// LoadProfile loads the specified AppArmor profile.
func LoadProfile(profile string) error {
	f, err := os.OpenFile("/proc/self/attr/exec", os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()

	p := "exec " + profile
	if _, err := f.Write([]byte(p)); err != nil {
		return fmt.Errorf("failed to set apparmor profile (%s)", err)
	}
	return nil
}
