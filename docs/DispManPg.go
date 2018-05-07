/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package docs

import (
	"fmt"
	"os/exec"

	"github.com/golang/glog"
)

// DispManPage will display the man page related to the first argument or an
// error if the man page system is not installed and configured properly.
func DispManPg(pg string) {

	out, err := exec.Command("man", pg).Output()
	if err != nil {
		glog.Info("ERROR: please make sure that 'man' is installed on your system and the \nsingularity man pages are on your MANPATH")
		glog.Fatal(err)
	}

	fmt.Printf("%s\n", out)
}
