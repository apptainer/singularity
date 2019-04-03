// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package imgbuild

import (
	"bytes"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	imgbuildConfig "github.com/sylabs/singularity/internal/pkg/runtime/engines/imgbuild/config"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/singularity/rpc/client"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
)

// CreateContainer creates a container
func (engine *EngineOperations) CreateContainer(pid int, rpcConn net.Conn) error {
	if engine.CommonConfig.EngineName != imgbuildConfig.Name {
		return fmt.Errorf("engineName configuration doesn't match runtime name")
	}

	rpcOps := &client.RPC{
		Client: rpc.NewClient(rpcConn),
		Name:   engine.CommonConfig.EngineName,
	}
	if rpcOps.Client == nil {
		return fmt.Errorf("failed to initialiaze RPC client")
	}

	rootfs := engine.EngineConfig.Rootfs()

	st, err := os.Stat(rootfs)
	if err != nil {
		return fmt.Errorf("stat on %s failed", rootfs)
	}

	if !st.IsDir() {
		return fmt.Errorf("%s is not a directory", rootfs)
	}

	sessionPath, err := filepath.EvalSymlinks(buildcfg.SESSIONDIR)
	if err != nil {
		return fmt.Errorf("failed to resolved session directory %s: %s", buildcfg.SESSIONDIR, err)
	}

	// sensible mount point options to avoid accidental system settings override
	flags := uintptr(syscall.MS_BIND | syscall.MS_NOSUID | syscall.MS_NOEXEC | syscall.MS_NODEV | syscall.MS_RDONLY)

	sylog.Debugf("Mounting image directory %s\n", rootfs)
	_, err = rpcOps.Mount(rootfs, sessionPath, "", syscall.MS_BIND, "errors=remount-ro")
	if err != nil {
		return fmt.Errorf("failed to mount directory filesystem %s: %s", rootfs, err)
	}

	dest := filepath.Join(sessionPath, "tmp")
	sylog.Debugf("Mounting /tmp at %s\n", dest)
	_, err = rpcOps.Mount("/tmp", dest, "", syscall.MS_BIND, "")
	if err != nil {
		return fmt.Errorf("mount /tmp failed: %s", err)
	}

	dest = filepath.Join(sessionPath, "var", "tmp")
	sylog.Debugf("Mounting /var/tmp at %s\n", dest)
	_, err = rpcOps.Mount("/var/tmp", dest, "", syscall.MS_BIND, "")
	if err != nil {
		return fmt.Errorf("mount /var/tmp failed: %s", err)
	}

	// run setup/files sections here to allow injection of custom /etc/hosts or /etc/resolv.conf
	if engine.EngineConfig.RunSection("setup") && engine.EngineConfig.Recipe.BuildData.Setup.Script != "" {
		// Run %setup script here
		setup := exec.Command("/bin/sh", "-cex", engine.EngineConfig.Recipe.BuildData.Setup.Script)
		setup.Env = engine.EngineConfig.OciConfig.Process.Env
		setup.Stdout = os.Stdout
		setup.Stderr = os.Stderr

		sylog.Infof("Running setup scriptlet\n")
		if err := setup.Start(); err != nil {
			sylog.Fatalf("failed to start %%setup proc: %v\n", err)
		}
		if err := setup.Wait(); err != nil {
			sylog.Fatalf("setup proc: %v\n", err)
		}
	}

	if engine.EngineConfig.RunSection("files") {
		sylog.Debugf("Copying files from host")
		if err := engine.copyFiles(); err != nil {
			return fmt.Errorf("unable to copy files to container fs: %v", err)
		}
	}

	dest = filepath.Join(sessionPath, "proc")
	sylog.Debugf("Mounting /proc at %s\n", dest)
	_, err = rpcOps.Mount("/proc", dest, "", flags, "")
	if err != nil {
		return fmt.Errorf("mount proc failed: %s", err)
	}
	_, err = rpcOps.Mount("", dest, "", syscall.MS_REMOUNT|flags, "")
	if err != nil {
		return fmt.Errorf("remount proc failed: %s", err)
	}

	dest = filepath.Join(sessionPath, "sys")
	sylog.Debugf("Mounting /sys at %s\n", dest)
	_, err = rpcOps.Mount("/sys", dest, "", flags, "")
	if err != nil {
		return fmt.Errorf("mount sys failed: %s", err)
	}
	_, err = rpcOps.Mount("", dest, "", syscall.MS_REMOUNT|flags, "")
	if err != nil {
		return fmt.Errorf("remount sys failed: %s", err)
	}

	dest = filepath.Join(sessionPath, "dev")
	sylog.Debugf("Mounting /dev at %s\n", dest)
	_, err = rpcOps.Mount("/dev", dest, "", syscall.MS_BIND|syscall.MS_REC, "")
	if err != nil {
		return fmt.Errorf("mount /dev failed: %s", err)
	}

	dest = filepath.Join(sessionPath, "etc", "resolv.conf")
	sylog.Debugf("Mounting /etc/resolv.conf at %s\n", dest)
	_, err = rpcOps.Mount("/etc/resolv.conf", dest, "", flags, "")
	if err != nil {
		return fmt.Errorf("mount /etc/resolv.conf failed: %s", err)
	}
	_, err = rpcOps.Mount("", dest, "", syscall.MS_REMOUNT|flags, "")
	if err != nil {
		return fmt.Errorf("remount /etc/resolv.conf failed: %s", err)
	}

	dest = filepath.Join(sessionPath, "etc", "hosts")
	sylog.Debugf("Mounting /etc/hosts at %s\n", dest)
	_, err = rpcOps.Mount("/etc/hosts", dest, "", flags, "")
	if err != nil {
		return fmt.Errorf("mount /etc/hosts failed: %s", err)
	}
	_, err = rpcOps.Mount("", dest, "", syscall.MS_REMOUNT|flags, "")
	if err != nil {
		return fmt.Errorf("remount /etc/hosts failed: %s", err)
	}

	sylog.Debugf("Chdir into %s\n", sessionPath)
	err = syscall.Chdir(sessionPath)
	if err != nil {
		return fmt.Errorf("change directory failed: %s", err)
	}

	sylog.Debugf("Set RPC mount propagation flag to PRIVATE")
	_, err = rpcOps.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, "")
	if err != nil {
		return err
	}

	sylog.Debugf("Chroot into %s\n", buildcfg.SESSIONDIR)
	_, err = rpcOps.Chroot(buildcfg.SESSIONDIR, "pivot")
	if err != nil {
		sylog.Debugf("Fallback to move/chroot")
		_, err = rpcOps.Chroot(buildcfg.SESSIONDIR, "move")
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

func (engine *EngineOperations) copyFiles() error {
	var output, stderr bytes.Buffer
	files := types.Files{}
	for _, f := range engine.EngineConfig.Recipe.BuildData.Files {
		if f.Args == "" {
			files = f
		}
	}
	// iterate through filetransfers
	for _, transfer := range files.Files {
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
		transfer.Dst = filepath.Join(engine.EngineConfig.Rootfs(), transfer.Dst)
		sylog.Infof("Copying %v to %v", transfer.Src, transfer.Dst)
		copy := exec.Command("/bin/cp", "-fLr", transfer.Src, transfer.Dst)
		copy.Stdout = &output
		copy.Stderr = &stderr
		if err := copy.Run(); err != nil {
			return fmt.Errorf("while copying %v to %v: %v: %v", transfer.Src, transfer.Dst, err, stderr.String())
		}
	}

	return nil
}
