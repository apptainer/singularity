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
	config "github.com/singularityware/singularity/internal/pkg/runtime/engine/singularity/config"
	"github.com/singularityware/singularity/pkg/configs"

	"github.com/spf13/cobra"
    "github.com/singularityware/singularity/docs"
)

var formats string = `

*.sqsh              SquashFS format.  Native to Singularity 2.4+

*.img               This is the native Singularity image format for all
                    Singularity versions < 2.4.

*.tar\*              Tar archives are exploded to a temporary directory and
                    run within that directory (and cleaned up after). The
                    contents of the archive is a root file system with root
                    being in the current directory. All compression
                    suffixes are supported.

directory/          Container directories that contain a valid root file
                    system.

instance://*        A local running instance of a container. (See the
                    instance command group.)

shub://*            A container hosted on Singularity Hub

docker://*          A container hosted on Docker Hub`

var execUse string = `exec [exec options...] <container> ...`

var execShort string = `execute any program within the given container image`

var execLong string = `
singularity exec supports the following formats:` + formats

var execExamples string = `
$ singularity exec /tmp/Debian.img cat /etc/debian_version
$ singularity exec /tmp/Debian.img python ./hello_world.py
$ cat hello_world.py | singularity exec /tmp/Debian.img python
$ sudo singularity exec --writable /tmp/Debian.img apt-get update
$ singularity exec instance://my_instance ps -ef`

var shellUse string = `shell [shell options...] <container>`

var shellShort string = `obtain an interactive shell (/bin/bash) within the container image`

var shellLong string = `
singularity shell supports the following formats:` + formats

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

var runUse string = `run [run options...] <container>`

var runShort string = ``

var runLong string = `
This command will launch a Singularity container and execute a runscript
if one is defined for that container. The runscript is a file at
'/singularity'. If this file is present (and executable) then this
command will execute that file within the container automatically. All
arguments following the container name will be passed directly to the
runscript.

singularity run accepts the following container formats:` + formats

var runExamples string = `
# Here we see that the runscript prints "Hello world: "
$ singularity exec /tmp/Debian.img cat /singularity
#!/bin/sh
echo "Hello world: "

# It runs with our inputs when we run the image
$ singularity run /tmp/Debian.img one two three
Hello world: one two three

# Note that this does the same thing
$ ./tmp/Debian.img one two three
`

func init() {
	actionCmds := []*cobra.Command{
		ExecCmd,
		ShellCmd,
		RunCmd,
	}

    // TODO : the next n lines of code are repeating too much but I don't 
    // know how to shorten them tonight 
    execHelp := func(c *cobra.Command, args []string) {
        docs.DispManPg("singularity-exec")
    }
    ExecCmd.SetHelpFunc(execHelp)

    shellHelp := func(c *cobra.Command, args []string) {
        docs.DispManPg("singularity-shell")
    }
    ShellCmd.SetHelpFunc(shellHelp)

    runHelp := func(c *cobra.Command, args []string) {
        docs.DispManPg("singularity-run")
    }
    RunCmd.SetHelpFunc(runHelp)

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

	SingularityCmd.AddCommand(ExecCmd)
	SingularityCmd.AddCommand(ShellCmd)
	SingularityCmd.AddCommand(RunCmd)

    ExecCmd.Flags().SetInterspersed(false)
    ShellCmd.Flags().SetInterspersed(false)
    RunCmd.Flags().SetInterspersed(false)

}

// execCmd represents the exec command
var ExecCmd = &cobra.Command{
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		a := append([]string{"/.singularity.d/actions/exec"}, args[1:]...)
		execWrapper(cmd, args[0], a)
	},

	Use: execUse,
    Short: execShort,
    Long: execLong,
	Example: execExamples,
}

// shellCmd represents the shell command
var ShellCmd = &cobra.Command{
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		a := append([]string{"/.singularity.d/actions/shell"}, args[1:]...)
		execWrapper(cmd, args[0], a)
	},

	Use: shellUse,
    Short: shellShort,
    Long: shellLong,
	Example: shellExamples,
}

// runCmd represents the run command
var RunCmd = &cobra.Command{
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		a := append([]string{"/.singularity.d/actions/run"}, args[1:]...)
		execWrapper(cmd, args[0], a)
	},

	Use: runUse,
    Short: runShort,
    Long: runLong,
	Example: runExamples,
}

// TODO: Let's stick this in another file so that that CLI is just CLI
func execWrapper(cobraCmd *cobra.Command, image string, args []string) {
	lvl := "0"
	if configs.BUILDTREE == "" {
		log.Fatal("buildtree not defined at compile time, exiting")
	}

	wrapper := configs.BUILDTREE + "/wrapper-suid"

	oci, runtime := config.NewSingularityConfig("new")
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
		wrapper = configs.BUILDTREE + "/wrapper"
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
