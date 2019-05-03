// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Package stest provides a testing framework to run tests from script.
package stest

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"

	"mvdan.cc/sh/v3/interp"
)

// create-tmpdir builtin
// usage:
// create-tmpdir
func createTmpDir(ctx context.Context, mc interp.ModuleCtx, args []string) error {
	dir, err := ioutil.TempDir(os.Getenv("TESTDIR"), "stestdir-")
	if err != nil {
		return err
	}
	fmt.Fprintf(mc.Stdout, "%s\n", dir)

	cleanup := func() error {
		os.RemoveAll(dir)
		return nil
	}
	desc := fmt.Sprintf("delete directory %s", dir)
	RegisterAtExit(ctx, &AtExitFn{Fn: cleanup, Desc: desc})
	return nil
}

// create-tmpfile builtin
// usage:
// create-tmpfile
func createTmpFile(ctx context.Context, mc interp.ModuleCtx, args []string) error {
	f, err := ioutil.TempFile(os.Getenv("TESTDIR"), "stestfile-")
	if err != nil {
		return err
	}
	file := f.Name()
	fmt.Fprintf(mc.Stdout, "%s\n", file)

	cleanup := func() error {
		os.Remove(file)
		return nil
	}
	desc := fmt.Sprintf("delete file %s", file)
	RegisterAtExit(ctx, &AtExitFn{Fn: cleanup, Desc: desc})
	f.Close()
	return nil
}

// has-succeeded builtin
// usage:
// has-succeeded
func hasSucceeded(ctx context.Context, mc interp.ModuleCtx, args []string) error {
	t := GetTesting(ctx)
	if t.Failed() {
		return interp.ExitStatus(1)
	}
	return nil
}

// which-os builtin
// usage:
// which-os
func whichOS(ctx context.Context, mc interp.ModuleCtx, args []string) error {
	_, err := fmt.Fprintf(mc.Stdout, "%s\n", runtime.GOOS)
	return err
}

func init() {
	RegisterCommandBuiltin("create-tmpdir", createTmpDir)
	RegisterCommandBuiltin("create-tmpfile", createTmpFile)
	RegisterCommandBuiltin("has-succeeded", hasSucceeded)
	RegisterCommandBuiltin("which-os", whichOS)
}
