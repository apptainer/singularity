// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package imgbuild

import (
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

	if st.IsDir() == false {
		return fmt.Errorf("%s is not a directory", rootfs)
	}

	sessionPath, err := filepath.EvalSymlinks(buildcfg.SESSIONDIR)
	if err != nil {
		return fmt.Errorf("failed to resolved session directory %s: %s", buildcfg.SESSIONDIR, err)
	}

	sylog.Debugf("Mounting image directory %s\n", rootfs)
	_, err = rpcOps.Mount(rootfs, sessionPath, "", syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_NODEV, "errors=remount-ro")
	if err != nil {
		return fmt.Errorf("failed to mount directory filesystem %s: %s", rootfs, err)
	}

	sylog.Debugf("Mounting proc at %s\n", filepath.Join(sessionPath, "proc"))
	_, err = rpcOps.Mount("/proc", filepath.Join(sessionPath, "proc"), "", syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_REC, "")
	if err != nil {
		return fmt.Errorf("mount proc failed: %s", err)
	}

	sylog.Debugf("Mounting sysfs at %s\n", filepath.Join(sessionPath, "sys"))
	_, err = rpcOps.Mount("sysfs", filepath.Join(sessionPath, "sys"), "sysfs", syscall.MS_NOSUID, "")
	if err != nil {
		return fmt.Errorf("mount sys failed: %s", err)
	}

	sylog.Debugf("Mounting dev at %s\n", filepath.Join(sessionPath, "dev"))
	_, err = rpcOps.Mount("/dev", filepath.Join(sessionPath, "dev"), "", syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_REC, "")
	if err != nil {
		return fmt.Errorf("mount /dev failed: %s", err)
	}

	sylog.Debugf("Mounting tmp at %s\n", filepath.Join(sessionPath, "tmp"))
	_, err = rpcOps.Mount("/tmp", filepath.Join(sessionPath, "tmp"), "", syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_REC, "")
	if err != nil {
		return fmt.Errorf("mount /tmp failed: %s", err)
	}

	sylog.Debugf("Mounting var/tmp at %s\n", filepath.Join(sessionPath, "var/tmp"))
	_, err = rpcOps.Mount("/var/tmp", filepath.Join(sessionPath, "var/tmp"), "", syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_REC, "")
	if err != nil {
		return fmt.Errorf("mount /var/tmp failed: %s", err)
	}

	sylog.Debugf("Mounting /etc/resolv.conf at %s\n", filepath.Join(sessionPath, "etc/resolv.conf"))
	_, err = rpcOps.Mount("/etc/resolv.conf", filepath.Join(sessionPath, "etc/resolv.conf"), "", syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_REC, "")
	if err != nil {
		return fmt.Errorf("mount /etc/resolv.conf failed: %s", err)
	}

	sylog.Debugf("Mounting /etc/hosts at %s\n", filepath.Join(sessionPath, "etc/hosts"))
	_, err = rpcOps.Mount("/etc/hosts", filepath.Join(sessionPath, "etc/hosts"), "", syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_REC, "")
	if err != nil {
		return fmt.Errorf("mount /etc/hosts failed: %s", err)
	}

	if engine.EngineConfig.RunSection("setup") && engine.EngineConfig.Recipe.BuildData.Setup != "" {
		// Run %setup script here
		setup := exec.Command("/bin/sh", "-cex", engine.EngineConfig.Recipe.BuildData.Setup)
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

	sylog.Debugf("Chdir into %s\n", sessionPath)
	err = syscall.Chdir(sessionPath)
	if err != nil {
		return fmt.Errorf("change directory failed: %s", err)
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
	// iterate through filetransfers
	for _, transfer := range engine.EngineConfig.Recipe.BuildData.Files {
		// sanity
		if transfer.Src == "" {
			sylog.Warningf("Attempt to copy file with no name...")
			continue
		}
		// dest = source if not specified
		if transfer.Dst == "" {
			transfer.Dst = transfer.Src
		}
		sylog.Infof("Copying %v to %v", transfer.Src, transfer.Dst)
		// copy each file into bundle rootfs
		transfer.Dst = filepath.Join(engine.EngineConfig.Rootfs(), transfer.Dst)
		copy := exec.Command("/bin/cp", "-fLr", transfer.Src, transfer.Dst)
		if err := copy.Run(); err != nil {
			return fmt.Errorf("While copying %v to %v: %v", transfer.Src, transfer.Dst, err)
		}
	}

	return nil
}
