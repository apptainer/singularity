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

// execCmd represents the exec command
var execCmd = &cobra.Command{
	Use: "exec",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("exec called")
	},
}

func init() {
	singularityCmd.AddCommand(execCmd)

	// ultimately all of this should go into a seperate function to be shared
	// between exec, run, and shell
	execCmd.PersistentFlags().String("overlay", "o", "")
	execCmd.PersistentFlags().String("shell", "s", "")
	execCmd.PersistentFlags().BoolP("userns", "u", false, "")
	execCmd.PersistentFlags().BoolP("readonly", "r", false, "")
	execCmd.PersistentFlags().String("home", "H", "")
	execCmd.PersistentFlags().String("workdir", "W", "")
	execCmd.PersistentFlags().String("scratchdir", "S", "")
	execCmd.PersistentFlags().String("app", "a", "")
	execCmd.PersistentFlags().String("bind", "B", "")
	execCmd.PersistentFlags().BoolP("contain", "c", false, "")
	execCmd.PersistentFlags().BoolP("containall", "C", false, "")
	execCmd.PersistentFlags().BoolP("cleanenv", "e", false, "")
	execCmd.PersistentFlags().BoolP("pid", "p", false, "")
	execCmd.PersistentFlags().BoolP("ipc", "i", false, "")
	execCmd.PersistentFlags().BoolP("uts", "", false, "")
	execCmd.PersistentFlags().BoolP("hostname", "", false, "")
	execCmd.PersistentFlags().String("pwd", "p", "")
	execCmd.PersistentFlags().BoolP("nv", "", false, "")
	execCmd.PersistentFlags().BoolP("fakeroot", "f", false, "")
	execCmd.PersistentFlags().BoolP("keep-privs", "", false, "")
	execCmd.PersistentFlags().BoolP("no-privs", "", false, "")
	execCmd.PersistentFlags().BoolP("add-caps", "", false, "")
	execCmd.PersistentFlags().BoolP("drop-caps", "", false, "")
	execCmd.PersistentFlags().BoolP("allow-setuid", "", false, "")

	execCmd.SetHelpTemplate(`
USAGE: singularity [...] exec [exec options...] <container path> <command>

This command will allow you to execute any program within the given
container image.

EXEC OPTIONS:
    -a|--app            Exec a command relevant to an application directory
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
    
    $ singularity exec /tmp/Debian.img cat /etc/debian_version
    $ singularity exec /tmp/Debian.img python ./hello_world.py
    $ cat hello_world.py | singularity exec /tmp/Debian.img python
    $ sudo singularity exec --writable /tmp/Debian.img apt-get update
    $ singularity exec instance://my_instance ps -ef 

For additional help, please visit our public documentation pages which are
found at:

    http://singularity.lbl.gov/
`)

	execCmd.SetUsageTemplate(`
USAGE: singularity [...] exec [exec options...] <container path> <command>
    `)
}
