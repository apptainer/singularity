// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package assemblers

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/satori/go.uuid"
	"github.com/sylabs/sif/pkg/sif"
	"github.com/sylabs/singularity/src/pkg/build/types"
	"github.com/sylabs/singularity/src/pkg/build/types/parser"
	"github.com/sylabs/singularity/src/pkg/buildcfg"
	"github.com/sylabs/singularity/src/pkg/sylog"
	"github.com/sylabs/singularity/src/runtime/engines/config"
	"github.com/sylabs/singularity/src/runtime/engines/singularity"
)

// SIFAssembler doesnt store anything
type SIFAssembler struct {
}

func createSIF(path string, definition []byte, squashfile string) (err error) {
	// general info for the new SIF file creation
	cinfo := sif.CreateInfo{
		Pathname:   path,
		Launchstr:  sif.HdrLaunch,
		Sifversion: sif.HdrVersion,
		ID:         uuid.NewV4(),
	}

	// data we need to create a definition file descriptor
	definput := sif.DescriptorInput{
		Datatype: sif.DataDeffile,
		Groupid:  sif.DescrDefaultGroup,
		Link:     sif.DescrUnusedLink,
		Data:     definition,
	}
	definput.Size = int64(binary.Size(definput.Data))

	// add this descriptor input element to creation descriptor slice
	cinfo.InputDescr = append(cinfo.InputDescr, definput)

	// data we need to create a system partition descriptor
	parinput := sif.DescriptorInput{
		Datatype: sif.DataPartition,
		Groupid:  sif.DescrDefaultGroup,
		Link:     sif.DescrUnusedLink,
		Fname:    squashfile,
	}
	// open up the data object file for this descriptor
	if parinput.Fp, err = os.Open(parinput.Fname); err != nil {
		return fmt.Errorf("while opening partition file: %s", err)
	}
	defer parinput.Fp.Close()
	fi, err := parinput.Fp.Stat()
	if err != nil {
		return fmt.Errorf("while calling start on partition file: %s", err)
	}
	parinput.Size = fi.Size()

	err = parinput.SetPartExtra(sif.FsSquash, sif.PartPrimSys, sif.GetSIFArch(runtime.GOARCH))
	if err != nil {
		return
	}

	// add this descriptor input element to the list
	cinfo.InputDescr = append(cinfo.InputDescr, parinput)

	// remove anything that may exist at the build destination at last moment
	os.RemoveAll(path)

	// test container creation with two partition input descriptors
	if _, err := sif.CreateContainer(cinfo); err != nil {
		return fmt.Errorf("while creating container: %s", err)
	}

	return nil
}

func getMksquashfsPath() (string, error) {
	// parse singularity configuration file
	c := &singularity.FileConfig{}
	if err := config.Parser(buildcfg.SYSCONFDIR+"/singularity/singularity.conf", c); err != nil {
		return "", fmt.Errorf("Unable to parse singularity.conf file: %s", err)
	}

	// look in admin defined mksquashfs location
	if c.MksquashfsPath != "" {
		mksquashfs := filepath.Join(c.MksquashfsPath, "mksquashfs")

		if _, err := os.Stat(mksquashfs); os.IsNotExist(err) {
			return "", fmt.Errorf("mksquashfs cannot be found at custom location: %v", c.MksquashfsPath)
		}

		return mksquashfs, nil
	}

	// look for mksquashfs in standard locations
	mksquashfs, err := exec.LookPath("mksquashfs")
	if err != nil {
		return "", fmt.Errorf("mksquashfs cannot be found on this system")
	}

	return mksquashfs, nil
}

// Assemble creates a SIF image from a Bundle
func (a *SIFAssembler) Assemble(b *types.Bundle, path string) (err error) {
	defer os.RemoveAll(b.Path)

	sylog.Infof("Creating SIF file...")

	// convert definition to plain text
	var buf bytes.Buffer
	parser.WriteDefinitionFile(&(b.Recipe), &buf)
	def := buf.Bytes()

	mksquashfs, err := getMksquashfsPath()
	if err != nil {
		return
	}

	f, err := ioutil.TempFile(b.Path, "squashfs-")
	squashfsPath := f.Name() + ".img"
	f.Close()
	os.Remove(f.Name())
	os.Remove(squashfsPath)

	args := []string{b.Rootfs(), squashfsPath, "-noappend"}

	// build squashfs with all-root flag when building as a user
	if syscall.Getuid() != 0 {
		args = append(args, "-all-root")
	}

	mksquashfsCmd := exec.Command(mksquashfs, args...)
	err = mksquashfsCmd.Run()
	defer os.Remove(squashfsPath)
	if err != nil {
		return
	}

	err = createSIF(path, def, squashfsPath)
	if err != nil {
		return
	}

	return
}
