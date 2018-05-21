// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package runtime

/*
#include <unistd.h>
#include "lib/image/image.h"
#include "lib/util/config_parser.h"
*/
// #cgo CFLAGS: -I../../../c/lib
// #cgo LDFLAGS: -L../../../../../builddir/lib -lruntime -luuid
import "C"

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
	"path"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/singularityware/singularity/src/pkg/buildcfg"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/loop"
	runtimeconfig "github.com/singularityware/singularity/src/runtime/workflows/workflows/singularity/config"
	"github.com/singularityware/singularity/src/runtime/workflows/workflows/singularity/rpc/client"
)

// CreateContainer creates a container
func (engine *Engine) CreateContainer(rpcConn net.Conn) error {
	if engine.RuntimeSpec.RuntimeName != runtimeconfig.Name {
		return fmt.Errorf("engineName configuration doesn't match runtime name")
	}

	rpcOps := &client.RPC{
		Client: rpc.NewClient(rpcConn),
		Name:   engine.RuntimeSpec.RuntimeName,
	}
	if rpcOps.Client == nil {
		return fmt.Errorf("failed to initialiaze RPC client")
	}

	_, err := rpcOps.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, "")
	if err != nil {
		return fmt.Errorf("mount / failed: %s", err)
	}

	st, err := os.Stat(engine.OciConfig.RuntimeOciSpec.Root.Path)
	if err != nil {
		return fmt.Errorf("stat on %s failed", engine.OciConfig.RuntimeOciSpec.Root.Path)
	}

	rootfs := engine.OciConfig.RuntimeOciSpec.Root.Path

	userNS := false
	pidNS := false

	for _, namespace := range engine.OciConfig.RuntimeOciSpec.Linux.Namespaces {
		switch namespace.Type {
		case specs.UserNamespace:
			userNS = true
		case specs.PIDNamespace:
			pidNS = true
		}
	}

	C.singularity_config_init()

	imageObject := C.singularity_image_init(C.CString(rootfs), 0)

	info := new(loop.LoopInfo64)
	mountType := ""

	switch C.singularity_image_type(&imageObject) {
	case 1:
		mountType = "squashfs"
		info.Offset = uint64(C.uint(imageObject.offset))
		info.SizeLimit = uint64(C.uint(imageObject.size))
	case 2:
		mountType = "ext3"
		info.Offset = uint64(C.uint(imageObject.offset))
		info.SizeLimit = uint64(C.uint(imageObject.size))
	}

	if st.IsDir() == false && !userNS {
		var number int
		info.Flags = loop.FlagsAutoClear
		number, err = rpcOps.LoopDevice(rootfs, os.O_RDONLY, *info)
		if err != nil {
			return err
		}

		path := fmt.Sprintf("/dev/loop%d", number)
		sylog.Debugf("Mounting loop device %s\n", path)
		_, err = rpcOps.Mount(path, buildcfg.CONTAINER_FINALDIR, mountType, syscall.MS_NOSUID|syscall.MS_RDONLY|syscall.MS_NODEV, "errors=remount-ro")
		if err != nil {
			return fmt.Errorf("failed to mount %s filesystem: %s", mountType, err)
		}
	} else {
		sylog.Debugf("Mounting image directory %s\n", rootfs)
		_, err = rpcOps.Mount(rootfs, buildcfg.CONTAINER_FINALDIR, "", syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_RDONLY|syscall.MS_NODEV, "errors=remount-ro")
		if err != nil {
			return fmt.Errorf("failed to mount directory filesystem %s: %s", rootfs, err)
		}
	}

	if pidNS {
		sylog.Debugf("Mounting proc at %s\n", path.Join(buildcfg.CONTAINER_FINALDIR, "proc"))
		_, err = rpcOps.Mount("proc", path.Join(buildcfg.CONTAINER_FINALDIR, "proc"), "proc", syscall.MS_NOSUID, "")
		if err != nil {
			return fmt.Errorf("mount proc failed: %s", err)
		}
	} else {
		sylog.Debugf("Mounting proc at %s\n", path.Join(buildcfg.CONTAINER_FINALDIR, "proc"))
		_, err = rpcOps.Mount("/proc", path.Join(buildcfg.CONTAINER_FINALDIR, "proc"), "", syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_REC, "")
		if err != nil {
			return fmt.Errorf("mount proc failed: %s", err)
		}
	}
	if !userNS {
		sylog.Debugf("Mounting sysfs at %s\n", path.Join(buildcfg.CONTAINER_FINALDIR, "sys"))
		_, err = rpcOps.Mount("sysfs", path.Join(buildcfg.CONTAINER_FINALDIR, "sys"), "sysfs", syscall.MS_NOSUID, "")
		if err != nil {
			return fmt.Errorf("mount sys failed: %s", err)
		}
	} else {
		sylog.Debugf("Mounting sysfs at %s\n", path.Join(buildcfg.CONTAINER_FINALDIR, "sys"))
		_, err = rpcOps.Mount("/sys", path.Join(buildcfg.CONTAINER_FINALDIR, "sys"), "", syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_REC, "")
		if err != nil {
			return fmt.Errorf("mount sys failed: %s", err)
		}
	}

	sylog.Debugf("Mounting home at %s\n", path.Join(buildcfg.CONTAINER_FINALDIR, "home"))
	_, err = rpcOps.Mount("/home", path.Join(buildcfg.CONTAINER_FINALDIR, "home"), "", syscall.MS_BIND, "")
	if err != nil {
		return fmt.Errorf("mount /home failed: %s", err)
	}

	sylog.Debugf("Mounting dev at %s\n", path.Join(buildcfg.CONTAINER_FINALDIR, "dev"))
	_, err = rpcOps.Mount("/dev", path.Join(buildcfg.CONTAINER_FINALDIR, "dev"), "", syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_REC, "")
	if err != nil {
		return fmt.Errorf("mount /dev failed: %s", err)
	}

	sylog.Debugf("Mounting /etc/passwd at %s\n", path.Join(buildcfg.CONTAINER_FINALDIR, "etc/passwd"))
	_, err = rpcOps.Mount("/etc/passwd", path.Join(buildcfg.CONTAINER_FINALDIR, "etc/passwd"), "", syscall.MS_BIND, "")
	if err != nil {
		return fmt.Errorf("mount /etc/passwd failed: %s", err)
	}

	sylog.Debugf("Mounting /etc/group at %s\n", path.Join(buildcfg.CONTAINER_FINALDIR, "etc/group"))
	_, err = rpcOps.Mount("/etc/group", path.Join(buildcfg.CONTAINER_FINALDIR, "etc/group"), "", syscall.MS_BIND, "")
	if err != nil {
		return fmt.Errorf("mount /etc/group failed: %s", err)
	}

	sylog.Debugf("Mounting staging dir %s into final dir %s\n", buildcfg.CONTAINER_FINALDIR, buildcfg.SESSIONDIR)
	_, err = rpcOps.Mount(buildcfg.CONTAINER_FINALDIR, buildcfg.SESSIONDIR, "", syscall.MS_BIND|syscall.MS_REC, "")
	if err != nil {
		return fmt.Errorf("mount staging directory failed: %s", err)
	}

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
