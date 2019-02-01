// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"fmt"
	"os"

	"github.com/sylabs/singularity/cmd/internal/cli"
)

func main() {
	if err := cli.SingularityCmd.GenBashCompletionFile(os.Args[1]); err != nil {
		fmt.Println(err)
		return
	}
}
