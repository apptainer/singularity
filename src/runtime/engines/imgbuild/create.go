// Copyright (c) 2018, Sylabs Inc. All rights reserved.
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

	"github.com/singularityware/singularity/src/pkg/buildcfg"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/runtime/engines/singularity/rpc/client"
)

// CreateContainer creates a container
func (engine *EngineOperations) CreateContainer(pid int, rpcConn net.Conn) error {
	if engine.CommonConfig.EngineName != Name {
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

	sylog.Debugf("Mounting image directory %s\n", rootfs)
	_, err = rpcOps.Mount(rootfs, buildcfg.SESSIONDIR, "", syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_NODEV, "errors=remount-ro")
	if err != nil {
		return fmt.Errorf("failed to mount directory filesystem %s: %s", rootfs, err)
	}

	sylog.Debugf("Mounting proc at %s\n", filepath.Join(buildcfg.SESSIONDIR, "proc"))
	_, err = rpcOps.Mount("/proc", filepath.Join(buildcfg.SESSIONDIR, "proc"), "", syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_REC, "")
	if err != nil {
		return fmt.Errorf("mount proc failed: %s", err)
	}

	sylog.Debugf("Mounting sysfs at %s\n", filepath.Join(buildcfg.SESSIONDIR, "sys"))
	_, err = rpcOps.Mount("sysfs", filepath.Join(buildcfg.SESSIONDIR, "sys"), "sysfs", syscall.MS_NOSUID, "")
	if err != nil {
		return fmt.Errorf("mount sys failed: %s", err)
	}

	sylog.Debugf("Mounting dev at %s\n", filepath.Join(buildcfg.SESSIONDIR, "dev"))
	_, err = rpcOps.Mount("/dev", filepath.Join(buildcfg.SESSIONDIR, "dev"), "", syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_REC, "")
	if err != nil {
		return fmt.Errorf("mount /dev failed: %s", err)
	}

	sylog.Debugf("Mounting tmp at %s\n", filepath.Join(buildcfg.SESSIONDIR, "tmp"))
	_, err = rpcOps.Mount("/tmp", filepath.Join(buildcfg.SESSIONDIR, "tmp"), "", syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_REC, "")
	if err != nil {
		return fmt.Errorf("mount /tmp failed: %s", err)
	}

	sylog.Debugf("Mounting var/tmp at %s\n", filepath.Join(buildcfg.SESSIONDIR, "var/tmp"))
	_, err = rpcOps.Mount("/var/tmp", filepath.Join(buildcfg.SESSIONDIR, "var/tmp"), "", syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_REC, "")
	if err != nil {
		return fmt.Errorf("mount /var/tmp failed: %s", err)
	}

	sylog.Debugf("Mounting /etc/resolv.conf at %s\n", filepath.Join(buildcfg.SESSIONDIR, "etc/resolv.conf"))
	_, err = rpcOps.Mount("/etc/resolv.conf", filepath.Join(buildcfg.SESSIONDIR, "etc/resolv.conf"), "", syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_REC, "")
	if err != nil {
		return fmt.Errorf("mount /etc/resolv.conf failed: %s", err)
	}

	sylog.Debugf("Mounting /etc/hosts at %s\n", filepath.Join(buildcfg.SESSIONDIR, "etc/hosts"))
	_, err = rpcOps.Mount("/etc/hosts", filepath.Join(buildcfg.SESSIONDIR, "etc/hosts"), "", syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_REC, "")
	if err != nil {
		return fmt.Errorf("mount /etc/hosts failed: %s", err)
	}

	sylog.Debugf("Set RPC mount propagation flag to SLAVE")
	_, err = rpcOps.Mount("", "/", "", syscall.MS_SLAVE|syscall.MS_REC, "")
	if err != nil {
		return fmt.Errorf("mount /etc/hosts failed: %s", err)
	}

	// Run %setup script here
	setup := exec.Command("/bin/sh", "-c", engine.EngineConfig.Recipe.BuildData.Setup)
	setup.Env = engine.EngineConfig.OciConfig.Process.Env
	setup.Stdout = os.Stdout
	setup.Stderr = os.Stderr

	sylog.Infof("Running %%setup script\n")
	if err := setup.Start(); err != nil {
		sylog.Fatalf("failed to start %%setup proc: %v\n", err)
	}
	if err := setup.Wait(); err != nil {
		sylog.Fatalf("setup proc: %v\n", err)
	}
	sylog.Infof("Finished running %%setup script. exit status 0\n")

	sylog.Debugf("Chdir into %s\n", buildcfg.SESSIONDIR)
	err = syscall.Chdir(buildcfg.SESSIONDIR)
	if err != nil {
		return fmt.Errorf("change directory failed: %s", err)
	}

	sylog.Debugf("Chroot into %s\n", buildcfg.SESSIONDIR)
	_, err = rpcOps.Chroot(buildcfg.SESSIONDIR)
	if err != nil {
		return fmt.Errorf("chroot failed: %s", err)
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
