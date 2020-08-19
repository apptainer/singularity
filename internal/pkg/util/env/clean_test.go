// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package env

import (
	"reflect"
	"strings"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/runtime/engine/config/oci"
	"github.com/sylabs/singularity/internal/pkg/runtime/engine/config/oci/generate"
	"github.com/sylabs/singularity/internal/pkg/test"
)

func TestSetContainerEnv(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	tt := []struct {
		name           string
		cleanEnv       bool
		homeDest       string
		env            []string
		resultEnv      []string
		singularityEnv map[string]string
	}{
		{
			name:     "no SINGULARITYENV_",
			homeDest: "/home/tester",
			env: []string{
				"LD_LIBRARY_PATH=/.singularity.d/libs",
				"HOME=/home/john",
				"SOME_INVALID_VAR:test",
				"SINGULARITYENV_=invalid",
				"PS1=test",
				"TERM=xterm-256color",
				"PATH=/usr/games:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
				"LANG=C",
				"SINGULARITY_CONTAINER=/tmp/lolcow.sif",
				"PWD=/tmp",
				"LC_ALL=C",
				"SINGULARITY_NAME=lolcow.sif",
			},
			resultEnv: []string{
				"PS1=test",
				"TERM=xterm-256color",
				"LANG=C",
				"PWD=/tmp",
				"LC_ALL=C",
				"HOME=/home/tester",
				"PATH=" + DefaultPath,
			},
		},
		{
			name:     "exclude PATH",
			homeDest: "/home/tester",
			env: []string{
				"LD_LIBRARY_PATH=/.singularity.d/libs",
				"HOME=/home/john",
				"PS1=test",
				"SOCIOPATH=VolanDeMort",
				"TERM=xterm-256color",
				"PATH=/usr/games:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
				"LANG=C",
				"SINGULARITYENV_LD_LIBRARY_PATH=/my/custom/libs",
				"SINGULARITY_CONTAINER=/tmp/lolcow.sif",
				"PWD=/tmp",
				"LC_ALL=C",
				"SINGULARITY_NAME=lolcow.sif",
			},
			resultEnv: []string{
				"PS1=test",
				"SOCIOPATH=VolanDeMort",
				"TERM=xterm-256color",
				"LANG=C",
				"PWD=/tmp",
				"LC_ALL=C",
				"HOME=/home/tester",
				"PATH=" + DefaultPath,
			},
			singularityEnv: map[string]string{
				"LD_LIBRARY_PATH": "/my/custom/libs",
			},
		},
		{
			name:     "special PATH envs",
			homeDest: "/home/tester",
			env: []string{
				"LD_LIBRARY_PATH=/.singularity.d/libs",
				"HOME=/home/john",
				"SINGULARITYENV_APPEND_PATH=/sylabs/container",
				"PS1=test",
				"TERM=xterm-256color",
				"SINGULARITYENV_PATH=/my/path",
				"PATH=/usr/games:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
				"LANG=C",
				"SINGULARITY_CONTAINER=/tmp/lolcow.sif",
				"PWD=/tmp",
				"LC_ALL=C",
				"SINGULARITYENV_PREPEND_PATH=/foo/bar",
				"SINGULARITY_NAME=lolcow.sif",
			},
			resultEnv: []string{
				"PS1=test",
				"TERM=xterm-256color",
				"LANG=C",
				"PWD=/tmp",
				"LC_ALL=C",
				"HOME=/home/tester",
				"PATH=" + DefaultPath,
			},
			singularityEnv: map[string]string{
				"SING_USER_DEFINED_PREPEND_PATH": "/foo/bar",
				"SING_USER_DEFINED_PATH":         "/my/path",
				"SING_USER_DEFINED_APPEND_PATH":  "/sylabs/container",
			},
		},
		{
			name:     "clean envs",
			cleanEnv: true,
			homeDest: "/home/tester",
			env: []string{
				"LD_LIBRARY_PATH=/.singularity.d/libs",
				"HOME=/home/john",
				"PS1=test",
				"TERM=xterm-256color",
				"PATH=/usr/games:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
				"LANG=C",
				"SINGULARITY_CONTAINER=/tmp/lolcow.sif",
				"PWD=/tmp",
				"LC_ALL=C",
				"SINGULARITY_NAME=lolcow.sif",
				"SINGULARITYENV_FOO=VAR",
				"CLEANENV=TRUE",
			},
			resultEnv: []string{
				"LANG=C",
				"TERM=xterm-256color",
				"HOME=/home/tester",
				"PATH=" + DefaultPath,
			},
			singularityEnv: map[string]string{
				"FOO": "VAR",
			},
		},
		{
			name:     "always pass keys",
			cleanEnv: true,
			homeDest: "/home/tester",
			env: []string{
				"LD_LIBRARY_PATH=/.singularity.d/libs",
				"HOME=/home/john",
				"PS1=test",
				"TERM=xterm-256color",
				"PATH=/usr/games:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
				"LANG=C",
				"SINGULARITY_CONTAINER=/tmp/lolcow.sif",
				"PWD=/tmp",
				"LC_ALL=C",
				"http_proxy=http_proxy",
				"https_proxy=https_proxy",
				"no_proxy=no_proxy",
				"all_proxy=all_proxy",
				"ftp_proxy=ftp_proxy",
				"HTTP_PROXY=http_proxy",
				"HTTPS_PROXY=https_proxy",
				"NO_PROXY=no_proxy",
				"ALL_PROXY=all_proxy",
				"FTP_PROXY=ftp_proxy",
				"SINGULARITY_NAME=lolcow.sif",
				"SINGULARITYENV_FOO=VAR",
				"CLEANENV=TRUE",
			},
			resultEnv: []string{
				"LANG=C",
				"TERM=xterm-256color",
				"http_proxy=http_proxy",
				"https_proxy=https_proxy",
				"no_proxy=no_proxy",
				"all_proxy=all_proxy",
				"ftp_proxy=ftp_proxy",
				"HTTP_PROXY=http_proxy",
				"HTTPS_PROXY=https_proxy",
				"NO_PROXY=no_proxy",
				"ALL_PROXY=all_proxy",
				"FTP_PROXY=ftp_proxy",
				"HOME=/home/tester",
				"PATH=" + DefaultPath,
			},
			singularityEnv: map[string]string{
				"FOO": "VAR",
			},
		},
		{
			name:     "SINGULARITYENV_PATH",
			cleanEnv: false,
			homeDest: "/home/tester",
			env: []string{
				"SINGULARITYENV_PATH=/my/path",
			},
			resultEnv: []string{
				"HOME=/home/tester",
				"PATH=" + DefaultPath,
			},
			singularityEnv: map[string]string{
				"SING_USER_DEFINED_PATH": "/my/path",
			},
		},
		{
			name:     "SINGULARITYENV_LANG with cleanenv",
			cleanEnv: true,
			homeDest: "/home/tester",
			env: []string{
				"SINGULARITYENV_LANG=en",
			},
			resultEnv: []string{
				"HOME=/home/tester",
				"PATH=" + DefaultPath,
			},
			singularityEnv: map[string]string{
				"LANG": "en",
			},
		},
		{
			name:     "SINGULARITYENV_HOME",
			cleanEnv: false,
			homeDest: "/home/tester",
			env: []string{
				"SINGULARITYENV_HOME=/my/home",
			},
			resultEnv: []string{
				"HOME=/home/tester",
				"PATH=" + DefaultPath,
			},
			singularityEnv: map[string]string{},
		},
		{
			name:     "SINGULARITYENV_LD_LIBRARY_PATH",
			cleanEnv: false,
			homeDest: "/home/tester",
			env: []string{
				"SINGULARITYENV_LD_LIBRARY_PATH=/my/libs",
			},
			resultEnv: []string{
				"HOME=/home/tester",
				"PATH=" + DefaultPath,
			},
			singularityEnv: map[string]string{
				"LD_LIBRARY_PATH": "/my/libs",
			},
		},
		{
			name:     "SINGULARITYENV_LD_LIBRARY_PATH with cleanenv",
			cleanEnv: true,
			homeDest: "/home/tester",
			env: []string{
				"SINGULARITYENV_LD_LIBRARY_PATH=/my/libs",
			},
			resultEnv: []string{
				"LANG=C",
				"HOME=/home/tester",
				"PATH=" + DefaultPath,
			},
			singularityEnv: map[string]string{
				"LD_LIBRARY_PATH": "/my/libs",
			},
		},
		{
			name:     "SINGULARITYENV_HOST after HOST",
			cleanEnv: false,
			homeDest: "/home/tester",
			env: []string{
				"HOST=myhost",
				"SINGULARITYENV_HOST=myhostenv",
			},
			resultEnv: []string{
				"HOME=/home/tester",
				"PATH=" + DefaultPath,
			},
			singularityEnv: map[string]string{
				"HOST": "myhostenv",
			},
		},
		{
			name:     "SINGULARITYENV_HOST before HOST",
			cleanEnv: false,
			homeDest: "/home/tester",
			env: []string{
				"SINGULARITYENV_HOST=myhostenv",
				"HOST=myhost",
			},
			resultEnv: []string{
				"HOME=/home/tester",
				"PATH=" + DefaultPath,
			},
			singularityEnv: map[string]string{
				"HOST": "myhostenv",
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ociConfig := &oci.Config{}
			generator := generate.New(&ociConfig.Spec)

			senv := SetContainerEnv(generator, tc.env, tc.cleanEnv, tc.homeDest)
			if !equal(t, ociConfig.Process.Env, tc.resultEnv) {
				t.Fatalf("unexpected envs:\n want: %v\ngot: %v", tc.resultEnv, ociConfig.Process.Env)
			}
			if tc.singularityEnv != nil && !reflect.DeepEqual(senv, tc.singularityEnv) {
				t.Fatalf("unexpected singularity env:\n want: %v\ngot: %v", tc.singularityEnv, senv)
			}
		})
	}
}

// equal tells whether a and b contain the same elements in the
// same order. A nil argument is equivalent to an empty slice.
func equal(t *testing.T, a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if c := strings.Compare(v, b[i]); c != 0 {
			return false
		}
	}
	return true
}
