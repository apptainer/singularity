// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"os"
	"os/exec"

	"github.com/singularityware/singularity/src/docs"
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
	uuid     string
)

func init() {
	SifCmd.Flags().SetInterspersed(false)
	SingularityCmd.AddCommand(SifCmd)
	// Create
	//SifCmd.AddCommand(SifCreate)
	/*SifCreate.Flags().StringVarP(&deffile, "deffile", "D", "", "include definitions file 'deffile'")
	SifCreate.Flags().StringVarP(&partfile, "partfile", "P", "", "include file system partition `partfile'")
	SifCreate.Flags().StringVarP(&content, "CONTENT", "c", "", "freeform partition content string")
	SifCreate.Flags().StringVarP(&fstype, "FSTYPE", "f", "", "filesystem type: EXT3, SQUASHFS")
	SifCreate.Flags().StringVarP(&parttype, "PARTTYPE", "p", "", "filesystem partition type: SYSTEM, DATA, OVERLAY")
	SifCreate.Flags().StringVarP(&uuid, "uuid", "u", "", "pass a uuid to use instead of generating a new one")
	// List
	SifCmd.AddCommand(SifList)
	// Dump
	SifCmd.AddCommand(SifDump)
	// Header ifHeader
	SifCmd.AddCommand(SifHeader)
	// Info
	SifCmd.AddCommand(SifInfo)
	// Del
	SifCmd.AddCommand(SifDel)*/
}

var sif = buildcfg.SBINDIR + "/sif"

// SifCmd represent the sif CLI cmd
var SifCmd = &cobra.Command{
	Use:   docs.SifUse,
	Short: docs.SifShort,
	Long:  docs.SifLong,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sif := exec.Command(sif, args...)

		sif.Stdout = os.Stdout
		sif.Stderr = os.Stderr
		sif.Stdin = os.Stdin

		if err := sif.Start(); err != nil {
			sylog.Fatalf("failed to start sif: %v\n", err)
		}
		if err := sif.Wait(); err != nil {
			sylog.Fatalf("sif failed: %v\n", err)
		}

	},
	DisableFlagParsing: true,
}

/*// SifCreate sif create cmd
var SifCreate = &cobra.Command{
	Use:     docs.SifCreateUse,
	Short:   docs.SifCreateShort,
	Example: docs.SifCreateExample,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var argv []string
		argv = append(argv, "create")
		if partfile != "" {
			argv = append(argv, []string{"-P", partfile}...)
		}
		if fstype != "" {
			argv = append(argv, []string{"-f", fstype}...)
		}
		if parttype != "" {
			argv = append(argv, []string{"-p", parttype}...)
		}
		if content != "" {
			argv = append(argv, []string{"-c", content}...)
		}
		if uuid != "" {
			argv = append(argv, []string{"-c", uuid}...)
		}
		argv = append(argv, args...)
		SifCmd := exec.Command(sif, argv...)
		SifCmd.Stdout = os.Stdout
		SifCmd.Stderr = os.Stderr

	},
}

// SifList sif list subcommand
var SifList = &cobra.Command{
	Use:     docs.SifListUse,
	Short:   docs.SifListShort,
	Example: docs.SifListExample,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		argv := []string{"list", args[0]}
		SifCmd := exec.Command(sif, argv...)
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
	Use:     docs.SifInfoUse,
	Short:   docs.SifInfoShort,
	Example: docs.SifInfoExample,
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {

		argv := append([]string{"info"}, args...)
		SifCmd := exec.Command(sif, argv...)
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
	Use:   docs.SifDumpUse,
	Short: docs.SifDumpShort,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {

		argv := append([]string{"dump"}, args...)
		SifCmd := exec.Command(sif, argv...)
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
	Use:   docs.SifDelUse,
	Short: docs.SifDelShort,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {

		argv := append([]string{"del"}, args...)
		SifCmd := exec.Command(sif, argv...)
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
	Use:     docs.SifHeaderUse,
	Short:   docs.SifHeaderShort,
	Example: docs.SifHeaderExample,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		argv := []string{"header", args[0]}
		SifCmd := exec.Command(sif, argv...)
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
*/
