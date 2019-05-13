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
	"os/exec"
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

// LookupCommand searches for command path based on current PATH sets
// in script.
func LookupCommand(command string, env expand.Environ) (string, error) {
	oldPath := os.Getenv("PATH")
	vr := env.Get("PATH")

	os.Setenv("PATH", vr.String())
	path, err := exec.LookPath(command)
	if err != nil {
		return "", nil
	}
	os.Setenv("PATH", oldPath)

	return path, nil
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
	f, err := os.Open(script)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	exec := func(ctx context.Context, path string, args []string) error {
		if tb, ok := testBuiltins[args[0]]; ok {
			te := ctx.Value(testExecContext).(*testExec)
			mc, _ := interp.FromModuleContext(ctx)
			if tb.Index >= 0 {
				if len(args) < tb.Index {
					te.t.Errorf("wrong usage of test builtin %s", args[0])
					return interp.ShellExitStatus(1)
				}
				failed := false
				te.t.Run(args[tb.Index], func(sub *testing.T) {
					var subTe testExec
					subTe.t = sub

					ctx := context.TODO()
					ctx = context.WithValue(ctx, testExecContext, &subTe)

					if err := tb.Fn(ctx, mc, args[1:]); err != nil {
						sub.Errorf("%sERROR: %-30s", removeFunctionLine(), err)
						failed = true
					}
				})
				if failed {
					return interp.ExitStatus(1)
				}
			} else {
				if err := tb.Fn(ctx, mc, args[1:]); err != nil {
					te.t.Errorf("%sERROR: %-30s", removeFunctionLine(), err)
					return interp.ExitStatus(1)
				}
			}
			return interp.ExitStatus(0)
		} else if cb, ok := commandBuiltins[args[0]]; ok {
			mc, _ := interp.FromModuleContext(ctx)
			return cb.Fn(ctx, mc, args[1:])
		} else if path == "" {
			return fmt.Errorf("%q test/command builtin doesn't exist", args[0])
		}
		return interp.DefaultExec(ctx, path, args)
	}
	runner, _ := interp.New(
		interp.StdIO(os.Stdin, os.Stdout, os.Stderr),
		interp.Module(interp.ModuleExec(exec)),
	)
	parser := syntax.NewParser()
	runner.Params = []string{script}

	t.Run(name, func(t *testing.T) {
		var te testExec
		var atExitFunctions []*AtExitFn

		te.atExitFunctions = &atExitFunctions
		te.t = t
		te.runner = runner

		ctx := context.TODO()
		ctx = context.WithValue(ctx, testExecContext, &te)

		defer func() {
			for i := len(atExitFunctions) - 1; i >= 0; i-- {
				if err := atExitFunctions[i].Fn(); err != nil {
					t.Logf("%sLOG: %s: %-30s", removeFunctionLine(), atExitFunctions[i].Desc, err)
				}
			}
		}()

		parser.Stmts(f, func(st *syntax.Stmt) bool {
			line := st.Cmd.Pos().Line()
			err := runner.Run(ctx, st)
			if err == nil {
				return true
			}

			switch err.(type) {
			case interp.ExitStatus:
				// continue execution
				return true
			default:
				if _, has := err.(interp.ShellExitStatus); has {
					// equivalent of t.Fatal
					if err != interp.ShellExitStatus(0) {
						t.Errorf("%sERROR: %s exited (at line %d): %-30s", removeFunctionLine(), script, line, err)
					}
				} else if err != nil {
					// trigger a test error and stop parsing
					t.Errorf("%sERROR: execution failed in %s (at line %d) with error: %-30s", removeFunctionLine(), script, line, err)
				}
			}
			return false
		})

		// if test-skip-script is called this function won't be executed
		if _, has := runner.Funcs["atexit"]; has {
			if err := runner.Run(ctx, runner.Funcs["atexit"].Cmd); err != nil {
				t.Errorf("%sERROR: function atexit returned an error: %s", removeFunctionLine(), err)
			}
		}
	})
}
