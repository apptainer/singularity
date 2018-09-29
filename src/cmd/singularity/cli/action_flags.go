// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"log"
	"os"

	"github.com/spf13/pflag"
	"github.com/sylabs/singularity/src/pkg/util/user"
)

// actionflags.go contains flag variables for action-like commands to draw from
var (
	AppName     string
	BindPaths   []string
	HomePath    string
	OverlayPath []string
	ScratchPath []string
	WorkdirPath string
	PwdPath     string
	ShellPath   string
	Hostname    string
	Network     string
	NetworkArgs []string
	DNS         string
	Security    []string
	CgroupsPath string
	Namespace   []string

	IsBoot          bool
	IsFakeroot      bool
	IsCleanEnv      bool
	IsContained     bool
	IsContainAll    bool
	IsWritable      bool
	IsWritableTmpfs bool
	Nvidia          bool
	NoHome          bool
	NoInit          bool

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
	initPrivilegeVars()
}

// initPathVars initializes flags that take a string argument
func initPathVars() {
	// --app
	actionFlags.StringVar(&AppName, "app", "", "Set container app to run")

	// -B|--bind
	actionFlags.StringSliceVarP(&BindPaths, "bind", "B", []string{}, "A user-bind path specification.  spec has the format src[:dest[:opts]], where src and dest are outside and inside paths.  If dest is not given, it is set equal to src.  Mount options ('opts') may be specified as 'ro' (read-only) or 'rw' (read/write, which is the default). Multiple bind paths can be given by a comma separated list.")
	actionFlags.SetAnnotation("bind", "argtag", []string{"<spec>"})
	actionFlags.SetAnnotation("bind", "envkey", []string{"BIND", "BINDPATH"})

	// -H|--home
	actionFlags.StringVarP(&HomePath, "home", "H", getHomeDir(), "A home directory specification.  spec can either be a src path or src:dest pair.  src is the source path of the home directory outside the container and dest overrides the home directory within the container.")
	actionFlags.SetAnnotation("home", "argtag", []string{"<spec>"})
	actionFlags.SetAnnotation("home", "envkey", []string{"HOME"})

	// -o|--overlay
	actionFlags.StringSliceVarP(&OverlayPath, "overlay", "o", []string{}, "Use an overlayFS image for persistent data storage or as read-only layer of container.")
	actionFlags.SetAnnotation("overlay", "argtag", []string{"<path>"})
	actionFlags.SetAnnotation("overlay", "envkey", []string{"OVERLAY", "OVERLAYIMAGE"})

	// -S|--scratch
	actionFlags.StringSliceVarP(&ScratchPath, "scratch", "S", []string{}, "Include a scratch directory within the container that is linked to a temporary dir (use -W to force location)")
	actionFlags.SetAnnotation("scratch", "argtag", []string{"<path>"})
	actionFlags.SetAnnotation("scratch", "envkey", []string{"SCRATCH", "SCRATCHDIR"})

	// -W|--workdir
	actionFlags.StringVarP(&WorkdirPath, "workdir", "W", "", "Working directory to be used for /tmp, /var/tmp and $HOME (if -c/--contain was also used)")
	actionFlags.SetAnnotation("workdir", "argtag", []string{"<path>"})
	actionFlags.SetAnnotation("workdir", "envkey", []string{"WORKDIR"})

	// -s|--shell
	actionFlags.StringVarP(&ShellPath, "shell", "s", "", "Path to program to use for interactive shell")
	actionFlags.SetAnnotation("shell", "argtag", []string{"<path>"})
	actionFlags.SetAnnotation("shell", "envkey", []string{"SHELL"})

	// --pwd
	actionFlags.StringVar(&PwdPath, "pwd", "", "Initial working directory for payload process inside the container")
	actionFlags.SetAnnotation("pwd", "argtag", []string{"<path>"})
	actionFlags.SetAnnotation("pwd", "envkey", []string{"PWD", "TARGET_PWD"})

	// --hostname
	actionFlags.StringVar(&Hostname, "hostname", "", "Set container hostname")
	actionFlags.SetAnnotation("hostname", "argtag", []string{"<name>"})
	actionFlags.SetAnnotation("hostname", "envkey", []string{"HOSTNAME"})

	// --network
	actionFlags.StringVar(&Network, "network", "bridge", "Specify desired network type separated by commas, each network will bring up a dedicated interface inside container")
	actionFlags.SetAnnotation("network", "argtag", []string{"<name>"})
	actionFlags.SetAnnotation("network", "envkey", []string{"NETWORK"})

	// --network-args
	actionFlags.StringSliceVar(&NetworkArgs, "network-args", []string{}, "Specify network arguments to pass to CNI plugins")
	actionFlags.SetAnnotation("network-args", "argtag", []string{"<name>"})
	actionFlags.SetAnnotation("network-args", "envkey", []string{"NETWORK_ARGS"})

	// --dns
	actionFlags.StringVar(&DNS, "dns", "", "List of DNS server separated by commas to add in resolv.conf")
	actionFlags.SetAnnotation("dns", "envkey", []string{"DNS"})

	// --security
	actionFlags.StringSliceVar(&Security, "security", []string{}, "Enable security features (SELinux, Apparmor, Seccomp)")
	actionFlags.SetAnnotation("security", "argtag", []string{""})

	// --apply-cgroups
	actionFlags.StringVar(&CgroupsPath, "apply-cgroups", "", "Apply cgroups from file for container processes (requires root privileges)")
	actionFlags.SetAnnotation("cgroups-profile", "argtag", []string{"<path>"})

	// --namespace
	actionFlags.StringSliceVarP(&Namespace, "namespace", "n", []string{}, "List of namespaces (pid,ipc,network,uts,user) for container to join separated by commas")
	actionFlags.SetAnnotation("namespace", "argtag", []string{"<list>"})
}

// initBoolVars initializes flags that take a boolean argument
func initBoolVars() {
	// --boot
	actionFlags.BoolVar(&IsBoot, "boot", false, "Execute /sbin/init to boot container (root only)")
	actionFlags.SetAnnotation("boot", "envkey", []string{"BOOT"})

	// -f|--fakeroot
	actionFlags.BoolVarP(&IsFakeroot, "fakeroot", "f", false, "Run container in new user namespace as uid 0")
	actionFlags.SetAnnotation("fakeroot", "envkey", []string{"FAKEROOT"})

	// -e|--cleanenv
	actionFlags.BoolVarP(&IsCleanEnv, "cleanenv", "e", false, "Clean environment before running container")
	actionFlags.SetAnnotation("cleanenv", "envkey", []string{"CLEANENV"})

	// -c|--contain
	actionFlags.BoolVarP(&IsContained, "contain", "c", false, "Use minimal /dev and empty other directories (e.g. /tmp and $HOME) instead of sharing filesystems from your host.")
	actionFlags.SetAnnotation("contain", "envkey", []string{"CONTAIN"})

	// -C|--containall
	actionFlags.BoolVarP(&IsContainAll, "containall", "C", false, "Contain not only file systems, but also PID, IPC, and environment")
	actionFlags.SetAnnotation("containall", "envkey", []string{"CONTAINALL"})

	// --nv
	actionFlags.BoolVar(&Nvidia, "nv", false, "Enable experimental Nvidia support")
	actionFlags.SetAnnotation("nv", "envkey", []string{"NV"})

	// -w|--writable
	actionFlags.BoolVarP(&IsWritable, "writable", "w", false, "By default all Singularity containers are available as read only. This option makes the file system accessible as read/write.")
	actionFlags.SetAnnotation("writable", "envkey", []string{"WRITABLE"})

	// --writable-tmpfs
	actionFlags.BoolVar(&IsWritableTmpfs, "writable-tmpfs", false, "Makes the file system accessible as read-write with non persistent data (with overlay support only).")

	// --no-home
	actionFlags.BoolVar(&NoHome, "no-home", false, "Do NOT mount users home directory if home is not the current working directory.")
	actionFlags.SetAnnotation("no-home", "envkey", []string{"NO_HOME"})

	// --no-init
	actionFlags.BoolVar(&NoInit, "no-init", false, "Do NOT start shim process with --pid.")
	actionFlags.SetAnnotation("no-init", "envkey", []string{"NO_INIT", "NOSHIMINIT"})
}

// initPrivilegeVars initializes flags that manipulate privileges
func initPrivilegeVars() {
	// --keep-privs
	actionFlags.BoolVar(&KeepPrivs, "keep-privs", false, "Let root user keep privileges in container")
	actionFlags.SetAnnotation("keep-privs", "envkey", []string{"KEEP_PRIVS"})

	// --no-privs
	actionFlags.BoolVar(&NoPrivs, "no-privs", false, "Drop all privileges from root user in container")
	actionFlags.SetAnnotation("no-privs", "envkey", []string{"NO_PRIVS"})

	// --add-caps
	actionFlags.StringVar(&AddCaps, "add-caps", "", "A comma separated capability list to add")
	actionFlags.SetAnnotation("add-caps", "envkey", []string{"ADD_CAPS"})

	// --drop-caps
	actionFlags.StringVar(&DropCaps, "drop-caps", "", "A comma separated capability list to drop")
	actionFlags.SetAnnotation("drop-caps", "envkey", []string{"DROP_CAPS"})

	// --allow-setuid
	actionFlags.BoolVar(&AllowSUID, "allow-setuid", false, "Allow setuid binaries in container (root only)")
	actionFlags.SetAnnotation("allow-setuid", "envkey", []string{"ALLOW_SETUID"})
}
