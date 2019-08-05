// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/app/singularity"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/cmdline"
)

var (
	cacheCleanDry   bool
	cacheCleanForce bool
	cacheCleanTypes []string
	cacheCleanNames []string
)

// -T|--type
var cacheCleanTypesFlag = cmdline.Flag{
	ID:           "cacheCleanTypes",
	Value:        &cacheCleanTypes,
	DefaultValue: []string{"all"},
	Name:         "type",
	ShortHand:    "T",
	Usage:        "a list of cache types to clean (possible values: library, oci, shub, blob, net, oras, all)",
}

// -N|--name
var cacheCleanNameFlag = cmdline.Flag{
	ID:           "cacheCleanNameFlag",
	Value:        &cacheCleanNames,
	DefaultValue: []string{},
	Name:         "name",
	ShortHand:    "N",
	Usage:        "specify a container cache to clean (will clear all cache with the same name)",
}

// -n|--dry-run
var cacheCleanDryFlag = cmdline.Flag{
	ID:           "cacheCleanDryFlag",
	Value:        &cacheCleanDry,
	DefaultValue: false,
	Name:         "dry-run",
	ShortHand:    "n",
	Usage:        "operate in dry run mode and do not actually clean the cache",
}

// --force
var cacheCleanForceFlag = cmdline.Flag{
	ID:           "cacheCleanForceFlag",
	Value:        &cacheCleanForce,
	DefaultValue: false,
	Name:         "force",
	Usage:        "suppress any prompts and clean the cache",
}

func init() {
	cmdManager.RegisterFlagForCmd(&cacheCleanTypesFlag, CacheCleanCmd)
	cmdManager.RegisterFlagForCmd(&cacheCleanNameFlag, CacheCleanCmd)
	cmdManager.RegisterFlagForCmd(&cacheCleanDryFlag, CacheCleanCmd)
	cmdManager.RegisterFlagForCmd(&cacheCleanForceFlag, CacheCleanCmd)
}

// CacheCleanCmd is 'singularity cache clean' and will clear your local singularity cache
var CacheCleanCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := cacheCleanCmd(); err != nil {
			os.Exit(2)
		}
	},

	Use:     docs.CacheCleanUse,
	Short:   docs.CacheCleanShort,
	Long:    docs.CacheCleanLong,
	Example: docs.CacheCleanExample,
}

func cacheCleanCmd() error {
	if !cacheCleanPrompt() {
		sylog.Infof("Cache cleanup cancelled.")
		return nil
	}

	// We create a handle to access the current image cache
	imgCache := getCacheHandle()
	err := singularity.CleanSingularityCache(imgCache, !cacheCleanDry, cacheCleanTypes, cacheCleanNames)
	if err != nil {
		sylog.Fatalf("Failed while clean cache: %v", err)
	}
	return nil
}

func cacheCleanPrompt() bool {
	if cacheCleanForce {
		return true
	}

	fmt.Print(`This will delete everything in your cache (containers from all sources and OCI blobs). 
Hint: You can see exactly what would be deleted by canceling and using the --dry-run option.
Do you want to continue? [N/y] `)

	r := bufio.NewReader(os.Stdin)
	input, err := r.ReadString('\n')
	if err != nil {
		sylog.Fatalf("Could not read user's input: %s", err)
	}

	return strings.ToLower(input) == "y\n"
}
