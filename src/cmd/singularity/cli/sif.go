// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"os"
	"os/exec"

	"github.com/singularityware/singularity/src/pkg/buildcfg"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/spf13/cobra"
)

var (
	deffile  string
	partfile string
	content  string
	fstype   string
	parttype string
	uuID     string
)

func init() {
	SifCmd.Flags().SetInterspersed(false)
	SingularityCmd.AddCommand(SifCmd)
	// Create
	SifCmd.AddCommand(SifCreate)
	SifCreate.Flags().StringVarP(&deffile, "deffile", "D", "", "include definitions file 'deffile'")
	SifCreate.Flags().StringVarP(&partfile, "partfile", "P", "", "include file system partition `partfile'")
	SifCreate.Flags().StringVarP(&content, "CONTENT", "c", "", "freeform partition content string")
	SifCreate.Flags().StringVarP(&fstype, "FSTYPE", "f", "", "filesystem type: EXT3, SQUASHFS")
	SifCreate.Flags().StringVarP(&parttype, "PARTTYPE", "p", "", "filesystem partition type: SYSTEM, DATA, OVERLAY")
	SifCreate.Flags().StringVarP(&uuID, "uuid", "u", "", "pass a uuid to use instead of generating a new one")
	// List
	SifCmd.AddCommand(SifList)
	// Dump
	SifCmd.AddCommand(SifDump)
	// Header ifHeader
	SifCmd.AddCommand(SifHeader)
	// Info
	SifCmd.AddCommand(SifInfo)
	// Del
	SifCmd.AddCommand(SifDel)
}

var sif = buildcfg.SBINDIR + "/sif"
var (
	sifCreateExample = `
sif create -P /tmp/fs.squash -f "SQUASHFS" -p "SYSTEM" -c "Linux" /tmp/container.sif`
	sifListExample = `
sif list /tmp/container.sif
Container uuid: 2b88f62f-be4f-4143-8a7a-061c49a68249
Created on: Fri May 25 17:23:04 2018
Modified on: Fri May 25 17:23:04 2018
----------------------------------------------------

Descriptor list:
ID   |GROUP   |LINK    |SIF POSITION (start-end)  |TYPE
------------------------------------------------------------------------------
1    |1       |NONE    |3328-2010367              |FS.Img (Squashfs/System)`
	sifInfoExample = `
sif info 1 container.sif
Descriptor info:
---------------------------
desc type: FS.Img
desc id: 1
group id: 1
link: NONE
fileoff: 3328
filelen: 2007040
fstype: Squashfs
parttype: System
content: LINUX
---------------------------`
	sifHeaderExample = `
sif header hah.sif
================ SIF Header ================
launch: #!/usr/bin/env run-singularity

magic: SIF_MAGIC
version: 0
arch: AMD64
uuid: 2b88f62f-be4f-4143-8a7a-061c49a68249
creation time: Fri May 25 17:23:04 2018
modification time: Fri May 25 17:23:04 2018
number of descriptors: 1
start of descriptors in file: 120
length of descriptors in file: 104
start of data in file: 3328
length of data in file: 1MB
============================================`
)

// SifCmd represent the sif CLI cmd
var SifCmd = &cobra.Command{
	Use:  "sif",
	Args: cobra.MinimumNArgs(1),
	Run:  nil,
}

// SifCreate sif create cmd
var SifCreate = &cobra.Command{
	Use:     "create [option] <file>",
	Short:   "Create a new sif file with input data objects",
	Example: sifCreateExample,
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var argc []string
		argc = append(argc, "create")
		if partfile != "" {
			argc = append(argc, []string{"-P", partfile}...)
		}
		if fstype != "" {
			argc = append(argc, []string{"-f", fstype}...)
		}
		if parttype != "" {
			argc = append(argc, []string{"-p", parttype}...)
		}
		if content != "" {
			argc = append(argc, []string{"-c", content}...)
		}
		if uuID != "" {
			argc = append(argc, []string{"-c", uuID}...)
		}
		argc = append(argc, args...)
		SifCmd := exec.Command(sif, argc...)
		SifCmd.Stdout = os.Stdout
		SifCmd.Stderr = os.Stderr

		if err := SifCmd.Start(); err != nil {
			sylog.Fatalf("%v", err)
		}
		if err := SifCmd.Wait(); err != nil {
			sylog.Fatalf("%v", err)
		}
	},
}

// SifList sif list subcommand
var SifList = &cobra.Command{
	Use:     "list <file>",
	Short:   "List SIF data descriptors from an input SIF file",
	Example: sifListExample,
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		argc := []string{"list", args[0]}
		SifCmd := exec.Command(sif, argc...)
		SifCmd.Stdout = os.Stdout
		SifCmd.Stderr = os.Stderr

		if err := SifCmd.Start(); err != nil {
			sylog.Fatalf("%v", err)
		}
		if err := SifCmd.Wait(); err != nil {
			sylog.Fatalf("%v", err)
		}
	},
}

// SifInfo sif info subcommand
var SifInfo = &cobra.Command{
	Use:     "info [id] <file>",
	Short:   "Print data object descriptor info",
	Example: sifInfoExample,
	Args:    cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {

		argc := append([]string{"info"}, args...)
		SifCmd := exec.Command(sif, argc...)
		SifCmd.Stdout = os.Stdout
		SifCmd.Stderr = os.Stderr

		if err := SifCmd.Start(); err != nil {
			sylog.Fatalf("%v", err)
		}
		if err := SifCmd.Wait(); err != nil {
			sylog.Fatalf("%v", err)
		}
	},
}

// SifDump sif dump subcommand
var SifDump = &cobra.Command{
	Use:     "dump [id] <file>",
	Short:   "Display data object content",
	Example: sifInfoExample,
	Args:    cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {

		argc := append([]string{"info"}, args...)
		SifCmd := exec.Command(sif, argc...)
		SifCmd.Stdout = os.Stdout
		SifCmd.Stderr = os.Stderr

		if err := SifCmd.Start(); err != nil {
			sylog.Fatalf("%v", err)
		}
		if err := SifCmd.Wait(); err != nil {
			sylog.Fatalf("%v", err)
		}
	},
}

// SifDel sif del subcommand
var SifDel = &cobra.Command{
	Use:     "del [id] <file>",
	Short:   "Delete a specified set of descriptor+object",
	Example: sifInfoExample,
	Args:    cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {

		argc := append([]string{"del"}, args...)
		SifCmd := exec.Command(sif, argc...)
		SifCmd.Stdout = os.Stdout
		SifCmd.Stderr = os.Stderr

		if err := SifCmd.Start(); err != nil {
			sylog.Fatalf("%v", err)
		}
		if err := SifCmd.Wait(); err != nil {
			sylog.Fatalf("%v", err)
		}
	},
}

// SifHeader sif header subcommand
var SifHeader = &cobra.Command{
	Use:     "header <file>",
	Short:   "Display SIF header",
	Example: sifHeaderExample,
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		argc := []string{"list", args[0]}
		SifCmd := exec.Command(sif, argc...)
		SifCmd.Stdout = os.Stdout
		SifCmd.Stderr = os.Stderr

		if err := SifCmd.Start(); err != nil {
			sylog.Fatalf("%v", err)
		}
		if err := SifCmd.Wait(); err != nil {
			sylog.Fatalf("%v", err)
		}
	},
}
