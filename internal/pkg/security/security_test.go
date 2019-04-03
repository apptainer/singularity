// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package security

import (
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"

	"github.com/sylabs/singularity/internal/pkg/security/apparmor"

	"github.com/sylabs/singularity/internal/pkg/security/selinux"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func TestGetParam(t *testing.T) {
	paramTests := []struct {
		security []string
		feature  string
		result   string
	}{
		{
			security: []string{"seccomp:test"},
			feature:  "seccomp",
			result:   "test",
		},
		{
			security: []string{"test:test"},
			feature:  "seccomp",
			result:   "",
		},
		{
			security: []string{"seccomp:test", "uid:1000"},
			feature:  "uid",
			result:   "1000",
		},
	}
	for _, p := range paramTests {
		r := GetParam(p.security, p.feature)
		if p.result != r {
			t.Errorf("unexpected result for param %v, returned %s instead of %s", p.security, r, p.result)
		}
	}
}

func TestConfigure(t *testing.T) {
	test.EnsurePrivilege(t)

	specs := []struct {
		desc          string
		spec          specs.Spec
		expectFailure bool
		disabled      bool
	}{
		{
			desc: "empty security spec",
			spec: specs.Spec{},
		},
		{
			desc: "both SELinux context and apparmor profile",
			spec: specs.Spec{
				Process: &specs.Process{
					SelinuxLabel:    "test",
					ApparmorProfile: "test",
				},
			},
			expectFailure: true,
		},
		{
			desc: "with bad SELinux context",
			spec: specs.Spec{
				Process: &specs.Process{
					SelinuxLabel: "test",
				},
			},
			expectFailure: true,
			disabled:      !selinux.Enabled(),
		},
		{
			desc: "with unconfined SELinux context",
			spec: specs.Spec{
				Process: &specs.Process{
					SelinuxLabel: "unconfined_u:unconfined_r:unconfined_t",
				},
			},
			disabled: !selinux.Enabled(),
		},
		{
			desc: "with bad apparmor profile",
			spec: specs.Spec{
				Process: &specs.Process{
					SelinuxLabel: "__test__",
				},
			},
			expectFailure: true,
			disabled:      !apparmor.Enabled(),
		},
		{
			desc: "with unconfined apparmor profile",
			spec: specs.Spec{
				Process: &specs.Process{
					SelinuxLabel: "unconfined",
				},
			},
			disabled: !apparmor.Enabled(),
		},
	}

	for _, s := range specs {
		if s.disabled {
			continue
		}
		err := Configure(&s.spec)
		if err != nil && !s.expectFailure {
			t.Errorf("unexpected failure with %s", s.desc)
		}
	}
}
