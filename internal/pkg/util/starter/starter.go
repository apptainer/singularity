// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package starter

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/runtime/engine/config"
	"golang.org/x/sys/unix"
)

// CommandOp represents a function type passed to Exec/Run allowing
// to customize the starter command execution.
type CommandOp func(*Command)

// WithStdout allows to pass a custom output stream to starter
// command. Output stream is ignored for Exec as it uses the
// caller stream.
func WithStdout(stdout io.Writer) CommandOp {
	return func(c *Command) {
		c.stdout = stdout
	}
}

// WithStderr allows to pass a custom error stream to starter
// command. Error stream is ignored for Exec as it uses the
// caller stream.
func WithStderr(stderr io.Writer) CommandOp {
	return func(c *Command) {
		c.stderr = stderr
	}
}

// WithStdin allows to pass a custom input stream to starter
// command. Input stream is ignored for Exec as it uses the
// caller stream.
func WithStdin(stdin io.Reader) CommandOp {
	return func(c *Command) {
		c.stdin = stdin
	}
}

// UseSuid sets if the starter command uses either the setuid
// binary or the unprivileged binary. The unprivileged binary
// is used by default if this operation is not passed to Run/Exec.
func UseSuid(suid bool) CommandOp {
	return func(c *Command) {
		if suid {
			c.path = filepath.Join(buildcfg.LIBEXECDIR, "singularity/bin/starter-suid")
			return
		}
		c.path = filepath.Join(buildcfg.LIBEXECDIR, "singularity/bin/starter")
	}
}

// LoadOverlayModule sets LOAD_OVERLAY_MODULE environment variable
// which tell starter to load overlay kernel module.
func LoadOverlayModule(load bool) CommandOp {
	return func(c *Command) {
		if load {
			c.env = append(c.env, "LOAD_OVERLAY_MODULE=1")
		}
	}
}

// Command a starter command to execute.
type Command struct {
	path   string
	env    []string
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

// Exec executes the starter binary in place of the caller if
// there is no error. This function never returns on success.
func Exec(name string, config *config.Common, ops ...CommandOp) error {
	c := new(Command)
	if err := c.init(config, ops...); err != nil {
		return fmt.Errorf("while initializing starter command: %s", err)
	}
	err := unix.Exec(c.path, []string{name}, c.env)
	return fmt.Errorf("while executing %s: %s", c.path, err)
}

// Run executes the starter binary and returns once starter
// finished its execution.
func Run(name string, config *config.Common, ops ...CommandOp) error {
	c := new(Command)
	if err := c.init(config, ops...); err != nil {
		return fmt.Errorf("while initializing starter command: %s", err)
	}

	cmd := exec.Command(c.path)
	cmd.Args = []string{name}
	cmd.Env = c.env
	cmd.Stdin = c.stdin
	cmd.Stdout = c.stdout
	cmd.Stderr = c.stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("while running %s: %s", c.path, err)
	}
	return nil
}

func (c *Command) init(config *config.Common, ops ...CommandOp) error {
	c.path = filepath.Join(buildcfg.LIBEXECDIR, "singularity/bin/starter")

	for _, op := range ops {
		op(c)
	}

	sylog.Debugf("Use starter binary %s", c.path)

	if _, err := os.Stat(c.path); os.IsNotExist(err) {
		return fmt.Errorf("%s not found, please check your installation", c.path)
	}

	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("while marshaling config: %s", err)
	}

	pipeFd, err := sendData(data)
	if err != nil {
		return fmt.Errorf("while sending configuration data: %s", err)
	}

	env := []string{sylog.GetEnvVar(), fmt.Sprintf("PIPE_EXEC_FD=%d", pipeFd)}
	c.env = append(c.env, env...)

	return nil
}
