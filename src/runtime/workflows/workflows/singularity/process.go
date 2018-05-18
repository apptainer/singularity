/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package runtime

import (
	"log"
	"os"
	"syscall"
)

func (engine *RuntimeEngine) PrestartProcess() error {
	/* seccomp setup goes here */
	return nil
}

func (engine *RuntimeEngine) StartProcess() error {
	//    if cconf.isInstance == C.uchar(0) {
	os.Setenv("PS1", "shell> ")
	os.Chdir("/")
	args := engine.OciConfig.RuntimeOciSpec.Process.Args
	err := syscall.Exec(args[0], args, os.Environ())
	if err != nil {
		log.Fatalln("exec failed:", err)
	}
	/*    }  else {
	          err := syscall.Exec("/bin/sleep", []string{"/bin/sleep", "60"}, os.Environ())
	          if err != nil {
	              log.Fatalln("exec failed:", err)
	          }
	      }
	*/
	return nil
}
