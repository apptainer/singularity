// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"testing"
	"time"

	expect "github.com/Netflix/go-expect"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
)

// SingularityCmdResultOp is a function type executed
// by ExpectExit to process and test execution result.
type SingularityCmdResultOp func(*testing.T, *SingularityCmdResult)

// SingularityCmdResult holds the result for a Singularity command
// execution test.
type SingularityCmdResult struct {
	Stdout  []byte
	Stderr  []byte
	FullCmd string
}

// MatchType defines the type of match for ExpectOuput and ExpectError
// functions
type MatchType uint8

const (
	// ContainMatch is for contain match
	ContainMatch MatchType = iota
	// ExactMatch is for exact match
	ExactMatch
	// RegexMatch is for regular expression match
	RegexMatch
)

// streamType defines a stream type
type streamType uint8

const (
	// outputStream is the command output stream
	outputStream streamType = iota
	// errorStream is the command error stream
	errorStream
)

func (r *SingularityCmdResult) expectMatch(mt MatchType, stream streamType, pattern string) error {
	var output string
	var streamName string

	switch stream {
	case outputStream:
		output = string(r.Stdout)
		streamName = "output"
	case errorStream:
		output = string(r.Stderr)
		streamName = "error"
	}

	switch mt {
	case ContainMatch:
		if !strings.Contains(output, pattern) {
			return fmt.Errorf(
				"Command %q:\nExpect %s stream contains:\n%s\nCommand %s stream:\n%s",
				r.FullCmd, streamName, pattern, streamName, output,
			)
		}
	case ExactMatch:
		// get rid of the trailing newline
		if strings.TrimSuffix(output, "\n") != pattern {
			return fmt.Errorf(
				"Command %q:\nExpect %s stream exact match:\n%s\nCommand %s output:\n%s",
				r.FullCmd, streamName, pattern, streamName, output,
			)
		}
	case RegexMatch:
		matched, err := regexp.MatchString(pattern, output)
		if err != nil {
			return fmt.Errorf(
				"compilation of regular expression %q failed: %s",
				pattern, err,
			)
		}
		if !matched {
			return fmt.Errorf(
				"Command %q:\nExpect %s stream match regular expression:\n%s\nCommand %s output:\n%s",
				r.FullCmd, streamName, pattern, streamName, output,
			)
		}
	}

	return nil
}

// ExpectOutput tests if the command output stream match the
// pattern string based on the type of match.
func ExpectOutput(mt MatchType, pattern string) SingularityCmdResultOp {
	return func(t *testing.T, r *SingularityCmdResult) {
		err := r.expectMatch(mt, outputStream, pattern)
		if err != nil {
			t.Error(err)
		}
	}
}

// ExpectOutputf tests if the command output stream match the
// formatted string pattern based on the type of match.
func ExpectOutputf(mt MatchType, formatPattern string, a ...interface{}) SingularityCmdResultOp {
	return func(t *testing.T, r *SingularityCmdResult) {
		pattern := fmt.Sprintf(formatPattern, a...)
		err := r.expectMatch(mt, outputStream, pattern)
		if err != nil {
			t.Error(err)
		}
	}
}

// ExpectError tests if the command error stream match the
// pattern string based on the type of match.
func ExpectError(mt MatchType, pattern string) SingularityCmdResultOp {
	return func(t *testing.T, r *SingularityCmdResult) {
		err := r.expectMatch(mt, errorStream, pattern)
		if err != nil {
			t.Error(err)
		}
	}
}

// ExpectErrorf tests if the command error stream match the
// pattern string based on the type of match.
func ExpectErrorf(mt MatchType, formatPattern string, a ...interface{}) SingularityCmdResultOp {
	return func(t *testing.T, r *SingularityCmdResult) {
		pattern := fmt.Sprintf(formatPattern, a...)
		err := r.expectMatch(mt, errorStream, pattern)
		if err != nil {
			t.Error(err)
		}
	}
}

// SingularityConsoleOp is a function type passed to ConsoleRun
// to execute interactive commands.
type SingularityConsoleOp func(*testing.T, *expect.Console)

// ConsoleExpectf reads from the console until the provided formatted string
// is read or an error occurs.
func ConsoleExpectf(format string, args ...interface{}) SingularityConsoleOp {
	return func(t *testing.T, c *expect.Console) {
		if o, err := c.Expectf(format, args...); err != nil {
			expected := fmt.Sprintf(format, args...)
			t.Logf("\nConsole output: %s\nExpected output: %s", o, expected)
			t.Errorf("error while reading from the console: %s", err)
		}
	}
}

// ConsoleExpect reads from the console until the provided string is read or
// an error occurs.
func ConsoleExpect(s string) SingularityConsoleOp {
	return func(t *testing.T, c *expect.Console) {
		if o, err := c.ExpectString(s); err != nil {
			t.Logf("\nConsole output: %s\nExpected output: %s", o, s)
			t.Errorf("error while reading from the console: %s", err)
		}
	}
}

// ConsoleSend writes a string to the console.
func ConsoleSend(s string) SingularityConsoleOp {
	return func(t *testing.T, c *expect.Console) {
		if _, err := c.Send(s); err != nil {
			t.Errorf("error while writing string to the console: %s", err)
		}
	}
}

// ConsoleSendLine writes a string to the console with a trailing newline.
func ConsoleSendLine(s string) SingularityConsoleOp {
	return func(t *testing.T, c *expect.Console) {
		if _, err := c.SendLine(s); err != nil {
			t.Errorf("error while writing string to the console: %s", err)
		}
	}
}

// SingularityCmdOp is a function type passed to RunCommand
// used to define the test execution context.
type SingularityCmdOp func(*singularityCmd)

// singularityCmd defines a Singularity command execution test.
type singularityCmd struct {
	args       []string
	envs       []string
	dir        string
	privileged bool
	subTest    bool
	stdin      io.Reader
	preFn      func(*testing.T)
	postFn     func(*testing.T)
	consoleFn  SingularityCmdOp
	console    *expect.Console
	resultFn   SingularityCmdOp
	result     *SingularityCmdResult
	waitErr    error
	t          *testing.T
}

// WithCommand sets the singularity command to execute.
func WithCommand(command string) SingularityCmdOp {
	return func(s *singularityCmd) {
		cmd := strings.Split(command, " ")
		s.args = append(cmd, s.args...)
	}
}

// WithArgs sets the singularity command arguments.
func WithArgs(args ...string) SingularityCmdOp {
	return func(s *singularityCmd) {
		if len(args) > 0 {
			s.args = append(s.args, args...)
		}
	}
}

// WithEnv sets environment variables to use while running a
// singularity command.
func WithEnv(envs []string) SingularityCmdOp {
	return func(s *singularityCmd) {
		if len(envs) > 0 {
			s.envs = append(s.envs, envs...)
		}
	}
}

// WithDir sets the current working directory for the execution of a command.
func WithDir(dir string) SingularityCmdOp {
	return func(s *singularityCmd) {
		if dir != "" {
			s.dir = dir
		}
	}
}

// WithPrivileges sets whether a singularity command must be
// executed with privileges or not. PreRun, InRun, PostRun
// are also executed with privileges.
func WithPrivileges(privileged bool) SingularityCmdOp {
	return func(s *singularityCmd) {
		s.privileged = privileged
	}
}

// WithStdin sets a reader to use as input data to pass
// to the singularity command.
func WithStdin(r io.Reader) SingularityCmdOp {
	return func(s *singularityCmd) {
		s.stdin = r
	}
}

// WithoutSubTest marks singularity command as not being
// part of a subtest. May be useful for e2e helpers.
func WithoutSubTest() SingularityCmdOp {
	return func(s *singularityCmd) {
		s.subTest = false
	}
}

// ConsoleRun sets console operations to interact with the
// running command.
func ConsoleRun(consoleOps ...SingularityConsoleOp) SingularityCmdOp {
	return func(s *singularityCmd) {
		if s.consoleFn == nil {
			s.consoleFn = ConsoleRun(consoleOps...)
			return
		}
		for _, op := range consoleOps {
			op(s.t, s.console)
		}
		s.console.ExpectEOF()
	}
}

// PreRun sets a function to execute before running
// the singularity command (executed with privileges
// if WithPrivileges(true) is passed to RunCommand).
func PreRun(fn func(*testing.T)) SingularityCmdOp {
	return func(s *singularityCmd) {
		s.preFn = fn
	}
}

// PostRun sets a function to execute when the singularity
// command execution finished (executed with privileges if
// WithPrivileges(true) is passed to RunCommand). PostRun
// is executed in all cases even when the command execution
// failed, it's the responsibility of the caller to check if the
// test failed with t.Failed().
func PostRun(fn func(*testing.T)) SingularityCmdOp {
	return func(s *singularityCmd) {
		s.postFn = fn
	}
}

// ExpectExit is called once the command completed and before
// PostRun function in order to check the exit code returned. This
// function is always required by RunCommand and can call additional
// test functions processing the command result like ExpectOutput,
// ExpectError.
func ExpectExit(code int, resultOps ...SingularityCmdResultOp) SingularityCmdOp {
	return func(s *singularityCmd) {
		if s.resultFn == nil {
			s.resultFn = ExpectExit(code, resultOps...)
			return
		}

		r := s.result
		t := s.t

		if t.Failed() {
			return
		}

		err := s.waitErr
		switch x := err.(type) {
		case *exec.ExitError:
			if status, ok := x.Sys().(syscall.WaitStatus); ok {
				if code != status.ExitStatus() {
					t.Logf("\n%q output:\n%s%s\n", r.FullCmd, string(r.Stderr), string(r.Stdout))
					t.Errorf("got %d as exit code and was expecting %d)", status.ExitStatus(), code)
					return
				}
			}
		default:
			if err != nil {
				t.Errorf("command execution of %q failed: %s", r.FullCmd, err)
				return
			}
		}

		if code == 0 && err != nil {
			t.Logf("\n%q output:\n%s%s\n", r.FullCmd, string(r.Stderr), string(r.Stdout))
			t.Errorf("unexpected failure while executing %q", r.FullCmd)
			return
		} else if code != 0 && err == nil {
			t.Logf("\n%q output:\n%s%s\n", r.FullCmd, string(r.Stderr), string(r.Stdout))
			t.Errorf("unexpected success while executing %q", r.FullCmd)
			return
		}

		for _, op := range resultOps {
			op(t, r)
		}
	}
}

// RunSingularity executes a singularity command within an test execution
// context.
func RunSingularity(t *testing.T, name string, cmdOps ...SingularityCmdOp) {
	if name == "" {
		t.Errorf("you must provide a test name")
		return
	}

	s := new(singularityCmd)
	s.subTest = true

	for _, op := range cmdOps {
		op(s)
	}
	if s.resultFn == nil {
		t.Errorf("ExpectExit is missing in cmdOps argument")
		return
	}

	fn := func(t *testing.T) {
		s.result = new(SingularityCmdResult)
		s.result.FullCmd = fmt.Sprintf("singularity %s", strings.Join(s.args, " "))

		var (
			stdout bytes.Buffer
			stderr bytes.Buffer
		)

		s.t = t

		cmd := exec.Command("singularity", s.args...)

		cmd.Env = s.envs
		if len(cmd.Env) == 0 {
			cmd.Env = os.Environ()
		}
		if s.privileged {
			cacheDirEnv := fmt.Sprintf("%s=%s", cache.DirEnv, cacheDirPriv)
			cmd.Env = append(cmd.Env, cacheDirEnv)
		}

		cmd.Dir = s.dir
		cmd.Stdin = s.stdin
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if s.consoleFn != nil {
			var err error

			s.console, err = expect.NewTestConsole(
				t,
				expect.WithStdout(cmd.Stdout),
				expect.WithDefaultTimeout(1*time.Second),
			)
			if err != nil {
				t.Errorf("console initialization failed: %s", err)
				return
			}
			defer s.console.Close()

			cmd.Stdin = s.console.Tty()
			cmd.Stdout = s.console.Tty()
			cmd.Stderr = s.console.Tty()

			cmd.SysProcAttr = &syscall.SysProcAttr{
				Setctty: true,
				Setsid:  true,
				Ctty:    int(s.console.Tty().Fd()),
			}
		}

		if s.preFn != nil {
			s.preFn(t)
			// if PreRun call t.Error(f) or t.Skip(f), don't
			// execute the command and return
			if t.Failed() || t.Skipped() {
				return
			}
		}
		if s.postFn != nil {
			defer s.postFn(t)
		}

		t.Logf("Running command %q", s.result.FullCmd)
		if err := cmd.Start(); err != nil {
			t.Errorf("command execution of %q failed: %s", s.result.FullCmd, err)
			return
		}

		if s.consoleFn != nil {
			s.consoleFn(s)
			if t.Failed() {
				cmd.Process.Signal(os.Kill)
				return
			}
		}

		s.waitErr = cmd.Wait()
		s.result.Stdout = stdout.Bytes()
		s.result.Stderr = stderr.Bytes()
		s.resultFn(s)
	}

	if s.privileged {
		fn = Privileged(fn)
	}
	if s.subTest {
		t.Run(name, fn)
	} else {
		fn(t)
	}
}
