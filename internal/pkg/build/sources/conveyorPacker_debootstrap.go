// Copyright (c) 2018-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/hpcng/singularity/internal/pkg/util/fs"
	"github.com/hpcng/singularity/pkg/build/types"
	"github.com/hpcng/singularity/pkg/sylog"
	"github.com/hpcng/singularity/pkg/util/namespaces"
)

// debootstrapArchs is a map of GO Archs to official Debian ports
// https://www.debian.org/ports/
var debootstrapArchs = map[string]string{
	"386":      "i386",
	"amd64":    "amd64",
	"arm":      "armhf",
	"arm64":    "arm64",
	"ppc64le":  "ppc64el",
	"mipsle":   "mipsel",
	"mips64le": "mips64el",
	"s390x":    "s390x",
}

// DebootstrapConveyorPacker holds stuff that needs to be packed into the bundle
type DebootstrapConveyorPacker struct {
	b         *types.Bundle
	mirrorurl string
	osversion string
	include   string
}

// prepareFakerootEnv prepares a build environment to
// make fakeroot working with debootstrap.
func (cp *DebootstrapConveyorPacker) prepareFakerootEnv(ctx context.Context) (func(), error) {
	truePath, err := exec.LookPath("true")
	if err != nil {
		return nil, fmt.Errorf("while searching true command: %s", err)
	}
	mountPath, err := exec.LookPath("mount")
	if err != nil {
		return nil, fmt.Errorf("while searching mount command: %s", err)
	}
	mknodPath, err := exec.LookPath("mknod")
	if err != nil {
		return nil, fmt.Errorf("while searching mknod command: %s", err)
	}

	procFsPath := "/proc/filesystems"

	devs := []string{
		"/dev/null",
		"/dev/random",
		"/dev/urandom",
		"/dev/zero",
	}

	devPath := filepath.Join(cp.b.RootfsPath, "dev")
	if err := os.Mkdir(devPath, 0755); err != nil {
		return nil, fmt.Errorf("while creating %s: %s", devPath, err)
	}

	innerCtx, cancel := context.WithCancel(ctx)

	umountFn := func() {
		cancel()

		syscall.Unmount(mountPath, syscall.MNT_DETACH)
		syscall.Unmount(mknodPath, syscall.MNT_DETACH)
		for _, d := range devs {
			path := filepath.Join(cp.b.RootfsPath, d)
			syscall.Unmount(path, syscall.MNT_DETACH)
		}
	}

	// bind /bin/true on top of mount/mknod command
	// so debootstrap wouldn't fail while preparing
	// chroot environment
	if err := syscall.Mount(truePath, mountPath, "", syscall.MS_BIND, ""); err != nil {
		return umountFn, fmt.Errorf("while mounting %s to %s: %s", truePath, mountPath, err)
	}
	if err := syscall.Mount(truePath, mknodPath, "", syscall.MS_BIND, ""); err != nil {
		return umountFn, fmt.Errorf("while mounting %s to %s: %s", truePath, mknodPath, err)
	}

	// very dirty workaround to address issue with makedev
	// package installation during ubuntu bootstrap, we watch
	// for the creation of $ROOTFS/sbin/MAKEDEV and truncate
	// the file to obtain an equivalent of /bin/true, for makedev
	// post-configuration package we also need to create at least
	// one /dev/ttyX file
	go func() {
		makedevPath := filepath.Join(cp.b.RootfsPath, "/sbin/MAKEDEV")
		for {
			select {
			case <-innerCtx.Done():
				break
			case <-time.After(100 * time.Millisecond):
				if _, err := os.Stat(makedevPath); err == nil {
					os.Truncate(makedevPath, 0)
					os.Create(filepath.Join(cp.b.RootfsPath, "/dev/tty1"))
					break
				}
			}
		}
	}()

	// debootstrap look at /proc/filesystems to check
	// if sysfs is present, we bind /dev/null on top
	// of /proc/filesystems to trick debootstrap to not
	// mount /sys
	if err := syscall.Mount("/dev/null", procFsPath, "", syscall.MS_BIND, ""); err != nil {
		return umountFn, fmt.Errorf("while mounting /dev/null to %s: %s", procFsPath, err)
	}

	// mount required block devices
	for _, p := range devs {
		rootfsPath := filepath.Join(cp.b.RootfsPath, p)
		if err := fs.Touch(rootfsPath); err != nil {
			return umountFn, fmt.Errorf("while creating %s: %s", rootfsPath, err)
		}
		if err := syscall.Mount(p, rootfsPath, "", syscall.MS_BIND, ""); err != nil {
			return umountFn, fmt.Errorf("while mounting %s to %s: %s", p, rootfsPath, err)
		}
	}

	return umountFn, nil
}

// Get downloads container information from the specified source
func (cp *DebootstrapConveyorPacker) Get(ctx context.Context, b *types.Bundle) (err error) {
	cp.b = b

	// check for debootstrap on system(script using "singularity_which" not sure about its importance)
	debootstrapPath, err := exec.LookPath("debootstrap")
	if err != nil {
		return fmt.Errorf("debootstrap is not in PATH... Perhaps 'apt-get install' it: %v", err)
	}

	if err = cp.getRecipeHeaderInfo(); err != nil {
		return err
	}

	if os.Getuid() != 0 {
		return fmt.Errorf("you must be root to build with debootstrap")
	}

	// Debian port arch values do not always match GOARCH values, so we need to look it up.
	debArch, ok := debootstrapArchs[runtime.GOARCH]
	if !ok {
		return fmt.Errorf("Debian arch not known for GOARCH %s", runtime.GOARCH)
	}

	insideUserNs, setgroupsAllowed := namespaces.IsInsideUserNamespace(os.Getpid())
	if insideUserNs && setgroupsAllowed {
		umountFn, err := cp.prepareFakerootEnv(ctx)
		if umountFn != nil {
			defer umountFn()
		}
		if err != nil {
			return fmt.Errorf("while preparing fakeroot build environment: %s", err)
		}
	}

	// run debootstrap command
	cmd := exec.Command(debootstrapPath, `--variant=minbase`, `--exclude=openssl,udev,debconf-i18n,e2fsprogs`, `--include=apt,`+cp.include, `--arch=`+debArch, cp.osversion, cp.b.RootfsPath, cp.mirrorurl)

	sylog.Debugf("\n\tDebootstrap Path: %s\n\tIncludes: apt(default),%s\n\tDetected Arch: %s\n\tOSVersion: %s\n\tMirrorURL: %s\n", debootstrapPath, cp.include, runtime.GOARCH, cp.osversion, cp.mirrorurl)

	// run debootstrap
	out, err := cmd.CombinedOutput()

	io.Copy(os.Stdout, bytes.NewReader(out))

	if err != nil {
		dumpLog := func(fn string) {
			if _, err := os.Stat(fn); os.IsNotExist(err) {
				return
			}

			fh, err := os.Open(fn)
			if err != nil {
				sylog.Debugf("Cannot open %s: %#v", fn, err)
				return
			}
			defer fh.Close()

			log, err := ioutil.ReadAll(fh)
			if err != nil {
				sylog.Debugf("Cannot read %s: %#v", fn, err)
				return
			}

			sylog.Debugf("Contents of %s:\n%s", fn, log)
		}

		dumpLog(filepath.Join(cp.b.RootfsPath, "debootstrap/debootstrap.log"))

		return fmt.Errorf("while debootstrapping: %v", err)
	}

	return nil
}

// Pack puts relevant objects in a Bundle!
func (cp *DebootstrapConveyorPacker) Pack(context.Context) (*types.Bundle, error) {

	//change root directory permissions to 0755
	if err := os.Chmod(cp.b.RootfsPath, 0755); err != nil {
		return nil, fmt.Errorf("while changing bundle rootfs perms: %v", err)
	}

	err := cp.insertBaseEnv(cp.b)
	if err != nil {
		return nil, fmt.Errorf("while inserting base environtment: %v", err)
	}

	err = cp.insertRunScript(cp.b)
	if err != nil {
		return nil, fmt.Errorf("while inserting runscript: %v", err)
	}

	return cp.b, nil
}

func (cp *DebootstrapConveyorPacker) getRecipeHeaderInfo() (err error) {
	var ok bool

	//get mirrorURL, OSVerison, and Includes components to definition
	cp.mirrorurl, ok = cp.b.Recipe.Header["mirrorurl"]
	if !ok {
		return fmt.Errorf("invalid debootstrap header, no mirrorurl specified")
	}

	cp.osversion, ok = cp.b.Recipe.Header["osversion"]
	if !ok {
		return fmt.Errorf("invalid debootstrap header, no osversion specified")
	}

	include := cp.b.Recipe.Header["include"]

	//check for include environment variable and add it to requires string
	include += ` ` + os.Getenv("INCLUDE")

	//trim leading and trailing whitespace
	include = strings.TrimSpace(include)

	//convert Requires string to comma separated list
	cp.include = strings.Replace(include, ` `, `,`, -1)

	return nil
}

func (cp *DebootstrapConveyorPacker) insertBaseEnv(b *types.Bundle) (err error) {
	if err = makeBaseEnv(b.RootfsPath); err != nil {
		return
	}
	return nil
}

func (cp *DebootstrapConveyorPacker) insertRunScript(b *types.Bundle) (err error) {
	f, err := os.Create(b.RootfsPath + "/.singularity.d/runscript")
	if err != nil {
		return
	}

	defer f.Close()

	_, err = f.WriteString("#!/bin/sh\n")
	if err != nil {
		return
	}

	if err != nil {
		return
	}

	f.Sync()

	err = os.Chmod(b.RootfsPath+"/.singularity.d/runscript", 0755)
	if err != nil {
		return
	}

	return nil
}

// CleanUp removes any tmpfs owned by the conveyorPacker on the filesystem
func (cp *DebootstrapConveyorPacker) CleanUp() {
	cp.b.Remove()
}
