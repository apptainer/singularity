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
	"reflect"
	"sort"
	"strings"
	"testing"

	"mvdan.cc/sh/v3/interp"
)

func newIOStream() (io.Reader, io.ReadWriter, io.ReadWriter) {
	var stdin bytes.Buffer
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	return &stdin, &stdout, &stderr
}

type bufferCloser struct {
	bytes.Buffer
}

func (*bufferCloser) Close() error {
	return nil
}

type testShellBuiltin struct {
	builtin string
	fn      func(*Shell) ShellBuiltin
}

type testOpenHandler struct {
	path string
	fn   func(*Shell) OpenHandler
}

func TestInterpreter(t *testing.T) {
	var err error
	var shell *Shell

	_, err = New(nil, "empty_reader", nil, nil, interp.StdIO(newIOStream()))
	if err == nil {
		t.Fatalf("unexpected error with empty reader: %s", err)
	}

	tests := []struct {
		name         string
		argv         []string
		env          []string
		script       string
		expectOut    string
		expectErr    string
		shellBuiltin *testShellBuiltin
		openHandler  *testOpenHandler
		expectExit   uint8
	}{
		{
			name:      "hello stdout",
			script:    "echo 'hello'",
			expectOut: "hello",
		},
		{
			name:      "hello stderr",
			script:    "echo 'hello' 1>&2",
			expectErr: "hello",
		},
		{
			name:   "exit 0",
			script: "exit 0",
		},
		{
			name:       "exit 1",
			script:     "exit 1",
			expectExit: 1,
		},
		{
			name:      "echo arg one",
			script:    "echo ${1}",
			argv:      []string{"arg1"},
			expectOut: "arg1",
		},
		{
			name:      "echo env",
			script:    "echo $PATH",
			env:       []string{"PATH=/bin"},
			expectOut: "/bin",
		},
		{
			name:   "spawn true",
			script: "/bin/true",
		},
		{
			name:       "spawn false",
			script:     "/bin/false",
			expectExit: 1,
		},
		{
			name:      "exec builtin",
			script:    "exec true && echo 'hello'",
			env:       []string{"PATH=/bin"},
			expectOut: "/bin/true",
			shellBuiltin: &testShellBuiltin{
				builtin: "exec",
				fn: func(shell *Shell) ShellBuiltin {
					return func(ctx context.Context, args []string) error {
						hc := interp.HandlerCtx(ctx)
						cmd, err := shell.LookPath(ctx, args[0])
						if err != nil {
							return err
						}
						fmt.Fprintf(hc.Stdout, "%s\n", cmd)
						return nil
					}
				},
			},
		},
		{
			name:      "testing builtin",
			script:    "testing argument",
			expectOut: "argument",
			shellBuiltin: &testShellBuiltin{
				builtin: "testing",
				fn: func(shell *Shell) ShellBuiltin {
					return func(ctx context.Context, args []string) error {
						hc := interp.HandlerCtx(ctx)
						fmt.Fprintf(hc.Stdout, "%s\n", args[0])
						return nil
					}
				},
			},
		},
		{
			name:   "export env",
			script: "export FOO=bar && exec /bin/true",
			shellBuiltin: &testShellBuiltin{
				builtin: "exec",
				fn: func(shell *Shell) ShellBuiltin {
					return func(ctx context.Context, args []string) error {
						hc := interp.HandlerCtx(ctx)
						for _, env := range GetEnv(hc) {
							if env == "FOO=bar" {
								return nil
							}
						}
						return fmt.Errorf("no FOO environment variable")
					}
				},
			},
		},
		{
			name:      "source handler",
			script:    ". /virtual/file",
			expectOut: "a virtual file",
			openHandler: &testOpenHandler{
				path: "/virtual/file",
				fn: func(shell *Shell) OpenHandler {
					return func(path string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
						bc := new(bufferCloser)
						bc.WriteString("echo 'a virtual file'\n")
						return bc, nil
					}
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var scriptBuf bytes.Buffer
			var stdin bytes.Buffer
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			scriptBuf.WriteString(tt.script)

			shell, err = New(&scriptBuf, "buffer", tt.argv, tt.env, interp.StdIO(&stdin, &stdout, &stderr))
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if tt.shellBuiltin != nil {
				shell.RegisterShellBuiltin(tt.shellBuiltin.builtin, tt.shellBuiltin.fn(shell))
			}
			if tt.openHandler != nil {
				shell.RegisterOpenHandler(tt.openHandler.path, tt.openHandler.fn(shell))
			}
			if err := shell.Run(); err != nil {
				if tt.expectExit != 0 && shell.Status() != tt.expectExit {
					t.Fatalf("unexpected exit status for %s: got %d instead of %d", tt.name, shell.Status(), tt.expectExit)
				} else if tt.expectExit == 0 {
					t.Fatalf("unexpected error: %s", err)
				}
			}

			errStr := strings.TrimSpace(stderr.String())
			outStr := strings.TrimSpace(stdout.String())

			if tt.expectErr != "" && errStr != tt.expectErr {
				t.Fatalf("unexpected stderr: %s instead of %s", errStr, tt.expectErr)
			}
			if tt.expectOut != "" && outStr != tt.expectOut {
				t.Fatalf("unexpected stdout: %s instead of %s", outStr, tt.expectOut)
			}
		})
	}
}

func TestEvaluateEnv(t *testing.T) {
	tests := []struct {
		name      string
		script    string
		argv      []string
		env       []string
		resultEnv []string
		expectErr bool
	}{
		{
			name:      "EmptyScript",
			script:    "",
			resultEnv: []string{},
		},
		{
			name:      "SingleEnv",
			script:    "FOO=bar",
			resultEnv: []string{"FOO=bar"},
		},
		{
			name:      "ExternalEnv",
			env:       []string{"FOO=bar"},
			script:    "BAR=$FOO",
			resultEnv: []string{"BAR=bar"},
		},
		{
			name:      "ExternalEnvExport",
			env:       []string{"FOO=bar"},
			script:    "BAR=foo && export FOO",
			resultEnv: []string{"FOO=bar", "BAR=foo"},
		},
		{
			name:      "ExternalEnvOverwrite",
			env:       []string{"FOO=bar"},
			script:    "FOO=overwrite",
			resultEnv: []string{"FOO=overwrite"},
		},
		{
			name:      "ExecFailure",
			script:    "/bin/true",
			resultEnv: []string{},
			expectErr: true,
		},
		{
			name:      "OpenFailure",
			script:    "echo hello > /tmp/hello",
			resultEnv: []string{},
			expectErr: true,
		},
		{
			name:      "NonExistentVar",
			script:    "FOO=$FAKE",
			resultEnv: []string{"FOO="},
		},
		{
			name:      "ArgZeroVar",
			script:    "FOO=$0",
			resultEnv: []string{"FOO=singularity"},
		},
		{
			name:      "ArgOneVar",
			argv:      []string{"bar"},
			script:    "FOO=$1",
			resultEnv: []string{"FOO=bar"},
		},
		{
			name:      "AllArgVar",
			argv:      []string{"bar", "-a", "foo"},
			script:    "FOO=\"$@\"",
			resultEnv: []string{"FOO=bar -a foo"},
		},
	}

	// Since mvdan.cc/sh/v3@v3.4.0 some default vars will be set:
	//    HOME IFS OPTIND PWD UID GID
	// These don't adversely impact our downstream container environment, but
	// must be accounted for here.
	// https://github.com/mvdan/sh/commit/f4c774aa15046ef006508e182fde10c4b56876fa
	// https://github.com/mvdan/sh/commit/d48a421feafd08247e3b19a6f26b31008ab858c7
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	shDefaults := []string{
		fmt.Sprintf("HOME=%s", os.Getenv("HOME")),
		"IFS= \t\n",
		"OPTIND=1",
		fmt.Sprintf("PWD=%s", pwd),
		fmt.Sprintf("UID=%d", os.Getuid()),
		fmt.Sprintf("GID=%d", os.Getgid()),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, err := EvaluateEnv([]byte(tt.script), tt.argv, tt.env)
			if !tt.expectErr && err != nil {
				t.Fatalf("unexpected error: %s", err)
			} else if tt.expectErr && err == nil {
				t.Fatalf("unexpected success")
			} else if !tt.expectErr {
				tt.resultEnv = append(tt.resultEnv, shDefaults...)
				sort.Strings(tt.resultEnv)
				sort.Strings(env)
				if !reflect.DeepEqual(tt.resultEnv, env) {
					t.Fatalf("unexpected variables:\nwant %v\ngot %v", tt.resultEnv, env)
				}
			}
		})
	}
}
