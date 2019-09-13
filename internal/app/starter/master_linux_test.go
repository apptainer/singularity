// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package starter

import (
	"testing"

	"github.com/sylabs/singularity/internal/pkg/runtime/engine"
	"github.com/sylabs/singularity/internal/pkg/test"
)

// TODO: actually we can't really test Master function which is
// part of the main function, as it exits, it would require mock at
// some point and that would make code more complex than necessary.
// createContainer and startContainer are quickly tested and only
// cover case with bad socket file descriptors or non socket file
// file descriptor (stderr).

func TestCreateContainer(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	var fatal error
	fatalChan := make(chan error, 1)

	tests := []struct {
		name         string
		rpcSocket    int
		containerPid int
		engine       *engine.Engine
		shallPass    bool
	}{
		{
			name:         "nil engine; bad rpcSocket",
			rpcSocket:    -1,
			containerPid: -1,
			engine:       nil,
			shallPass:    false,
		},
		{
			name:         "nil engine; wrong socket",
			rpcSocket:    2,
			containerPid: -1,
			engine:       nil,
			shallPass:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			go createContainer(tt.rpcSocket, tt.containerPid, tt.engine, fatalChan)
			// createContainer is creating a separate thread and we sync with that
			// thread through a channel similarly to the createContainer function itself,
			// as well as the Master function.
			// For more details, please refer to the master_linux.go code.
			fatal = <-fatalChan
			if tt.shallPass && fatal != nil {
				t.Fatalf("test %s expected to succeed but failed: %s", tt.name, fatal)
			} else if !tt.shallPass && fatal == nil {
				t.Fatalf("test %s expected to fail but succeeded", tt.name)
			} else if tt.shallPass && fatal == nil {
				// test succeed
			} else if !tt.shallPass && fatal != nil {
				// test succeed
			}
		})
	}
}

func TestStartContainer(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	var fatal error
	fatalChan := make(chan error, 1)

	tests := []struct {
		name         string
		masterSocket int
		containerPid int
		engine       *engine.Engine
		shallPass    bool
	}{
		{
			name:         "nil engine; bad masterSocket",
			masterSocket: -1,
			containerPid: -1,
			engine:       nil,
			shallPass:    false,
		},
		{
			name:         "nil engine; wrong socket",
			masterSocket: 2,
			containerPid: -1,
			engine:       nil,
			shallPass:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			go startContainer(tt.masterSocket, tt.containerPid, tt.engine, fatalChan)
			fatal = <-fatalChan
			if tt.shallPass && fatal != nil {
				t.Fatalf("test %s expected to succeed but failed: %s", tt.name, fatal)
			} else if !tt.shallPass && fatal == nil {
				t.Fatalf("test %s expected to fail but succeeded", tt.name)
			} else if tt.shallPass && fatal == nil {
				// test succeed
			} else if !tt.shallPass && fatal != nil {
				// test succeed
			}
		})
	}
}
