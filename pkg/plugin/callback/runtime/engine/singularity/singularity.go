// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the URIs of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"os"
	"syscall"

	"github.com/sylabs/singularity/pkg/runtime/engine/config"
)

// MonitorContainer callback allows to monitor container process.
// The plugin callback must implement the signal handler responsible
// of tracking container process status, it's also responsible to
// propagate signals to container process. If more than one plugin
// uses this callback the runtime aborts its execution.
// This callback is called in:
// - internal/pkg/runtime/engine/singularity/monitor_linux.go
type MonitorContainer func(config *config.Common, pid int, signals chan os.Signal) (syscall.WaitStatus, error)

// PostStartProcess callback is called after the container process
// started. It's a good place to add custom logger and/or notifier.
// This callback is called in:
// - internal/pkg/runtime/engine/singularity/process_linux.go
type PostStartProcess func(config *config.Common, pid int) error

// RegisterImageDriver callback is called before the container
// creation setup to register an image driver.
// This callback is called in:
// - internal/pkg/runtime/engine/singularity/container_linux.go
type RegisterImageDriver func(unprivileged bool) error
