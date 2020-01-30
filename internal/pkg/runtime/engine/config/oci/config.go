// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oci

import (
	"encoding/json"
	"fmt"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	cseccomp "github.com/seccomp/containers-golang"
	"github.com/sylabs/singularity/internal/pkg/runtime/engine/config/oci/generate"
	"github.com/sylabs/singularity/internal/pkg/security/seccomp"
)

// Config is the OCI runtime configuration.
type Config struct {
	generate.Generator
	specs.Spec
}

// MarshalJSON implements json.Marshaler.
func (c *Config) MarshalJSON() ([]byte, error) {
	return json.Marshal(&c.Spec)
}

// UnmarshalJSON implements json.Unmarshaler.
func (c *Config) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &c.Spec); err != nil {
		return err
	}
	c.Generator = *generate.New(&c.Spec)
	return nil
}

// DefaultConfig returns an OCI config generator with a
// default OCI configuration.
func DefaultConfig() (*generate.Generator, error) {
	var err error

	config := specs.Spec{
		Version:  specs.Version,
		Hostname: "mrsdalloway",
	}

	config.Root = &specs.Root{
		Path:     "rootfs",
		Readonly: false,
	}
	config.Process = &specs.Process{
		Terminal: false,
		Args: []string{
			"sh",
		},
	}

	config.Process.User = specs.User{}
	config.Process.Env = []string{
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"TERM=xterm",
	}
	config.Process.Cwd = "/"
	config.Process.Rlimits = []specs.POSIXRlimit{
		{
			Type: "RLIMIT_NOFILE",
			Hard: uint64(1024),
			Soft: uint64(1024),
		},
	}

	config.Process.Capabilities = &specs.LinuxCapabilities{
		Bounding: []string{
			"CAP_CHOWN",
			"CAP_DAC_OVERRIDE",
			"CAP_FSETID",
			"CAP_FOWNER",
			"CAP_MKNOD",
			"CAP_NET_RAW",
			"CAP_SETGID",
			"CAP_SETUID",
			"CAP_SETFCAP",
			"CAP_SETPCAP",
			"CAP_NET_BIND_SERVICE",
			"CAP_SYS_CHROOT",
			"CAP_KILL",
			"CAP_AUDIT_WRITE",
		},
		Permitted: []string{
			"CAP_CHOWN",
			"CAP_DAC_OVERRIDE",
			"CAP_FSETID",
			"CAP_FOWNER",
			"CAP_MKNOD",
			"CAP_NET_RAW",
			"CAP_SETGID",
			"CAP_SETUID",
			"CAP_SETFCAP",
			"CAP_SETPCAP",
			"CAP_NET_BIND_SERVICE",
			"CAP_SYS_CHROOT",
			"CAP_KILL",
			"CAP_AUDIT_WRITE",
		},
		Inheritable: []string{
			"CAP_CHOWN",
			"CAP_DAC_OVERRIDE",
			"CAP_FSETID",
			"CAP_FOWNER",
			"CAP_MKNOD",
			"CAP_NET_RAW",
			"CAP_SETGID",
			"CAP_SETUID",
			"CAP_SETFCAP",
			"CAP_SETPCAP",
			"CAP_NET_BIND_SERVICE",
			"CAP_SYS_CHROOT",
			"CAP_KILL",
			"CAP_AUDIT_WRITE",
		},
		Effective: []string{
			"CAP_CHOWN",
			"CAP_DAC_OVERRIDE",
			"CAP_FSETID",
			"CAP_FOWNER",
			"CAP_MKNOD",
			"CAP_NET_RAW",
			"CAP_SETGID",
			"CAP_SETUID",
			"CAP_SETFCAP",
			"CAP_SETPCAP",
			"CAP_NET_BIND_SERVICE",
			"CAP_SYS_CHROOT",
			"CAP_KILL",
			"CAP_AUDIT_WRITE",
		},
		Ambient: []string{
			"CAP_CHOWN",
			"CAP_DAC_OVERRIDE",
			"CAP_FSETID",
			"CAP_FOWNER",
			"CAP_MKNOD",
			"CAP_NET_RAW",
			"CAP_SETGID",
			"CAP_SETUID",
			"CAP_SETFCAP",
			"CAP_SETPCAP",
			"CAP_NET_BIND_SERVICE",
			"CAP_SYS_CHROOT",
			"CAP_KILL",
			"CAP_AUDIT_WRITE",
		},
	}
	config.Mounts = []specs.Mount{
		{
			Destination: "/proc",
			Type:        "proc",
			Source:      "proc",
			Options:     []string{"nosuid", "noexec", "nodev"},
		},
		{
			Destination: "/dev",
			Type:        "tmpfs",
			Source:      "tmpfs",
			Options:     []string{"nosuid", "strictatime", "mode=755", "size=65536k"},
		},
		{
			Destination: "/dev/pts",
			Type:        "devpts",
			Source:      "devpts",
			Options:     []string{"nosuid", "noexec", "newinstance", "ptmxmode=0666", "mode=0620", "gid=5"},
		},
		{
			Destination: "/dev/shm",
			Type:        "tmpfs",
			Source:      "shm",
			Options:     []string{"nosuid", "noexec", "nodev", "mode=1777", "size=65536k"},
		},
		{
			Destination: "/dev/mqueue",
			Type:        "mqueue",
			Source:      "mqueue",
			Options:     []string{"nosuid", "noexec", "nodev"},
		},
		{
			Destination: "/sys",
			Type:        "sysfs",
			Source:      "sysfs",
			Options:     []string{"nosuid", "noexec", "nodev", "ro"},
		},
	}
	config.Linux = &specs.Linux{
		Resources: &specs.LinuxResources{
			Devices: []specs.LinuxDeviceCgroup{
				{
					Allow:  false,
					Access: "rwm",
				},
			},
		},
		Namespaces: []specs.LinuxNamespace{
			{
				Type: "pid",
			},
			{
				Type: "network",
			},
			{
				Type: "ipc",
			},
			{
				Type: "uts",
			},
			{
				Type: "mount",
			},
		},
	}

	if seccomp.Enabled() {
		config.Linux.Seccomp, err = cseccomp.GetDefaultProfile(&config)
		if err != nil {
			return nil, fmt.Errorf("failed to get seccomp default profile: %s", err)
		}
	}

	return &generate.Generator{Config: &config}, nil
}
