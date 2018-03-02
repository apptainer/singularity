/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package main

/*
#include "cgo_scontainer.c"
*/
// #cgo CFLAGS: -I../c
import "C"

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"log"
	"loop"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"path"
	"rpc/client"
	"strings"
	"syscall"
	"unsafe"
)

func bool2int(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

func main() {
	var stage = flag.Int("stage", 0, "run stage 1 or 2")
	var socket = flag.Int("socket", 0, "socket process communication descriptor")
	var rpcfd = flag.Int("rpc", 0, "rpc communication descriptor")

	flag.Parse()

	if flag.NFlag() < 2 {
		flag.Usage()
		os.Exit(1)
	}

	if *stage == 0 || *socket == 0 {
		err := errors.New("Bad arguments\n\n")
		fmt.Print(err)

		flag.Usage()
		os.Exit(1)
	}

	cconf := C.cconf
	var spec specs.Spec

	/* get json configuration */
	jstr := C.GoStringN(C.json_conf, C.int(cconf.jsonConfSize))
	C.free(unsafe.Pointer(C.json_conf))

	decoder := json.NewDecoder(strings.NewReader(jstr))
	err := decoder.Decode(&spec)
	if err != nil {
		log.Fatalln("read json configuration failed")
	}

	comm := os.NewFile(uintptr(*socket), "comm")

	if *stage == 1 {
		cconf.isInstance = C.uchar(bool2int(false))
		cconf.noNewPrivs = C.uchar(bool2int(spec.Process.NoNewPrivileges))

		cconf.uidMapping.containerID = C.uid_t(os.Getuid())
		cconf.uidMapping.hostID = C.uid_t(os.Getuid())
		cconf.uidMapping.size = 1
		cconf.gidMapping.containerID = C.gid_t(os.Getgid())
		cconf.gidMapping.hostID = C.gid_t(os.Getgid())
		cconf.gidMapping.size = 1

		for _, namespace := range spec.Linux.Namespaces {
			switch namespace.Type {
			case specs.UserNamespace:
				cconf.userNS = C.uchar(bool2int(true))
				cconf.nsFlags |= syscall.CLONE_NEWUSER
			case specs.IPCNamespace:
				cconf.nsFlags |= syscall.CLONE_NEWIPC
			case specs.UTSNamespace:
				cconf.nsFlags |= syscall.CLONE_NEWUTS
			case specs.PIDNamespace:
				cconf.nsFlags |= syscall.CLONE_NEWPID
			case specs.NetworkNamespace:
				cconf.nsFlags |= syscall.CLONE_NEWNET
			case specs.MountNamespace:
				cconf.nsFlags |= syscall.CLONE_NEWNS
			}
		}

		jsonConf, err := json.Marshal(spec)
		if err != nil {
			log.Fatalln("serialize json configuration failed")
		}

		cconf.jsonConfSize = C.uint(len(jsonConf))

		cconfPayload := C.GoBytes(unsafe.Pointer(&cconf), C.sizeof_struct_cConfig)
		if _, err := comm.Write(cconfPayload); err != nil {
			log.Fatalln("write C configuration failed")
		}
		if _, err := comm.Write(jsonConf); err != nil {
			log.Fatalln("write json configuration failed")
		}
	} else if *stage == 2 {
		/* wait childs process */
		rpcChild := make(chan os.Signal, 1)
		signal.Notify(rpcChild, syscall.SIGCHLD)

		rpcSocket := os.NewFile(uintptr(*rpcfd), "rpc")

		if C.child_stage2 == 0 {
			conn, err := net.FileConn(rpcSocket)
			rpcSocket.Close()
			if err != nil {
				log.Fatalln("communication error")
			}

			rpcOps := new(client.RpcOps)
			rpcOps.Client = rpc.NewClient(conn)

			_, err = rpcOps.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, "")
			if err != nil {
				log.Fatalln("mount / failed:", err)
			}

			st, err := os.Stat(spec.Root.Path)
			if err != nil {
				log.Fatalf("stat on %s failed\n", spec.Root.Path)
			}

			rootfs := spec.Root.Path

			if st.IsDir() == false && cconf.userNS == C.uchar(0) {
				info := new(loop.LoopInfo64)
				info.Offset = 31
				info.Flags = loop.FlagsAutoClear
				var number int
				number, err = rpcOps.LoopDevice(spec.Root.Path, os.O_RDONLY, *info)
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
			os.Exit(0)
		}

		/* seccomp setup goes here */

		code := 0
		rpcSocket.Close()

		var status syscall.WaitStatus
	sigloop:
		for {
			select {
			case _ = (<-rpcChild):
				/*
				 * waiting 2 childs signal there, since Linux can merge signals, we wait for all childs
				 * when first SIGCHLD received
				 */
				for {
					pid, err := syscall.Wait4(-1, &status, syscall.WNOHANG|syscall.WUNTRACED, nil)
					if err == syscall.ECHILD {
						/* no more childs */
						signal.Stop(rpcChild)
						close(rpcChild)
						break sigloop
					}
					if pid > 0 {
						code += status.ExitStatus()
					}
				}
			}
		}
		if code != 0 {
			log.Fatalln("Container setup failed")
		}

		/* force close on exec on socket file descriptor to distinguish an exec success and error */
		_, _, errsys := syscall.RawSyscall(syscall.SYS_FCNTL, uintptr(*socket), syscall.F_SETFD, syscall.FD_CLOEXEC)
		if errsys != 0 {
			log.Fatalln("set close-on-exec failed:", errsys)
		}

		if cconf.isInstance == C.uchar(0) {
			os.Setenv("PS1", "shell> ")
			args := spec.Process.Args
			err := syscall.Exec(args[0], args, os.Environ())
			if err != nil {
				log.Fatalln("exec failed:", err)
			}
		} /* else {
			err := syscall.Exec("/bin/sleep", []string{"/bin/sleep", "60"}, os.Environ())
			if err != nil {
				log.Fatalln("exec failed:", err)
			}
			os.Exit(1)
		}*/
	}
}
