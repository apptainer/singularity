// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package assemblers

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"syscall"

	uuid "github.com/satori/go.uuid"
	"github.com/sylabs/sif/pkg/sif"
	"github.com/sylabs/singularity/internal/pkg/build/metadata"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
	"github.com/sylabs/singularity/pkg/image/packer"
	"github.com/sylabs/singularity/pkg/util/crypt"
)

// SIFAssembler doesnt store anything
type SIFAssembler struct {
	GzipFlag       bool
	MksquashfsPath string
}

type encryptionOptions struct {
	keyInfo   crypt.KeyInfo
	plaintext []byte
}

func createSIF(path string, definition, ociConf []byte, squashfile string, encOpts *encryptionOptions) (err error) {
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

	if encOpts != nil {
		sifType = sif.FsEncryptedSquashfs
	}

	err = parinput.SetPartExtra(sifType, sif.PartPrimSys, sif.GetSIFArch(runtime.GOARCH))
	if err != nil {
		return
	}

	// add this descriptor input element to the list
	cinfo.InputDescr = append(cinfo.InputDescr, parinput)

	if encOpts != nil {
		data, err := crypt.EncryptKey(encOpts.keyInfo, encOpts.plaintext)
		if err != nil {
			return fmt.Errorf("while encrypting filesystem key: %s", err)
		}

		if data != nil {
			syspartID := uint32(len(cinfo.InputDescr))
			part := sif.DescriptorInput{
				Datatype: sif.DataCryptoMessage,
				Groupid:  sif.DescrDefaultGroup,
				Link:     syspartID,
				Data:     data,
				Size:     int64(len(data)),
			}

			// extra data needed for the creation of a signature descriptor
			err := part.SetCryptoMsgExtra(sif.FormatPEM, sif.MessageRSAOAEP)
			if err != nil {
				return err
			}

			cinfo.InputDescr = append(cinfo.InputDescr, part)
		}
	}

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

// Assemble creates a SIF image from a Bundle
func (a *SIFAssembler) Assemble(b *types.Bundle, path string) error {
	sylog.Infof("Creating SIF file...")

	s := packer.NewSquashfs()
	s.MksquashfsPath = a.MksquashfsPath

	f, err := ioutil.TempFile(b.TmpDir, "squashfs-")
	if err != nil {
		return fmt.Errorf("while creating temporary file for squashfs: %v", err)
	}

	fsPath := f.Name()
	f.Close()
	defer os.Remove(fsPath)

	flags := []string{"-noappend"}
	// build squashfs with all-root flag when building as a user
	if syscall.Getuid() != 0 {
		flags = append(flags, "-all-root")
	}
	// specify compression if needed
	if a.GzipFlag {
		flags = append(flags, "-comp", "gzip")
	}

	if err := s.Create([]string{b.RootfsPath}, fsPath, flags); err != nil {
		return fmt.Errorf("while creating squashfs: %v", err)
	}

	var encOpts *encryptionOptions

	if b.Opts.EncryptionKeyInfo != nil {
		plaintext, err := crypt.NewPlaintextKey(*b.Opts.EncryptionKeyInfo)
		if err != nil {
			return fmt.Errorf("unable to obtain encryption key: %+v", err)
		}

		// A dm-crypt device needs to be created with squashfs
		cryptDev := &crypt.Device{}

		// TODO (schebro): Fix #3876
		// Detach the following code from the squashfs creation. SIF can be
		// created first and encrypted after. This gives the flexibility to
		// encrypt an existing SIF
		loopPath, err := cryptDev.EncryptFilesystem(fsPath, plaintext)
		if err != nil {
			return fmt.Errorf("unable to encrypt filesystem at %s: %+v", fsPath, err)
		}
		defer os.Remove(loopPath)

		fsPath = loopPath

		encOpts = &encryptionOptions{
			keyInfo:   *b.Opts.EncryptionKeyInfo,
			plaintext: plaintext,
		}

	}

	err = createSIF(path, b.Recipe.Raw, b.JSONObjects["oci-config"], fsPath, encOpts)
	if err != nil {
		return fmt.Errorf("while creating SIF: %v", err)
	}

	labels := make(map[string]map[string]string, 1)
	//	labels["system-partition"] = make(map[string]string, 1)

	// Get the old image labels first
	if b.RunSection("labels") && len(b.Recipe.ImageData.Labels) > 0 {
		for key, value := range b.Recipe.ImageData.Labels {
			labels[key] = make(map[string]string, 1)
			for foo, bar := range value {
				labels[key][foo] = bar
				fmt.Printf("OLD LABELS: %s : %s : %s\n", key, foo, bar)
				//labels[key][foo] = bar
			}
		}
	}

	fmt.Printf("RECIPE_LABELS: %+v\n", b.Recipe.ImageData.Labels)
	fmt.Printf("JSON_LABELS: %+v\n", b.JSONLabels)

	// Copy the labels
	for k, v := range b.JSONLabels {
		labels[k] = make(map[string]string, 1)
		for foo, bar := range v {
			labels[k][foo] = bar
		}
	}

	//for k, v := range b.Recipe.ImageData.Labels {
	//	labels[k] = make(map[string]string, 1)
	//	for foo, bar := range v {
	//		labels[k][foo] = bar
	//	}
	//}

	sylog.Infof("Inserting Metadata Labels...")

	// load the container to add the metadata
	fimg, err := sif.LoadContainer(path, false)
	if err != nil {
		return fmt.Errorf("failed to load sif container file: %s", err)
	}
	defer fimg.UnloadContainer()

	descr, err := getDescr(&fimg)
	if err != nil {
		return fmt.Errorf("no primary partition found: %s", err)
	}
	groupid := descr[0].Groupid

	// Make the new org.label-schema, overidding the old ones
	metadata.GetImageInfoLabels(labels, &fimg, b)

	// make new map into json
	text, err := json.MarshalIndent(labels, "", "    ")
	if err != nil {
		return err
	}

	// Add the metadata
	err = sifAddMetadata(&fimg, groupid, uint32(0), text)
	if err != nil {
		return fmt.Errorf("failed adding metadata block to SIF container file: %s", err)
	}

	return nil
}

func sifAddMetadata(fimg *sif.FileImage, groupid, link uint32, data []byte) error {
	// data we need to create a signature descriptor
	siginput := sif.DescriptorInput{
		Datatype: sif.DataLabels,
		Groupid:  groupid,
		Link:     link,
		Fname:    "image-metadata",
		Data:     data,
	}
	siginput.Size = int64(binary.Size(siginput.Data))

	// add new signature data object to SIF file
	err := fimg.AddObject(siginput)
	if err != nil {
		return err
	}

	return nil
}

func getDescr(fimg *sif.FileImage) ([]*sif.Descriptor, error) {
	descr := make([]*sif.Descriptor, 1)
	var err error

	descr[0], _, err = fimg.GetPartPrimSys()
	if err != nil {
		return nil, fmt.Errorf("no primary partition found")
	}

	return descr, nil
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
