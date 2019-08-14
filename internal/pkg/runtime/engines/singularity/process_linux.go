// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"debug/elf"
	"encoding/binary"
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
	"unsafe"

	"github.com/sylabs/singularity/internal/pkg/security"
	singularity "github.com/sylabs/singularity/pkg/runtime/engines/singularity/config"

	"github.com/sylabs/singularity/internal/pkg/util/user"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sylabs/singularity/internal/pkg/instance"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"golang.org/x/crypto/ssh/terminal"
)

const defaultShell = "/bin/sh"

// Convert an ELF architecture into a GOARCH-style string. This is not an
// exhaustive list, so there is a default for rare cases. Adapted from
// https://golang.org/src/cmd/internal/objfile/elf.go
func elfToGoArch(elfFile *elf.File) string {
	switch elfFile.Machine {
	case elf.EM_386:
		return "386"
	case elf.EM_X86_64:
		return "amd64"
	case elf.EM_ARM:
		return "arm"
	case elf.EM_AARCH64:
		return "arm64"
	case elf.EM_PPC64:
		if elfFile.ByteOrder == binary.LittleEndian {
			return "ppc64le"
		}
		return "ppc64"
	case elf.EM_S390:
		return "s390x"
	}
	return "UNKNOWN"
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
	oldpath := os.Getenv("PATH")
	defer func() {
		os.Setenv("PATH", oldpath)
		e.EngineConfig.OciConfig.Process.Args = args
		e.EngineConfig.OciConfig.Process.Env = env
	}()

	engine.setPathEnv()

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

func (engine *EngineOperations) runFuseDriver(name string, program []string, fd int) error {
	sylog.Debugf("Running FUSE driver for %s as %v, fd %d", name, program, fd)

	fh := os.NewFile(uintptr(fd), "fd-"+name)
	if fh == nil {
		// this should never happen
		return errors.New("cannot map /dev/fuse file descriptor to a file handle")
	}
	// the master process does not need this file descriptor after
	// running the program, make sure it gets closed; ignore any
	// errors that happen here
	defer fh.Close()

	// The assumption is that the plugin prepared "Program" in such
	// a way that it's missing the last parameter and that must
	// correspond to /dev/fd/N. Instead of making assumptions as to
	// how many and which file descriptors are open at this point,
	// simply assume that it's possible to map the existing file
	// descriptor to the name number in the new process.
	//
	// "newFd" should be the same as "fd", but do not assume that
	// either.
	newFd := fh.Fd()
	fdDevice := fmt.Sprintf("/dev/fd/%d", newFd)
	args := append(program, fdDevice)

	// set PATH for the command
	oldpath := os.Getenv("PATH")
	defer func() {
		os.Setenv("PATH", oldpath)
	}()
	engine.setPathEnv()

	cmd := exec.Command(args[0], args[1:]...)

	// Add the /dev/fuse file descriptor to the list of file
	// descriptors to be passed to the new process.
	//
	// ExtraFiles is an array of *os.File, with the position of each
	// entry determining the resulting file descriptor number. Since
	// we are passing /dev/fd/N above, place our file handle at
	// position N-3, so that it gets mapped to file descriptor N in
	// the new process (the Go library will set things up so that
	// stdin, stdout and stderr are 0, 1, and 2, so the first
	// element of ExtraFiles gets 3).
	cmd.ExtraFiles = make([]*os.File, newFd-3+1)
	cmd.ExtraFiles[newFd-3] = fh
	// The FUSE driver will get SIGQUIT if the parent dies.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGQUIT,
	}

	if err := cmd.Start(); err != nil {
		sylog.Debugf("cannot start program %v: %v\n", args, err)
		return err
	}

	return nil
}

// setupFuseDrivers runs the operations required by FUSE drivers before
// the user process starts
func setupFuseDrivers(engine *EngineOperations) error {
	// close file descriptors open for FUSE mount
	for _, name := range engine.EngineConfig.GetPluginFuseMounts() {
		var cfg struct {
			Fuse singularity.FuseInfo
		}
		if err := engine.EngineConfig.GetPluginConfig(name, &cfg); err != nil {
			return err
		}

		if err := engine.runFuseDriver(name, cfg.Fuse.Program, cfg.Fuse.DevFuseFd); err != nil {
			return err
		}

		syscall.Close(cfg.Fuse.DevFuseFd)
	}

	return nil
}

// preStartProcess does the final set up before starting the user's
// process.
func preStartProcess(engine *EngineOperations) error {
	// TODO(mem): most of the StartProcess method should be here, as
	// it's doing preparation for actually starting the user
	// process.
	//
	// For now it's limited to doing the final set up for FUSE
	// drivers
	if err := setupFuseDrivers(engine); err != nil {
		return err
	}

	return nil
}

// StartProcess starts the process
func (e *EngineOperations) StartProcess(masterConn net.Conn) error {
	if err := preStartProcess(e); err != nil {
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
		if err := syscall.Close(int(img.Fd)); err != nil {
			return fmt.Errorf("failed to close file descriptor for %s", img.Path)
		}
	}

	for _, fd := range e.EngineConfig.GetOpenFd() {
		if err := syscall.Close(fd); err != nil {
			return fmt.Errorf("aborting failed to close file descriptor: %s", err)
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
			self, errElf := elf.Open(shell)
			if errElf != nil {
				return fmt.Errorf("failed to open %s for inspection: %s", shell, errElf)
			}
			defer self.Close()
			if elfArch := elfToGoArch(self); elfArch != runtime.GOARCH {
				return fmt.Errorf("image targets %s, cannot run on %s", elfArch, runtime.GOARCH)
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
					var status syscall.WaitStatus

					wpid, err := syscall.Wait4(-1, &status, syscall.WNOHANG, nil)
					if wpid <= 0 || err != nil {
						// We break the loop since an error occurred
						break
					}

					if wpid == cmd.Process.Pid {
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

// PostStartProcess will execute code in master context after execution of container
// process, typically to write instance state/config files or execute post start OCI hook
func (e *EngineOperations) PostStartProcess(pid int) error {
	sylog.Debugf("Post start process")

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

		return file.Update()
	}
	return nil
}
