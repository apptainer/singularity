// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oci

import (
	"fmt"
	"io"
	"net"
	"os"
	osexec "os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/util/unix"

	"github.com/sylabs/singularity/internal/pkg/instance"
	"github.com/sylabs/singularity/internal/pkg/util/exec"

	"github.com/sylabs/singularity/internal/pkg/security"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

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

	if engine.EngineConfig.EmptyProcess {
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

	args := engine.EngineConfig.OciConfig.Process.Args
	env := engine.EngineConfig.OciConfig.Process.Env

	os.Setenv("PATH", "/bin:/usr/bin:/sbin:/usr/sbin:/usr/local/bin:/usr/local/sbin")

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
		if _, err := syscall.Setsid(); err != nil {
			return err
		}
		if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, os.Stdin.Fd(), uintptr(syscall.TIOCSCTTY), 1); err != 0 {
			return fmt.Errorf("failed to set crontrolling terminal: %s", err.Error())
		}
	}

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

	if err := security.Configure(&engine.EngineConfig.OciConfig.Spec); err != nil {
		return fmt.Errorf("failed to apply security configuration: %s", err)
	}

	err = syscall.Exec(args[0], args, env)

	// write data to just tell Smaster to not execute PostStartProcess
	// in case of failure
	if _, err := masterConn.Write([]byte("t")); err != nil {
		sylog.Errorf("fail to send data to Smaster: %s", err)
	}

	return fmt.Errorf("exec %s failed: %s", args[0], err)
}

// PreStartProcess will be executed in smaster context
func (engine *EngineOperations) PreStartProcess() error {
	var master *os.File

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
		master = os.Stdin
	}

	file, err := instance.Get(engine.CommonConfig.ContainerID)
	socket := filepath.Join(filepath.Dir(file.Path), engine.CommonConfig.ContainerID+".sock")
	engine.EngineConfig.State.Annotations["io.sylabs.runtime.oci.attach-socket"] = socket

	l, err := unix.CreateSocket(socket)
	if err != nil {
		return err
	}

	if err := engine.updateState("created"); err != nil {
		return err
	}

	go engine.handleStream(master, l)

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
	mux     sync.Mutex
	writers []io.Writer
}

func (mw *multiWriter) Write(p []byte) (n int, err error) {
	mw.mux.Lock()
	defer mw.mux.Unlock()

	for _, w := range mw.writers {
		n, err = w.Write(p)
		if err != nil {
			return
		}
		if n != len(p) {
			err = io.ErrShortWrite
			return
		}
	}
	return len(p), nil
}

func (mw *multiWriter) Add(writer io.Writer) {
	mw.mux.Lock()
	mw.writers = append(mw.writers, writer)
	mw.mux.Unlock()
}

func MultiWriter(writers ...io.Writer) *multiWriter {
	allwriters := make([]io.Writer, 0, len(writers))

	for _, w := range writers {
		if mw, ok := w.(*multiWriter); ok {
			allwriters = append(allwriters, mw.writers...)
		} else {
			allwriters = append(allwriters, w)
		}
	}
	return &multiWriter{writers: allwriters}
}

type TestWriter struct{}

func (t *TestWriter) Write(p []byte) (n int, err error) {
	// duplicate stream example
	return len(p), nil
}

func (engine *EngineOperations) handleStream(master *os.File, l net.Listener) {
	var err error

	defer l.Close()

	numClient := -1
	maxClient := 10
	a := make([]net.Conn, maxClient)
	var mw *multiWriter

	tee := io.TeeReader(master, &TestWriter{})

	for {
		numClient++
		if numClient == maxClient {
			continue
		}
		a[numClient], err = l.Accept()
		if err != nil {
			sylog.Fatalf("%s", err)
		}

		b := a[numClient]

		if mw == nil {
			mw = MultiWriter(b)
			go func() {
				io.Copy(mw, tee)
			}()
		} else {
			mw.Add(b)
		}

		go func() {
			io.Copy(master, b)
			b.Close()
		}()
	}
}
