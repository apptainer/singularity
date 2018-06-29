// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"fmt"
	"os/exec"
	"runtime"
)

//"github.com/singularityware/singularity/src/pkg/image"

// DebootstrapConveyor holds stuff that needs to be packed into the bundle
type DebootstrapConveyor struct {
	recipe Definition
}

// DebootstrapConveyorPacker only needs to hold the conveyor to have the needed data to pack
type DebootstrapConveyorPacker struct {
	DebootstrapConveyor
}

// Get downloads container information from the specified source
func (c *DebootstrapConveyor) Get(recipe Definition) (err error) {

	c.recipe = recipe

	//get mirrorURL and OSVerison components to definition
	OSVersion, ok := recipe.Header["MirrorURL"]
	if ok != false {
		return fmt.Errorf("Invalid debootstrap header, no MirrorURL specified")
	}

	MirrorURL, ok := recipe.Header["OSVersion"]
	if ok != false {
		return fmt.Errorf("Invalid debootstrap header, no OSVersion specified")
	}

	//check for debootstrap on system(script using "singularity_which" not sure about its importance)
	debootstrapPath, err := exec.LookPath("debootstrap")
	if err != nil {
		return fmt.Errorf("debootstrap is not in PATH... Perhaps 'apt-get install' it?%v", debootstrapPath)
	}

	//Dont know what this does...
	//REQUIRES=`echo "${INCLUDE:-}" | sed -e 's/\s/,/g'`
	requires := ""

	//run debootstrap command
	//$DEBOOTSTRAP_PATH --variant=minbase --exclude=openssl,udev,debconf-i18n,e2fsprogs --include=apt,$REQUIRES --arch=$ARCH '$OSVERSION' '$SINGULARITY_ROOTFS' '$MIRRORURL'
	cmd := exec.Command(debootstrapPath, `--variant=minbase --exclude=openssl,udev,debconf-i18n,e2fsprogs`, `--include=apt,`+requires, `--arch=`+runtime.GOARCH, OSVersion, MirrorURL)

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

	return b, nil
}
