// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
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

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
	"github.com/sylabs/singularity/pkg/util/namespaces"
)

// DebootstrapConveyorPacker holds stuff that needs to be packed into the bundle
type DebootstrapConveyorPacker struct {
	b         *types.Bundle
	mirrorurl string
	osversion string
	include   string
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
	cmd := exec.Command(debootstrapPath, `--variant=minbase`, `--exclude=openssl,udev,debconf-i18n,e2fsprogs`, `--include=apt,`+cp.include, `--arch=`+runtime.GOARCH, cp.osversion, cp.b.RootfsPath, cp.mirrorurl)

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
