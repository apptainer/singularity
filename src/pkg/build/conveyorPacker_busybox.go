// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/singularityware/singularity/src/pkg/sylog"
)

// BusyBoxConveyor only needs to hold the conveyor to have the needed data to pack
type BusyBoxConveyor struct {
	recipe Definition
	src    string
	tmpfs  string
	b      *Bundle
}

// BusyBoxConveyorPacker only needs to hold the conveyor to have the needed data to pack
type BusyBoxConveyorPacker struct {
	BusyBoxConveyor
}

// Get just stores the source
func (c *BusyBoxConveyor) Get(recipe Definition) (err error) {

	c.recipe = recipe

	c.b, err = NewBundle("")
	if err != nil {
		return
	}

	//get mirrorURL, OSVerison, and Includes components to definition
	mirrorurl, ok := recipe.Header["mirrorurl"]
	if !ok {
		return fmt.Errorf("Invalid busybox header, no MirrurURL specified")
	}

	err = c.insertBaseEnv()
	if err != nil {
		return fmt.Errorf("While inserting base environment: %v", err)
	}

	err = c.insertBaseFiles()
	if err != nil {
		return fmt.Errorf("While inserting files: %v", err)
	}

	busyBoxPath, err := c.insertBusyBox(mirrorurl)
	if err != nil {
		return fmt.Errorf("While inserting busybox: %v", err)
	}

	cmd := exec.Command(busyBoxPath, `--install`, filepath.Join(c.b.Rootfs(), "/bin"))

	sylog.Debugf("\n\tBusyBox Path: %s\n\tMirrorURL: %s\n", busyBoxPath, mirrorurl)

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("While performing busybox install: %v", err)
	}

	return nil
}

// Pack puts relevant objects in a Bundle!
func (cp *BusyBoxConveyorPacker) Pack() (b *Bundle, err error) {

	err = cp.insertRunScript()
	if err != nil {
		return nil, fmt.Errorf("While inserting base environment: %v", err)
	}

	cp.b.Recipe = cp.recipe

	return cp.b, nil
}

func (c *BusyBoxConveyor) insertBaseFiles() (err error) {

	ioutil.WriteFile(filepath.Join(c.b.Rootfs(), "/etc/passwd"), []byte("root:!:0:0:root:/root:/bin/sh"), 0664)
	if err != nil {
		return
	}

	ioutil.WriteFile(filepath.Join(c.b.Rootfs(), "/etc/group"), []byte(" root:x:0:"), 0664)
	if err != nil {
		return
	}

	ioutil.WriteFile(filepath.Join(c.b.Rootfs(), "/etc/hosts"), []byte("127.0.0.1   localhost localhost.localdomain localhost4 localhost4.localdomain4"), 0664)
	if err != nil {
		return
	}

	return
}

func (c *BusyBoxConveyor) insertBusyBox(mirrorurl string) (busyBoxPath string, err error) {

	os.Mkdir(filepath.Join(c.b.Rootfs(), "/bin"), 0755)

	resp, err := http.Get(mirrorurl)
	if err != nil {
		return "", fmt.Errorf("While performing http request: %v", err)
	}
	defer resp.Body.Close()

	f, err := os.Create(filepath.Join(c.b.Rootfs(), "/bin/busybox"))
	if err != nil {
		return
	}
	defer f.Close()

	bytesWritten, err := io.Copy(f, resp.Body)
	if err != nil {
		return
	}

	//Simple check to make sure file received is the correct size
	if bytesWritten != resp.ContentLength {
		return "", fmt.Errorf("File received is not the right size. Supposed to be: %v  Actually: %v", resp.ContentLength, bytesWritten)
	}

	err = os.Chmod(f.Name(), 0755)
	if err != nil {
		return
	}

	return filepath.Join(c.b.Rootfs(), "/bin/busybox"), nil
}

func (c *BusyBoxConveyor) insertBaseEnv() (err error) {
	if err = makeBaseEnv(c.b.Rootfs()); err != nil {
		return
	}
	return nil
}

func (cp *BusyBoxConveyorPacker) insertRunScript() (err error) {

	ioutil.WriteFile(filepath.Join(cp.b.Rootfs(), "/.singularity.d/runscript"), []byte("#!/bin/sh\n"), 0755)
	if err != nil {
		return
	}

	return nil
}
