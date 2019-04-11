// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package ociruntime

import (
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

const (
	// Creating represent creating status during container lifecycle
	Creating = "creating"
	// Created represent created status during container lifecycle
	Created = "created"
	// Running represent running status during container lifecycle
	Running = "running"
	// Stopped represent stopped status during container lifecycle
	Stopped = "stopped"
	// Paused represent paused status during container lifecycle
	Paused = "paused"
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
	Pause          bool       `json:"pause,omitempty"`
	Resume         bool       `json:"resume,omitempty"`
}
