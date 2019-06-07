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
	"io"
	"os"
	"time"

	"github.com/sylabs/sif/pkg/sif"
)

// Header displays a SIF file global header
func Header(file string) error {
	fimg, err := sif.LoadContainer(file, true)
	if err != nil {
		return err
	}
	defer func() {
		if err := fimg.UnloadContainer(); err != nil {
			fmt.Println("Error unloading container: ", err)
		}
	}()

	fmt.Print(fimg.FmtHeader())

	return nil
}

// List displays a list of all active descriptors from a SIF file
func List(file string) error {
	fimg, err := sif.LoadContainer(file, true)
	if err != nil {
		return err
	}
	defer func() {
		if err := fimg.UnloadContainer(); err != nil {
			fmt.Println("Error unloading container: ", err)
		}
	}()

	fmt.Println("Container id:", fimg.Header.ID)
	fmt.Println("Created on:  ", time.Unix(fimg.Header.Ctime, 0))
	fmt.Println("Modified on: ", time.Unix(fimg.Header.Mtime, 0))
	fmt.Println("----------------------------------------------------")

	fmt.Println("Descriptor list:")

	fmt.Print(fimg.FmtDescrList())

	return nil
}

// Info displays detailed info about a descriptor from a SIF file
func Info(descr uint64, file string) error {
	fimg, err := sif.LoadContainer(file, true)
	if err != nil {
		return err
	}
	defer func() {
		if err := fimg.UnloadContainer(); err != nil {
			fmt.Println("Error unloading container: ", err)
		}
	}()

	fmt.Print(fimg.FmtDescrInfo(uint32(descr)))

	return nil
}

// Dump extracts and outputs a data object from a SIF file
func Dump(descr uint64, file string) error {
	fimg, err := sif.LoadContainer(file, true)
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
			if _, err := fimg.Fp.Seek(v.Fileoff, 0); err != nil {
				return fmt.Errorf("while seeking to data object: %s", err)
			}
			if _, err := io.CopyN(os.Stdout, fimg.Fp, v.Filelen); err != nil {
				return fmt.Errorf("while copying data object to stdout: %s", err)
			}

			return nil
		}
	}

	return fmt.Errorf("descriptor not in range or currently unused")
}
