/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package cli

import (
	"fmt"
	"log"
	"os/user"

	"github.com/spf13/cobra"
)

// Local Flags for instance start command
var (
	bindPath    []string
	homePath    string
	overlayPath string
	scratchPath string
	workdirPath string
	hostname    string
	nvidia      bool

	isBoot      bool
	isFakeroot  bool
	isContained bool
	isWritable  bool

	netNamespace  bool
	utsNamespace  bool
	userNamespace bool

	allowSUID bool
	keepPrivs bool
	noPrivs   bool
	addCaps   []string
	dropCaps  []string
)

var instanceStartExamples string = `
    $ singularity instance.start /tmp/my-sql.img mysql

    $ singularity shell instance://mysql
    Singularity my-sql.img> pwd
    /home/mibauer/mysql
    Singularity my-sql.img> ps
    PID TTY          TIME CMD
      1 pts/0    00:00:00 sinit
      2 pts/0    00:00:00 bash
      3 pts/0    00:00:00 ps
    Singularity my-sql.img> 
    
    $ singularity instance.stop /tmp/my-sql.img mysql
    Stopping /tmp/my-sql.img mysql`

func getHomeDir() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
		return ""
	}

	return usr.HomeDir
}

func init() {
	instanceStartCmds := []*cobra.Command{
		instanceStartCmd,
		instanceDotStartCmd,
	}

	for _, cmd := range instanceStartCmds {
		// -B|--bind
		cmd.LocalFlags().StringSliceVarP(&bindPath, "bind", "B", []string{}, "Path(s) to bind mount inside container")

		// -c|--contain
		cmd.LocalFlags().BoolVarP(&isContained, "contain", "c", false, "Use minimal /dev and empty other directories (e.g. /tmp and $HOME).")

		// -H|--home
		cmd.LocalFlags().StringVarP(&homePath, "home", "H", getHomeDir(), "A home directory specification")

		// -n|--net
		cmd.LocalFlags().BoolVarP(&netNamespace, "net", "n", false, "Run container in a new NET namespace")

		// --uts
		cmd.LocalFlags().BoolVar(&utsNamespace, "uts", false, "Run container in a new UTS namespace")

		// --nv
		cmd.LocalFlags().BoolVar(&nvidia, "nv", false, "Enable experimental Nvidia support")

		// -o|--overlay
		cmd.LocalFlags().StringVarP(&overlayPath, "overlay", "o", "", "Use a persistent overlayFS via a writable image")

		// -S|--scratch
		cmd.LocalFlags().StringVarP(&scratchPath, "scratch", "S", "", "Include a scratch directory within the container")

		// -W|--workdir
		cmd.LocalFlags().StringVarP(&workdirPath, "workdir", "W", "", "Working directory to be used for /tmp, /var/tmp")

		// -w|--writable // Not applicable in 3.x
		//cmd.LocalFlags().BoolVarP(&isWritable, "writable", "-w", false, )

		// -u|--userns
		cmd.LocalFlags().BoolVarP(&userNamespace, "userns", "u", false, "Run container in a new USER namespace")

		// --hostname
		cmd.LocalFlags().StringVar(&hostname, "hostname", "", "Set container hostname")

		// --boot
		cmd.LocalFlags().BoolVar(&isBoot, "boot", false, "Execute /sbin/init to boot container (root only)")

		// -f|--fakeroot
		cmd.LocalFlags().BoolVarP(&isFakeroot, "fakeroot", "f", false, `Run container in new user namespace as uid 0`)

		// --keep-privs
		cmd.LocalFlags().BoolVar(&keepPrivs, "keep-privs", false, "Let root user keep privileges in container")

		// --no-privs
		cmd.LocalFlags().BoolVar(&noPrivs, "no-privs", true, "Drop all privileges from root user in container")

		// --add-caps
		cmd.LocalFlags().StringSliceVar(&addCaps, "add-caps", []string{}, "A comma separated capability list to add")

		// --drop-caps
		cmd.LocalFlags().StringSliceVar(&dropCaps, "drop-caps", []string{}, "A comma separated capability list to drop")

		// --allow-setuid
		cmd.LocalFlags().BoolVar(&allowSUID, "allow-suid", false, "Allow setuid binaries in container (root only)")
	}

	singularityCmd.AddCommand(instanceDotStartCmd)
}

var instanceStartCmd = &cobra.Command{
	Use:  "start [options...] <container path> <instance name>",
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("starting instance")
	},
	Example: instanceStartExamples,
}

var instanceDotStartCmd = &cobra.Command{
	Use:  "instance.start [options...] <container path> <instance name>",
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("starting instance")
	},
	Example: instanceStartExamples,
	Hidden:  true,
}
