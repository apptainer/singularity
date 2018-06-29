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
)

//"github.com/singularityware/singularity/src/pkg/image"

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
		return fmt.Errorf("debootstrap is not in PATH... Perhaps 'apt-get install' it?%v", debootstrapPath)
	}

	c.tmpfs, err = ioutil.TempDir("", "temp-debootstrap-")
	if err != nil {
		return
	}

	//get mirrorURL, OSVerison, and Includes components to definition
	MirrorURL, ok := recipe.Header["MirrorURL"]
	if !ok {
		return fmt.Errorf("Invalid debootstrap header, no MirrorURL specified")
	}

	OSVersion, ok := recipe.Header["OSVersion"]
	if !ok {
		return fmt.Errorf("Invalid debootstrap header, no OSVersion specified")
	}

	Requires, _ := recipe.Header["Include"]

	//check for include environment variable and add it to requires string
	Requires += ` ` + os.Getenv("INCLUDE")

	//trim leading and trailing whitespace
	Requires = strings.TrimSpace(Requires)

	//convert Requires string to comma separated list
	Requires = strings.Replace(Requires, ` `, `,`, -1)

	if os.Getuid() != 0 {
		return fmt.Errorf("You must be root to build with debootstrap")
	}

	//run debootstrap command
	cmd := exec.Command(debootstrapPath, `--variant=minbase`, `--exclude=openssl,udev,debconf-i18n,e2fsprogs`, `--include=apt,`+Requires, `--arch=`+runtime.GOARCH, OSVersion, c.tmpfs, MirrorURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	//run debootstrap
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("Debootstrap failed with error: %v", err)
	}

	return nil
}

// Pack puts relevant objects in a Bundle!
func (cp *DebootstrapConveyorPacker) Pack() (b *Bundle, err error) {
	b, err = NewBundle()
	if err != nil {
		return
	}

	//remove old rootfs
	os.RemoveAll(b.Rootfs())

	//move downloaded files from tmpdir to bundle
	err = os.Rename(cp.tmpfs, b.Rootfs())
	if err != nil {
		return nil, fmt.Errorf("Failed to move rootfs into bundles rootfs: %v", err)
	}

	return b, nil
}
