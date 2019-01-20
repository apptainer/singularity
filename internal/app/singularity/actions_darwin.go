// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func handleOCI(cmd *cobra.Command, u string) (string, error) {
	return "", fmt.Errorf("unsupported on this platform")
}

func handleLibrary(u string) (string, error) {
	return "", fmt.Errorf("unsupported on this platform")
}

func handleShub(u string) (string, error) {
	return "", fmt.Errorf("unsupported on this platform")
}

func handleNet(u string) (string, error) {
	return "", fmt.Errorf("unsupported on this platform")
}

// TODO: Let's stick this in another file so that that CLI is just CLI
func execStarter(cobraCmd *cobra.Command, image string, args []string, name string) {
}
