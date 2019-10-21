// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
)

// BusyBoxConveyor only needs to hold the conveyor to have the needed data to pack
type BusyBoxConveyor struct {
	b *types.Bundle
}

// BusyBoxConveyorPacker only needs to hold the conveyor to have the needed data to pack
type BusyBoxConveyorPacker struct {
	BusyBoxConveyor
}

// Get just stores the source
func (c *BusyBoxConveyor) Get(ctx context.Context, b *types.Bundle) (err error) {
	c.b = b

	// get mirrorURL, OSVerison, and Includes components to definition
	mirrorurl, ok := b.Recipe.Header["mirrorurl"]
	if !ok {
		return fmt.Errorf("invalid busybox header, no mirrur url specified")
	}

	err = c.insertBaseEnv()
	if err != nil {
		return fmt.Errorf("while inserting base environment: %v", err)
	}

	err = c.insertBaseFiles()
	if err != nil {
		return fmt.Errorf("while inserting files: %v", err)
	}

	busyBoxPath, err := c.insertBusyBox(mirrorurl)
	if err != nil {
		return fmt.Errorf("while inserting busybox: %v", err)
	}

	cmd := exec.Command(busyBoxPath, `--install`, filepath.Join(c.b.RootfsPath, "/bin"))

	sylog.Debugf("\n\tBusyBox Path: %s\n\tMirrorURL: %s\n", busyBoxPath, mirrorurl)

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("while performing busybox install: %v", err)
	}

	return nil
}

// Pack puts relevant objects in a Bundle!
func (cp *BusyBoxConveyorPacker) Pack(context.Context) (b *types.Bundle, err error) {
	err = cp.insertRunScript()
	if err != nil {
		return nil, fmt.Errorf("while inserting base environment: %v", err)
	}

	return cp.b, nil
}

func (c *BusyBoxConveyor) insertBaseFiles() error {
	if err := ioutil.WriteFile(filepath.Join(c.b.RootfsPath, "/etc/passwd"), []byte("root:!:0:0:root:/root:/bin/sh"), 0664); err != nil {
		return err
	}

	if err := ioutil.WriteFile(filepath.Join(c.b.RootfsPath, "/etc/group"), []byte(" root:x:0:"), 0664); err != nil {
		return err
	}

	if err := ioutil.WriteFile(filepath.Join(c.b.RootfsPath, "/etc/hosts"), []byte("127.0.0.1   localhost localhost.localdomain localhost4 localhost4.localdomain4"), 0664); err != nil {
		return err
	}

	return nil
}

func (c *BusyBoxConveyor) insertBusyBox(mirrorurl string) (busyBoxPath string, err error) {
	os.Mkdir(filepath.Join(c.b.RootfsPath, "/bin"), 0755)

	resp, err := http.Get(mirrorurl)
	if err != nil {
		return "", fmt.Errorf("while performing http request: %v", err)
	}
	defer resp.Body.Close()

	f, err := os.Create(filepath.Join(c.b.RootfsPath, "/bin/busybox"))
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
		return "", fmt.Errorf("file received is not the right size. supposed to be: %v actually: %v", resp.ContentLength, bytesWritten)
	}

	err = os.Chmod(f.Name(), 0755)
	if err != nil {
		return
	}

	return filepath.Join(c.b.RootfsPath, "/bin/busybox"), nil
}

func (c *BusyBoxConveyor) insertBaseEnv() (err error) {
	if err = makeBaseEnv(c.b.RootfsPath); err != nil {
		return
	}
	return nil
}

func (cp *BusyBoxConveyorPacker) insertRunScript() error {
	if err := ioutil.WriteFile(filepath.Join(cp.b.RootfsPath, "/.singularity.d/runscript"), []byte("#!/bin/sh\n"), 0755); err != nil {
		return err
	}

	return nil
}

// CleanUp removes any tmpfs owned by the conveyorPacker on the filesystem
func (c *BusyBoxConveyor) CleanUp() {
	c.b.Remove()
}
