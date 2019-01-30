// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"log"
	"os"
	"runtime"

	"github.com/spf13/pflag"
	"github.com/sylabs/singularity/internal/pkg/util/user"
)

// actionflags.go contains flag variables for action-like commands to draw from
var (
	AppName         string
	BindPaths       []string
	HomePath        string
	OverlayPath     []string
	ScratchPath     []string
	WorkdirPath     string
	PwdPath         string
	ShellPath       string
	Hostname        string
	Network         string
	NetworkArgs     []string
	DNS             string
	Security        []string
	CgroupsPath     string
	VmRam		string
	VmCpu		string
	ContainLibsPath []string

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
	NoNvidia        bool
	VM              bool
	isSyOS		bool

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
	// --app
	actionFlags.StringVar(&AppName, "app", "", "set an application to run inside a container")
	actionFlags.SetAnnotation("app", "envkey", []string{"APP", "APPNAME"})

	// -B|--bind
	actionFlags.StringSliceVarP(&BindPaths, "bind", "B", []string{}, "a user-bind path specification.  spec has the format src[:dest[:opts]], where src and dest are outside and inside paths.  If dest is not given, it is set equal to src.  Mount options ('opts') may be specified as 'ro' (read-only) or 'rw' (read/write, which is the default). Multiple bind paths can be given by a comma separated list.")
	actionFlags.SetAnnotation("bind", "argtag", []string{"<spec>"})
	actionFlags.SetAnnotation("bind", "envkey", []string{"BIND", "BINDPATH"})

	// -H|--home
	actionFlags.StringVarP(&HomePath, "home", "H", getHomeDir(), "a home directory specification.  spec can either be a src path or src:dest pair.  src is the source path of the home directory outside the container and dest overrides the home directory within the container.")
	actionFlags.SetAnnotation("home", "argtag", []string{"<spec>"})
	actionFlags.SetAnnotation("home", "envkey", []string{"HOME"})

	// -o|--overlay
	actionFlags.StringSliceVarP(&OverlayPath, "overlay", "o", []string{}, "use an overlayFS image for persistent data storage or as read-only layer of container")
	actionFlags.SetAnnotation("overlay", "argtag", []string{"<path>"})
	actionFlags.SetAnnotation("overlay", "envkey", []string{"OVERLAY", "OVERLAYIMAGE"})

	// -S|--scratch
	actionFlags.StringSliceVarP(&ScratchPath, "scratch", "S", []string{}, "include a scratch directory within the container that is linked to a temporary dir (use -W to force location)")
	actionFlags.SetAnnotation("scratch", "argtag", []string{"<path>"})
	actionFlags.SetAnnotation("scratch", "envkey", []string{"SCRATCH", "SCRATCHDIR"})

	// -W|--workdir
	actionFlags.StringVarP(&WorkdirPath, "workdir", "W", "", "working directory to be used for /tmp, /var/tmp and $HOME (if -c/--contain was also used)")
	actionFlags.SetAnnotation("workdir", "argtag", []string{"<path>"})
	actionFlags.SetAnnotation("workdir", "envkey", []string{"WORKDIR"})

	// -s|--shell
	actionFlags.StringVarP(&ShellPath, "shell", "s", "", "path to program to use for interactive shell")
	actionFlags.SetAnnotation("shell", "argtag", []string{"<path>"})
	actionFlags.SetAnnotation("shell", "envkey", []string{"SHELL"})

	// --pwd
	actionFlags.StringVar(&PwdPath, "pwd", "", "initial working directory for payload process inside the container")
	actionFlags.SetAnnotation("pwd", "argtag", []string{"<path>"})
	actionFlags.SetAnnotation("pwd", "envkey", []string{"PWD", "TARGET_PWD"})

	// --hostname
	actionFlags.StringVar(&Hostname, "hostname", "", "set container hostname")
	actionFlags.SetAnnotation("hostname", "argtag", []string{"<name>"})
	actionFlags.SetAnnotation("hostname", "envkey", []string{"HOSTNAME"})

	// --network
	actionFlags.StringVar(&Network, "network", "bridge", "specify desired network type separated by commas, each network will bring up a dedicated interface inside container")
	actionFlags.SetAnnotation("network", "argtag", []string{"<name>"})
	actionFlags.SetAnnotation("network", "envkey", []string{"NETWORK"})

	// --network-args
	actionFlags.StringSliceVar(&NetworkArgs, "network-args", []string{}, "specify network arguments to pass to CNI plugins")
	actionFlags.SetAnnotation("network-args", "argtag", []string{"<name>"})
	actionFlags.SetAnnotation("network-args", "envkey", []string{"NETWORK_ARGS"})

	// --dns
	actionFlags.StringVar(&DNS, "dns", "", "list of DNS server separated by commas to add in resolv.conf")
	actionFlags.SetAnnotation("dns", "envkey", []string{"DNS"})

	// --security
	actionFlags.StringSliceVar(&Security, "security", []string{}, "enable security features (SELinux, Apparmor, Seccomp)")
	actionFlags.SetAnnotation("security", "argtag", []string{""})
	actionFlags.SetAnnotation("security", "envkey", []string{"SECURITY"})

	// --apply-cgroups
	actionFlags.StringVar(&CgroupsPath, "apply-cgroups", "", "apply cgroups from file for container processes (requires root privileges)")
	actionFlags.SetAnnotation("apply-cgroups", "argtag", []string{"<path>"})
	actionFlags.SetAnnotation("apply-cgroups", "envkey", []string{"APPLY_CGROUPS"})

	// --vm-ram
	actionFlags.StringVar(&VmRam,  "vm-ram", "1024", "Amount of RAM in MiB to allocate to Virtual Machine (implies --vm)")
	actionFlags.SetAnnotation("vm-ram", "argtag", []string{"<size>"})
	actionFlags.SetAnnotation("vm-ram", "envkey", []string{"VM_RAM"})

	// --vm-cpu
	actionFlags.StringVar(&VmCpu,  "vm-cpu", "1", "Number of CPU cores to allocate to Virtual Machine (implies --vm)")
	actionFlags.SetAnnotation("vm-cpu", "argtag", []string{"<CPU #>"})
	actionFlags.SetAnnotation("vm-cpu", "envkey", []string{"VM_CPU"})

	// hidden flag to handle SINGULARITY_CONTAINLIBS environment variable
	actionFlags.StringSliceVar(&ContainLibsPath, "containlibs", []string{}, "")
	actionFlags.Lookup("containlibs").Hidden = true
	actionFlags.SetAnnotation("containlibs", "envkey", []string{"CONTAINLIBS"})

	// hidden flags to handle docker credentials
	actionFlags.StringVar(&dockerUsername, "docker-username", "", "specify a username for docker authentication")
	actionFlags.Lookup("docker-username").Hidden = true
	actionFlags.SetAnnotation("docker-username", "envkey", []string{"DOCKER_USERNAME"})

	actionFlags.StringVar(&dockerPassword, "docker-password", "", "specify a password for docker authentication")
	actionFlags.Lookup("docker-password").Hidden = true
	actionFlags.SetAnnotation("docker-password", "envkey", []string{"DOCKER_PASSWORD"})

	// hidden flag to handle SINGULARITY_TMPDIR environment variable
	actionFlags.StringVar(&tmpDir, "tmpdir", "", "specify a temporary directory to use for build")
	actionFlags.Lookup("tmpdir").Hidden = true
	actionFlags.SetAnnotation("tmpdir", "envkey", []string{"TMPDIR"})
}

// initBoolVars initializes flags that take a boolean argument
func initBoolVars() {
	// --boot
	actionFlags.BoolVar(&IsBoot, "boot", false, "execute /sbin/init to boot container (root only)")
	actionFlags.SetAnnotation("boot", "envkey", []string{"BOOT"})

	// -f|--fakeroot
	actionFlags.BoolVarP(&IsFakeroot, "fakeroot", "f", false, "run container in new user namespace as uid 0 (experimental)")
	actionFlags.Lookup("fakeroot").Hidden = true
	actionFlags.SetAnnotation("fakeroot", "envkey", []string{"FAKEROOT"})

	// -e|--cleanenv
	actionFlags.BoolVarP(&IsCleanEnv, "cleanenv", "e", false, "clean environment before running container")
	actionFlags.SetAnnotation("cleanenv", "envkey", []string{"CLEANENV"})

	// -c|--contain
	actionFlags.BoolVarP(&IsContained, "contain", "c", false, "use minimal /dev and empty other directories (e.g. /tmp and $HOME) instead of sharing filesystems from your host")
	actionFlags.SetAnnotation("contain", "envkey", []string{"CONTAIN"})

	// -C|--containall
	actionFlags.BoolVarP(&IsContainAll, "containall", "C", false, "contain not only file systems, but also PID, IPC, and environment")
	actionFlags.SetAnnotation("containall", "envkey", []string{"CONTAINALL"})

	// --nv
	actionFlags.BoolVar(&Nvidia, "nv", false, "enable experimental Nvidia support")
	actionFlags.SetAnnotation("nv", "envkey", []string{"NV"})

	// -w|--writable
	actionFlags.BoolVarP(&IsWritable, "writable", "w", false, "by default all Singularity containers are available as read only. This option makes the file system accessible as read/write.")
	actionFlags.SetAnnotation("writable", "envkey", []string{"WRITABLE"})

	// --writable-tmpfs
	actionFlags.BoolVar(&IsWritableTmpfs, "writable-tmpfs", false, "makes the file system accessible as read-write with non persistent data (with overlay support only)")
	actionFlags.SetAnnotation("writable-tmpfs", "envkey", []string{"WRITABLE_TMPFS"})

	// --no-home
	actionFlags.BoolVar(&NoHome, "no-home", false, "do NOT mount users home directory if home is not the current working directory")
	actionFlags.SetAnnotation("no-home", "envkey", []string{"NO_HOME"})

	// --no-init
	actionFlags.BoolVar(&NoInit, "no-init", false, "do NOT start shim process with --pid")
	actionFlags.SetAnnotation("no-init", "envkey", []string{"NO_INIT", "NOSHIMINIT"})

	// --nohttps
	actionFlags.BoolVar(&noHTTPS, "nohttps", false, "do NOT use HTTPS, for communicating with local docker registry")
	actionFlags.SetAnnotation("nohttps", "envkey", []string{"NOHTTPS"})

	// --docker-login
	actionFlags.BoolVar(&dockerLogin, "docker-login", false, "interactive prompt for docker authentication")
	actionFlags.SetAnnotation("docker-login", "envkey", []string{"DOCKER_LOGIN"})

	// hidden flag to disable nvidia bindings when 'always use nv = yes'
	actionFlags.BoolVar(&NoNvidia, "no-nv", false, "")
	actionFlags.Lookup("no-nv").Hidden = true
	actionFlags.SetAnnotation("no-nv", "envkey", []string{"NV_OFF", "NO_NV"})

	// --vm
	if runtime.GOOS == "darwin" {
		actionFlags.BoolVar(&VM, "vm", true, "enable VM support")
	} else {
		actionFlags.BoolVar(&VM, "vm", false, "enable VM support")
	}
	actionFlags.SetAnnotation("vm", "envkey", []string{"VM"})

	// --syos
	// TODO: Keep this in production?
	actionFlags.BoolVar(&isSyOS, "syos", false, "execute SyOS shell")
	actionFlags.SetAnnotation("syos", "envkey", []string{"SYOS"})

}

// initNamespaceVars initializes flags that take toggle namespace support
func initNamespaceVars() {
	// -p|--pid
	actionFlags.BoolVarP(&PidNamespace, "pid", "p", false, "run container in a new PID namespace")
	actionFlags.SetAnnotation("pid", "envkey", []string{"PID", "UNSHARE_PID"})

	// -i|--ipc
	actionFlags.BoolVarP(&IpcNamespace, "ipc", "i", false, "run container in a new IPC namespace")
	actionFlags.SetAnnotation("ipc", "envkey", []string{"IPC", "UNSHARE_IPC"})

	// -n|--net
	actionFlags.BoolVarP(&NetNamespace, "net", "n", false, "run container in a new network namespace (sets up a bridge network interface by default)")
	actionFlags.SetAnnotation("net", "envkey", []string{"NET", "UNSHARE_NET"})

	// --uts
	actionFlags.BoolVar(&UtsNamespace, "uts", false, "run container in a new UTS namespace")
	actionFlags.SetAnnotation("uts", "envkey", []string{"UTS", "UNSHARE_UTS"})

	// -u|--userns
	actionFlags.BoolVarP(&UserNamespace, "userns", "u", false, "run container in a new user namespace, allowing Singularity to run completely unprivileged on recent kernels. This disables some features of Singularity, for example it only works with sandbox images.")
	actionFlags.SetAnnotation("userns", "envkey", []string{"USERNS", "UNSHARE_USERNS"})
}

// initPrivilegeVars initializes flags that manipulate privileges
func initPrivilegeVars() {
	// --keep-privs
	actionFlags.BoolVar(&KeepPrivs, "keep-privs", false, "let root user keep privileges in container")
	actionFlags.SetAnnotation("keep-privs", "envkey", []string{"KEEP_PRIVS"})

	// --no-privs
	actionFlags.BoolVar(&NoPrivs, "no-privs", false, "drop all privileges from root user in container")
	actionFlags.SetAnnotation("no-privs", "envkey", []string{"NO_PRIVS"})

	// --add-caps
	actionFlags.StringVar(&AddCaps, "add-caps", "", "a comma separated capability list to add")
	actionFlags.SetAnnotation("add-caps", "envkey", []string{"ADD_CAPS"})

	// --drop-caps
	actionFlags.StringVar(&DropCaps, "drop-caps", "", "a comma separated capability list to drop")
	actionFlags.SetAnnotation("drop-caps", "envkey", []string{"DROP_CAPS"})

	// --allow-setuid
	actionFlags.BoolVar(&AllowSUID, "allow-setuid", false, "allow setuid binaries in container (root only)")
	actionFlags.SetAnnotation("allow-setuid", "envkey", []string{"ALLOW_SETUID"})
}
