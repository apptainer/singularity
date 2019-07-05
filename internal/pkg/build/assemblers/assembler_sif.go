// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package assemblers

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	uuid "github.com/satori/go.uuid"
	"github.com/sylabs/sif/pkg/sif"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/runtime/engines/config"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
	singularityConfig "github.com/sylabs/singularity/pkg/runtime/engines/singularity/config"
	"github.com/sylabs/singularity/pkg/util/crypt"
)

// SIFAssembler doesnt store anything
type SIFAssembler struct {
}

func createSIF(path string, definition, ociConf []byte, squashfile string, encrypted bool) (err error) {
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

	if len(ociConf) > 0 {
		// data we need to create a definition file descriptor
		ociInput := sif.DescriptorInput{
			Datatype: sif.DataGenericJSON,
			Groupid:  sif.DescrDefaultGroup,
			Link:     sif.DescrUnusedLink,
			Data:     ociConf,
			Fname:    "oci-config.json",
		}
		ociInput.Size = int64(binary.Size(ociInput.Data))

		// add this descriptor input element to creation descriptor slice
		cinfo.InputDescr = append(cinfo.InputDescr, ociInput)
	}

	// data we need to create a system partition descriptor
	parinput := sif.DescriptorInput{
		Datatype: sif.DataPartition,
		Groupid:  sif.DescrDefaultGroup,
		Link:     sif.DescrUnusedLink,
		Fname:    squashfile,
	}
	// open up the data object file for this descriptor
	fp, err := os.Open(parinput.Fname)
	if err != nil {
		return fmt.Errorf("while opening partition file: %s", err)
	}

	defer fp.Close()

	fi, err := fp.Stat()
	if err != nil {
		return fmt.Errorf("while calling stat on partition file: %s", err)
	}

	parinput.Fp = fp
	parinput.Size = fi.Size()

	sifType := sif.FsSquash

	if encrypted {
		sifType = sif.FsEncryptedSquashfs
	}

	err = parinput.SetPartExtra(sifType, sif.PartPrimSys, sif.GetSIFArch(runtime.GOARCH))
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

	// chown the sif file to the calling user
	if uid, gid, ok := changeOwner(); ok {
		if err := os.Chown(path, uid, gid); err != nil {
			return fmt.Errorf("while changing image ownership: %s", err)
		}
	}

	return nil
}

func getMksquashfsPath() (string, error) {
	// Parse singularity configuration file
	c := &singularityConfig.FileConfig{}
	if err := config.Parser(buildcfg.SYSCONFDIR+"/singularity/singularity.conf", c); err != nil {
		return "", fmt.Errorf("Unable to parse singularity.conf file: %s", err)
	}

	// p is either "" or the string value in the conf file
	p := c.MksquashfsPath

	// If the path contains the binary name use it as is, otherwise add mksquashfs via filepath.Join
	if !strings.HasSuffix(c.MksquashfsPath, "mksquashfs") {
		p = filepath.Join(c.MksquashfsPath, "mksquashfs")
	}

	// exec.LookPath functions on absolute paths (ignoring $PATH) as well
	return exec.LookPath(p)
}

// Assemble creates a SIF image from a Bundle
func (a *SIFAssembler) Assemble(b *types.Bundle, path string) (err error) {
	sylog.Infof("Creating SIF file...")

	var fsPath string

	mksquashfs, err := getMksquashfsPath()
	if err != nil {
		return fmt.Errorf("While searching for mksquashfs: %v", err)
	}
	f, err := ioutil.TempFile(b.Path, "squashfs-")
	fsPath = f.Name()
	f.Close()
	defer os.Remove(fsPath)
	args := []string{b.Rootfs(), fsPath, "-noappend"}

	// build squashfs with all-root flag when building as a user
	if syscall.Getuid() != 0 {
		args = append(args, "-all-root")
	}

	mksquashfsCmd := exec.Command(mksquashfs, args...)
	stderr, err := mksquashfsCmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("While setting up stderr pipe: %v", err)
	}

	if err := mksquashfsCmd.Start(); err != nil {
		return fmt.Errorf("While starting mksquashfs: %v", err)
	}

	errOut, err := ioutil.ReadAll(stderr)
	if err != nil {
		return fmt.Errorf("While reading mksquashfs stderr: %v", err)
	}

	if err := mksquashfsCmd.Wait(); err != nil {
		return fmt.Errorf("While running mksquashfs: %v: %s", err, strings.Replace(string(errOut), "\n", " ", -1))
	}

	if b.Opts.Encrypted {
		// A dm-crypt device needs to be created with squashfs
		cryptDev := &crypt.Device{}

		// TODO (schebro): Fix #3876
		// Detach the following code from the squashfs creation. SIF can be
		// created first and encrypted after. This gives the flexibility to
		// encrypt an existing SIF
		key, err := cryptDev.ReadKeyFromStdin(true)
		if err != nil {
			return fmt.Errorf("unable to read key from stdin")
		}
		loopPath, cryptName, err := cryptDev.FormatCryptDevice(fsPath, key)
		if err != nil {
			return fmt.Errorf("unable to format crypt device: %s", cryptName)
		}

		defer os.Remove(loopPath)

		err = cryptDev.CloseCryptDevice(cryptName)
		if err != nil {
			return fmt.Errorf("unable to close crypt device: %s", cryptName)
		}
		fsPath = loopPath
	}

	err = createSIF(path, b.Recipe.Raw, b.JSONObjects["oci-config"], fsPath, b.Opts.Encrypted)
	if err != nil {
		return fmt.Errorf("While creating SIF: %v", err)
	}

	return
}

// changeOwner check the command being called with sudo with the environment
// variable SUDO_COMMAND. Pattern match that for the singularity bin
func changeOwner() (int, int, bool) {
	r := regexp.MustCompile("(singularity)")
	sudoCmd := os.Getenv("SUDO_COMMAND")
	if !r.MatchString(sudoCmd) {
		return 0, 0, false
	}

	if os.Getenv("SUDO_USER") == "" || syscall.Getuid() != 0 {
		return 0, 0, false
	}

	_uid := os.Getenv("SUDO_UID")
	_gid := os.Getenv("SUDO_GID")
	if _uid == "" || _gid == "" {
		sylog.Warningf("Env vars SUDO_UID or SUDO_GID are not set, won't call chown over built SIF")

		return 0, 0, false
	}

	uid, err := strconv.Atoi(_uid)
	if err != nil {
		sylog.Warningf("Error while calling strconv: %v", err)

		return 0, 0, false
	}
	gid, err := strconv.Atoi(_gid)
	if err != nil {
		sylog.Warningf("Error while calling strconv : %v", err)

		return 0, 0, false
	}

	return uid, gid, true
}
