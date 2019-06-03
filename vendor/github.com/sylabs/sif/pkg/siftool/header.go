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

// Header implements 'siftool header' sub-command
func Header() *cobra.Command {
	return &cobra.Command{
		Use:   "header <containerfile>",
		Short: "Display SIF global headers",
		Args:  cobra.ExactArgs(1),

		RunE: func(cmd *cobra.Command, args []string) error {
			return siftool.Header(args[0])
		},
		DisableFlagsInUseLine: true,
	}
}
