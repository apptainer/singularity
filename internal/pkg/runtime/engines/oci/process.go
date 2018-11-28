// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oci

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	osexec "os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/kr/pty"

	"golang.org/x/crypto/ssh/terminal"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/internal/pkg/util/unix"
	"github.com/sylabs/singularity/pkg/ociruntime"
	"github.com/sylabs/singularity/pkg/util/rlimit"

	"github.com/sylabs/singularity/internal/pkg/instance"
	"github.com/sylabs/singularity/internal/pkg/util/exec"

	"github.com/sylabs/singularity/internal/pkg/security"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

func setRlimit(rlimits []specs.POSIXRlimit) error {
	var resources []string

	for _, rl := range rlimits {
		if err := rlimit.Set(rl.Type, rl.Soft, rl.Hard); err != nil {
			return err
		}
		for _, t := range resources {
			if t == rl.Type {
				return fmt.Errorf("%s was already set", t)
			}
		}
		resources = append(resources, rl.Type)
	}

	return nil
}

func (engine *EngineOperations) emptyProcess(masterConn net.Conn) error {
	// pause process, by sending data to Smaster the process will
	// be paused with SIGSTOP signal
	if _, err := masterConn.Write([]byte("t")); err != nil {
		return fmt.Errorf("failed to pause process: %s", err)
	}

	// block on read waiting SIGCONT signal
	data := make([]byte, 1)
	if _, err := masterConn.Read(data); err != nil {
		return fmt.Errorf("failed to receive ack from Smaster: %s", err)
	}

	masterConn.Close()

	var status syscall.WaitStatus
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGCHLD, syscall.SIGINT, syscall.SIGTERM)

	if err := security.Configure(&engine.EngineConfig.OciConfig.Spec); err != nil {
		return fmt.Errorf("failed to apply security configuration: %s", err)
	}

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
func (engine *EngineOperations) StartProcess(masterConn net.Conn) error {
	cwd := engine.EngineConfig.OciConfig.Process.Cwd

	if cwd == "" {
		cwd = "/"
	}

	if !filepath.IsAbs(cwd) {
		return fmt.Errorf("cwd property must be an absolute path")
	}

	if err := os.Chdir(cwd); err != nil {
		return fmt.Errorf("can't enter in current working directory: %s", err)
	}

	if err := setRlimit(engine.EngineConfig.OciConfig.Process.Rlimits); err != nil {
		return err
	}

	if engine.EngineConfig.EmptyProcess {
		return engine.emptyProcess(masterConn)
	}

	args := engine.EngineConfig.OciConfig.Process.Args
	env := engine.EngineConfig.OciConfig.Process.Env

	for _, e := range engine.EngineConfig.OciConfig.Process.Env {
		if strings.HasPrefix(e, "PATH=") {
			os.Setenv("PATH", e[5:])
		}
	}

	bpath, err := osexec.LookPath(args[0])
	if err != nil {
		return fmt.Errorf("%s", err)
	}
	args[0] = bpath

	if engine.EngineConfig.MasterPts != -1 {
		slaveFd := engine.EngineConfig.SlavePts
		if err := syscall.Dup3(slaveFd, int(os.Stdin.Fd()), 0); err != nil {
			return err
		}
		if err := syscall.Dup3(slaveFd, int(os.Stdout.Fd()), 0); err != nil {
			return err
		}
		if err := syscall.Dup3(slaveFd, int(os.Stderr.Fd()), 0); err != nil {
			return err
		}
		if err := syscall.Close(engine.EngineConfig.MasterPts); err != nil {
			return err
		}
		if err := syscall.Close(slaveFd); err != nil {
			return err
		}
		if terminal.IsTerminal(int(os.Stdin.Fd())) {
			if _, err := syscall.Setsid(); err != nil {
				return err
			}
			if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, os.Stdin.Fd(), uintptr(syscall.TIOCSCTTY), 1); err != 0 {
				return fmt.Errorf("failed to set crontrolling terminal: %s", err.Error())
			}
		} else {
			os.Stdin.Close()
		}
	} else {
		if err := syscall.Dup3(int(os.Stdout.Fd()), int(os.Stderr.Fd()), 0); err != nil {
			return err
		}
	}

	if !engine.EngineConfig.Exec {
		// pause process, by sending data to Smaster the process will
		// be paused with SIGSTOP signal
		if _, err := masterConn.Write([]byte("t")); err != nil {
			return fmt.Errorf("failed to pause process: %s", err)
		}

		// block on read waiting SIGCONT signal
		data := make([]byte, 1)
		if _, err := masterConn.Read(data); err != nil {
			return fmt.Errorf("failed to receive ack from Smaster: %s", err)
		}
	}

	if err := security.Configure(&engine.EngineConfig.OciConfig.Spec); err != nil {
		return fmt.Errorf("failed to apply security configuration: %s", err)
	}

	err = syscall.Exec(args[0], args, env)

	if !engine.EngineConfig.Exec {
		// write data to just tell Smaster to not execute PostStartProcess
		// in case of failure
		if _, err := masterConn.Write([]byte("t")); err != nil {
			sylog.Errorf("fail to send data to Smaster: %s", err)
		}
	}

	return fmt.Errorf("exec %s failed: %s", args[0], err)
}

// PreStartProcess will be executed in smaster context
func (engine *EngineOperations) PreStartProcess(pid int, masterConn net.Conn, fatalChan chan error) error {
	var master *os.File

	// stop container process
	syscall.Kill(pid, syscall.SIGSTOP)

	hooks := engine.EngineConfig.OciConfig.Hooks
	if hooks != nil {
		for _, h := range hooks.Prestart {
			if err := exec.Hook(&h, &engine.EngineConfig.State); err != nil {
				return err
			}
		}
	}

	if engine.EngineConfig.MasterPts != -1 {
		master = os.NewFile(uintptr(engine.EngineConfig.MasterPts), "master-pts")
	} else {
		master = os.Stdout
	}

	file, err := instance.Get(engine.CommonConfig.ContainerID)
	socket := filepath.Join(filepath.Dir(file.Path), "attach.sock")
	engine.EngineConfig.State.Annotations[ociruntime.AnnotationAttachSocket] = socket

	attach, err := unix.CreateSocket(socket)
	if err != nil {
		return err
	}

	socket = filepath.Join(filepath.Dir(file.Path), "control.sock")
	engine.EngineConfig.State.Annotations[ociruntime.AnnotationControlSocket] = socket
	control, err := unix.CreateSocket(socket)
	if err != nil {
		return err
	}

	if err := engine.updateState("created"); err != nil {
		return err
	}

	logPath := engine.EngineConfig.GetLogPath()
	if logPath == "" {
		containerID := engine.CommonConfig.ContainerID
		dir, err := instance.GetDirPrivileged(containerID)
		if err != nil {
			return err
		}
		logPath = filepath.Join(dir, containerID+".log")
	}

	logger, err := NewLogger(logPath)
	if err != nil {
		return err
	}

	go engine.handleControl(master, control, logger, fatalChan)
	go engine.handleStream(master, attach, logger, fatalChan)

	// since paused process block on read, send it an
	// ACK so when it will receive SIGCONT, the process
	// will continue execution normally
	if _, err := masterConn.Write([]byte("s")); err != nil {
		return fmt.Errorf("failed to send ACK to start process: %s", err)
	}

	// wait container process execution
	data := make([]byte, 1)

	if _, err := masterConn.Read(data); err != io.EOF {
		return err
	}

	return nil
}

// PostStartProcess will execute code in smaster context after execution of container
// process, typically to write instance state/config files or execute post start OCI hook
func (engine *EngineOperations) PostStartProcess(pid int) error {
	if err := engine.updateState("running"); err != nil {
		return err
	}

	hooks := engine.EngineConfig.OciConfig.Hooks
	if hooks != nil {
		for _, h := range hooks.Poststart {
			if err := exec.Hook(&h, &engine.EngineConfig.State); err != nil {
				sylog.Warningf("%s", err)
			}
		}
	}

	return nil
}

type multiWriter struct {
	sync.Mutex
	writers []io.Writer
}

func (mw *multiWriter) Write(p []byte) (n int, err error) {
	mw.Lock()
	defer mw.Unlock()

	l := len(p)

	for _, w := range mw.writers {
		n, err = w.Write(p)
		if err != nil {
			return
		}
		if n != l {
			err = io.ErrShortWrite
			return
		}
	}

	return l, nil
}

func (mw *multiWriter) Add(writer io.Writer) {
	mw.Lock()
	mw.writers = append(mw.writers, writer)
	mw.Unlock()
}

func (mw *multiWriter) Del(writer io.Writer) {
	mw.Lock()
	for i, w := range mw.writers {
		if writer == w {
			mw.writers = append(mw.writers[:i], mw.writers[i+1:]...)
			break
		}
	}
	mw.Unlock()
}

type Logger struct {
	reader      *io.PipeReader
	writer      *io.PipeWriter
	buffer      []byte
	bufferMutex sync.Mutex
	file        *os.File
	fileMutex   sync.Mutex
}

func NewLogger(logPath string) (*Logger, error) {
	logger := &Logger{}
	logger.reader, logger.writer = io.Pipe()

	if err := logger.openFile(logPath); err != nil {
		return nil, err
	}

	go logger.scan()

	return logger, nil
}

func (l *Logger) openFile(path string) (err error) {
	oldmask := syscall.Umask(0)
	defer syscall.Umask(oldmask)

	l.file, err = os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)

	return err
}

func (l *Logger) ScanOutput(data []byte, atEOF bool) (advance int, token []byte, err error) {
	length := len(data)

	if atEOF && length == 0 {
		return 0, nil, nil
	}

	l.bufferMutex.Lock()
	defer l.bufferMutex.Unlock()
	l.buffer = data

	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		l.buffer = nil
		return i + 1, data[0 : i+1], nil
	}

	if atEOF {
		return length, data[0:length], nil
	}

	return 0, nil, nil
}

func (l *Logger) GetBuffer() []byte {
	l.bufferMutex.Lock()
	defer l.bufferMutex.Unlock()

	return l.buffer
}

func (l *Logger) GetWriter() *io.PipeWriter {
	return l.writer
}

func (l *Logger) scan() {
	scanner := bufio.NewScanner(l.reader)
	scanner.Split(l.ScanOutput)

	for scanner.Scan() {
		l.fileMutex.Lock()
		fmt.Fprintf(l.file, "%s stdout F %s", time.Now().Format(time.RFC3339Nano), scanner.Text())
		l.fileMutex.Unlock()
	}

	l.reader.Close()
	l.writer.Close()
	l.file.Close()
}

func (l *Logger) ReOpenFile() {
	l.fileMutex.Lock()
	defer l.fileMutex.Unlock()

	path := l.file.Name()

	l.file.Close()

	l.openFile(path)
}

func (engine *EngineOperations) handleStream(master *os.File, l net.Listener, logger *Logger, fatalChan chan error) {
	hasTerminal := engine.EngineConfig.OciConfig.Process.Terminal
	defer l.Close()

	mw := &multiWriter{}
	mw.Add(logger.GetWriter())

	go func() {
		io.Copy(mw, master)
	}()

	for {
		c, err := l.Accept()
		if err != nil {
			fatalChan <- err
			return
		}

		go func() {
			mw.Add(c)
			c.Write(logger.GetBuffer())
			if hasTerminal {
				io.Copy(master, c)
			} else {
				io.Copy(ioutil.Discard, c)
			}
			mw.Del(c)
			c.Close()
		}()
	}
}

func (engine *EngineOperations) handleControl(master *os.File, l net.Listener, logger *Logger, fatalChan chan error) {
	for {
		ctrl := &ociruntime.Control{}

		c, err := l.Accept()
		if err != nil {
			fatalChan <- err
			return
		}
		dec := json.NewDecoder(c)
		if err := dec.Decode(ctrl); err != nil {
			fatalChan <- err
			return
		}

		c.Close()

		if ctrl.ConsoleSize != nil {
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
			logger.ReOpenFile()
		}
	}
}
