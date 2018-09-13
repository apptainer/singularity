// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"reflect"
	"syscall"
	"unsafe"

	"github.com/singularityware/singularity/src/pkg/util/mainthread"
	"github.com/singularityware/singularity/src/pkg/util/user"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/singularityware/singularity/src/pkg/instance"
	"github.com/singularityware/singularity/src/pkg/sylog"
)

// StartProcess starts the process
func (engine *EngineOperations) StartProcess(masterConn net.Conn) error {
	isInstance := engine.EngineConfig.GetInstance()
	bootInstance := (isInstance && engine.EngineConfig.GetBootInstance())
	shimProcess := false

	if err := os.Chdir(engine.EngineConfig.OciConfig.Process.Cwd); err != nil {
		if err := os.Chdir(engine.EngineConfig.GetHomeDest()); err != nil {
			os.Chdir("/")
		}
	}

	args := engine.EngineConfig.OciConfig.Process.Args
	env := engine.EngineConfig.OciConfig.Process.Env

	if engine.EngineConfig.OciConfig.Linux != nil {
		namespaces := engine.EngineConfig.OciConfig.Linux.Namespaces
		for _, ns := range namespaces {
			if ns.Type == specs.PIDNamespace {
				if !engine.EngineConfig.GetNoInit() {
					shimProcess = true
				}
				break
			}
		}
	}

	if (!isInstance && !shimProcess) || bootInstance || engine.EngineConfig.GetInstanceJoin() {
		err := syscall.Exec(args[0], args, env)
		return fmt.Errorf("exec %s failed: %s", args[0], err)
	}

	// Spawn and wait container process, signal handler
	cmd := exec.Command(args[0], args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = env

	var status syscall.WaitStatus
	errChan := make(chan error, 1)
	signals := make(chan os.Signal, 1)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("exec %s failed: %s", args[0], err)
	}

	go func() {
		errChan <- cmd.Wait()
	}()

	// Modify argv argument and program name shown in /proc/self/comm
	name := "sinit"

	argv0str := (*reflect.StringHeader)(unsafe.Pointer(&os.Args[0]))
	argv0 := (*[1 << 30]byte)(unsafe.Pointer(argv0str.Data))[:argv0str.Len]
	progname := make([]byte, argv0str.Len)

	if len(name) > argv0str.Len {
		return fmt.Errorf("program name too short")
	}

	copy(progname, name)
	copy(argv0, progname)

	ptr := unsafe.Pointer(&progname[0])
	if _, _, err := syscall.Syscall(syscall.SYS_PRCTL, syscall.PR_SET_NAME, uintptr(ptr), 0); err != 0 {
		return syscall.Errno(err)
	}

	// Manage all signals
	signal.Notify(signals)

	masterConn.Close()

	for {
		select {
		case s := <-signals:
			sylog.Debugf("Received signal %s", s.String())
			switch s {
			case syscall.SIGCHLD:
				for {
					wpid, err := syscall.Wait4(-1, &status, syscall.WNOHANG, nil)
					if wpid <= 0 || err != nil {
						break
					}
				}
			default:
				if isInstance {
					if s != syscall.SIGCONT {
						syscall.Kill(-1, s.(syscall.Signal))
					}
				} else {
					// kill ourself with SIGKILL whatever signal was received
					syscall.Kill(syscall.Gettid(), syscall.SIGKILL)
				}
			}
		case err := <-errChan:
			if e, ok := err.(*exec.ExitError); ok {
				if status, ok := e.Sys().(syscall.WaitStatus); ok {
					if status.Signaled() {
						syscall.Kill(syscall.Gettid(), syscall.SIGKILL)
					}
					os.Exit(status.ExitStatus())
				}
				return fmt.Errorf("command exit with error: %s", err)
			}
			if !isInstance {
				os.Exit(0)
			}
		}
	}
}

// PostStartProcess will execute code in smaster context after execution of container
// process, typically to write instance state/config files or execute post start OCI hook
func (engine *EngineOperations) PostStartProcess(pid int) error {
	sylog.Debugf("Post start process")

	if engine.EngineConfig.GetInstance() {
		uid := os.Getuid()
		gid := os.Getgid()
		name := engine.CommonConfig.ContainerID
		privileged := true

		if engine.EngineConfig.OciConfig.Linux != nil {
			for _, ns := range engine.EngineConfig.OciConfig.Linux.Namespaces {
				if ns.Type == specs.UserNamespace {
					privileged = false
					break
				}
			}
		}

		file, err := instance.Add(name, privileged)
		if err != nil {
			return err
		}

		file.Config, err = json.Marshal(engine.CommonConfig)
		if err != nil {
			return err
		}

		pw, err := user.GetPwUID(uint32(uid))
		if err != nil {
			return err
		}
		file.User = pw.Name
		file.Pid = pid
		file.PPid = os.Getpid()
		file.Image = engine.EngineConfig.GetImage()

		if privileged {
			var err error

			mainthread.Execute(func() {
				if err = syscall.Setresuid(0, 0, uid); err != nil {
					err = fmt.Errorf("failed to escalate uid privileges")
					return
				}
				if err = syscall.Setresgid(0, 0, gid); err != nil {
					err = fmt.Errorf("failed to escalate gid privileges")
					return
				}
				if err = file.Update(); err != nil {
					return
				}
				if err = syscall.Setresgid(gid, gid, 0); err != nil {
					err = fmt.Errorf("failed to escalate gid privileges")
					return
				}
				if err := syscall.Setresuid(uid, uid, 0); err != nil {
					err = fmt.Errorf("failed to escalate uid privileges")
					return
				}
			})

			return err
		}

		return file.Update()
	}
	return nil
}
