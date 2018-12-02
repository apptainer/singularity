// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package ociruntime

import (
	"github.com/opencontainers/runtime-spec/specs-go"
)

const (
	// AnnotationCreatedAt is used to pass creation timestamp in annotations.
	AnnotationCreatedAt = "io.sylabs.runtime.oci.created_at"
	// AnnotationStartedAt is used to pass startup timestamp in annotations.
	AnnotationStartedAt = "io.sylabs.runtime.oci.starter_at"
	// AnnotationFinishedAt is used to pass finished timestamp in annotations.
	AnnotationFinishedAt = "io.sylabs.runtime.oci.finished_at"
	// AnnotationExitCode is used to pass exit code in annotations.
	AnnotationExitCode = "io.sylabs.runtime.oci.exit-code"
	// AnnotationExitDesc is used to pass exit descrition (e.g. reson) in annotations.
	AnnotationExitDesc = "io.sylabs.runtime.oci.exit-desc"
	// AnnotationAttachSocket is used to pass attach socket path in annotations.
	AnnotationAttachSocket = "io.sylabs.runtime.oci.attach-socket"
	// AnnotationControlSocket is used to pass control socket path in annotations.
	AnnotationControlSocket = "io.sylabs.runtime.oci.control-socket"
)

// Control is used to pass information for container control
// like terminal resize or log file reopen
type Control struct {
	ConsoleSize    *specs.Box `json:"consoleSize,omitempty"`
	ReopenLog      bool       `json:"reopenLog,omitempty"`
	StartContainer bool       `json:"startContainer,omitempty"`
}
