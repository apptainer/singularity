// Copyright (c) 2020, Control Command Inc. All rights reserved.
// Copyright (c) 2019-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/hpcng/singularity/docs"
	"github.com/hpcng/singularity/internal/app/singularity"
	"github.com/hpcng/singularity/internal/pkg/client/library"
	"github.com/hpcng/singularity/internal/pkg/util/interactive"
	"github.com/hpcng/singularity/pkg/cmdline"
	"github.com/hpcng/singularity/pkg/sylog"
	"github.com/spf13/cobra"
)

func init() {
	addCmdInit(func(cmdManager *cmdline.CommandManager) {
		cmdManager.RegisterCmd(deleteImageCmd)
		cmdManager.RegisterFlagForCmd(&deleteForceFlag, deleteImageCmd)
		cmdManager.RegisterFlagForCmd(&deleteImageArchFlag, deleteImageCmd)
		cmdManager.RegisterFlagForCmd(&deleteImageTimeoutFlag, deleteImageCmd)
		cmdManager.RegisterFlagForCmd(&deleteLibraryURIFlag, deleteImageCmd)
	})
}

var (
	deleteForce     bool
	deleteForceFlag = cmdline.Flag{
		ID:           "deleteForceFlag",
		Value:        &deleteForce,
		DefaultValue: false,
		Name:         "force",
		ShortHand:    "F",
		Usage:        "delete image without confirmation",
		EnvKeys:      []string{"FORCE"},
	}
)

var (
	deleteImageArch     string
	deleteImageArchFlag = cmdline.Flag{
		ID:           "deleteImageArchFlag",
		Value:        &deleteImageArch,
		DefaultValue: runtime.GOARCH,
		Name:         "arch",
		ShortHand:    "A",
		Usage:        "specify requested image arch",
		EnvKeys:      []string{"ARCH"},
	}
)

var (
	deleteImageTimeout     int
	deleteImageTimeoutFlag = cmdline.Flag{
		ID:           "deleteImageTimeoutFlag",
		Value:        &deleteImageTimeout,
		DefaultValue: 15,
		Name:         "timeout",
		ShortHand:    "T",
		Hidden:       true,
		Usage:        "specify delete timeout in seconds",
		EnvKeys:      []string{"TIMEOUT"},
	}
)

var (
	deleteLibraryURI     string
	deleteLibraryURIFlag = cmdline.Flag{
		ID:           "deleteLibraryURIFlag",
		Value:        &deleteLibraryURI,
		DefaultValue: "",
		Name:         "library",
		Usage:        "delete images from the provided library",
		EnvKeys:      []string{"LIBRARY"},
	}
)

var deleteImageCmd = &cobra.Command{
	Use:     docs.DeleteUse,
	Short:   docs.DeleteShort,
	Long:    docs.DeleteLong,
	Example: docs.DeleteExample,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sylog.Debugf("Using library service URI: %s", deleteLibraryURI)

		imageRef, err := library.NormalizeLibraryRef(args[0])
		if err != nil {
			sylog.Fatalf("Error parsing library ref: %v", err)
		}

		if deleteLibraryURI != "" && imageRef.Host != "" {
			sylog.Fatalf("Conflicting arguments; do not use --library with a library URI containing host name")
		}

		var libraryURI string
		if deleteLibraryURI != "" {
			libraryURI = deleteLibraryURI
		} else if imageRef.Host != "" {
			// override libraryURI if ref contains host name
			libraryURI = "https://" + imageRef.Host
		}

		r := fmt.Sprintf("%s:%s", imageRef.Path, imageRef.Tags[0])

		if !deleteForce {
			y, err := interactive.AskYNQuestion("n", "Are you sure you want to delete %s (%s) [N/y] ", r, deleteImageArch)
			if err != nil {
				sylog.Fatalf(err.Error())
			}
			if y == "n" {
				return
			}
		}

		libraryConfig, err := getLibraryClientConfig(libraryURI)
		if err != nil {
			sylog.Fatalf("Error while getting library client config: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.TODO(), time.Duration(deleteImageTimeout)*time.Second)
		defer cancel()

		if err := singularity.DeleteImage(ctx, libraryConfig, r, deleteImageArch); err != nil {
			sylog.Fatalf("Unable to delete image from library: %s\n", err)
		}

		sylog.Infof("Image %s (%s) deleted.", r, deleteImageArch)
	},
}
