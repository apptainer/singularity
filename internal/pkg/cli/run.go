/*
Copyright (c) 2018, Sylabs, Inc. All rights reserved.
This software is licensed under a 3-clause BSD license.  Please
consult LICENSE file distributed with the sources of this project regarding
your rights to use or distribute this software.
*/

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use: "run",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("run called")
	},
}

func init() {
	singularityCmd.AddCommand(runCmd)

	// ultimately all of this should go into a seperate function to be shared
	// between exec, run, and shell
	runCmd.PersistentFlags().String("overlay", "o", "")
	runCmd.PersistentFlags().String("shell", "s", "")
	runCmd.PersistentFlags().BoolP("userns", "u", false, "")
	runCmd.PersistentFlags().BoolP("readonly", "r", false, "")
	runCmd.PersistentFlags().String("home", "H", "")
	runCmd.PersistentFlags().String("workdir", "W", "")
	runCmd.PersistentFlags().String("scratchdir", "S", "")
	runCmd.PersistentFlags().String("app", "a", "")
	runCmd.PersistentFlags().String("bind", "B", "")
	runCmd.PersistentFlags().BoolP("contain", "c", false, "")
	runCmd.PersistentFlags().BoolP("containall", "C", false, "")
	runCmd.PersistentFlags().BoolP("cleanenv", "e", false, "")
	runCmd.PersistentFlags().BoolP("pid", "p", false, "")
	runCmd.PersistentFlags().BoolP("ipc", "i", false, "")
	runCmd.PersistentFlags().BoolP("uts", "", false, "")
	runCmd.PersistentFlags().BoolP("hostname", "", false, "")
	runCmd.PersistentFlags().String("pwd", "p", "")
	runCmd.PersistentFlags().BoolP("nv", "", false, "")
	runCmd.PersistentFlags().BoolP("fakeroot", "f", false, "")
	runCmd.PersistentFlags().BoolP("keep-privs", "", false, "")
	runCmd.PersistentFlags().BoolP("no-privs", "", false, "")
	runCmd.PersistentFlags().BoolP("add-caps", "", false, "")
	runCmd.PersistentFlags().BoolP("drop-caps", "", false, "")
	runCmd.PersistentFlags().BoolP("allow-setuid", "", false, "")

	runCmd.SetHelpTemplate(`
USAGE: singularity [...] run [run options...] <container path> [...]

This command will launch a Singularity container and execute a runscript
if one is defined for that container. The runscript is a file at
'/singularity'. If this file is present (and executable) then this
command will execute that file within the container automatically. All
arguments following the container name will be passed directly to the
runscript.


RUN OPTIONS:
    -a|--app            Run an app's runscript instead of the default one
    -B|--bind <spec>    A user-bind path specification.  spec has the format
                        src[:dest[:opts]], where src and dest are outside and
                        inside paths.  If dest is not given, it is set equal
                        to src.  Mount options ('opts') may be specified as
                        'ro' (read-only) or 'rw' (read/write, which is the 
                        default). This option can be called multiple times.
    -c|--contain        Use minimal /dev and empty other directories (e.g. /tmp
                        and /home/ubuntu) instead of sharing filesystems on your host
    -C|--containall     Contain not only file systems, but also PID and IPC
    -e|--cleanenv       Clean environment before running container
    -H|--home <spec>    A home directory specification.  spec can either be a
                        src path or src:dest pair.  src is the source path
                        of the home directory outside the container and dest
                        overrides the home directory within the container
    -i|--ipc            Run container in a new IPC namespace
    -n|--net            Run container in a new network namespace (loopback is
                        only network device active)
    --nv                Enable experimental Nvidia support
    -o|--overlay        Use a persistent overlayFS via a writable image
    -p|--pid            Run container in a new PID namespace
    --pwd               Initial working directory for payload process inside
                        the container
    -S|--scratch <path> Include a scratch directory within the container that 
                        is linked to a temporary dir (use -W to force location)
    -u|--userns         Run container in a new user namespace (this allows
                        Singularity to run completely unprivileged on recent
                        kernels and doesn't support all features)
    -W|--workdir        Working directory to be used for /tmp, /var/tmp and
                        /home/ubuntu (if -c/--contain was also used)
    -r|--readonly       By default all Singularity containers are available as
                        writable if they contain a writable partition. This
                        option makes the file system accessible as read only.
                        

CONTAINER FORMATS SUPPORTED:
    *.sqsh              SquashFS format.  Native to Singularity 2.4+
    *.img               This is the native Singularity image format for all
                        Singularity versions < 2.4.
    *.tar*              Tar archives are exploded to a temporary directory and
                        run within that directory (and cleaned up after). The
                        contents of the archive is a root file system with root
                        being in the current directory. Compression suffixes as
                        '.gz' and '.bz2' are supported.
    directory/          Container directories that contain a valid root file
                        system.
    instance://*        A local running instance of a container. (See the
                        instance command group.)
    shub://*            A container hosted on Singularity Hub
    docker://*          A container hosted on Docker Hub


EXAMPLES:

    # Here we see that the runscript prints "Hello world: "
    $ singularity exec /tmp/Debian.img cat /singularity
    #!/bin/sh
    echo "Hello world: "

    # It runs with our inputs when we run the image
    $ singularity run /tmp/Debian.img one two three
    Hello world: one two three

    # Note that this does the same thing
    $ ./tmp/Debian.img one two three

For additional help, please visit our public documentation pages which are
found at:

    http://singularity.lbl.gov/

    `)

	runCmd.SetUsageTemplate(`
USAGE: singularity [...] run [run options...] <container path> [...]
    `)
}
