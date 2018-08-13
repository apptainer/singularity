// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package security

import (
	"syscall"

	oci "github.com/opencontainers/runtime-spec/specs-go"
)

// OciNamespaceFlags returns the nsFlags bitfield that is specified in
// the oci config
func OciNamespaceFlags(l *oci.Linux) (retflags uint) {
	for _, namespace := range l.Namespaces {
		switch namespace.Type {
		case oci.UserNamespace:
			retflags |= syscall.CLONE_NEWUSER
		case oci.IPCNamespace:
			retflags |= syscall.CLONE_NEWIPC
		case oci.UTSNamespace:
			retflags |= syscall.CLONE_NEWUTS
		case oci.PIDNamespace:
			retflags |= syscall.CLONE_NEWPID
		case oci.NetworkNamespace:
			retflags |= syscall.CLONE_NEWNET
		case oci.MountNamespace:
			retflags |= syscall.CLONE_NEWNS
		case oci.CgroupNamespace:
			retflags |= 0x2000000
		}
	}

	return
}
