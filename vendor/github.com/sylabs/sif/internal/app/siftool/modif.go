// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// Copyright (c) 2018, Divya Cote <divya.cote@gmail.com> All rights reserved.
// Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
// Copyright (c) 2017, Yannick Cote <yhcote@gmail.com> All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package siftool

import (
	"fmt"
	"log"
	"os"

	uuid "github.com/satori/go.uuid"
	"github.com/sylabs/sif/pkg/sif"
)

// New creates a new empty SIF file
func New(file string) error {
	cinfo := sif.CreateInfo{
		Pathname:   file,
		Launchstr:  sif.HdrLaunch,
		Sifversion: sif.HdrVersion,
		ID:         uuid.NewV4(),
	}

	_, err := sif.CreateContainer(cinfo)
	if err != nil {
		return err
	}

	return nil
}

// AddOptions contains the options when adding a section to a SIF file
type AddOptions struct {
	Datatype   *int64
	Parttype   *int64
	Partfs     *int64
	Partarch   *int64
	Signhash   *int64
	Signentity *string
	Groupid    *int64
	Link       *int64
	Alignment  *int
	Filename   *string
}

// Add adds a data object to a SIF file
func Add(containerFile, dataFile string, opts AddOptions) error {
	var err error
	var d sif.Datatype
	var a string

	switch *opts.Datatype {
	case 1:
		d = sif.DataDeffile
	case 2:
		d = sif.DataEnvVar
	case 3:
		d = sif.DataLabels
	case 4:
		d = sif.DataPartition
	case 5:
		d = sif.DataSignature
	case 6:
		d = sif.DataGenericJSON
	case 7:
		d = sif.DataGeneric
	default:
		log.Printf("error: -datatype flag is required with a valid range\n\n")
		return fmt.Errorf("usage")
	}

	if *opts.Filename == "" {
		*opts.Filename = dataFile
	}

	// data we need to create a new descriptor
	input := sif.DescriptorInput{
		Datatype:  d,
		Groupid:   sif.DescrGroupMask | uint32(*opts.Groupid),
		Link:      uint32(*opts.Link),
		Alignment: *opts.Alignment,
		Fname:     *opts.Filename,
	}

	if dataFile == "-" {
		input.Fp = os.Stdin
	} else {
		// open up the data object file for this descriptor
		fp, err := os.Open(dataFile)
		if err != nil {
			return err
		}
		defer fp.Close()

		input.Fp = fp

		fi, err := fp.Stat()
		if err != nil {
			return err
		}
		input.Size = fi.Size()
	}

	if d == sif.DataPartition {
		if sif.Fstype(*opts.Partfs) == -1 || sif.Parttype(*opts.Parttype) == -1 || *opts.Partarch == -1 {
			return fmt.Errorf("with partition datatype, -partfs, -parttype and -partarch must be passed")
		}

		switch *opts.Partarch {
		case 1:
			a = sif.HdrArch386
		case 2:
			a = sif.HdrArchAMD64
		case 3:
			a = sif.HdrArchARM
		case 4:
			a = sif.HdrArchARM64
		case 5:
			a = sif.HdrArchPPC64
		case 6:
			a = sif.HdrArchPPC64le
		case 7:
			a = sif.HdrArchMIPS
		case 8:
			a = sif.HdrArchMIPSle
		case 9:
			a = sif.HdrArchMIPS64
		case 10:
			a = sif.HdrArchMIPS64le
		case 11:
			a = sif.HdrArchS390x
		default:
			log.Printf("error: -partarch flag is required with a valid range\n\n")
			return fmt.Errorf("usage")
		}

		err := input.SetPartExtra(sif.Fstype(*opts.Partfs), sif.Parttype(*opts.Parttype), a, []byte{0})
		if err != nil {
			return err
		}
	} else if d == sif.DataSignature {
		if sif.Hashtype(*opts.Signhash) == -1 || *opts.Signentity == "" {
			return fmt.Errorf("with signature datatype, -signhash and -signentity must be passed")
		}

		if err := input.SetSignExtra(sif.Hashtype(*opts.Signhash), *opts.Signentity); err != nil {
			return err
		}
	}

	// load SIF image file
	fimg, err := sif.LoadContainer(containerFile, false)
	if err != nil {
		return err
	}
	defer func() {
		if err := fimg.UnloadContainer(); err != nil {
			fmt.Println("Error unloading container: ", err)
		}
	}()

	// add new data object to SIF file
	if err = fimg.AddObject(input); err != nil {
		return err
	}

	return nil
}

// Del deletes a specified object descriptor and data from the SIF file
func Del(descr uint64, file string) error {
	fimg, err := sif.LoadContainer(file, false)
	if err != nil {
		return err
	}
	defer func() {
		if err := fimg.UnloadContainer(); err != nil {
			fmt.Println("Error unloading container: ", err)
		}
	}()

	for _, v := range fimg.DescrArr {
		if !v.Used {
			continue
		} else if v.ID == uint32(descr) {
			if err := fimg.DeleteObject(uint32(descr), 0); err != nil {
				return err
			}

			return nil
		}
	}

	return fmt.Errorf("descriptor not in range or currently unused")
}

// Setprim sets the primary system partition of the SIF file
func Setprim(descr uint64, file string) error {
	fimg, err := sif.LoadContainer(file, false)
	if err != nil {
		return err
	}
	defer func() {
		if err := fimg.UnloadContainer(); err != nil {
			fmt.Println("Error unloading container: ", err)
		}
	}()

	for _, v := range fimg.DescrArr {
		if !v.Used {
			continue
		} else if v.ID == uint32(descr) {
			if err := fimg.SetPrimPart(uint32(descr)); err != nil {
				return err
			}

			return nil
		}
	}

	return fmt.Errorf("descriptor not in range or currently unused")
}
