// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/internal/pkg/instance"
	"github.com/sylabs/singularity/internal/pkg/plugin"
	"github.com/sylabs/singularity/internal/pkg/security"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/env"
	"github.com/sylabs/singularity/internal/pkg/util/machine"
	"github.com/sylabs/singularity/internal/pkg/util/user"
	singularitycallback "github.com/sylabs/singularity/pkg/plugin/callback/runtime/engine/singularity"
	"github.com/sylabs/singularity/pkg/util/rlimit"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/sys/unix"
)

const defaultShell = "/bin/sh"

// StartProcess is called during stage2 after RPC server finished
// environment preparation. This is the container process itself.
//
// No additional privileges can be gained during this call (unless container
// is executed as root intentionally) as starter will set uid/euid/suid
// to the targetUID (PrepareConfig will set it by calling starter.Config.SetTargetUID).
func (e *EngineOperations) StartProcess(masterConn net.Conn) error {
	// Manage all signals.
	// Queue them until they're ready to be handled below.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals)

	if err := e.runFuseDrivers(true, -1); err != nil {
		return err
	}

	isInstance := e.EngineConfig.GetInstance()
	bootInstance := isInstance && e.EngineConfig.GetBootInstance()
	shimProcess := false

	if err := os.Chdir(e.EngineConfig.OciConfig.Process.Cwd); err != nil {
		if err := os.Chdir(e.EngineConfig.GetHomeDest()); err != nil {
			os.Chdir("/")
		}
	}

	if err := e.checkExec(); err != nil {
		return err
	}

	if e.EngineConfig.File.MountDev == "minimal" || e.EngineConfig.GetContain() {
		// If on a terminal, reopen /dev/console so /proc/self/fd/[0-2
		//   will point to /dev/console.  This is needed so that tty and
		//   ttyname() on el6 will return the correct answer.  Newer
		//   ttyname() functions might work because they will search
		//   /dev if the value of /proc/self/fd/X doesn't exist, but
		//   they won't work if another /dev/pts/X is allocated in its
		//   place.  Also, programs that don't use ttyname() and instead
		//   directly do readlink() on /proc/self/fd/X need this.
		for fd := 0; fd <= 2; fd++ {
			if !terminal.IsTerminal(fd) {
				continue
			}
			consfile, err := os.OpenFile("/dev/console", os.O_RDWR, 0600)
			if err != nil {
				sylog.Debugf("Could not open minimal /dev/console, skipping replacing tty descriptors")
				break
			}
			sylog.Debugf("Replacing tty descriptors with /dev/console")
			consfd := int(consfile.Fd())
			for ; fd <= 2; fd++ {
				if !terminal.IsTerminal(fd) {
					continue
				}
				syscall.Close(fd)
				syscall.Dup3(consfd, fd, 0)
			}
			consfile.Close()
			break
		}
	}

	args := e.EngineConfig.OciConfig.Process.Args
	env := e.EngineConfig.OciConfig.Process.Env

	if e.EngineConfig.OciConfig.Linux != nil {
		namespaces := e.EngineConfig.OciConfig.Linux.Namespaces
		for _, ns := range namespaces {
			if ns.Type == specs.PIDNamespace {
				if !e.EngineConfig.GetNoInit() {
					shimProcess = true
				}
				break
			}
		}
	}

	for _, img := range e.EngineConfig.GetImageList() {
		// bad file descriptor error is ignored because
		// the file descriptor has been previously closed
		// in this loop, happens when a SIF image contains
		// overlay partition in it as each SIF overlay
		// partition is considered as a single image with
		// different offset/size but pointing to the same
		// opened image file descriptor
		if err := syscall.Close(int(img.Fd)); err != nil && err != syscall.EBADF {
			return fmt.Errorf("failed to close file descriptor for %s: %s", img.Path, err)
		}
	}

	for _, fd := range e.EngineConfig.GetOpenFd() {
		if err := syscall.Close(fd); err != nil {
			return fmt.Errorf("aborting failed to close file descriptor: %s", err)
		}
	}

	// restore the stack size limit for setuid workflow
	for _, limit := range e.EngineConfig.OciConfig.Process.Rlimits {
		if limit.Type == "RLIMIT_STACK" {
			if err := rlimit.Set(limit.Type, limit.Soft, limit.Hard); err != nil {
				return fmt.Errorf("while restoring stack size limit: %s", err)
			}
			break
		}
	}

	if err := security.Configure(&e.EngineConfig.OciConfig.Spec); err != nil {
		return fmt.Errorf("failed to apply security configuration: %s", err)
	}

	if (!isInstance && !shimProcess) || bootInstance || e.EngineConfig.GetInstanceJoin() {
		err := syscall.Exec(args[0], args, env)
		if err != nil {
			// We know the shell exists at this point, so let's inspect its architecture
			shell := e.EngineConfig.GetShell()
			if shell == "" {
				shell = defaultShell
			}
			elfArch, elfErr := machine.ArchFromElf(shell)
			if elfErr != nil && elfErr != machine.ErrUnknownArch {
				return fmt.Errorf("failed to open %s for inspection: %s", shell, elfErr)
			} else if elfErr == machine.ErrUnknownArch {
				elfArch = "unknown architecture"
			}
			if elfArch != runtime.GOARCH {
				return fmt.Errorf("image targets '%s', cannot run on '%s'", elfArch, runtime.GOARCH)
			}
			// Assume a missing shared library on ENOENT
			if err == syscall.ENOENT {
				return fmt.Errorf("exec %s failed: a shared library is likely missing in the image", args[0])
			}
			// Return the raw error as a last resort
			return fmt.Errorf("exec %s failed: %s", args[0], err)
		}
	}

	// Spawn and wait container process, signal handler
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: isInstance,
	}

	errChan := make(chan error, 1)
	statusChan := make(chan syscall.WaitStatus, 1)

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

	masterConn.Close()

	for {
		select {
		case s := <-signals:
			sylog.Debugf("Received signal %s", s.String())
			switch s {
			case syscall.SIGCHLD:
				for {
					var status syscall.WaitStatus

					wpid, err := syscall.Wait4(-1, &status, syscall.WNOHANG, nil)
					if wpid <= 0 || err != nil {
						// We break the loop since an error occurred
						break
					}

					if wpid == cmd.Process.Pid {
						e.stopFuseDrivers()
						statusChan <- status
					}
				}
			default:
				signal := s.(syscall.Signal)
				// EPERM and EINVAL are deliberately ignored because they can't be
				// returned in this context, this process is PID 1, so it has the
				// permissions to send signals to its childs and EINVAL would
				// mean to update the Go runtime or the kernel to something more
				// stable :)
				if isInstance {
					if err := syscall.Kill(-cmd.Process.Pid, signal); err == syscall.ESRCH {
						sylog.Debugf("No child process, exiting ...")
						os.Exit(128 + int(signal))
					}
				} else if e.EngineConfig.GetSignalPropagation() {
					if err := syscall.Kill(cmd.Process.Pid, signal); err == syscall.ESRCH {
						sylog.Debugf("No child process, exiting ...")
						os.Exit(128 + int(signal))
					}
				}
			}
		case err := <-errChan:
			if e, ok := err.(*exec.ExitError); ok {
				status, ok := e.Sys().(syscall.WaitStatus)
				if !ok {
					return fmt.Errorf("command exit with error: %s", err)
				}
				statusChan <- status
			} else if e, ok := err.(*os.SyscallError); ok {
				// handle possible race with Wait4 call above by ignoring ECHILD
				// error because child process was already catched
				if e.Err.(syscall.Errno) != syscall.ECHILD {
					sylog.Fatalf("error while waiting container process: %s", e.Error())
				}
			}
			if !isInstance {
				if len(statusChan) > 0 {
					status := <-statusChan
					if status.Signaled() {
						os.Exit(128 + int(status.Signal()))
					}
					os.Exit(status.ExitStatus())
				} else if err == nil {
					os.Exit(0)
				}
				sylog.Fatalf("command exited with unknown error: %s", err)
			}
		}
	}
}

// PostStartProcess is called from master after successful
// execution of the container process. It will write instance
// state/config files (if any).
//
// Additional privileges may be gained when running
// in suid flow. However, when a user namespace is requested and it is not
// a hybrid workflow (e.g. fakeroot), then there is no privileged saved uid
// and thus no additional privileges can be gained.
//
// Here, however, singularity engine does not escalate privileges.
func (e *EngineOperations) PostStartProcess(ctx context.Context, pid int) error {
	sylog.Debugf("Post start process")

	callbackType := (singularitycallback.PostStartProcess)(nil)
	callbacks, err := plugin.LoadCallbacks(callbackType)
	if err != nil {
		return fmt.Errorf("while loading plugins callbacks '%T': %s", callbackType, err)
	}
	for _, cb := range callbacks {
		if err := cb.(singularitycallback.PostStartProcess)(e.CommonConfig, pid); err != nil {
			return err
		}
	}

	if e.EngineConfig.GetInstance() {
		name := e.CommonConfig.ContainerID

		if err := os.Chdir("/"); err != nil {
			return fmt.Errorf("failed to change directory to /: %s", err)
		}

		file, err := instance.Add(name, instance.SingSubDir)
		if err != nil {
			return err
		}

		pw, err := user.CurrentOriginal()
		if err != nil {
			return err
		}
		file.User = pw.Name
		file.Pid = pid
		file.PPid = os.Getpid()
		file.Image = e.EngineConfig.GetImage()

		ip, err := e.getIP()
		if err != nil {
			sylog.Warningf("Could not get ip for %s: %s", pw.Name, err)
		}
		file.IP = ip

		// by default we add all namespaces except the user namespace which
		// is added conditionally. This delegates checks to the C starter code
		// which will determine if a namespace needs to be joined by
		// comparing namespace inodes
		path := fmt.Sprintf("/proc/%d/ns", pid)
		namespaces := []struct {
			nstype string
			ns     specs.LinuxNamespaceType
		}{
			{"pid", specs.PIDNamespace},
			{"uts", specs.UTSNamespace},
			{"ipc", specs.IPCNamespace},
			{"mnt", specs.MountNamespace},
			{"cgroup", specs.CgroupNamespace},
			{"net", specs.NetworkNamespace},
		}
		for _, n := range namespaces {
			nspath := filepath.Join(path, n.nstype)
			e.EngineConfig.OciConfig.AddOrReplaceLinuxNamespace(string(n.ns), nspath)
		}
		for _, ns := range e.EngineConfig.OciConfig.Linux.Namespaces {
			if ns.Type == specs.UserNamespace {
				nspath := filepath.Join(path, "user")
				e.EngineConfig.OciConfig.AddOrReplaceLinuxNamespace(specs.UserNamespace, nspath)
				file.UserNs = true
				break
			}
		}

		// grab configuration to store in instance file
		file.Config, err = json.Marshal(e.CommonConfig)
		if err != nil {
			return err
		}

		err = file.Update()

		// send SIGUSR1 to the parent process in order to tell it
		// to detach container process and run as instance.
		// Sleep a bit in case child would exit
		time.Sleep(100 * time.Millisecond)
		if err := syscall.Kill(os.Getppid(), syscall.SIGUSR1); err != nil {
			return err
		}

		return err
	}
	return nil
}

func (e *EngineOperations) setPathEnv() {
	env := e.EngineConfig.OciConfig.Process.Env
	for _, keyval := range env {
		if strings.HasPrefix(keyval, "PATH=") {
			os.Setenv("PATH", keyval[5:])
			break
		}
	}
}

func (e *EngineOperations) checkExec() error {
	shell := e.EngineConfig.GetShell()

	if shell == "" {
		shell = defaultShell
	}

	// Make sure the shell exists
	if _, err := os.Stat(shell); os.IsNotExist(err) {
		return fmt.Errorf("shell %s doesn't exist in container", shell)
	}

	args := e.EngineConfig.OciConfig.Process.Args
	env := e.EngineConfig.OciConfig.Process.Env

	// match old behavior of searching path
	oldPath := os.Getenv("PATH")
	defer func() {
		os.Setenv("PATH", oldPath)
		e.EngineConfig.OciConfig.Process.Args = args
		e.EngineConfig.OciConfig.Process.Env = env
	}()

	e.setPathEnv()

	// If args[0] is an absolute path, exec.LookPath() looks for
	// this file directly instead of within PATH
	if _, err := exec.LookPath(args[0]); err == nil {
		return nil
	}

	// If args[0] isn't executable (either via PATH or absolute path),
	// look for alternative approaches to handling it
	switch args[0] {
	case "/.singularity.d/actions/exec":
		if p, err := exec.LookPath("/.exec"); err == nil {
			args[0] = p
			return nil
		}
		if p, err := exec.LookPath(args[1]); err == nil {
			sylog.Warningf("container does not have %s, calling %s directly", args[0], args[1])
			args[1] = p
			args = args[1:]
			return nil
		}
		return fmt.Errorf("no executable %s found", args[1])
	case "/.singularity.d/actions/shell":
		if p, err := exec.LookPath("/.shell"); err == nil {
			args[0] = p
			return nil
		}
		if p, err := exec.LookPath(shell); err == nil {
			sylog.Warningf("container does not have %s, calling %s directly", args[0], shell)
			args[0] = p
			return nil
		}
		return fmt.Errorf("no %s found inside container", shell)
	case "/.singularity.d/actions/run":
		if p, err := exec.LookPath("/.run"); err == nil {
			args[0] = p
			return nil
		}
		if p, err := exec.LookPath("/singularity"); err == nil {
			args[0] = p
			return nil
		}
		return fmt.Errorf("no run driver found inside container")
	case "/.singularity.d/actions/start":
		if _, err := exec.LookPath(shell); err != nil {
			return fmt.Errorf("no %s found inside container, can't run instance", shell)
		}
		args = []string{shell, "-c", `echo "instance start script not found"`}
		return nil
	case "/.singularity.d/actions/test":
		if p, err := exec.LookPath("/.test"); err == nil {
			args[0] = p
			return nil
		}
		return fmt.Errorf("no test driver found inside container")
	}

	return fmt.Errorf("no %s found inside container", args[0])
}

// runFuseDrivers execute FUSE drivers and returns the list of FUSE process ID.
func (e *EngineOperations) runFuseDrivers(fromContainer bool, usernsFd int) error {
	// set PATH for the command
	oldpath := os.Getenv("PATH")
	defer func() {
		os.Setenv("PATH", oldpath)
	}()

	if fromContainer {
		e.setPathEnv()
	} else {
		os.Setenv("PATH", env.DefaultPath)
	}

	for _, fd := range e.EngineConfig.GetUnixSocketPair() {
		if fd >= 0 {
			unix.Close(fd)
		}
	}

	var usernsFh *os.File

	if usernsFd >= 0 {
		usernsFh = os.NewFile(uintptr(usernsFd), "/proc/self/ns/user")
		if usernsFh == nil {
			// this should never happen
			return errors.New("cannot map /proc/self/ns/user file descriptor to a file handle")
		}
		defer usernsFh.Close()
	}

	fuseMounts := e.EngineConfig.GetFuseMount()
	for i := range fuseMounts {
		if fromContainer != fuseMounts[i].FromContainer {
			syscall.Close(fuseMounts[i].Fd)
			continue
		}

		mnt := fuseMounts[i].MountPoint
		program := fuseMounts[i].Program
		fd := fuseMounts[i].Fd

		sylog.Debugf("Running FUSE driver for %s as %v, fd %d", mnt, program, fd)

		fh := os.NewFile(uintptr(fd), "/dev/fuse")
		if fh == nil {
			// this should never happen
			return errors.New("cannot map /dev/fuse file descriptor to a file handle")
		}
		// the master process does not need this file descriptor after
		// running the program, make sure it gets closed; ignore any
		// errors that happen here
		defer fh.Close()

		// as we pass file handle as first element in ExtraFiles
		// the fuse file descriptor becomes 3 for the FUSE program
		args := append(program, "/dev/fd/3")

		// add -f to run FUSE in foreground mode
		if !fuseMounts[i].Daemon {
			args = append(args, "-f")
		}

		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout

		// Add the /dev/fuse file descriptor to the list of file
		// descriptors to be passed to the new process.
		// The Go library will set things up so that stdin, stdout
		// and stderr are 0, 1, and 2, so the first element of
		// ExtraFiles gets 3
		cmd.ExtraFiles = make([]*os.File, 1)
		cmd.ExtraFiles[0] = fh

		// Add /proc/<container_pid>/ns/user file descriptor for nsenter
		// so it could join the container user namespace by using /dev/fd/4
		if usernsFh != nil {
			cmd.ExtraFiles = append(cmd.ExtraFiles, usernsFh)
		}

		if fuseMounts[i].Daemon {
			if err := cmd.Run(); err != nil {
				cmdline := strings.Join(args, " ")
				return fmt.Errorf("could not start program %s: %s", cmdline, err)
			}
		} else {
			if err := cmd.Start(); err != nil {
				cmdline := strings.Join(args, " ")
				return fmt.Errorf("could not start program %s: %s", cmdline, err)
			}
			fuseMounts[i].Cmd = cmd
		}
	}

	return nil
}

// stopFuseDrivers notifies FUSE drivers running in foreground mode
// with a SIGTERM signal.
func (e *EngineOperations) stopFuseDrivers() {
	for _, fuseMount := range e.EngineConfig.GetFuseMount() {
		if fuseMount.Cmd != nil {
			cmd := fuseMount.Cmd
			if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
				sylog.Warningf("Can not send SIGTERM to FUSE process: %s", err)
				continue
			}
			mnt := fuseMount.MountPoint
			_, err := cmd.Process.Wait()
			if err != nil {
				sylog.Warningf("FUSE process for mount point %s terminated with error: %s", mnt, err)
			} else {
				sylog.Debugf("FUSE process for mount point %s terminated", mnt)
			}
		}
	}
}

func (e *EngineOperations) getIP() (string, error) {
	if e.EngineConfig.Network == nil {
		return "", nil
	}

	net := strings.Split(e.EngineConfig.GetNetwork(), ",")

	ip, err := e.EngineConfig.Network.GetNetworkIP(net[0], "4")
	if err == nil {
		return ip.String(), nil
	}
	sylog.Warningf("Could not get ipv4 %s", err)

	ip, err = e.EngineConfig.Network.GetNetworkIP(net[0], "6")
	if err == nil {
		return ip.String(), nil
	}
	sylog.Warningf("Could not get ipv6 %s", err)

	return "", errors.New("could not get ip")
}
