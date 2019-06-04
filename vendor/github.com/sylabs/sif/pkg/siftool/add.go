// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
// Copyright (c) 2017, Yannick Cote <yhcote@gmail.com> All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package siftool

import (
	"github.com/spf13/cobra"
	"github.com/sylabs/sif/internal/app/siftool"
	"github.com/sylabs/sif/pkg/sif"
)

// Add implements 'siftool add' sub-command
func Add() *cobra.Command {
	ret := &cobra.Command{
		Use:   "add [OPTIONS] <containerfile> <dataobjectfile>",
		Short: "Add a data object to a SIF file",
		Args:  cobra.ExactArgs(2),
	}

	opts := siftool.AddOptions{
		Datatype: ret.Flags().Int64("datatype", -1, `the type of data to add
[NEEDED, no default]:
  1-Deffile,   2-EnvVar,    3-Labels,
  4-Partition, 5-Signature, 6-GenericJSON`),
		Parttype: ret.Flags().Int64("parttype", -1, `the type of partition (with -datatype 4-Partition)
[NEEDED, no default]:
  1-System,    2-PrimSys,   3-Data,
  4-Overlay`),
		Partfs: ret.Flags().Int64("partfs", -1, `the filesystem used (with -datatype 4-Partition)
[NEEDED, no default]:
  1-Squash,    2-Ext3,      3-ImmuObj,
  4-Raw`),
		Partarch: ret.Flags().Int64("partarch", -1, `the main architecture used (with -datatype 4-Partition)
[NEEDED, no default]:
  1-386,       2-amd64,     3-arm,
  4-arm64,     5-ppc64,     6-ppc64le,
  7-mips,      8-mipsle,    9-mips64,
  10-mips64le, 11-s390x`),
		Signhash: ret.Flags().Int64("signhash", -1, `the signature hash used (with -datatype 5-Signature)
[NEEDED, no default]:
  1-SHA256,    2-SHA384,    3-SHA512,
  4-BLAKE2S,   5-BLAKE2B`),
		Signentity: ret.Flags().String("signentity", "", `the entity that signs (with -datatype 5-Signature)
[NEEDED, no default]:
  example: 433FE984155206BD962725E20E8713472A879943`),
		Groupid:   ret.Flags().Int64("groupid", sif.DescrUnusedGroup, "set groupid [default: DescrUnusedGroup]"),
		Link:      ret.Flags().Int64("link", sif.DescrUnusedLink, "set link pointer [default: DescrUnusedLink]"),
		Alignment: ret.Flags().Int("alignment", 0, "set alignment constraint [default: aligned on page size]"),
		Filename:  ret.Flags().String("filename", "", "set logical filename/handle [default: input filename]"),
	}

	ret.RunE = func(cmd *cobra.Command, args []string) error {
		return siftool.Add(args[0], args[1], opts)
	}

	// function to set flag.DefVal to the "zero-value"
	fn := func(name, setdef string) {
		fl := ret.Flags().Lookup(name)
		if fl == nil {
			return
		}

		fl.DefValue = setdef
	}

	// set the DefVal fields for all the siftool add flags
	fn("datatype", "0")
	fn("parttype", "0")
	fn("partfs", "0")
	fn("partarch", "0")
	fn("signhash", "0")
	fn("signentity", "")
	fn("groupid", "0")
	fn("link", "0")
	fn("alignment", "0")

	return ret
}
