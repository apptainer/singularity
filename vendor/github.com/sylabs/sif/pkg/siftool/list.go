// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// Copyright (c) 2018, Divya Cote <divya.cote@gmail.com> All rights reserved.
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

// List implements 'siftool list' sub-command
func List() *cobra.Command {
	return &cobra.Command{
		Use:   "list <containerfile>",
		Short: "List object descriptors from SIF files",
		Args:  cobra.ExactArgs(1),

		RunE: func(cmd *cobra.Command, args []string) error {
			return siftool.List(args[0])
		},
		DisableFlagsInUseLine: true,
	}
}
