// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	osignal "os/signal"
	"sync"
	"syscall"

	"github.com/kr/pty"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/oci"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/ociruntime"
	"github.com/sylabs/singularity/pkg/util/unix"
	"golang.org/x/crypto/ssh/terminal"
)

func resize(controlSocket string, oversized bool) {
	ctrl := &ociruntime.Control{}
	ctrl.ConsoleSize = &specs.Box{}

	c, err := unix.Dial(controlSocket)
	if err != nil {
		sylog.Errorf("failed to connect to control socket")
		return
	}
	defer c.Close()

	rows, cols, err := pty.Getsize(os.Stdin)
	if err != nil {
		sylog.Errorf("terminal resize error: %s", err)
		return
	}

	ctrl.ConsoleSize.Height = uint(rows)
	ctrl.ConsoleSize.Width = uint(cols)

	if oversized {
		ctrl.ConsoleSize.Height++
		ctrl.ConsoleSize.Width++
	}

	enc := json.NewEncoder(c)
	if err != nil {
		sylog.Errorf("%s", err)
		return
	}

	if err := enc.Encode(ctrl); err != nil {
		sylog.Errorf("%s", err)
		return
	}
}

func attach(engineConfig *oci.EngineConfig, run bool) error {
	var ostate *terminal.State
	var conn net.Conn
	var wg sync.WaitGroup

	state := &engineConfig.State

	if state.AttachSocket == "" {
		return fmt.Errorf("attach socket not available, container state: %s", state.Status)
	}
	if state.ControlSocket == "" {
		return fmt.Errorf("control socket not available, container state: %s", state.Status)
	}

	hasTerminal := engineConfig.OciConfig.Process.Terminal && terminal.IsTerminal(0)

	var err error
	conn, err = unix.Dial(state.AttachSocket)
	if err != nil {
		return err
	}
	defer conn.Close()

	if hasTerminal {
		ostate, _ = terminal.MakeRaw(0)
		resize(state.ControlSocket, true)
		resize(state.ControlSocket, false)
	}

	wg.Add(1)

	go func() {
		// catch SIGWINCH signal for terminal resize
		signals := make(chan os.Signal, 1)
		pid := state.Pid
		osignal.Notify(signals)

		for {
			s := <-signals
			switch s {
			case syscall.SIGWINCH:
				if hasTerminal {
					resize(state.ControlSocket, false)
				}
			default:
				syscall.Kill(pid, s.(syscall.Signal))
			}
		}
	}()

	// Pipe session to bash and visa-versa
	go func() {
		if !run {
			io.Copy(os.Stdout, conn)
		} else {
			io.Copy(ioutil.Discard, conn)
		}
		wg.Done()
	}()

	go func() {
		io.Copy(conn, os.Stdin)
	}()

	wg.Wait()

	if hasTerminal {
		fmt.Printf("\r")
		return terminal.Restore(0, ostate)
	}

	return nil
}

// OciAttach attaches console to a running container
func OciAttach(containerID string) error {
	engineConfig, err := getEngineConfig(containerID)
	if err != nil {
		return err
	}

	defer exitContainer(containerID, false)

	return attach(engineConfig, false)
}
