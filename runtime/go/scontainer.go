/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package main

/*
#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/prctl.h>
#include <signal.h>
#include <string.h>
#include <errno.h>
#include <linux/securebits.h>
#include <linux/capability.h>
#include <sys/syscall.h>

#include "../c/wrapper.h"

// Support only 64 bits sets, since kernel 2.6.25
#define CAPSET_MAX          40

#ifdef _LINUX_CAPABILITY_VERSION_3
#  define LINUX_CAPABILITY_VERSION  _LINUX_CAPABILITY_VERSION_3
#elif defined(_LINUX_CAPABILITY_VERSION_2)
#  define LINUX_CAPABILITY_VERSION  _LINUX_CAPABILITY_VERSION_2
#else
#  error Linux 64 bits capability set not supported
#endif // _LINUX_CAPABILITY_VERSION_3


static int capget(cap_user_header_t hdrp, cap_user_data_t datap) {
    return syscall(__NR_capget, hdrp, datap);
}

static int capset(cap_user_header_t hdrp, const cap_user_data_t datap) {
    return syscall(__NR_capset, hdrp, datap);
}

char *json_conf = NULL;
struct cConfig cconf;
pid_t child_stage2 = 0;

//
// drop privileges here to restrain users to access sensitive
// resources in /proc/<pid> during container setup
//
__attribute__((constructor)) static void init(void) {
    uid_t uid = getuid();
    gid_t gid = getgid();
    struct __user_cap_header_struct header;
    struct __user_cap_data_struct data[2];
    int stage = strtoul(getenv("STAGE"), NULL, 10);
    int fd = strtoul(getenv("SOCKET"), NULL, 10);
    int ret;

    if ( prctl(PR_SET_PDEATHSIG, SIGKILL) < 0 ) {
        exit(1);
    }

    if ( (ret = read(fd, &cconf, sizeof(cconf))) != sizeof(cconf) ) {
        printf("read failed %d %d\n", ret, fd);
        exit(1);
    }

    json_conf = (char *)malloc(cconf.jsonConfSize);
    if ( json_conf == NULL ) {
        printf("memory allocation failed\n");
        exit(1);
    }

    if ( (ret = read(fd, json_conf, cconf.jsonConfSize)) != cconf.jsonConfSize ) {
        printf("read json configuration failed\n");
        exit(1);
    }

    if ( stage == 2 ) {
        child_stage2 = fork();
    }

    if ( child_stage2 < 0 ) {
        printf("Failed to spawn child\n");
        exit(1);
    }

    if ( cconf.userNS == 1 || cconf.isSuid == 0 ) {
        return;
    }

    header.version = LINUX_CAPABILITY_VERSION;
    header.pid = 0;

    if ( capget(&header, data) < 0 ) {
        printf("Failed to get processus capabilities");
        exit(1);
    }

    if ( child_stage2 > 0 ) {
        data[1].inheritable = (__u32)(cconf.capInheritable >> 32);
        data[0].inheritable = (__u32)(cconf.capInheritable & 0xFFFFFFFF);
        data[1].permitted = (__u32)(cconf.capPermitted >> 32);
        data[0].permitted = (__u32)(cconf.capPermitted & 0xFFFFFFFF);
        data[1].effective = (__u32)(cconf.capEffective >> 32);
        data[0].effective = (__u32)(cconf.capEffective & 0xFFFFFFFF);
    } else {
        data[1].inheritable = data[1].permitted = data[1].effective = 0;
        data[0].inheritable = data[0].permitted = data[0].effective = 0;
        cconf.capBounding = 0;
        cconf.capAmbient = 0;
    }

    if ( prctl(PR_SET_SECUREBITS, SECBIT_NO_SETUID_FIXUP|SECBIT_NO_SETUID_FIXUP_LOCKED) < 0 ) {
        printf("securebits: %s\n", strerror(errno));
        exit(1);
    }

    if ( setresgid(gid, gid, gid) < 0 ) {
        printf("error gid\n");
        exit(1);
    }
    if ( setresuid(uid, uid, uid) < 0 ) {
        printf("error uid\n");
        exit(1);
    }

    if ( prctl(PR_SET_PDEATHSIG, SIGKILL) < 0 ) {
        exit(1);
    }

    int last_cap;
    for ( last_cap = CAPSET_MAX; ; last_cap-- ) {
        if ( prctl(PR_CAPBSET_READ, last_cap) > 0 || last_cap == 0 ) {
            break;
        }
    }

    int caps_index;
    for ( caps_index = 0; caps_index <= last_cap; caps_index++ ) {
        if ( !(cconf.capBounding & (1ULL << caps_index)) ) {
            if ( prctl(PR_CAPBSET_DROP, caps_index) < 0 ) {
                printf("Failed to drop bounding capabilities set: %s\n", strerror(errno));
                exit(1);
            }
        }
    }

#ifdef PR_SET_NO_NEW_PRIVS
    if ( cconf.noNewPrivs ) {
        if ( prctl(PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0) < 0 ) {
            printf("Failed to set no new privs flag: %s", strerror(errno));
            exit(1);
        }
    }
#endif

    if ( capset(&header, data) < 0 ) {
        printf("Failed to set processus capabilities");
        exit(1);
    }

#ifdef PR_CAP_AMBIENT
    // set ambient capabilities if supported
    int i;
    for (i = 0; i <= CAPSET_MAX; i++ ) {
        if ( (cconf.capAmbient & (1ULL << i)) ) {
            if ( prctl(PR_CAP_AMBIENT, PR_CAP_AMBIENT_RAISE, i, 0, 0) < 0 ) {
                printf("Failed to set ambient capability: %s\n", strerror(errno));
                exit(1);
            }
        }
    }
#endif
}
*/
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
