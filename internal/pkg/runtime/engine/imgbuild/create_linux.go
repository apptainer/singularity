// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package imgbuild

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/build/files"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	imgbuildConfig "github.com/sylabs/singularity/internal/pkg/runtime/engine/imgbuild/config"
	"github.com/sylabs/singularity/internal/pkg/runtime/engine/singularity/rpc/client"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
	"github.com/sylabs/singularity/pkg/util/namespaces"
)

// CreateContainer is called from master process to prepare container
// environment, e.g. perform mount operations, etc.
//
// Additional privileges required for setup may be gained when running
// in suid flow. However, when a user namespace is requested and it is not
// a hybrid workflow (e.g. fakeroot), then there is no privileged saved uid
// and thus no additional privileges can be gained.
//
// Specifically in imgbuild engine, no additional privileges are gained. Container
// setup (e.g. mount operations) where privileges may be required is performed
// by calling RPC server methods (see internal/app/starter/rpc_linux.go for details).
//
// Note that imgbuild engine is usually called by root user or by fakeroot engine, so technically this
// method may already be run with escalated privileges.
func (e *EngineOperations) CreateContainer(ctx context.Context, pid int, rpcConn net.Conn) error {
	if e.CommonConfig.EngineName != imgbuildConfig.Name {
		return fmt.Errorf("engineName configuration doesn't match runtime name")
	}

	rpcOps := &client.RPC{
		Client: rpc.NewClient(rpcConn),
		Name:   e.CommonConfig.EngineName,
	}
	if rpcOps.Client == nil {
		return fmt.Errorf("failed to initialiaze RPC client")
	}

	insideUserNs, setgroups := namespaces.IsInsideUserNamespace(os.Getpid())
	// if we are running inside a user namespace, at this stage we
	// are a root user in this user namespace, but if setgroups is
	// denied build may not work correctly, so warn user about that
	if insideUserNs && !setgroups {
		sylog.Warningf("Running inside a user namespace, but setgroups is denied, build may not work correctly")
	}

	rootfs := e.EngineConfig.RootfsPath

	st, err := os.Stat(rootfs)
	if err != nil {
		return fmt.Errorf("stat on %s failed: %v", rootfs, err)
	}

	if !st.IsDir() {
		return fmt.Errorf("%s is not a directory", rootfs)
	}

	sessionPath, err := filepath.EvalSymlinks(buildcfg.SESSIONDIR)
	if err != nil {
		return fmt.Errorf("failed to resolved session directory %s: %s", buildcfg.SESSIONDIR, err)
	}

	flags := uintptr(syscall.MS_NOSUID | syscall.MS_NOEXEC | syscall.MS_NODEV)
	tmpfsOpts := "mode=0755,size=2m"

	if err := rpcOps.Mount("tmpfs", sessionPath, "tmpfs", flags, tmpfsOpts); err != nil {
		return fmt.Errorf("failed to mount tmpfs filesystem on %s: %s", sessionPath, err)
	}

	sessionRootFs := filepath.Join(sessionPath, "rootfs")
	if err := os.Mkdir(sessionRootFs, 0755); err != nil {
		return fmt.Errorf("failed to create %s: %s", sessionRootFs, err)
	}

	// sensible mount point options to avoid accidental system settings override
	flags = uintptr(syscall.MS_BIND | syscall.MS_NOSUID | syscall.MS_NOEXEC | syscall.MS_NODEV | syscall.MS_RDONLY)
	if insideUserNs {
		flags = uintptr(syscall.MS_BIND | syscall.MS_REC)
	}

	sylog.Debugf("Mounting image directory %s\n", rootfs)
	if err := rpcOps.Mount(rootfs, sessionRootFs, "", syscall.MS_BIND, "errors=remount-ro"); err != nil {
		return fmt.Errorf("failed to mount directory filesystem %s: %s", rootfs, err)
	}

	dest := filepath.Join(sessionRootFs, "tmp")
	sylog.Debugf("Mounting /tmp at %s\n", dest)
	if err := rpcOps.Mount("/tmp", dest, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("mount /tmp failed: %s", err)
	}

	dest = filepath.Join(sessionRootFs, "var", "tmp")
	sylog.Debugf("Mounting /var/tmp at %s\n", dest)
	if err := rpcOps.Mount("/var/tmp", dest, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("mount /var/tmp failed: %s", err)
	}

	// run setup/files sections here to allow injection of custom /etc/hosts or /etc/resolv.conf
	if e.EngineConfig.RunSection("setup") && e.EngineConfig.Recipe.BuildData.Setup.Script != "" {
		// Run %setup script here
		e.runScriptSection("setup", e.EngineConfig.Recipe.BuildData.Setup, true)
	}

	if e.EngineConfig.RunSection("files") {
		sylog.Debugf("Copying files from host")
		if err := e.copyFiles(); err != nil {
			return fmt.Errorf("unable to copy files to container fs: %v", err)
		}
	}

	dest = filepath.Join(sessionRootFs, "proc")
	sylog.Debugf("Mounting /proc at %s\n", dest)
	if err := rpcOps.Mount("/proc", dest, "", flags, ""); err != nil {
		return fmt.Errorf("mount proc failed: %s", err)
	}
	if !insideUserNs {
		if err := rpcOps.Mount("", dest, "", syscall.MS_REMOUNT|flags, ""); err != nil {
			return fmt.Errorf("remount proc failed: %s", err)
		}
	}

	dest = filepath.Join(sessionRootFs, "sys")
	sylog.Debugf("Mounting /sys at %s\n", dest)
	if err := rpcOps.Mount("/sys", dest, "", flags, ""); err != nil {
		return fmt.Errorf("mount sys failed: %s", err)
	}
	if !insideUserNs {
		if err := rpcOps.Mount("", dest, "", syscall.MS_REMOUNT|flags, ""); err != nil {
			return fmt.Errorf("remount sys failed: %s", err)
		}
	}

	dest = filepath.Join(sessionRootFs, "dev")
	sylog.Debugf("Mounting /dev at %s\n", dest)
	if err := rpcOps.Mount("/dev", dest, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("mount /dev failed: %s", err)
	}

	// copy /etc/resolv.conf to tmpfs and bind copy into container
	sessionResolv, err := stageFile("/etc/resolv.conf", sessionPath)
	if err != nil {
		return err
	}

	dest = filepath.Join(sessionRootFs, "etc", "resolv.conf")
	sylog.Debugf("Mounting %s at %s\n", sessionResolv, dest)
	if err := rpcOps.Mount(sessionResolv, dest, "", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("mount %s failed: %s", sessionResolv, err)
	}

	// copy /etc/hosts to tmpfs and bind copy into container
	sessionHosts, err := stageFile("/etc/hosts", sessionPath)
	if err != nil {
		return err
	}

	dest = filepath.Join(sessionRootFs, "etc", "hosts")
	sylog.Debugf("Mounting %s at %s\n", sessionHosts, dest)
	if err := rpcOps.Mount(sessionHosts, dest, "", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("mount %s failed: %s", sessionHosts, err)
	}

	sylog.Debugf("Chdir into %s\n", sessionRootFs)
	err = syscall.Chdir(sessionRootFs)
	if err != nil {
		return fmt.Errorf("change directory failed: %s", err)
	}

	sylog.Debugf("Set RPC mount propagation flag to PRIVATE")
	if err := rpcOps.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		return err
	}

	sylog.Debugf("Chroot into %s\n", sessionRootFs)
	_, err = rpcOps.Chroot(sessionRootFs, "pivot")
	if err != nil {
		sylog.Debugf("Fallback to move/chroot")
		_, err = rpcOps.Chroot(sessionRootFs, "move")
		if err != nil {
			return fmt.Errorf("chroot failed: %s", err)
		}
	}

	sylog.Debugf("Chdir into / to avoid errors\n")
	err = syscall.Chdir("/")
	if err != nil {
		return fmt.Errorf("change directory failed: %s", err)
	}
	if err := rpcOps.Client.Close(); err != nil {
		return fmt.Errorf("can't close connection with rpc server: %s", err)
	}

	return nil
}

func stageFile(source, destDir string) (string, error) {
	sessionFile := filepath.Join(destDir, filepath.Base(source))
	stageFile, err := os.Create(sessionFile)
	if err != nil {
		return "", fmt.Errorf("failed to create staging %s file: %s", sessionFile, err)
	}
	defer stageFile.Close()

	content, err := ioutil.ReadFile(source)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %s", source, err)
	}
	if _, err := stageFile.Write(content); err != nil {
		return "", fmt.Errorf("failed to copy %s content to %s: %s", source, sessionFile, err)
	}

	return sessionFile, nil
}

func (e *EngineOperations) copyFiles() error {
	filesSection := types.Files{}
	for _, f := range e.EngineConfig.Recipe.BuildData.Files {
		if f.Args == "" {
			filesSection = f
		}
	}
	// iterate through filetransfers
	for _, transfer := range filesSection.Files {
		// sanity
		if transfer.Src == "" {
			sylog.Warningf("Attempt to copy file with no name, skipping.")
			continue
		}
		// dest = source if not specified
		if transfer.Dst == "" {
			transfer.Dst = transfer.Src
		}
		// copy each file into bundle rootfs
		transfer.Dst = files.AddPrefix(e.EngineConfig.RootfsPath, transfer.Dst)
		sylog.Infof("Copying %v to %v", transfer.Src, transfer.Dst)
		if err := files.Copy(transfer.Src, transfer.Dst); err != nil {
			return err
		}
	}

	return nil
}

// runScriptSection executes the provided script by piping the
// script to /bin/sh command.
func (e *EngineOperations) runScriptSection(name string, s types.Script, setEnv bool) {
	args := []string{"-ex"}
	// trim potential trailing comment from args and append to args list
	args = append(args, strings.Fields(strings.Split(s.Args, "#")[0])...)

	envs := []string{}
	if setEnv {
		envs = e.EngineConfig.OciConfig.Process.Env
	}

	sylog.Infof("Running %s scriptlet\n", name)

	var b bytes.Buffer
	b.WriteString(s.Script)

	cmd := exec.Command("/bin/sh", args...)
	cmd.Env = envs
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = &b

	if err := cmd.Run(); err != nil {
		sylog.Fatalf("failed to execute %%%s proc: %v\n", name, err)
	}
}
