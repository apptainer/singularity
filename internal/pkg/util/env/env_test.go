// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package env

import (
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

func TestSetFromList(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	type args struct {
		environ []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "Good_envs", args: args{[]string{"LD_LIBRARY_PATH=/.singularity.d/libs", "HOME=/home/tester",
			"PS1=test", "TERM=xterm-256color", "PATH=/usr/games:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			"LANG=C", "SINGULARITY_CONTAINER=/tmp/lolcow.sif", "PWD=/tmp", "LC_ALL=C",
			"SINGULARITY_NAME=lolcow.sif"}}, wantErr: false},
		{name: "Bad_envs", args: args{[]string{"LD_LIBRARY_PATH=/.singularity.d/libs", "HOME=/home/tester",
			"PS1=test", "TERM=xterm-256color", "PATH=/usr/games:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			"LANG=C", "SINGULARITY_CONTAINER=/tmp/lolcow.sif", "TEST", "LC_ALL=C",
			"SINGULARITY_NAME=lolcow.sif"}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := SetFromList(tt.args.environ); (err != nil) != tt.wantErr {
				t.Errorf("SetFromList() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
