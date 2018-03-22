package runtime

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"path"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/singularityware/singularity/internal/pkg/runtime/engine/singularity/config"
	"github.com/singularityware/singularity/internal/pkg/runtime/engine/singularity/rpc/client"
	"github.com/singularityware/singularity/pkg/util/loop"
)

func (engine *RuntimeEngine) CreateContainer(rpcConn net.Conn) error {
	if engine.RuntimeSpec.RuntimeName != config.Name {
		log.Fatalln("engineName configuration doesn't match runtime name")
	}
	rpcOps := &client.Rpc{rpc.NewClient(rpcConn), engine.RuntimeSpec.RuntimeName}
	if rpcOps.Client == nil {
		log.Fatalln("Failed to initialiaze RPC client")
	}

	_, err := rpcOps.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, "")
	if err != nil {
		log.Fatalln("mount / failed:", err)
	}

	st, err := os.Stat(engine.OciConfig.RuntimeOciSpec.Root.Path)
	if err != nil {
		log.Fatalf("stat on %s failed\n", engine.OciConfig.RuntimeOciSpec.Root.Path)
	}

	rootfs := engine.OciConfig.RuntimeOciSpec.Root.Path

	userNS := false

	for _, namespace := range engine.OciConfig.RuntimeOciSpec.Linux.Namespaces {
		switch namespace.Type {
		case specs.UserNamespace:
			userNS = true
		}
	}

	if st.IsDir() == false && !userNS {
		info := new(loop.LoopInfo64)
		info.Offset = 31
		info.Flags = loop.FlagsAutoClear
		var number int
		number, err = rpcOps.LoopDevice(rootfs, os.O_RDONLY, *info)
		if err != nil {
			fmt.Println(err)
		}
		path := fmt.Sprintf("/dev/loop%d", number)
		rootfs = "/tmp/testing"
		_, err = rpcOps.Mount(path, rootfs, "squashfs", syscall.MS_NOSUID|syscall.MS_RDONLY|syscall.MS_NODEV, "errors=remount-ro")
		if err != nil {
			fmt.Println("mount squashfs:", err)
		}
	}

	_, err = rpcOps.Mount("proc", path.Join(rootfs, "proc"), "proc", syscall.MS_NOSUID, "")
	if err != nil {
		log.Fatalln("mount proc failed:", err)
	}
	_, err = rpcOps.Mount("sysfs", path.Join(rootfs, "sys"), "sysfs", syscall.MS_NOSUID, "")
	if err != nil {
		log.Fatalln("mount sys failed:", err)
	}
	_, err = rpcOps.Mount("/dev", path.Join(rootfs, "dev"), "", syscall.MS_BIND|syscall.MS_NOSUID|syscall.MS_REC, "")
	if err != nil {
		log.Fatalln("mount dev failed:", err)
	}
	_, err = rpcOps.Mount("/etc/passwd", path.Join(rootfs, "etc/passwd"), "", syscall.MS_BIND, "")
	if err != nil {
		log.Fatalln("mount /etc/passwd failed:", err)
	}
	_, err = rpcOps.Mount("/etc/group", path.Join(rootfs, "etc/group"), "", syscall.MS_BIND, "")
	if err != nil {
		log.Fatalln("mount /etc/group failed:", err)
	}
	_, err = rpcOps.Mount(rootfs, "/mnt", "", syscall.MS_BIND|syscall.MS_REC, "")
	if err != nil {
		log.Fatalln("mount failed:", err)
	}
	err = syscall.Chdir("/mnt")
	if err != nil {
		log.Fatalln("change directory failed:", err)
	}
	_, err = rpcOps.Chroot("/mnt")
	if err != nil {
		log.Fatalln("chroot failed:", err)
	}
	err = syscall.Chdir("/")
	if err != nil {
		log.Fatalln("change directory failed:", err)
	}
	if err := rpcOps.Client.Close(); err != nil {
		log.Fatalln("Can't close connection with rpc server")
	}

	return nil
}
