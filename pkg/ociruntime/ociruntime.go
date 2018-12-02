// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package ociruntime

import (
	"github.com/opencontainers/runtime-spec/specs-go"
)

// State represents the state of the container
type State struct {
	specs.State
	CreatedAt     *int64 `json:"createdAt,omitempty"`
	StartedAt     *int64 `json:"startedAt,omitempty"`
	FinishedAt    *int64 `json:"finishedAt,omitempty"`
	ExitCode      *int   `json:"exitCode,omitempty"`
	ExitDesc      string `json:"exitDesc,omitempty"`
	AttachSocket  string `json:"attachSocket,omitempty"`
	ControlSocket string `json:"controlSocket,omitempty"`
}

// Control is used to pass information for container control
// like terminal resize or log file reopen
type Control struct {
	ConsoleSize    *specs.Box `json:"consoleSize,omitempty"`
	ReopenLog      bool       `json:"reopenLog,omitempty"`
	StartContainer bool       `json:"startContainer,omitempty"`
}
