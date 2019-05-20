package exec

import (
	"bytes"
	stdexec "os/exec"
	"strings"
	"syscall"
	"testing"
)

type Cmd struct {
	*stdexec.Cmd
	cmdStr         string
	combinedOutput bytes.Buffer
}

func Command(name string, arg ...string) *Cmd {
	return &Cmd{
		Cmd: stdexec.Command(name, arg...),
	}
}

func (cmd *Cmd) ExecExpectCode(t *testing.T, code int) {
	cmd.Stdout = &cmd.combinedOutput
	cmd.Stderr = &cmd.combinedOutput

	if err := cmd.Start(); err != nil {
		t.Fatalf("cannot start command [%s]: %+v", cmd, err)
	}

	if err := cmd.Wait(); err != nil {
		switch err := err.(type) {
		case *stdexec.ExitError:
			ws, ok := err.Sys().(syscall.WaitStatus)
			if !ok {
				// this should never happen
				t.Fatalf("cannot get WaitStatus from %+v\nCommand: %s\n\nOutput:\n%s\n",
					err.Sys(),
					cmd,
					cmd.combinedOutput.String())
			}

			if es := ws.ExitStatus(); es != code {
				// The program has exited with an unexpected exit code
				t.Fatalf("unexpected exit code '%d', expecting '%d'\nCommand: %s\n\nOutput:\n%s\n",
					es,
					code,
					cmd,
					cmd.combinedOutput.String())
			}

		default:
			t.Fatalf("unexpected erro: %+v\nCommand: %s\n\nOutput:\n%s\n",
				err,
				cmd,
				cmd.combinedOutput.String())
		}
	} else if ec := cmd.ProcessState.ExitCode(); ec != code {
		// The program has exited with an unexpected exit code
		t.Fatalf("unexpected exit code '%d', expecting '%d'\nCommand: %s\n\nOutput:\n%s\n",
			ec,
			code,
			cmd,
			cmd.combinedOutput.String())
	}
}

func (cmd *Cmd) String() string {
	if len(cmd.cmdStr) > 0 {
		return cmd.cmdStr
	}

	var b strings.Builder
	b.WriteRune('\'')
	b.WriteString(cmd.Path)
	b.WriteRune('\'')
	if len(cmd.Args) > 1 {
		for _, arg := range cmd.Args[1:] {
			b.WriteRune(' ')
			b.WriteRune('\'')
			b.WriteString(arg)
			b.WriteRune('\'')
		}
	}

	cmd.cmdStr = b.String()

	return cmd.cmdStr
}
