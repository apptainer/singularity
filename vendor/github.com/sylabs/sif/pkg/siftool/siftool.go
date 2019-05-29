// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
// Copyright (c) 2017, Yannick Cote <yhcote@gmail.com> All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Package siftool implements cobra.Command structs for the siftool functionality. This
// allows for easy inclusion of siftool functions in the singularity cli
package siftool

import (
	"github.com/spf13/cobra"
)

const (
	siftoolLong = `
  A set of commands are provided to display elements such as the SIF global 
  header, the data object descriptors and to dump data objects. It is also 
  possible to modify a SIF file via this tool via the add/del commands.`
)

// Siftool is a program for Singularity Image Format (SIF) file manipulation.
//
// A set of commands are provided to display elements such as the SIF global
// header, the data object descriptors and to dump data objects. It is also
// possible to modify a SIF file via this tool via the add/del commands.
func Siftool() *cobra.Command {
	// Siftool is a program for Singularity Image Format (SIF) file manipulation.
	//
	// A set of commands are provided to display elements such as the SIF global
	// header, the data object descriptors and to dump data objects. It is also
	// possible to modify a SIF file via this tool via the add/del commands.
	var Siftool = &cobra.Command{
		Use:                   "sif",
		Short:                 "siftool is a program for Singularity Image Format (SIF) file manipulation",
		Long:                  siftoolLong,
		Aliases:               []string{"siftool"},
		DisableFlagsInUseLine: true,
	}

	Siftool.AddCommand(Header())
	Siftool.AddCommand(List())
	Siftool.AddCommand(Info())
	Siftool.AddCommand(Dump())
	Siftool.AddCommand(New())
	Siftool.AddCommand(Add())
	Siftool.AddCommand(Del())
	Siftool.AddCommand(Setprim())

	return Siftool
}
