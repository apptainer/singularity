// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
// Copyright (c) 2017, Yannick Cote <yhcote@gmail.com> All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package siftool

import (
	"github.com/spf13/cobra"
	"github.com/sylabs/sif/internal/app/siftool"
)

// New implements 'siftool new' sub-command
func New() *cobra.Command {
	return &cobra.Command{
		Use:   "new <containerfile>",
		Short: "Create a new empty SIF image file",
		Args:  cobra.ExactArgs(1),

		RunE: func(cmd *cobra.Command, args []string) error {
			return siftool.New(args[0])
		},
		DisableFlagsInUseLine: true,
	}
}
