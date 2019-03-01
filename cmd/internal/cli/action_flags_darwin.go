// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

// platformActionFlags is the list of actionFlags applicable to the
// target platform
var platformActionFlags = []string{
	"bind",
	"docker-login",
	"docker-password",
	"docker-username",
	"home",
	"nohttps",
	"tmpdir",
	"vm",
	"vm-cpu",
	"vm-err",
	"vm-ram",
}

// initPlatformDefaults customizes the default values for the flags in
// actionFlags to make them appropriate for the build target
func initPlatformDefaults() {
	// darwin defaults to running a VM
	vmFlag := actionFlags.Lookup("vm")
	vmFlag.Value.Set("true")
	vmFlag.Changed = false
	vmFlag.DefValue = "true"
}
