// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package exec

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/opencontainers/runtime-spec/specs-go"
)

// Hook execute an OCI hook command and pass state over stdin.
func Hook(ctx context.Context, hook *specs.Hook, state *specs.State) error {
	var cancel context.CancelFunc
	var timeout time.Duration
	var cmd *exec.Cmd

	if hook.Timeout != nil {
		timeout = time.Duration(*hook.Timeout) * 1000 * time.Millisecond
	}

	if timeout != 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	if ctx != nil {
		cmd = exec.CommandContext(ctx, hook.Path)
	} else {
		cmd = exec.Command(hook.Path)
	}

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state data: %s", err)
	}

	cmd.Stdin = bytes.NewReader(data)
	cmd.Env = hook.Env
	cmd.Args = hook.Args

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to execute hook %s: %s", hook.Path, err)
	}

	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("hook execution failed: %s", err)
	}

	if ctx != nil && ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("hook time out")
	}

	return err
}
