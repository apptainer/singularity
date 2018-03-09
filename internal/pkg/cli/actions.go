/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package cli

import (
	"fmt"

	sflags "github.com/singularityware/singularity/internal/pkg/cli/flags"
	"github.com/spf13/cobra"
)

var execExamples string = `
      $ singularity exec /tmp/Debian.img cat /etc/debian_version
      $ singularity exec /tmp/Debian.img python ./hello_world.py
      $ cat hello_world.py | singularity exec /tmp/Debian.img python
      $ sudo singularity exec --writable /tmp/Debian.img apt-get update
      $ singularity exec instance://my_instance ps -ef`

var shellExamples string = `
      $ singularity shell /tmp/Debian.img
      Singularity/Debian.img> pwd
      /home/gmk/test
      Singularity/Debian.img> exit
      
      $ singularity shell -C /tmp/Debian.img
      Singularity/Debian.img> pwd
      /home/gmk
      Singularity/Debian.img> ls -l
      total 0
      Singularity/Debian.img> exit
      
      $ sudo singularity shell -w /tmp/Debian.img
      $ sudo singularity shell --writable /tmp/Debian.img
      
      $ singularity shell instance://my_instance 
      
      $ singularity shell instance://my_instance
      Singularity: Invoking an interactive shell within container...
      Singularity container:~> ps -ef
      UID        PID  PPID  C STIME TTY          TIME CMD
      ubuntu       1     0  0 20:00 ?        00:00:00 /usr/local/bin/singularity/bin/sinit
      ubuntu       2     0  0 20:01 pts/8    00:00:00 /bin/bash --norc
      ubuntu       3     2  0 20:02 pts/8    00:00:00 ps -ef`

var runExamples string = `
`

func init() {
	actionCmds := []*cobra.Command{
		execCmd,
		shellCmd,
		runCmd,
	}

	for _, cmd := range actionCmds {
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("bind"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("contain"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("containall"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("cleanenv"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("home"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("ipc"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("net"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("nv"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("overlay"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("pid"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("uts"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("pwd"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("scratch"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("userns"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("workdir"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("hostname"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("fakeroot"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("keep-privs"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("no-privs"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("add-caps"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("drop-caps"))
		cmd.PersistentFlags().AddFlag(sflags.Flags.Lookup("allow-setuid"))
	}

	singularityCmd.AddCommand(execCmd)
	singularityCmd.AddCommand(shellCmd)
	singularityCmd.AddCommand(runCmd)

}

// execCmd represents the exec command
var execCmd = &cobra.Command{
	Use: "exec [exec options...]",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("exec called")
	},
	Example: execExamples,
}

// shellCmd represents the shell command
var shellCmd = &cobra.Command{
	Use: "shell [shell options...]",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("shell called")
	},
	Example: shellExamples,
}

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use: "run [run options]",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("run called")
	},
}
