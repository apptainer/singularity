package runtime

/*
#include <unistd.h>
#include "image/image.h"
#include "util/config_parser.h"
*/
// #cgo CFLAGS: -I../../../c/lib
// #cgo LDFLAGS: -L../../../../../builddir/lib -lruntime -luuid
import "C"

import (
	"fmt"
	"log"
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

func (engine *RuntimeEngine) CreateContainer(rpcConn net.Conn) error {
	if engine.RuntimeSpec.RuntimeName != runtimeconfig.Name {
		log.Fatalln("engineName configuration doesn't match runtime name")
		return fmt.Errorf("engineName configuration doesn't match runtime name")
	}
	rpcOps := &client.Rpc{
		Client: rpc.NewClient(rpcConn),
		Name:   engine.RuntimeSpec.RuntimeName,
	}
	if rpcOps.Client == nil {
		log.Fatalln("Failed to initialiaze RPC client")
		return fmt.Errorf("Failed to initialiaze RPC client")
	}

	_, err := rpcOps.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, "")
	if err != nil {
		log.Fatalln("mount / failed:", err)
		return err
	}

	st, err := os.Stat(engine.OciConfig.RuntimeOciSpec.Root.Path)
	if err != nil {
		log.Fatalf("stat on %s failed\n", engine.OciConfig.RuntimeOciSpec.Root.Path)
		return err
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
			log.Fatalln(err)
			return err
		}

		path := fmt.Sprintf("/dev/loop%d", number)
		sylog.Debugf("Mounting loop device %s\n", path)

		_, err = rpcOps.Mount(path, buildcfg.CONTAINER_MOUNTDIR, mountType, syscall.MS_NOSUID|syscall.MS_RDONLY|syscall.MS_NODEV, "errors=remount-ro")
		if err != nil {
			log.Fatalf("Failed to mount %s filesystem: %s\n", mountType, err)
			return err
		}
	}

	if pidNS {
		sylog.Debugf("Mounting proc at %s\n", path.Join(buildcfg.CONTAINER_MOUNTDIR, "proc"))
		_, err = rpcOps.Mount("proc", path.Join(buildcfg.CONTAINER_MOUNTDIR, "proc"), "proc", syscall.MS_NOSUID, "")
		if err != nil {
			log.Fatalln("mount proc failed:", err)
			return err
		}
	} else {
		sylog.Debugf("Mounting proc at %s\n", path.Join(buildcfg.CONTAINER_MOUNTDIR, "proc"))
		_, err = rpcOps.Mount("/proc", path.Join(buildcfg.CONTAINER_MOUNTDIR, "proc"), "", syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_REC, "")
		if err != nil {
			log.Fatalln("mount proc failed:", err)
			return err
		}
	}
	if !userNS {
		sylog.Debugf("Mounting sysfs at %s\n", path.Join(buildcfg.CONTAINER_MOUNTDIR, "sys"))
		_, err = rpcOps.Mount("sysfs", path.Join(buildcfg.CONTAINER_MOUNTDIR, "sys"), "sysfs", syscall.MS_NOSUID, "")
		if err != nil {
			log.Fatalln("mount sys failed:", err)
			return err
		}
	} else {
		sylog.Debugf("Mounting sysfs at %s\n", path.Join(buildcfg.CONTAINER_MOUNTDIR, "sys"))
		_, err = rpcOps.Mount("/sys", path.Join(buildcfg.CONTAINER_MOUNTDIR, "sys"), "", syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_REC, "")
		if err != nil {
			log.Fatalln("mount sys failed:", err)
			return err
		}
	}

	sylog.Debugf("Mounting dev at %s\n", path.Join(buildcfg.CONTAINER_MOUNTDIR, "dev"))
	_, err = rpcOps.Mount("/dev", path.Join(buildcfg.CONTAINER_MOUNTDIR, "dev"), "", syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_REC, "")
	if err != nil {
		log.Fatalln("mount dev failed:", err)
		return err
	}

	sylog.Debugf("Mounting /etc/passwd at %s\n", path.Join(buildcfg.CONTAINER_MOUNTDIR, "etc/passwd"))
	_, err = rpcOps.Mount("/etc/passwd", path.Join(buildcfg.CONTAINER_MOUNTDIR, "etc/passwd"), "", syscall.MS_BIND, "")
	if err != nil {
		log.Fatalln("mount /etc/passwd failed:", err)
		return err
	}

	sylog.Debugf("Mounting /etc/group at %s\n", path.Join(buildcfg.CONTAINER_MOUNTDIR, "etc/group"))
	_, err = rpcOps.Mount("/etc/group", path.Join(buildcfg.CONTAINER_MOUNTDIR, "etc/group"), "", syscall.MS_BIND, "")
	if err != nil {
		log.Fatalln("mount /etc/group failed:", err)
		return err
	}

	sylog.Debugf("Mounting staging dir %s into final dir %s\n", buildcfg.CONTAINER_MOUNTDIR, buildcfg.SESSIONDIR)
	_, err = rpcOps.Mount(buildcfg.CONTAINER_MOUNTDIR, buildcfg.SESSIONDIR, "", syscall.MS_BIND|syscall.MS_REC, "")
	if err != nil {
		log.Fatalln("mount failed:", err)
		return err
	}

	sylog.Debugf("Chdir into %s\n", buildcfg.SESSIONDIR)
	err = syscall.Chdir(buildcfg.SESSIONDIR)
	if err != nil {
		log.Fatalln("change directory failed:", err)
		return err
	}

	sylog.Debugf("Chroot into %s\n", buildcfg.SESSIONDIR)
	_, err = rpcOps.Chroot(buildcfg.SESSIONDIR)
	if err != nil {
		log.Fatalln("chroot failed:", err)
		return err
	}

	sylog.Debugf("Chdir into / to avoid errors\n")
	err = syscall.Chdir("/")
	if err != nil {
		log.Fatalln("change directory failed:", err)
		return err
	}
	if err := rpcOps.Client.Close(); err != nil {
		log.Fatalln("Can't close connection with rpc server:", err)
		return err
	}

	return nil
}
