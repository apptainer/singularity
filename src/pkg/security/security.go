// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package security

import (
	"fmt"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/singularityware/singularity/src/pkg/security/apparmor"
	"github.com/singularityware/singularity/src/pkg/security/seccomp"
	"github.com/singularityware/singularity/src/pkg/security/selinux"
	"github.com/singularityware/singularity/src/pkg/sylog"
)

// Configure applies security related configuration to current process
func Configure(config *specs.Spec) error {
	if config.Linux != nil && config.Linux.Seccomp != nil {
		if seccomp.Enabled() {
			if err := seccomp.LoadSeccompConfig(config.Linux.Seccomp); err != nil {
				return err
			}
		} else {
			sylog.Warningf("seccomp requested but not enabled")
		}
	}
	if config.Process != nil {
		if config.Process.SelinuxLabel != "" && config.Process.ApparmorProfile != "" {
			return fmt.Errorf("You can't specify both an apparmor profile and a SELinux label")
		}
		if config.Process.SelinuxLabel != "" {
			if selinux.Enabled() {
				if err := selinux.SetExecLabel(config.Process.SelinuxLabel); err != nil {
					return err
				}
			} else {
				sylog.Warningf("selinux is not enabled or supported on this system")
			}
		} else if config.Process.ApparmorProfile != "" {
			if apparmor.Enabled() {
				if err := apparmor.LoadProfile(config.Process.ApparmorProfile); err != nil {
					return err
				}
			} else {
				sylog.Warningf("apparmor is not enabled or supported on this system")
			}
		}
	}
	return nil
}
