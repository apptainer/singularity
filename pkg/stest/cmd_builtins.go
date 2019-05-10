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
	tmpDir := ""
	if len(args) == 1 {
		tmpDir = args[0]
	}
	dir, err := ioutil.TempDir(tmpDir, "stestdir-")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(mc.Stdout, "%s\n", dir)
	cleanup := func() error {
		os.RemoveAll(dir)
		return nil
	}
	desc := fmt.Sprintf("delete directory %s", dir)
	RegisterAtExit(ctx, &AtExitFn{Fn: cleanup, Desc: desc})
	return err
}

// create-tmpfile builtin
// usage:
// create-tmpfile
func createTmpFile(ctx context.Context, mc interp.ModuleCtx, args []string) error {
	tmpDir := ""
	if len(args) == 1 {
		tmpDir = args[0]
	}
	f, err := ioutil.TempFile(tmpDir, "stestfile-")
	if err != nil {
		return err
	}
	file := f.Name()
	_, err = fmt.Fprintf(mc.Stdout, "%s\n", file)

	cleanup := func() error {
		os.Remove(file)
		return nil
	}
	desc := fmt.Sprintf("delete file %s", file)
	RegisterAtExit(ctx, &AtExitFn{Fn: cleanup, Desc: desc})
	f.Close()
	return err
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
	RegisterCommandBuiltin("which-os", whichOS)
}
