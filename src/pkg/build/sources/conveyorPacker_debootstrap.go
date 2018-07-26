// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/singularityware/singularity/src/pkg/sylog"
)

// DebootstrapConveyor holds stuff that needs to be packed into the bundle
type DebootstrapConveyor struct {
	recipe Definition
	tmpfs  string
}

// DebootstrapConveyorPacker only needs to hold the conveyor to have the needed data to pack
type DebootstrapConveyorPacker struct {
	DebootstrapConveyor
}

// Get downloads container information from the specified source
func (c *DebootstrapConveyor) Get(recipe Definition) (err error) {

	c.recipe = recipe

	//check for debootstrap on system(script using "singularity_which" not sure about its importance)
	debootstrapPath, err := exec.LookPath("debootstrap")
	if err != nil {
		return fmt.Errorf("debootstrap is not in PATH... Perhaps 'apt-get install' it: %v", err)
	}

	c.tmpfs, err = ioutil.TempDir("", "temp-debootstrap-")
	if err != nil {
		return
	}

	//get mirrorURL, OSVerison, and Includes components to definition
	mirrorurl, ok := recipe.Header["mirrorurl"]
	if !ok {
		return fmt.Errorf("Invalid debootstrap header, no MirrorURL specified")
	}

	osversion, ok := recipe.Header["osversion"]
	if !ok {
		return fmt.Errorf("Invalid debootstrap header, no OSVersion specified")
	}

	include, _ := recipe.Header["include"]

	//check for include environment variable and add it to requires string
	include += ` ` + os.Getenv("INCLUDE")

	//trim leading and trailing whitespace
	include = strings.TrimSpace(include)

	//convert Requires string to comma separated list
	include = strings.Replace(include, ` `, `,`, -1)

	if os.Getuid() != 0 {
		return fmt.Errorf("You must be root to build with debootstrap")
	}

	//run debootstrap command
	cmd := exec.Command(debootstrapPath, `--variant=minbase`, `--exclude=openssl,udev,debconf-i18n,e2fsprogs`, `--include=apt,`+include, `--arch=`+runtime.GOARCH, osversion, c.tmpfs, mirrorurl)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	sylog.Debugf("\n\tDebootstrap Path: %s\n\tIncludes: apt(default),%s\n\tDetected Arch: %s\n\tOSVersion: %s\n\tMirrorURL: %s\n", debootstrapPath, include, runtime.GOARCH, osversion, mirrorurl)

	//run debootstrap
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("While debootstrapping: %v", err)
	}

	return nil
}

// Pack puts relevant objects in a Bundle!
func (cp *DebootstrapConveyorPacker) Pack() (b *Bundle, err error) {
	b, err = NewBundle("")
	if err != nil {
		return
	}

	//remove old rootfs
	os.RemoveAll(b.Rootfs())

	//move downloaded files from tmpdir to bundle
	err = os.Rename(cp.tmpfs, b.Rootfs())
	if err != nil {
		return nil, fmt.Errorf("While renaming bundle rootfs: %v", err)
	}

	//change root directory permissions to 0755
	if err := os.Chmod(b.Rootfs(), 0755); err != nil {
		return nil, fmt.Errorf("While changing bundle rootfs perms: %v", err)
	}

	err = cp.insertBaseEnv(b)
	if err != nil {
		return nil, fmt.Errorf("While inserting base environtment: %v", err)
	}

	err = cp.insertRunScript(b)
	if err != nil {
		return nil, fmt.Errorf("While inserting runscript: %v", err)
	}

	return b, nil
}

func (cp *DebootstrapConveyorPacker) insertBaseEnv(b *Bundle) (err error) {
	if err = makeBaseEnv(b.Rootfs()); err != nil {
		return
	}
	return nil
}

func (cp *DebootstrapConveyorPacker) insertRunScript(b *Bundle) (err error) {
	f, err := os.Create(b.Rootfs() + "/.singularity.d/runscript")
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

	err = os.Chmod(b.Rootfs()+"/.singularity.d/runscript", 0755)
	if err != nil {
		return
	}

	return nil
}
