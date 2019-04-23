// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build !build_engine or !linux

package imgbuild

// Init registers runtime engine, this method is called
// from cmd/starter/main_linux.go
func Init(name string) error {
	return nil
}
