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

	tt := []struct {
		name    string
		environ []string
		wantErr bool
	}{
		{
			name: "all ok",
			environ: []string{
				"LD_LIBRARY_PATH=/.singularity.d/libs",
				"HOME=/home/tester",
				"PS1=test",
				"TERM=xterm-256color",
				"PATH=/usr/games:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
				"LANG=C",
				"SINGULARITY_CONTAINER=/tmp/lolcow.sif",
				"PWD=/tmp",
				"LC_ALL=C",
				"SINGULARITY_NAME=lolcow.sif",
			},
			wantErr: false,
		},
		{
			name: "bad envs",
			environ: []string{
				"LD_LIBRARY_PATH=/.singularity.d/libs",
				"HOME=/home/tester",
				"PS1=test",
				"TERM=xterm-256color",
				"PATH=/usr/games:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
				"LANG=C",
				"SINGULARITY_CONTAINER=/tmp/lolcow.sif",
				"TEST",
				"LC_ALL=C",
				"SINGULARITY_NAME=lolcow.sif",
			},
			wantErr: true,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err := SetFromList(tc.environ)
			if tc.wantErr && err == nil {
				t.Fatalf("Expected error, but got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
