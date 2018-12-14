// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package env

import (
	"fmt"
	"strings"
	"testing"

	"github.com/opencontainers/runtime-tools/generate"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config/oci"
	"github.com/sylabs/singularity/internal/pkg/test"
)

func TestSetContainerEnv(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	ociConfig := &oci.Config{}
	generator := generate.Generator{Config: &ociConfig.Spec}

	type args struct {
		env       []string
		cleanEnv  bool
		homeDest  string
		resultEnv []string
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "NO_SINGULARITYENV_",
			args: args{[]string{"LD_LIBRARY_PATH=/.singularity.d/libs", "HOME=/home/tester",
				"PS1=test", "TERM=xterm-256color", "PATH=/usr/games:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
				"LANG=C", "SINGULARITY_CONTAINER=/tmp/lolcow.sif", "PWD=/tmp", "LC_ALL=C",
				"SINGULARITY_NAME=lolcow.sif"}, false, "/home/tester",
				[]string{"LD_LIBRARY_PATH=/.singularity.d/libs", "HOME=/home/tester", "PS1=test",
					"TERM=xterm-256color", "PATH=/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin",
					"LANG=C", "SINGULARITY_CONTAINER=/tmp/lolcow.sif", "PWD=/tmp", "LC_ALL=C",
					"SINGULARITY_NAME=lolcow.sif"},
			}},
		{name: "CLEANENV_true",
			args: args{[]string{"LD_LIBRARY_PATH=/.singularity.d/libs", "HOME=/home/tester",
				"PS1=test", "TERM=xterm-256color", "PATH=/usr/games:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
				"LANG=C", "SINGULARITY_CONTAINER=/tmp/lolcow.sif", "PWD=/tmp", "LC_ALL=C",
				"SINGULARITY_NAME=lolcow.sif", "SINGULARITYENV_FOO=VAR", "CLEANENV=TRUE"}, true, "/home/tester",
				[]string{"LD_LIBRARY_PATH=/.singularity.d/libs", "HOME=/home/tester", "PS1=test",
					"TERM=xterm-256color", "PATH=/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin",
					"LANG=C", "SINGULARITY_CONTAINER=/tmp/lolcow.sif", "PWD=/tmp", "LC_ALL=C",
					"SINGULARITY_NAME=lolcow.sif", "FOO=VAR"},
			}},
		{name: "alwaysPassKeys",
			args: args{[]string{"LD_LIBRARY_PATH=/.singularity.d/libs", "HOME=/home/tester",
				"PS1=test", "TERM=xterm-256color", "PATH=/usr/games:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
				"LANG=C", "SINGULARITY_CONTAINER=/tmp/lolcow.sif", "PWD=/tmp", "LC_ALL=C", "http_proxy=test_proxy", "no_proxy=noproxy",
				"ftp_proxy=ftpProxy", "SINGULARITY_NAME=lolcow.sif", "SINGULARITYENV_FOO=VAR", "CLEANENV=TRUE"}, true, "/home/tester",
				[]string{"LD_LIBRARY_PATH=/.singularity.d/libs", "HOME=/home/tester", "PS1=test", "TERM=xterm-256color", "PATH=/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin",
					"LANG=C", "SINGULARITY_CONTAINER=/tmp/lolcow.sif", "PWD=/tmp", "LC_ALL=C", "SINGULARITY_NAME=lolcow.sif", "FOO=VAR", "http_proxy=test_proxy", "no_proxy=noproxy", "ftp_proxy=ftpProxy"},
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetContainerEnv(&generator, tt.args.env, tt.args.cleanEnv, tt.args.homeDest)
			if !equal(ociConfig.Process.Env, tt.args.resultEnv) {
				fmt.Println(ociConfig.Process.Env)
				t.Fail()
			}
		})
	}
}

// equal tells whether a and b contain the same elements.
// A nil argument is equivalent to an empty slice.
func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		fmt.Println(v, b[i])
		if c := strings.Compare(v, b[i]); c != 0 {
			fmt.Println(v, b[i])
			return false
		}
	}
	return true
}
