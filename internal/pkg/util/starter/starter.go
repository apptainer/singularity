// Copyright (c) 2019-2020, Sylabs Inc. All rights reserved.
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

	"github.com/hpcng/singularity/internal/pkg/buildcfg"
	"github.com/hpcng/singularity/pkg/runtime/engine/config"
	"github.com/hpcng/singularity/pkg/sylog"
	"github.com/hpcng/singularity/pkg/util/rlimit"
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

// copyConfigToEnv checks that the current stack size is big enough
// to pass runtime configuration through environment variables.
// On linux RLIMIT_STACK determines the amount of space used for the
// process's command-line arguments and environment variables.
func copyConfigToEnv(data []byte) ([]string, error) {
	var configEnv []string

	const (
		// space size for singularity argument and environment variables
		// this is voluntary bigger than the real usage
		singularityArgSize = 4096

		// for kilobyte conversion
		kbyte = 1024

		// DO NOT MODIFY those format strings
		envConfigFormat      = buildcfg.ENGINE_CONFIG_ENV + "%d=%s"
		envConfigCountFormat = buildcfg.ENGINE_CONFIG_CHUNK_ENV + "=%d"
	)

	// get the current stack limit in kilobytes
	cur, max, err := rlimit.Get("RLIMIT_STACK")
	if err != nil {
		return nil, fmt.Errorf("failed to determine stack size: %s", err)
	}

	// stack size divided by four to determine the arguments+environments
	// size limit
	argSizeLimit := (cur / 4)

	// config length to be passed via environment variables + some space
	// for singularity first argument
	configLength := uint64(len(data)) + singularityArgSize

	// be sure everything fit with the current argument size limit
	if configLength <= argSizeLimit {
		i := 1
		offset := uint64(0)
		length := uint64(len(data))
		for i <= buildcfg.MAX_ENGINE_CONFIG_CHUNK {
			end := offset + buildcfg.MAX_CHUNK_SIZE
			if end > length {
				end = length
			}
			configEnv = append(configEnv, fmt.Sprintf(envConfigFormat, i, string(data[offset:end])))
			if end == length {
				break
			}
			offset = end
			i++
		}
		if i > buildcfg.MAX_ENGINE_CONFIG_CHUNK {
			return nil, fmt.Errorf("engine configuration too big > %d", buildcfg.MAX_ENGINE_CONFIG_SIZE)
		}
		configEnv = append(configEnv, fmt.Sprintf(envConfigCountFormat, i))
		return configEnv, nil
	}

	roundLimitKB := 4 * ((configLength / kbyte) + 1)
	hardLimitKB := max / kbyte
	// the hard limit is reached, maybe user screw up himself by
	// setting the hard limit with ulimit or this is a limit set
	// by administrator, in this case returns some hints
	if roundLimitKB > hardLimitKB {
		hint := "check if you didn't set the stack size hard limit with ulimit or ask to your administrator"
		return nil, fmt.Errorf("argument size hard limit reached (%d kbytes), could not pass configuration: %s", hardLimitKB, hint)
	}

	hint := fmt.Sprintf("use 'ulimit -S -s %d' and run it again", roundLimitKB)

	return nil, fmt.Errorf(
		"argument size limit is too low (%d bytes) to pass configuration (%d bytes): %s",
		argSizeLimit, configLength, hint,
	)
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

	envConfig, err := copyConfigToEnv(data)
	if err != nil {
		return fmt.Errorf("while copying engine configuration: %s", err)
	}

	c.env = append(c.env, sylog.GetEnvVar())
	c.env = append(c.env, envConfig...)

	return nil
}
