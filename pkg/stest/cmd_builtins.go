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
	"regexp"

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
	dir, err := ioutil.TempDir(tmpDir, "d-")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(mc.Stdout, "%s\n", dir)
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
	f, err := ioutil.TempFile(tmpDir, "f-")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(mc.Stdout, "%s\n", f.Name())
	f.Close()
	return err
}

// register-exit-func
// usage:
// register-exit-func <function_name>
func registerExitFunction(ctx context.Context, mc interp.ModuleCtx, args []string) error {
	funcName := args[0]
	runner := ctx.Value(testExecContext).(*testExec).runner

	if _, has := runner.Funcs[funcName]; has {
		atExitFunctions := ctx.Value(testExecContext).(*testExec).atExitFunctions
		for _, f := range *atExitFunctions {
			if f == funcName {
				return nil
			}
		}
		*atExitFunctions = append(*atExitFunctions, funcName)
		return nil
	}

	return fmt.Errorf("%s is not a function", funcName)
}

// escape-meta-regex
// usage:
// escape-meta-regex <string> or echo "test"|escape-meta-regex
func escapeMetaRegex(ctx context.Context, mc interp.ModuleCtx, args []string) error {
	str := ""
	if len(args) == 1 {
		str = args[0]
	} else {
		b, err := ioutil.ReadAll(mc.Stdin)
		if err != nil {
			return fmt.Errorf("escape-meta-regex error: %s", err)
		}
		str = string(b)
	}
	_, err := fmt.Fprintf(mc.Stdout, "%s\n", regexp.QuoteMeta(str))
	return err
}

func init() {
	RegisterCommandBuiltin("create-tmpdir", createTmpDir)
	RegisterCommandBuiltin("create-tmpfile", createTmpFile)
	RegisterCommandBuiltin("escape-meta-regex", escapeMetaRegex)
	RegisterCommandBuiltin("register-exit-func", registerExitFunction)
}
