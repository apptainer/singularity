// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build apparmor

package apparmor

import (
	"fmt"
	"io/ioutil"
	"os"
)

// LoadProfile write apparmor profile in /proc/self/attr/exec
func LoadProfile(profile string) error {
	data, err := ioutil.ReadFile("/sys/module/apparmor/parameters/enabled")
	if err == nil {
		if len(data) > 0 && data[0] == 'Y' {
			return writeProfile(profile)
		}
		return fmt.Errorf("apparmor is not enabled")
	}
	return fmt.Errorf("no apparmor support found")
}

func writeProfile(profile string) error {
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
