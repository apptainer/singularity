// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Package stest provides a testing framework to run tests from script.
package stest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"mvdan.cc/sh/v3/interp"
)

// expect-search builtin
// usage:
// expect-search output|error|combined "search_pattern" "TestName" command <command_args>
func expectSearch(ctx context.Context, mc interp.ModuleCtx, args []string) error {
	const regexPrefix = "regex:"

	if len(args) < 4 {
		return fmt.Errorf("expect-search requires at least 4 arguments")
	}

	var re *regexp.Regexp
	var d bytes.Buffer

	stream := args[0]
	search := args[1]

	if strings.HasPrefix(search, regexPrefix) {
		var err error
		re, err = regexp.Compile(strings.TrimPrefix(search, regexPrefix))
		if err != nil {
			return fmt.Errorf("error while compiling search pattern: %s", err)
		}
	}

	path, err := LookupCommand(args[3], mc.Env)
	if err != nil {
		return err
	}

	fullCmd := strings.Join(args[3:], " ")

	cmd := exec.Cmd{
		Path:  path,
		Args:  args[3:],
		Env:   ExecEnv(mc.Env),
		Dir:   mc.Dir,
		Stdin: mc.Stdin,
	}

	switch stream {
	case "error":
		if mc.Stderr != os.Stderr {
			cmd.Stderr = io.MultiWriter(&d, mc.Stderr)
		} else {
			cmd.Stderr = &d
		}
		if mc.Stdout != os.Stdout {
			cmd.Stdout = mc.Stdout
		}
	case "output":
		if mc.Stdout != os.Stdout {
			cmd.Stdout = io.MultiWriter(&d, mc.Stdout)
		} else {
			cmd.Stdout = &d
		}
		if mc.Stderr != os.Stderr {
			cmd.Stderr = mc.Stderr
		}
	case "combined":
		if mc.Stderr != os.Stderr {
			cmd.Stderr = io.MultiWriter(&d, mc.Stderr)
		} else {
			cmd.Stderr = &d
		}
		if mc.Stdout != os.Stdout {
			cmd.Stdout = io.MultiWriter(&d, mc.Stdout)
		} else {
			cmd.Stdout = &d
		}
	default:
		return fmt.Errorf("stream %s not supported", stream)
	}

	exitCode := 0

	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.Error); ok {
			return fmt.Errorf("error while executing %s: %s", fullCmd, err)
		}
		if x, ok := err.(*exec.ExitError); ok {
			if status, ok := x.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			}
		}
	}

	match := false
	if re != nil {
		if len(re.Find(d.Bytes())) > 0 {
			match = true
		}
	} else {
		if strings.Contains(d.String(), search) {
			match = true
		}
	}
	if !match {
		return fmt.Errorf("%s (%s stream doesn't contain pattern %q string): %s", fullCmd, stream, search, d.String())
	}

	return interp.ExitStatus(exitCode)
}

// expect-exit builtin
// usage:
// expect-exit 0 "TestName" command <command_args>
func expectExit(ctx context.Context, mc interp.ModuleCtx, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("expect-exit requires at least 3 arguments")
	}

	exitCode, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("failed to convert exit code %s: %s", args[0], err)
	}
	path, err := LookupCommand(args[2], mc.Env)
	if err != nil {
		return err
	}

	fullCmd := strings.Join(args[2:], " ")

	cmd := exec.Cmd{
		Path:  path,
		Args:  args[2:],
		Env:   ExecEnv(mc.Env),
		Dir:   mc.Dir,
		Stdin: mc.Stdin,
	}
	if mc.Stderr != os.Stderr {
		cmd.Stderr = mc.Stderr
	}
	if mc.Stdout != os.Stdout {
		cmd.Stdout = mc.Stdout
	}

	err = cmd.Run()
	switch x := err.(type) {
	case *exec.ExitError:
		if status, ok := x.Sys().(syscall.WaitStatus); ok {
			if exitCode != status.ExitStatus() {
				return fmt.Errorf("%s: %s (expected exit code %d got %d)", args[1], fullCmd, exitCode, status.ExitStatus())
			}
		}
	}
	if exitCode == 0 && err != nil {
		return fmt.Errorf("unexpected error while running command %q: %s", fullCmd, err)
	} else if exitCode != 0 && err == nil {
		return fmt.Errorf("unexpected success while running command %q", fullCmd)
	}
	return nil
}

// test-log builtin
// usage:
// test-log "Something logged"
func testLog(ctx context.Context, mc interp.ModuleCtx, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("test-log requires a string argument")
	}
	t := GetTesting(ctx)
	t.Logf("%sLOG: %-30s", removeFunctionLine(), args[0])
	return nil
}

// test-skip builtin
// usage:
// test-skip "TestName" "Skip reason"
func testSkip(ctx context.Context, mc interp.ModuleCtx, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("test-skip requires 2 arguments")
	}
	t := GetTesting(ctx)
	t.Skip(fmt.Sprintf("%sSKIP: %-30s", removeFunctionLine(), args[1]))
	return nil
}

// test-skip-script builtin
// usage:
// test-skip-script "Skip reason"
func testSkipScript(ctx context.Context, mc interp.ModuleCtx, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("test-skip-script requires a string argument")
	}
	t := GetTesting(ctx)
	t.Skip(fmt.Sprintf("%sSKIP: %-30s", removeFunctionLine(), args[0]))
	return nil
}

// test-error builtin
// usage:
// test-error "Error message"
func testError(ctx context.Context, mc interp.ModuleCtx, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("test-error requires a string argument")
	}
	t := GetTesting(ctx)
	t.Errorf("%sERROR: %-30s", removeFunctionLine(), args[0])
	return nil
}

func init() {
	RegisterTestBuiltin("expect-exit", expectExit, 2)
	RegisterTestBuiltin("expect-search", expectSearch, 3)
	RegisterTestBuiltin("test-skip", testSkip, 1)
	RegisterTestBuiltin("test-skip-script", testSkipScript, -1)
	RegisterTestBuiltin("test-log", testLog, -1)
	RegisterTestBuiltin("test-error", testError, -1)
}
