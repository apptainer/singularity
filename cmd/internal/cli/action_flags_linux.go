// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

// platformActionFlags is the list of actionFlags applicable to the
// target platform
var platformActionFlags = []string{
	"add-caps",
	"allow-setuid",
	"app",
	"apply-cgroups",
	"bind",
	"cleanenv",
	"contain",
	"containall",
	"containlibs",
	"dns",
	"docker-login",
	"docker-password",
	"docker-username",
	"drop-caps",
	"fakeroot",
	"home",
	"hostname",
	"ipc",
	"keep-privs",
	"net",
	"network",
	"network-args",
	"no-home",
	"nohttps",
	"no-init",
	"no-nv",
	"no-privs",
	"nv",
	"overlay",
	"pid",
	"pwd",
	"scratch",
	"security",
	"tmpdir",
	"userns",
	"uts",
	"vm",
	"vm-cpu",
	"vm-err",
	"vm-ram",
	"workdir",
	"writable",
	"writable-tmpfs",
}

// initPlatformDefaults customizes the default values for the flags in
// actionFlags to make them appropriate for the build target
func initPlatformDefaults() {
	// Linux does not have special defaults
}
