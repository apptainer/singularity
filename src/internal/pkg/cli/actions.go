/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package cli

import (
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"
	runtimeconfig "github.com/singularityware/singularity/src/internal/pkg/runtime/engine/singularity/config"
	"github.com/singularityware/singularity/src/pkg/buildcfg"

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
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("bind"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("contain"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("containall"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("cleanenv"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("home"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("ipc"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("net"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("nv"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("overlay"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("pid"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("uts"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("pwd"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("scratch"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("userns"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("workdir"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("hostname"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("fakeroot"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("keep-privs"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("no-privs"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("add-caps"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("drop-caps"))
		cmd.PersistentFlags().AddFlag(actionFlags.Lookup("allow-setuid"))
	}

	singularityCmd.AddCommand(execCmd)
	singularityCmd.AddCommand(shellCmd)
	singularityCmd.AddCommand(runCmd)

}

// execCmd represents the exec command
var execCmd = &cobra.Command{
	Use:  "exec [exec options...] <container> ...",
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		a := append([]string{"/.singularity.d/actions/exec"}, args[1:]...)
		execWrapper(cmd, args[0], a)
	},
	Example: execExamples,
}

// shellCmd represents the shell command
var shellCmd = &cobra.Command{
	Use:  "shell [shell options...] <container>",
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		a := append([]string{"/.singularity.d/actions/shell"}, args[1:]...)
		execWrapper(cmd, args[0], a)
	},
	Example: shellExamples,
}

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:  "run [run options...] <container>",
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		a := append([]string{"/.singularity.d/actions/run"}, args[1:]...)
		execWrapper(cmd, args[0], a)
	},
}

func execWrapper(cobraCmd *cobra.Command, image string, args []string) {
	lvl := "0"
	if buildcfg.BUILDDIR == "" {
		log.Fatal("buildtree not defined at compile time, exiting")
	}

	wrapper := buildcfg.SBINDIR + "/wrapper-suid"

	oci, runtime := runtimeconfig.NewSingularityConfig("new")
	oci.Root.SetPath(image)
	oci.Process.SetArgs(args)
	oci.Process.SetNoNewPrivileges(true)

	oci.RuntimeOciSpec.Linux = &specs.Linux{}
	namespaces := []specs.LinuxNamespace{}
	if NetNamespace {
		namespaces = append(namespaces, specs.LinuxNamespace{Type: specs.NetworkNamespace})
	}
	if UtsNamespace {
		namespaces = append(namespaces, specs.LinuxNamespace{Type: specs.UTSNamespace})
	}
	if PidNamespace {
		namespaces = append(namespaces, specs.LinuxNamespace{Type: specs.PIDNamespace})
	}
	if IpcNamespace {
		namespaces = append(namespaces, specs.LinuxNamespace{Type: specs.IPCNamespace})
	}
	if UserNamespace {
		namespaces = append(namespaces, specs.LinuxNamespace{Type: specs.UserNamespace})
		wrapper = buildcfg.SBINDIR + "/wrapper"
	}
	oci.RuntimeOciSpec.Linux.Namespaces = namespaces

	cmd := exec.Command(wrapper)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if verbose {
		lvl = "2"
	}
	if debug {
		lvl = "5"
	}

	cmd.Env = []string{"SINGULARITY_MESSAGELEVEL=" + lvl, "SRUNTIME=singularity"}
	j, err := runtime.GetConfig()
	if err != nil {
		log.Fatalln(err)
	}

	cmd.Stdin = strings.NewReader(string(j))
	err = cmd.Run()
	if err != nil {
		log.Fatalln(err)
	}
}
