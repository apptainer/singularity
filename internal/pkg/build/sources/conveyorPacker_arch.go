// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
	"github.com/sylabs/singularity/pkg/util/namespaces"
)

const (
	pacmanConfURL = "https://git.archlinux.org/svntogit/packages.git/plain/trunk/pacman.conf?h=packages/pacman"
)

// `pacstrap' installs the whole "base" package group, unless told otherwise.
// baseToSkip are "base" packages that won't be normally needed on a
// container system. $BASE_TO_INST are "base" packages not present in
// baseToSkip. The list of packages included in "base" group may (it surely
// will, one day) change in future, so baseToSkip will need an update from
// time to time. Here I'm referring to `base' group contents as of 30.08.2016.
var baseToSkip = map[string]bool{
	"cryptsetup":         true,
	"device-mapper":      true,
	"dhcpcd":             true,
	"iproute2":           true,
	"jfsutils":           true,
	"linux":              true,
	"lvm2":               true,
	"man-db":             true,
	"man-pages":          true,
	"mdadm":              true,
	"nano":               true,
	"netctl":             true,
	"openresolv":         true,
	"pciutils":           true,
	"pcmciautils":        true,
	"reiserfsprogs":      true,
	"s-nail":             true,
	"systemd-sysvcompat": true,
	"usbutils":           true,
	"vi":                 true,
	"xfsprogs":           true,
}

// ArchConveyorPacker only needs to hold the conveyor to have the needed data to pack
type ArchConveyorPacker struct {
	b *types.Bundle
}

// Get just stores the source
func (cp *ArchConveyorPacker) Get(ctx context.Context, b *types.Bundle) (err error) {
	cp.b = b

	//check for pacstrap on system
	pacstrapPath, err := exec.LookPath("pacstrap")
	if err != nil {
		return fmt.Errorf("pacstrap is not in PATH: %v", err)
	}

	//make sure architecture is supported
	if arch := runtime.GOARCH; arch != `amd64` {
		return fmt.Errorf("%v architecture is not supported", arch)
	}

	instList, err := getPacmanBaseList()
	if err != nil {
		return fmt.Errorf("while generating the installation list: %v", err)
	}

	pacConf, err := cp.getPacConf(pacmanConfURL)
	if err != nil {
		return fmt.Errorf("while getting pacman config: %v", err)
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

	args := []string{"-C", pacConf, "-c", "-d", "-G", "-M", cp.b.RootfsPath, "haveged"}
	args = append(args, instList...)

	pacCmd := exec.Command(pacstrapPath, args...)
	pacCmd.Stdout = os.Stdout
	pacCmd.Stderr = os.Stderr
	sylog.Debugf("\n\tPacstrap Path: %s\n\tPac Conf: %s\n\tRootfs: %s\n\tInstall List: %s\n", pacstrapPath, pacConf, cp.b.RootfsPath, instList)

	if err = pacCmd.Run(); err != nil {
		return fmt.Errorf("while pacstrapping: %v", err)
	}

	//Pacman package signing setup
	cmd := exec.Command("arch-chroot", cp.b.RootfsPath, "/bin/sh", "-c", "haveged -w 1024; pacman-key --init; pacman-key --populate archlinux")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("while setting up package signing: %v", err)
	}

	//Clean up haveged
	cmd = exec.Command("arch-chroot", cp.b.RootfsPath, "pacman", "-Rs", "--noconfirm", "haveged")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("while cleaning up packages: %v", err)
	}

	return nil
}

// Pack puts relevant objects in a Bundle!
func (cp *ArchConveyorPacker) Pack(context.Context) (b *types.Bundle, err error) {
	err = cp.insertBaseEnv()
	if err != nil {
		return nil, fmt.Errorf("while inserting base environment: %v", err)
	}

	err = cp.insertRunScript()
	if err != nil {
		return nil, fmt.Errorf("while inserting runscript: %v", err)
	}

	return cp.b, nil
}

func getPacmanBaseList() (instList []string, err error) {
	var output, stderr bytes.Buffer
	cmd := exec.Command("pacman", "-Sgq", "base")
	cmd.Stdout = &output
	cmd.Stderr = &stderr
	if err = cmd.Run(); err != nil {
		return nil, fmt.Errorf("%v: %v", err, stderr.String())
	}

	var toInstall []string
	scanner := bufio.NewScanner(&output)
	scanner.Split(bufio.ScanWords)

	for scanner.Scan() {
		if !baseToSkip[scanner.Text()] {
			toInstall = append(toInstall, scanner.Text())
		}
	}

	return toInstall, nil
}

func (cp *ArchConveyorPacker) getPacConf(pacmanConfURL string) (pacConf string, err error) {
	pacConfFile, err := ioutil.TempFile(cp.b.RootfsPath, "pac-conf-")
	if err != nil {
		return
	}

	resp, err := http.Get(pacmanConfURL)
	if err != nil {
		return "", fmt.Errorf("while performing http request: %v", err)
	}
	defer resp.Body.Close()

	bytesWritten, err := io.Copy(pacConfFile, resp.Body)
	if err != nil {
		return
	}

	//Simple check to make sure file received is the correct size
	if bytesWritten != resp.ContentLength {
		return "", fmt.Errorf("file received is not the right size. supposed to be: %v actually: %v", resp.ContentLength, bytesWritten)
	}

	return pacConfFile.Name(), nil
}

func (cp *ArchConveyorPacker) insertBaseEnv() (err error) {
	if err = makeBaseEnv(cp.b.RootfsPath); err != nil {
		return
	}
	return nil
}

func (cp *ArchConveyorPacker) insertRunScript() (err error) {
	err = ioutil.WriteFile(filepath.Join(cp.b.RootfsPath, "/.singularity.d/runscript"), []byte("#!/bin/sh\n"), 0755)
	if err != nil {
		return
	}

	return nil
}

// CleanUp removes any tmpfs owned by the conveyorPacker on the filesystem
func (cp *ArchConveyorPacker) CleanUp() {
	cp.b.Remove()
}
