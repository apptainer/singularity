// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Package stest provides a testing framework to run tests from script.
package stest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// AtExitFn defines a function called when a test script terminates.
type AtExitFn struct {
	Fn   func() error
	Desc string
}

// BuiltinFn defines a shell builtin function.
type BuiltinFn func(context.Context, interp.ModuleCtx, []string) error

// keep track of test execution context
type testExec struct {
	atExitFunctions *[]*AtExitFn
	t               *testing.T
	runner          *interp.Runner
}

// CommandBuiltin defines a command shell builtin.
type CommandBuiltin struct {
	Name string
	Fn   BuiltinFn
}

// TestBuiltin defines a test shell builtin.
type TestBuiltin struct {
	Name  string
	Index int
	Fn    BuiltinFn
}

// testExecContext context value for test execution
var testExecContext struct{}

// registered test builtins
var testBuiltins = make(map[string]*TestBuiltin)

// registered command builtins
var commandBuiltins = make(map[string]*CommandBuiltin)

// ExecEnv iterates over current script environment variables and
// returns a list of key-pair environment variable, typically to
// use them with exec.Command.
func ExecEnv(env expand.Environ) []string {
	list := make([]string, 0, 64)
	env.Each(func(name string, vr expand.Variable) bool {
		if vr.Exported {
			list = append(list, name+"="+vr.String())
		}
		return true
	})
	return list
}

// RegisterTestBuiltin registers a test builtin, typically called
// from init().
func RegisterTestBuiltin(name string, fn BuiltinFn, index int) error {
	if _, has := testBuiltins[name]; has {
		return fmt.Errorf("test builtin %q already exists", name)
	}
	testBuiltins[name] = &TestBuiltin{
		Name:  name,
		Index: index,
		Fn:    fn,
	}
	return nil
}

// RegisterCommandBuiltin registers a command builtin, typically called
// from init().
func RegisterCommandBuiltin(name string, fn BuiltinFn) error {
	if _, has := commandBuiltins[name]; has {
		return fmt.Errorf("command builtin %q already exists", name)
	}
	commandBuiltins[name] = &CommandBuiltin{
		Name: name,
		Fn:   fn,
	}
	return nil
}

// GetCommandBuiltin returns the named command builtin.
func GetCommandBuiltin(name string) *CommandBuiltin {
	return commandBuiltins[name]
}

// GetTestBuiltin returns the named test builtin.
func GetTestBuiltin(name string) *TestBuiltin {
	return testBuiltins[name]
}

// RegisterAtExit registers a function to execute when the script execution
// finished.
func RegisterAtExit(ctx context.Context, fn *AtExitFn) {
	atExitFunctions := ctx.Value(testExecContext).(*testExec).atExitFunctions
	*atExitFunctions = append(*atExitFunctions, fn)
}

// GetTesting returns the current test execution context.
func GetTesting(ctx context.Context) *testing.T {
	return ctx.Value(testExecContext).(*testExec).t
}

// SetEnv sets an environment variables, equivalent to "export NAME=VALUE"
func SetEnv(ctx context.Context, name string, value string) {
	runner := ctx.Value(testExecContext).(*testExec).runner
	vr := expand.Variable{Kind: expand.String, Exported: true, Str: value}
	runner.Vars[name] = vr
}

// get rid of function/line displayed by testing package
func removeFunctionLine() string {
	_, fn, line, _ := runtime.Caller(1)
	name := filepath.Base(fn)
	l := strconv.Itoa(line)
	sz := len(name) + len(l) + 3

	var b strings.Builder
	b.Grow(sz)
	for i := 0; i < sz; i++ {
		b.WriteByte('\b')
	}
	return b.String()
}

// RunScript executes the provided script from a test function as a main
// sub test with the provided name.
func RunScript(name, script string, t *testing.T) {
	var te testExec
	var testExecContext struct{}
	var atExitFunctions []*AtExitFn

	scriptPath, err := filepath.Abs(script)
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(script)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	te.atExitFunctions = &atExitFunctions

	exec := func(ctx context.Context, path string, args []string) error {
		if tb, ok := testBuiltins[args[0]]; ok {
			scriptTest := GetTesting(ctx)
			mc, _ := interp.FromModuleContext(ctx)
			if tb.Index >= 0 {
				scriptTest.Run(args[tb.Index], func(sub *testing.T) {
					te.t = sub
					if err := tb.Fn(ctx, mc, args[1:]); err != nil {
						sub.Fatalf("%sERROR: %-30s", removeFunctionLine(), err)
					}
				})
				te.t = scriptTest
			} else {
				if err := tb.Fn(ctx, mc, args[1:]); err != nil {
					scriptTest.Errorf("%sERROR: %-30s", removeFunctionLine(), err)
				}
			}
			return nil
		} else if cb, ok := commandBuiltins[args[0]]; ok {
			mc, _ := interp.FromModuleContext(ctx)
			return cb.Fn(ctx, mc, args[1:])
		} else if path == "" {
			return fmt.Errorf("%q: executable file not found in $PATH", args[0])
		}
		return interp.DefaultExec(ctx, path, args)
	}
	te.runner, _ = interp.New(
		interp.StdIO(os.Stdin, os.Stdout, os.Stderr),
		interp.Module(interp.ModuleExec(exec)),
	)
	parser := syntax.NewParser()

	t.Run(name, func(t *testing.T) {
		ctx := context.TODO()
		ctx = context.WithValue(ctx, testExecContext, &te)
		te.t = t

		atExit := func() {
			for i := len(atExitFunctions) - 1; i >= 0; i-- {
				if err := atExitFunctions[i].Fn(); err != nil {
					te.t.Logf("%sLOG: %s: %-30s", removeFunctionLine(), atExitFunctions[i].Desc, err)
				}
			}
		}

		parser.Stmts(f, func(st *syntax.Stmt) bool {
			line := st.Cmd.Pos().Line()
			if err := te.runner.Run(ctx, st); err != nil {
				atExit()
				te.t.Fatalf("%s%s failed (at line %d) with error: %-30s", removeFunctionLine(), scriptPath, line, err)
				return false
			}
			if te.t.Failed() {
				atExit()
				te.t.Fatalf("%s%s (at line %d)", removeFunctionLine(), scriptPath, line)
				return false
			}
			return true
		})

		atExit()
	})
}
