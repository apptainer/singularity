// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os/exec"
	"runtime"
	"os"

	"github.com/singularityware/singularity/src/pkg/sylog"
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

// ArchConveyor only needs to hold the conveyor to have the needed data to pack
type ArchConveyor struct {
	recipe Definition
	b      *Bundle
	src    string
	tmpfs  string
}

// ArchConveyorPacker only needs to hold the conveyor to have the needed data to pack
type ArchConveyorPacker struct {
	ArchConveyor
}

// Get just stores the source
func (c *ArchConveyor) Get(recipe Definition) (err error) {
	c.recipe = recipe

	//check for pacstrap on system
	pacstrapPath, err := exec.LookPath("pacstrap")
	if err != nil {
		return fmt.Errorf("pacstrap is not in PATH: %v", err)
	}

	//make sure architecture is supported
	if arch := runtime.GOARCH; arch != `amd64` && arch != `i686` {
		return fmt.Errorf("%v architecture is not supported", arch)
	}

	c.tmpfs, err = ioutil.TempDir("", "temp-arch-")
	if err != nil {
		return
	}

	c.b, err = NewBundle(c.tmpfs)
	if err != nil {
		return
	}

	instList, err := c.getInstList()
	if err != nil {
		return fmt.Errorf("While generating the installation list: %v", err)
	}

	pacConf, err := c.getPacConf(pacmanConfURL)
	if err != nil {
		return fmt.Errorf("While getting pacman config: %v", err)
	}

	args := []string{"-C", pacConf, "-c","-d","-G", "-M", c.b.Rootfs(), "haveged"}
	args = append(args, instList...)

	pacCmd := exec.Command(pacstrapPath,args...)
	pacCmd.Stdout = os.Stdout
	pacCmd.Stderr = os.Stderr
	sylog.Debugf("\n\tPacstrap Path: %s\n\tPac Conf: %s\n\tRootfs: %s\n\tInstall List: %s\n", pacstrapPath, pacConf, c.b.Rootfs(), instList)

	if err = pacCmd.Run(); err != nil {
		return fmt.Errorf("While pacstrapping: %v", err)
	}

	//Pacman package signing setup
	cmd := exec.Command("arch-chroot", c.b.Rootfs(), "/bin/sh", "-c", "haveged -w 1024; pacman-key --init; pacman-key --populate archlinux")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("While setting up package signing: %v", err)
	}

	//Clean up haveged
	cmd = exec.Command("arch-chroot", c.b.Rootfs(), "pacman", "-Rs", "--noconfirm", "haveged")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("While cleaning up packages: %v", err)
	}

	return nil
}

// Pack puts relevant objects in a Bundle!
func (cp *ArchConveyorPacker) Pack() (b *Bundle, err error) {

	err = cp.insertBaseEnv()
	if err != nil {
		return nil, fmt.Errorf("While inserting base environment: %v\n", err)
	}

	err = cp.insertRunScript()
	if err != nil {
		return nil, fmt.Errorf("While inserting runscript: %v\n", err)
	}

	cp.b.Recipe = cp.recipe

	return cp.b, nil
}

func (c *ArchConveyor) getInstList() (instList []string, err error) {

	r,w,err := os.Pipe()
	if err != nil {
		return
	}

	//feed output command into pipe while scanner reads from the other end
	go func() {
		cmd := exec.Command("pacman", "-Sgq", "base")
		cmd.Stdout = w
		defer w.Close()
		if err = cmd.Run(); err != nil {
			return 
		}
	}()

	var toInstall []string
	scanner := bufio.NewScanner(r)
	scanner.Split(bufio.ScanWords)

	for scanner.Scan() {
		if !baseToSkip[scanner.Text()] {
			toInstall = append(toInstall, scanner.Text())
		}
	}

	return toInstall, nil
}

func (c *ArchConveyor) getPacConf(pacmanConfURL string) (pacConf string, err error) {

	pacConfFile, err := ioutil.TempFile(c.tmpfs, "pac-conf-")
	if err != nil {
		return
	}

	resp, err := http.Get(pacmanConfURL)
	if err != nil {
		return "", fmt.Errorf("While performing http request: %v", err)
	}
	defer resp.Body.Close()

	bytesWritten, err := io.Copy(pacConfFile, resp.Body)
	if err != nil {
		return
	}

	//Simple check to make sure file received is the correct size
	if bytesWritten != resp.ContentLength {
		return "", fmt.Errorf("File received is not the right size. Supposed to be: %v  Actually: %v", resp.ContentLength, bytesWritten)
	}

	return pacConfFile.Name(), nil
}

func (cp *ArchConveyorPacker) insertBaseEnv() (err error) {
	if err = makeBaseEnv(cp.b.Rootfs()); err != nil {
		return
	}
	return nil
}

func (cp *ArchConveyorPacker) insertRunScript() (err error) {
	f, err := os.Create(cp.b.Rootfs() + "/.singularity.d/runscript")
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

	err = os.Chmod(cp.b.Rootfs()+"/.singularity.d/runscript", 0755)
	if err != nil {
		return
	}

	return nil
}
