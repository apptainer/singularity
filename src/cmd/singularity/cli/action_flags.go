// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"log"
	"os"

	"github.com/singularityware/singularity/src/pkg/util/user"
	"github.com/spf13/pflag"
)

// actionflags.go contains flag variables for action-like commands to draw from
var (
	BindPaths   []string
	HomePath    string
	OverlayPath []string
	ScratchPath []string
	WorkdirPath string
	PwdPath     string
	ShellPath   string
	Hostname    string

	IsBoot       bool
	IsFakeroot   bool
	IsCleanEnv   bool
	IsContained  bool
	IsContainAll bool
	IsWritable   bool
	Nvidia       bool
	NoHome       bool
	NoInit       bool

	NetNamespace  bool
	UtsNamespace  bool
	UserNamespace bool
	PidNamespace  bool
	IpcNamespace  bool

	AllowSUID bool
	KeepPrivs bool
	NoPrivs   bool
	AddCaps   string
	DropCaps  string
)

var actionFlags = pflag.NewFlagSet("ActionFlags", pflag.ExitOnError)

func getHomeDir() string {
	user, err := user.GetPwUID(uint32(os.Getuid()))
	if err != nil {
		log.Fatal(err)
		return ""
	}

	return user.Dir
}

func init() {
	initPathVars()
	initBoolVars()
	initNamespaceVars()
	initPrivilegeVars()
}

// initPathVars initializes flags that take a string argument
func initPathVars() {
	// -B|--bind
	actionFlags.StringSliceVarP(&BindPaths, "bind", "B", []string{}, "A user-bind path specification.  spec has the format src[:dest[:opts]], where src and dest are outside and inside paths.  If dest is not given, it is set equal to src.  Mount options ('opts') may be specified as 'ro' (read-only) or 'rw' (read/write, which is the default). Multiple bind paths can be given by a comma separated list.")
	actionFlags.SetAnnotation("bind", "argtag", []string{"<spec>"})

	// -H|--home
	actionFlags.StringVarP(&HomePath, "home", "H", getHomeDir(), "A home directory specification.  spec can either be a src path or src:dest pair.  src is the source path of the home directory outside the container and dest overrides the home directory within the container.")
	actionFlags.SetAnnotation("home", "argtag", []string{"<spec>"})

	// -o|--overlay
	actionFlags.StringSliceVarP(&OverlayPath, "overlay", "o", []string{}, "Use an overlayFS image for persistent data storage or as read-only layer of container.")
	actionFlags.SetAnnotation("overlay", "argtag", []string{"<path>"})

	// -S|--scratch
	actionFlags.StringSliceVarP(&ScratchPath, "scratch", "S", []string{}, "Include a scratch directory within the container that is linked to a temporary dir (use -W to force location)")
	actionFlags.SetAnnotation("scratch", "argtag", []string{"<path>"})

	// -W|--workdir
	actionFlags.StringVarP(&WorkdirPath, "workdir", "W", "", "Working directory to be used for /tmp, /var/tmp and $HOME (if -c/--contain was also used)")
	actionFlags.SetAnnotation("workdir", "argtag", []string{"<path>"})

	// -s|--shell
	actionFlags.StringVarP(&ShellPath, "shell", "s", "", "Path to program to use for interactive shell")
	actionFlags.SetAnnotation("shell", "argtag", []string{"<path>"})

	// --pwd
	actionFlags.StringVar(&PwdPath, "pwd", "", "Initial working directory for payload process inside the container")
	actionFlags.SetAnnotation("pwd", "argtag", []string{"<path>"})

	// --hostname
	actionFlags.StringVar(&Hostname, "hostname", "", "Set container hostname")
	actionFlags.SetAnnotation("hostname", "argtag", []string{"<name>"})
}

// initBoolVars initializes flags that take a boolean argument
func initBoolVars() {
	// --boot
	actionFlags.BoolVar(&IsBoot, "boot", false, "Execute /sbin/init to boot container (root only)")

	// -f|--fakeroot
	actionFlags.BoolVarP(&IsFakeroot, "fakeroot", "f", false, "Run container in new user namespace as uid 0")

	// -e|--cleanenv
	actionFlags.BoolVarP(&IsCleanEnv, "cleanenv", "e", false, "Clean environment before running container")

	// -c|--contain
	actionFlags.BoolVarP(&IsContained, "contain", "c", false, "Use minimal /dev and empty other directories (e.g. /tmp and $HOME) instead of sharing filesystems from your host.")

	// -C|--containall
	actionFlags.BoolVarP(&IsContainAll, "containall", "C", false, "Contain not only file systems, but also PID, IPC, and environment")

	// --nv
	actionFlags.BoolVar(&Nvidia, "nv", false, "Enable experimental Nvidia support")

	// -w|--writable
	actionFlags.BoolVarP(&IsWritable, "writable", "w", false, "By default all Singularity containers are available as read only. This option makes the file system accessible as read/write.")

	// --no-home
	actionFlags.BoolVar(&NoHome, "no-home", false, "Do NOT mount users home directory if home is not the current working directory.")

	// --no-init
	actionFlags.BoolVar(&NoInit, "no-init", false, "Do NOT start shim process with --pid.")
}

// initNamespaceVars initializes flags that take toggle namespace support
func initNamespaceVars() {
	// -p|--pid
	actionFlags.BoolVarP(&PidNamespace, "pid", "p", false, "Run container in a new PID namespace")

	// -i|--ipc
	actionFlags.BoolVarP(&IpcNamespace, "ipc", "i", false, "Run container in a new IPC namespace")

	// -n|--net
	actionFlags.BoolVarP(&NetNamespace, "net", "n", false, "Run container in a new network namespace (loopback is the only network device active).")

	// --uts
	actionFlags.BoolVar(&UtsNamespace, "uts", false, "Run container in a new UTS namespace")

	// -u|--userns
	actionFlags.BoolVarP(&UserNamespace, "userns", "u", false, "Run container in a new user namespace, allowing Singularity to run completely unprivileged on recent kernels. This may not support every feature of Singularity.")

}

// initPrivilegeVars initializes flags that manipulate privileges
func initPrivilegeVars() {
	// --keep-privs
	actionFlags.BoolVar(&KeepPrivs, "keep-privs", false, "Let root user keep privileges in container")

	// --no-privs
	actionFlags.BoolVar(&NoPrivs, "no-privs", false, "Drop all privileges from root user in container")

	// --add-caps
	actionFlags.StringVar(&AddCaps, "add-caps", "", "A comma separated capability list to add")

	// --drop-caps
	actionFlags.StringVar(&DropCaps, "drop-caps", "", "A comma separated capability list to drop")

	// --allow-setuid
	actionFlags.BoolVar(&AllowSUID, "allow-setuid", false, "Allow setuid binaries in container (root only)")
}
