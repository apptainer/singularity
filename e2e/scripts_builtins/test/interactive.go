// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package test

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	expect "github.com/Netflix/go-expect"
	"github.com/sylabs/singularity/pkg/stest"
	"mvdan.cc/sh/v3/interp"
)

// expect-exit-interactive builtin
// usage:
// expect-exit-interactive exit-code "TestName" <expect-script> command <command_args>
func expectExitInteractive(ctx context.Context, mc interp.ModuleCtx, args []string) error {
	if len(args) < 4 {
		return fmt.Errorf("expect-exit-interactive requires at least 3 arguments")
	}
	script, err := os.Open(args[2])
	if err != nil {
		return fmt.Errorf("failed to open script %s: %s", args[2], err)
	}

	c, err := expect.NewConsole(expect.WithDefaultTimeout(1 * time.Second))
	if err != nil {
		return err
	}
	defer c.Close()

	exitCode, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("failed to convert exit code %s: %s", args[0], err)
	}

	path, err := stest.LookupCommand(args[3], mc.Env)
	if err != nil {
		return err
	}

	fullCmd := strings.Join(args[3:], " ")

	cmd := exec.Cmd{
		Path:   path,
		Args:   args[3:],
		Env:    stest.ExecEnv(mc.Env),
		Dir:    mc.Env.Get("PWD").String(),
		Stdin:  c.Tty(),
		Stdout: c.Tty(),
		Stderr: c.Tty(),
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(script)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			cmd.Process.Signal(os.Kill)
			return fmt.Errorf("bad interactive script")
		}
		str := strings.Join(fields[1:], " ")
		switch fields[0] {
		case "expect":
			if fields[1] == "<EOF>" {
				c.Tty().Close()
				if _, err := c.ExpectEOF(); err != nil {
					cmd.Process.Signal(os.Kill)
					return fmt.Errorf("error while waiting EOF: %s", err)
				}
			} else {
				if _, err := c.ExpectString(str); err != nil {
					cmd.Process.Signal(os.Kill)
					return fmt.Errorf("error while trying to read %q: %s", str, err)
				}
			}
		case "send":
			if _, err := c.Send(str); err != nil {
				cmd.Process.Signal(os.Kill)
				return fmt.Errorf("error while sending %q: %s", str, err)
			}
		case "sendline":
			if _, err := c.SendLine(str); err != nil {
				cmd.Process.Signal(os.Kill)
				return fmt.Errorf("error while sending line %q: %s", str, err)
			}
		}
	}

	err = cmd.Wait()
	switch x := err.(type) {
	case *exec.ExitError:
		if status, ok := x.Sys().(syscall.WaitStatus); ok {
			if exitCode != status.ExitStatus() {
				return fmt.Errorf("%s: %s (expected exit code %d got %d)", args[1], fullCmd, exitCode, status.ExitStatus())
			}
		}
	}
	if exitCode == 0 && err != nil {
		return fmt.Errorf("unexpected error while running command %q interactively: %s", fullCmd, err)
	} else if exitCode != 0 && err == nil {
		return fmt.Errorf("unexpected success while running command %q interactively", fullCmd)
	}

	return nil
}

func init() {
	stest.RegisterTestBuiltin("expect-exit-interactive", expectExitInteractive, 2)
}
