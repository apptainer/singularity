// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
)

// DebootstrapConveyorPacker holds stuff that needs to be packed into the bundle
type DebootstrapConveyorPacker struct {
	b         *types.Bundle
	mirrorurl string
	osversion string
	include   string
}

// SetImgCache is a no-op for debootstrap; it does not use any image cache
func (cp *DebootstrapConveyorPacker) SetImgCache(*cache.ImgCache) (err error) {
	return nil
}

// Get downloads container information from the specified source
func (cp *DebootstrapConveyorPacker) Get(b *types.Bundle) (err error) {
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
		return fmt.Errorf("You must be root to build with debootstrap")
	}

	// run debootstrap command
	cmd := exec.Command(debootstrapPath, `--variant=minbase`, `--exclude=openssl,udev,debconf-i18n,e2fsprogs`, `--include=apt,`+cp.include, `--arch=`+runtime.GOARCH, cp.osversion, cp.b.Rootfs(), cp.mirrorurl)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	sylog.Debugf("\n\tDebootstrap Path: %s\n\tIncludes: apt(default),%s\n\tDetected Arch: %s\n\tOSVersion: %s\n\tMirrorURL: %s\n", debootstrapPath, cp.include, runtime.GOARCH, cp.osversion, cp.mirrorurl)

	// run debootstrap
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("While debootstrapping: %v", err)
	}

	return nil
}

// Pack puts relevant objects in a Bundle!
func (cp *DebootstrapConveyorPacker) Pack() (*types.Bundle, error) {

	//change root directory permissions to 0755
	if err := os.Chmod(cp.b.Rootfs(), 0755); err != nil {
		return nil, fmt.Errorf("While changing bundle rootfs perms: %v", err)
	}

	err := cp.insertBaseEnv(cp.b)
	if err != nil {
		return nil, fmt.Errorf("While inserting base environtment: %v", err)
	}

	err = cp.insertRunScript(cp.b)
	if err != nil {
		return nil, fmt.Errorf("While inserting runscript: %v", err)
	}

	return cp.b, nil
}

func (cp *DebootstrapConveyorPacker) getRecipeHeaderInfo() (err error) {
	var ok bool

	//get mirrorURL, OSVerison, and Includes components to definition
	cp.mirrorurl, ok = cp.b.Recipe.Header["mirrorurl"]
	if !ok {
		return fmt.Errorf("Invalid debootstrap header, no MirrorURL specified")
	}

	cp.osversion, ok = cp.b.Recipe.Header["osversion"]
	if !ok {
		return fmt.Errorf("Invalid debootstrap header, no OSVersion specified")
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
	if err = makeBaseEnv(b.Rootfs()); err != nil {
		return
	}
	return nil
}

func (cp *DebootstrapConveyorPacker) insertRunScript(b *types.Bundle) (err error) {
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

// CleanUp removes any tmpfs owned by the conveyorPacker on the filesystem
func (cp *DebootstrapConveyorPacker) CleanUp() {
	os.RemoveAll(cp.b.Path)
}
