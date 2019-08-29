// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oci

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	osexec "os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/kr/pty"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/internal/pkg/instance"
	"github.com/sylabs/singularity/internal/pkg/security"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/exec"
	"github.com/sylabs/singularity/pkg/ociruntime"
	"github.com/sylabs/singularity/pkg/util/copy"
	"github.com/sylabs/singularity/pkg/util/rlimit"
	"github.com/sylabs/singularity/pkg/util/unix"
)

func setRlimit(rlimits []specs.POSIXRlimit) error {
	resources := make(map[string]struct{})

	for _, rl := range rlimits {
		if err := rlimit.Set(rl.Type, rl.Soft, rl.Hard); err != nil {
			return err
		}
		if _, found := resources[rl.Type]; found {
			return fmt.Errorf("%s was already set", rl.Type)
		}
		resources[rl.Type] = struct{}{}
	}

	return nil
}

func (e *EngineOperations) emptyProcess(masterConn net.Conn) error {
	// pause process on next read
	if _, err := masterConn.Write([]byte("t")); err != nil {
		return fmt.Errorf("failed to pause process: %s", err)
	}

	// block on read start given
	data := make([]byte, 1)
	if _, err := masterConn.Read(data); err != nil {
		return fmt.Errorf("failed to receive ack from master: %s", err)
	}

	var status syscall.WaitStatus
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGCHLD, syscall.SIGINT, syscall.SIGTERM)

	if err := security.Configure(&e.EngineConfig.OciConfig.Spec); err != nil {
		return fmt.Errorf("failed to apply security configuration: %s", err)
	}

	masterConn.Close()

	for {
		s := <-signals
		switch s {
		case syscall.SIGCHLD:
			for {
				if pid, _ := syscall.Wait4(-1, &status, syscall.WNOHANG, nil); pid <= 0 {
					break
				}
			}
		case syscall.SIGINT, syscall.SIGTERM:
			os.Exit(0)
		}
	}
}

// StartProcess starts the process
func (e *EngineOperations) StartProcess(masterConn net.Conn) error {
	cwd := e.EngineConfig.OciConfig.Process.Cwd

	if cwd == "" {
		cwd = "/"
	}

	if !filepath.IsAbs(cwd) {
		return fmt.Errorf("cwd property must be an absolute path")
	}

	if err := os.Chdir(cwd); err != nil {
		return fmt.Errorf("can't enter in current working directory: %s", err)
	}

	if err := setRlimit(e.EngineConfig.OciConfig.Process.Rlimits); err != nil {
		return err
	}

	if e.EngineConfig.EmptyProcess {
		return e.emptyProcess(masterConn)
	}

	args := e.EngineConfig.OciConfig.Process.Args
	env := e.EngineConfig.OciConfig.Process.Env

	for _, e := range e.EngineConfig.OciConfig.Process.Env {
		if strings.HasPrefix(e, "PATH=") {
			os.Setenv("PATH", e[5:])
		}
	}

	bpath, err := osexec.LookPath(args[0])
	if err != nil {
		return fmt.Errorf("%s", err)
	}
	args[0] = bpath

	if e.EngineConfig.MasterPts != -1 {
		slaveFd := e.EngineConfig.SlavePts
		if err := syscall.Dup3(slaveFd, int(os.Stdin.Fd()), 0); err != nil {
			return err
		}
		if err := syscall.Dup3(slaveFd, int(os.Stdout.Fd()), 0); err != nil {
			return err
		}
		if err := syscall.Dup3(slaveFd, int(os.Stderr.Fd()), 0); err != nil {
			return err
		}
		if err := syscall.Close(e.EngineConfig.MasterPts); err != nil {
			return err
		}
		if err := syscall.Close(slaveFd); err != nil {
			return err
		}
		if _, err := syscall.Setsid(); err != nil {
			return err
		}
		if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, os.Stdin.Fd(), uintptr(syscall.TIOCSCTTY), 1); err != 0 {
			return fmt.Errorf("failed to set crontrolling terminal: %s", err.Error())
		}
	} else if e.EngineConfig.OutputStreams[1] != -1 {
		if err := syscall.Dup3(e.EngineConfig.OutputStreams[1], int(os.Stdout.Fd()), 0); err != nil {
			return err
		}
		if err := syscall.Close(e.EngineConfig.OutputStreams[1]); err != nil {
			return err
		}
		if err := syscall.Close(e.EngineConfig.OutputStreams[0]); err != nil {
			return err
		}

		if err := syscall.Dup3(e.EngineConfig.ErrorStreams[1], int(os.Stderr.Fd()), 0); err != nil {
			return err
		}
		if err := syscall.Close(e.EngineConfig.ErrorStreams[1]); err != nil {
			return err
		}
		if err := syscall.Close(e.EngineConfig.ErrorStreams[0]); err != nil {
			return err
		}

		if err := syscall.Dup3(e.EngineConfig.InputStreams[1], int(os.Stdin.Fd()), 0); err != nil {
			return err
		}
		if err := syscall.Close(e.EngineConfig.InputStreams[1]); err != nil {
			return err
		}
		if err := syscall.Close(e.EngineConfig.InputStreams[0]); err != nil {
			return err
		}
	}

	// trigger pre-start process
	if _, err := masterConn.Write([]byte("t")); err != nil {
		return fmt.Errorf("failed to pause process: %s", err)
	}
	if !e.EngineConfig.Exec {
		// block on read start given
		data := make([]byte, 1)
		if _, err := masterConn.Read(data); err != nil {
			return fmt.Errorf("failed to receive start signal: %s", err)
		}
	}

	if err := security.Configure(&e.EngineConfig.OciConfig.Spec); err != nil {
		return fmt.Errorf("failed to apply security configuration: %s", err)
	}

	err = syscall.Exec(args[0], args, env)
	return fmt.Errorf("exec %s failed: %s", args[0], err)
}

// PreStartProcess will be executed in master context
func (e *EngineOperations) PreStartProcess(pid int, masterConn net.Conn, fatalChan chan error) error {
	if e.EngineConfig.Exec {
		return nil
	}

	file, err := instance.Get(e.CommonConfig.ContainerID, instance.OciSubDir)
	if err != nil {
		return err
	}
	e.EngineConfig.State.AttachSocket = filepath.Join(filepath.Dir(file.Path), "attach.sock")

	attach, err := unix.CreateSocket(e.EngineConfig.State.AttachSocket)
	if err != nil {
		return err
	}

	e.EngineConfig.State.ControlSocket = filepath.Join(filepath.Dir(file.Path), "control.sock")

	control, err := unix.CreateSocket(e.EngineConfig.State.ControlSocket)
	if err != nil {
		return err
	}

	logPath := e.EngineConfig.GetLogPath()
	if logPath == "" {
		containerID := e.CommonConfig.ContainerID
		dir, err := instance.GetDir(containerID, instance.OciSubDir)
		if err != nil {
			return err
		}
		logPath = filepath.Join(dir, containerID+".log")
	}

	format := e.EngineConfig.GetLogFormat()
	formatter, ok := instance.LogFormats[format]
	if !ok {
		return fmt.Errorf("log format %s is not supported", format)
	}

	logger, err := instance.NewLogger(logPath, formatter)
	if err != nil {
		return err
	}

	pidFile := e.EngineConfig.GetPidFile()
	if pidFile != "" {
		if err := ioutil.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644); err != nil {
			return err
		}
	}

	if err := e.updateState(ociruntime.Created); err != nil {
		return err
	}

	start := make(chan bool, 1)

	go e.handleControl(masterConn, attach, control, logger, start, fatalChan)

	hooks := e.EngineConfig.OciConfig.Hooks
	if hooks != nil {
		for _, h := range hooks.Prestart {
			if err := exec.Hook(&h, &e.EngineConfig.State.State); err != nil {
				return err
			}
		}
	}

	// detach process
	syscall.Kill(os.Getppid(), syscall.SIGUSR1)

	// block until start event received
	<-start
	close(start)

	return nil
}

// PostStartProcess will execute code in master context after execution of container
// process, typically to write instance state/config files or execute post start OCI hook
func (e *EngineOperations) PostStartProcess(pid int) error {
	if err := e.updateState(ociruntime.Running); err != nil {
		return err
	}
	hooks := e.EngineConfig.OciConfig.Hooks
	if hooks != nil {
		for _, h := range hooks.Poststart {
			if err := exec.Hook(&h, &e.EngineConfig.State.State); err != nil {
				sylog.Warningf("%s", err)
			}
		}
	}
	return nil
}

func (e *EngineOperations) handleStream(l net.Listener, logger *instance.Logger, fatalChan chan error) {
	var stdout io.ReadWriteCloser
	var stderr io.ReadCloser
	var stdin io.WriteCloser
	var outputWriters *copy.MultiWriter
	var errorWriters *copy.MultiWriter
	var inputWriters *copy.MultiWriter
	var tbuf *copy.TerminalBuffer

	hasTerminal := e.EngineConfig.OciConfig.Process.Terminal

	inputWriters = &copy.MultiWriter{}
	outputWriters = &copy.MultiWriter{}
	outWriter, _ := logger.NewWriter("stdout", true)
	outputWriters.Add(outWriter)

	if hasTerminal {
		stdout = os.NewFile(uintptr(e.EngineConfig.MasterPts), "stream-master-pts")
		tbuf = copy.NewTerminalBuffer()
		outputWriters.Add(tbuf)
		inputWriters.Add(stdout)
	} else {
		outputStream := os.NewFile(uintptr(e.EngineConfig.OutputStreams[0]), "stdout-stream")
		errorStream := os.NewFile(uintptr(e.EngineConfig.ErrorStreams[0]), "error-stream")
		inputStream := os.NewFile(uintptr(e.EngineConfig.InputStreams[0]), "input-stream")
		stdout = outputStream
		stderr = errorStream
		stdin = inputStream
		outputWriters.Add(os.Stdout)
		inputWriters.Add(stdin)
	}

	if stderr != nil {
		errorWriters = &copy.MultiWriter{}
		errWriter, _ := logger.NewWriter("stderr", true)
		errorWriters.Add(errWriter)
		errorWriters.Add(os.Stderr)
	}

	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				fatalChan <- err
				return
			}

			go func() {
				outputWriters.Add(c)
				if stderr != nil {
					errorWriters.Add(c)
				}

				if tbuf != nil {
					c.Write(tbuf.Line())
				}

				io.Copy(inputWriters, c)

				outputWriters.Del(c)
				if stderr != nil {
					errorWriters.Del(c)
				}
				c.Close()
			}()
		}
	}()

	go func() {
		io.Copy(outputWriters, stdout)
		stdout.Close()
	}()

	if stderr != nil {
		go func() {
			io.Copy(errorWriters, stderr)
			stderr.Close()
		}()
	}
	if stdin != nil {
		go func() {
			io.Copy(inputWriters, os.Stdin)
			stdin.Close()
		}()
	}
}

func (e *EngineOperations) handleControl(masterConn net.Conn, attach net.Listener, control net.Listener, logger *instance.Logger, start chan bool, fatalChan chan error) {
	var master *os.File
	started := false

	if e.EngineConfig.OciConfig.Process.Terminal {
		master = os.NewFile(uintptr(e.EngineConfig.MasterPts), "control-master-pts")
	}

	for {
		c, err := control.Accept()
		if err != nil {
			fatalChan <- err
			return
		}
		dec := json.NewDecoder(c)
		ctrl := &ociruntime.Control{}
		if err := dec.Decode(ctrl); err != nil {
			fatalChan <- err
			return
		}

		if ctrl.StartContainer && !started {
			started = true

			e.handleStream(attach, logger, fatalChan)

			// since container process block on read, send it an
			// ACK so when it will receive data, the container
			// process will be executed
			if _, err := masterConn.Write([]byte("s")); err != nil {
				fatalChan <- fmt.Errorf("failed to send ACK to start process: %s", err)
				return
			}

			// send start event
			start <- true

			// wait status update
			e.waitStatusUpdate()
		}
		if ctrl.ConsoleSize != nil && master != nil {
			size := &pty.Winsize{
				Cols: uint16(ctrl.ConsoleSize.Width),
				Rows: uint16(ctrl.ConsoleSize.Height),
			}
			if err := pty.Setsize(master, size); err != nil {
				fatalChan <- err
				return
			}
		}
		if ctrl.ReopenLog {
			if err := logger.ReOpenFile(); err != nil {
				fatalChan <- err
				return
			}
		}
		if ctrl.Pause {
			if err := e.EngineConfig.Cgroups.Pause(); err != nil {
				fatalChan <- err
				return
			}
			if err := e.updateState(ociruntime.Paused); err != nil {
				fatalChan <- err
				return
			}
		}
		if ctrl.Resume {
			if err := e.updateState(ociruntime.Running); err != nil {
				fatalChan <- err
				return
			}
			if err := e.EngineConfig.Cgroups.Resume(); err != nil {
				fatalChan <- err
				return
			}
		}

		c.Close()
	}
}
