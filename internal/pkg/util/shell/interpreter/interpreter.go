// Copyright (c) 2020, Sylabs, Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license.  Please
// consult LICENSE.md file distributed with the sources of this project regarding
// your rights to use or distribute this software.

package interpreter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// ShellBuiltin defines function prototype for shell interpreter builtin registration.
type ShellBuiltin func(ctx context.Context, args []string) error

// OpenHandler defines function prototype for shell interpreter file handler registration.
type OpenHandler func(path string, flag int, perm os.FileMode) (io.ReadWriteCloser, error)

// Shell defines the shell interpreter.
type Shell struct {
	shellBuiltins map[string]ShellBuiltin
	openHandlers  map[string]OpenHandler
	name          string
	status        uint8
	reader        io.Reader
	runner        *interp.Runner
}

// execTimeout defines the execution timeout for commands executed by the
// shell interpreter (default: 1 minute).
var execTimeout = time.Minute

// defaultExecHandler is the default command execution handler if there is
// no registered shell builtin.
func defaultExecHandler(ctx context.Context, args []string) error {
	hc := interp.HandlerCtx(ctx)
	path, err := interp.LookPath(hc.Env, args[0])
	if err != nil {
		fmt.Fprintln(hc.Stderr, err)
		return interp.NewExitStatus(127)
	}

	ectx, cancel := context.WithTimeout(ctx, execTimeout)
	defer cancel()

	cmd := exec.CommandContext(ectx, path, args[1:]...)
	cmd.Env = append([]string{"PWD=" + hc.Dir}, GetEnv(hc)...)
	cmd.Dir = hc.Dir
	cmd.Stdin = hc.Stdin
	cmd.Stdout = hc.Stdout
	cmd.Stderr = hc.Stderr

	err = cmd.Run()

	switch x := err.(type) {
	case *exec.ExitError:
		if status, ok := x.Sys().(syscall.WaitStatus); ok {
			if status.Signaled() && ectx.Err() != nil {
				c := strings.Join(args, " ")
				return fmt.Errorf("command %q was killed after %s timeout", c, execTimeout)
			}
			return interp.NewExitStatus(uint8(status.ExitStatus()))
		}
		return interp.NewExitStatus(1)
	case *exec.Error:
		c := strings.Join(args, " ")
		return fmt.Errorf("command %q execution failed: %s", c, err)
	}

	return err
}

// GetEnv returns an the list of all exported environment variables within
// the context of the shell interpreter.
func GetEnv(hc interp.HandlerContext) []string {
	envMap := make(map[string]expand.Variable)

	// use of a map to remove duplicated variable, the latest wins
	hc.Env.Each(func(name string, vr expand.Variable) bool {
		envMap[name] = vr
		return true
	})

	environ := make([]string, 0, len(envMap))
	for k, v := range envMap {
		if v.Exported && v.Kind == expand.String {
			environ = append(environ, k+"="+v.Str)
		}
	}

	sort.Strings(environ)

	return environ
}

// New returns a shell interpreter instance.
func New(r io.Reader, name string, args []string, envs []string, runnerOptions ...interp.RunnerOption) (s *Shell, err error) {
	if r == nil {
		return nil, fmt.Errorf("nil reader")
	}

	s = &Shell{
		reader: r,
		name:   name,
	}

	opts := []interp.RunnerOption{
		interp.StdIO(os.Stdin, os.Stdout, os.Stderr),
		interp.ExecHandler(s.internalExecHandler()),
		interp.OpenHandler(s.internalOpenHandler()),
		interp.Params("--"),
		interp.Env(expand.ListEnviron(envs...)),
	}
	opts = append(opts, runnerOptions...)
	s.runner, err = interp.New(opts...)

	if err != nil {
		return nil, fmt.Errorf("while creating shell interpreter: %s", err)
	}

	s.runner.Dir, err = os.Getwd()
	if err != nil {
		s.runner.Dir = "/"
	}

	s.runner.Params = append(s.runner.Params, args...)

	return s, err
}

// internalExecHandler returns an ExecHandlerFunc used by default.
func (s *Shell) internalExecHandler() interp.ExecHandlerFunc {
	return func(ctx context.Context, args []string) error {
		if s.runner.Exited() {
			// special path for exec builtin keyword
			if builtin, ok := s.shellBuiltins["exec"]; ok {
				return builtin(ctx, args)
			}
		} else if builtin, ok := s.shellBuiltins[args[0]]; ok {
			return builtin(ctx, args[1:])
		} else {
			// declaration clause are normally handled by the interpreter
			// but when a builtin prefixed with a backslash is encountered
			// by example, the parser consider it as a call expression and
			// we get there, so basically what we do is to create a new parser
			// and evaluate it in the current shell interpreter
			switch args[0] {
			case "export", "local", "declare", "nameref", "readonly", "typeset":
				var b bytes.Buffer

				b.WriteString(strings.Join(args, " "))
				node, err := syntax.NewParser().Parse(&b, s.name)
				if err != nil {
					return err
				}

				// We run individual syntax.Stmt rather than the parsed syntax.File as the latter
				// implies an `exit`, and causes https://github.com/sylabs/singularity/issues/274
				// with the exit/trap changes in https://github.com/mvdan/sh/commit/fb5052e7a0109c9ef5553a310c05f3b8c04cca5f
				for _, stmt := range node.Stmts {
					if err := s.runner.Run(ctx, stmt); err != nil {
						return err
					}
				}
				return nil
			}
		}
		return defaultExecHandler(ctx, args)
	}
}

// internalOpenHandler returns an OpenHandlerFunc used by default.
func (s *Shell) internalOpenHandler() interp.OpenHandlerFunc {
	return func(ctx context.Context, path string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
		mc := interp.HandlerCtx(ctx)
		if !filepath.IsAbs(path) {
			path = filepath.Join(mc.Dir, path)
		}
		if handler, ok := s.openHandlers[path]; ok {
			return handler(path, flag, perm)
		}
		return os.OpenFile(path, flag, perm)
	}
}

// RegisterShellBuiltin registers a shell interpreter builtin.
func (s *Shell) RegisterShellBuiltin(name string, builtin ShellBuiltin) {
	if s.shellBuiltins == nil {
		s.shellBuiltins = make(map[string]ShellBuiltin)
	}
	s.shellBuiltins[name] = builtin
}

// RegisterOpenHandler registers a shell interpreter file handler.
func (s *Shell) RegisterOpenHandler(path string, handler OpenHandler) {
	if s.openHandlers == nil {
		s.openHandlers = make(map[string]OpenHandler)
	}
	s.openHandlers[path] = handler
}

// LookPath returns the absolute path for the command passed in argument
// within the context of the shell interpreter.
func (s *Shell) LookPath(ctx context.Context, cmd string) (string, error) {
	hc := interp.HandlerCtx(ctx)
	return interp.LookPath(hc.Env, cmd)
}

// Run runs the shell interpreter.
func (s *Shell) Run() error {
	ctx := context.TODO()

	parser := syntax.NewParser()
	node, err := parser.Parse(s.reader, s.name)
	if err != nil {
		return fmt.Errorf("while parsing script: %s", err)
	}

	if err := s.runner.Run(ctx, node); err != nil {
		if status, ok := interp.IsExitStatus(err); ok {
			s.status = status
		}
		return err
	}

	return nil
}

// Status returns the exit code status of the shell interpreter.
func (s *Shell) Status() uint8 {
	return s.status
}

// nonExportedEnv allows to initialize shell interpreter with all environment
// variables set as non exported variables.
type nonExportedEnv struct {
	envs map[string]expand.Variable
}

// newNonExportedEnv returns a localEnv instance associated to environment
// passed in argument.
func newNonExportedEnv(env []string) nonExportedEnv {
	local := nonExportedEnv{
		envs: make(map[string]expand.Variable),
	}
	for _, e := range env {
		e := strings.SplitN(e, "=", 2)
		local.envs[e[0]] = expand.Variable{Str: e[1], Kind: expand.String}
	}
	return local
}

// Get returns the named shell environment variable.
func (e nonExportedEnv) Get(name string) expand.Variable {
	if vr, ok := e.envs[name]; ok {
		return vr
	}
	return expand.Variable{}
}

// Each iterates over all environment variables by calling the
// function passed in argument for each variables.
func (e nonExportedEnv) Each(fn func(name string, vr expand.Variable) bool) {
	for name, vr := range e.envs {
		if !fn(name, vr) {
			return
		}
	}
}

// EvaluateEnv evaluates the environment variable script and returns
// the list of variables set in the script. Command execution is disabled
// along with redirection.
func EvaluateEnv(script []byte, args []string, envs []string) ([]string, error) {
	const stopBuiltin = "__stop__"

	var env []string

	// disable command execution and just handle stop builtin
	execHandler := func(ctx context.Context, args []string) error {
		if args[0] == stopBuiltin {
			env = GetEnv(interp.HandlerCtx(ctx))
			return nil
		}
		c := strings.Join(args, " ")
		return fmt.Errorf("could not execute %q: execution is disabled", c)
	}
	openHandler := func(ctx context.Context, path string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
		return nil, fmt.Errorf("could not open/create/modify %q: file feature is disabled", path)
	}

	opts := []interp.RunnerOption{
		interp.ExecHandler(execHandler),
		interp.OpenHandler(openHandler),
		interp.Env(newNonExportedEnv(envs)),
	}

	b := bytes.NewBuffer(script)
	// append stop builtin to the end of the script
	b.WriteString("\n" + stopBuiltin + "\n")

	shell, err := New(b, "singularity", args, nil, opts...)
	if err != nil {
		return nil, fmt.Errorf("while initializing shell interpreter: %s", err)
	}
	// set allexport option
	interp.Params("-a")(shell.runner)

	if err := shell.Run(); err != nil {
		return nil, fmt.Errorf("while evaluating environment script: %s", err)
	}

	return env, nil
}
