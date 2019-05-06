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
	"io/ioutil"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"mvdan.cc/sh/v3/interp"
)

// expect-search builtin
// usage:
// expect-search output|error "TestName" "search_pattern" command <command_args>
func expectSearch(ctx context.Context, mc interp.ModuleCtx, args []string) error {
	if len(args) < 4 {
		return fmt.Errorf("test-exit requires at least 4 arguments")
	}

	var readPipe io.ReadCloser

	stream := args[0]
	search := args[2]

	switch stream {
	case "output", "error":
	default:
		return fmt.Errorf("stream %s not supported", stream)
	}

	path, err := exec.LookPath(args[3])
	if err != nil {
		return err
	}

	fullCmd := strings.Join(args[3:], " ")

	cmd := exec.Command(path, args[4:]...)
	cmd.Dir = mc.Dir
	cmd.Env = ExecEnv(mc.Env)
	cmd.Stdin = mc.Stdin

	if stream == "error" {
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return err
		}
		readPipe = stderr
	} else if stream == "output" {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}
		readPipe = stdout
		cmd.Stderr = mc.Stderr
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	d, _ := ioutil.ReadAll(readPipe)
	if !bytes.Contains(d, []byte(search)) {
		return fmt.Errorf("%s (%s stream doesn't contain %s string) %s", fullCmd, stream, search, string(d))
	}

	err = cmd.Wait()
	switch err.(type) {
	case *exec.Error:
		return err
	}

	return nil
}

// expect-exit builtin
// usage:
// expect-exit 0 "TestName" command <command_args>
func expectExit(ctx context.Context, mc interp.ModuleCtx, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("test-exit requires at least 3 arguments")
	}

	exitCode, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("failed to convert exit code %s: %s", args[0], err)
	}
	path, err := exec.LookPath(args[2])
	if err != nil {
		return err
	}

	fullCmd := strings.Join(args[2:], " ")

	cmd := exec.Command(path, args[3:]...)
	cmd.Dir = mc.Dir
	cmd.Env = ExecEnv(mc.Env)
	cmd.Stdin = mc.Stdin

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
	RegisterTestBuiltin("expect-search", expectSearch, 2)
	RegisterTestBuiltin("test-skip", testSkip, 1)
	RegisterTestBuiltin("test-skip-script", testSkipScript, -1)
	RegisterTestBuiltin("test-log", testLog, -1)
	RegisterTestBuiltin("test-error", testError, -1)
}
