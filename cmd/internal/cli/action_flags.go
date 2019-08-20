// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/sylabs/singularity/internal/pkg/plugin"
	"github.com/sylabs/singularity/pkg/cmdline"
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
	VMRAM           string
	VMCPU           string
	VMIP            string
	ContainLibsPath []string
	encryptionPEM   string
	encryptionPass  string

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
	VMErr           bool
	NoNet           bool
	IsSyOS          bool
	disableCache    bool

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

// --app
var actionAppFlag = cmdline.Flag{
	ID:           "actionAppFlag",
	Value:        &AppName,
	DefaultValue: "",
	Name:         "app",
	Usage:        "set an application to run inside a container",
	EnvKeys:      []string{"APP", "APPNAME"},
	ExcludedOS:   []string{cmdline.Darwin},
}

// -B|--bind
var actionBindFlag = cmdline.Flag{
	ID:           "actionBindFlag",
	Value:        &BindPaths,
	DefaultValue: []string{},
	Name:         "bind",
	ShortHand:    "B",
	Usage:        "a user-bind path specification.  spec has the format src[:dest[:opts]], where src and dest are outside and inside paths.  If dest is not given, it is set equal to src.  Mount options ('opts') may be specified as 'ro' (read-only) or 'rw' (read/write, which is the default). Multiple bind paths can be given by a comma separated list.",
	EnvKeys:      []string{"BIND", "BINDPATH"},
	Tag:          "<spec>",
	EnvHandler:   cmdline.EnvAppendValue,
}

// -H|--home
var actionHomeFlag = cmdline.Flag{
	ID:           "actionHomeFlag",
	Value:        &HomePath,
	DefaultValue: CurrentUser.HomeDir,
	Name:         "home",
	ShortHand:    "H",
	Usage:        "a home directory specification.  spec can either be a src path or src:dest pair.  src is the source path of the home directory outside the container and dest overrides the home directory within the container.",
	EnvKeys:      []string{"HOME"},
	Tag:          "<spec>",
}

// -o|--overlay
var actionOverlayFlag = cmdline.Flag{
	ID:           "actionOverlayFlag",
	Value:        &OverlayPath,
	DefaultValue: []string{},
	Name:         "overlay",
	ShortHand:    "o",
	Usage:        "use an overlayFS image for persistent data storage or as read-only layer of container",
	EnvKeys:      []string{"OVERLAY", "OVERLAYIMAGE"},
	Tag:          "<path>",
	ExcludedOS:   []string{cmdline.Darwin},
}

// -S|--scratch
var actionScratchFlag = cmdline.Flag{
	ID:           "actionScratchFlag",
	Value:        &ScratchPath,
	DefaultValue: []string{},
	Name:         "scratch",
	ShortHand:    "S",
	Usage:        "include a scratch directory within the container that is linked to a temporary dir (use -W to force location)",
	EnvKeys:      []string{"SCRATCH", "SCRATCHDIR"},
	Tag:          "<path>",
	ExcludedOS:   []string{cmdline.Darwin},
}

// -W|--workdir
var actionWorkdirFlag = cmdline.Flag{
	ID:           "actionWorkdirFlag",
	Value:        &WorkdirPath,
	DefaultValue: "",
	Name:         "workdir",
	ShortHand:    "W",
	Usage:        "working directory to be used for /tmp, /var/tmp and $HOME (if -c/--contain was also used)",
	EnvKeys:      []string{"WORKDIR"},
	Tag:          "<path>",
	ExcludedOS:   []string{cmdline.Darwin},
}

// --disable-cache
var actionDisableCacheFlag = cmdline.Flag{
	ID:           "actionDisableCacheFlag",
	Value:        &disableCache,
	DefaultValue: false,
	Name:         "disable-cache",
	Usage:        "dont use cache, and dont create cache",
	EnvKeys:      []string{"DISABLE_CACHE"},
}

// -s|--shell
var actionShellFlag = cmdline.Flag{
	ID:           "actionShellFlag",
	Value:        &ShellPath,
	DefaultValue: "",
	Name:         "shell",
	ShortHand:    "s",
	Usage:        "path to program to use for interactive shell",
	EnvKeys:      []string{"SHELL"},
	Tag:          "<path>",
	ExcludedOS:   []string{cmdline.Darwin},
}

// --pwd
var actionPwdFlag = cmdline.Flag{
	ID:           "actionPwdFlag",
	Value:        &PwdPath,
	DefaultValue: "",
	Name:         "pwd",
	Usage:        "initial working directory for payload process inside the container",
	EnvKeys:      []string{"PWD", "TARGET_PWD"},
	Tag:          "<path>",
	ExcludedOS:   []string{cmdline.Darwin},
}

// --hostname
var actionHostnameFlag = cmdline.Flag{
	ID:           "actionHostnameFlag",
	Value:        &Hostname,
	DefaultValue: "",
	Name:         "hostname",
	Usage:        "set container hostname",
	EnvKeys:      []string{"HOSTNAME"},
	Tag:          "<name>",
	ExcludedOS:   []string{cmdline.Darwin},
}

// --network
var actionNetworkFlag = cmdline.Flag{
	ID:           "actionNetworkFlag",
	Value:        &Network,
	DefaultValue: "bridge",
	Name:         "network",
	Usage:        "specify desired network type separated by commas, each network will bring up a dedicated interface inside container",
	EnvKeys:      []string{"NETWORK"},
	Tag:          "<name>",
	ExcludedOS:   []string{cmdline.Darwin},
}

// --network-args
var actionNetworkArgsFlag = cmdline.Flag{
	ID:           "actionNetworkArgsFlag",
	Value:        &NetworkArgs,
	DefaultValue: []string{},
	Name:         "network-args",
	Usage:        "specify network arguments to pass to CNI plugins",
	EnvKeys:      []string{"NETWORK_ARGS"},
	Tag:          "<args>",
	ExcludedOS:   []string{cmdline.Darwin},
}

// --dns
var actionDNSFlag = cmdline.Flag{
	ID:           "actionDnsFlag",
	Value:        &DNS,
	DefaultValue: "",
	Name:         "dns",
	Usage:        "list of DNS server separated by commas to add in resolv.conf",
	EnvKeys:      []string{"DNS"},
	ExcludedOS:   []string{cmdline.Darwin},
}

// --security
var actionSecurityFlag = cmdline.Flag{
	ID:           "actionSecurityFlag",
	Value:        &Security,
	DefaultValue: []string{},
	Name:         "security",
	Usage:        "enable security features (SELinux, Apparmor, Seccomp)",
	EnvKeys:      []string{"SECURITY"},
	ExcludedOS:   []string{cmdline.Darwin},
}

// --apply-cgroups
var actionApplyCgroupsFlag = cmdline.Flag{
	ID:           "actionApplyCgroupsFlag",
	Value:        &CgroupsPath,
	DefaultValue: "",
	Name:         "apply-cgroups",
	Usage:        "apply cgroups from file for container processes (root only)",
	EnvKeys:      []string{"APPLY_CGROUPS"},
	ExcludedOS:   []string{cmdline.Darwin},
}

// --vm-ram
var actionVMRAMFlag = cmdline.Flag{
	ID:           "actionVMRAMFlag",
	Value:        &VMRAM,
	DefaultValue: "1024",
	Name:         "vm-ram",
	Usage:        "Amount of RAM in MiB to allocate to Virtual Machine (implies --vm)",
	Tag:          "<size>",
	EnvKeys:      []string{"VM_RAM"},
}

// --vm-cpu
var actionVMCPUFlag = cmdline.Flag{
	ID:           "actionVMCPUFlag",
	Value:        &VMCPU,
	DefaultValue: "1",
	Name:         "vm-cpu",
	Usage:        "Number of CPU cores to allocate to Virtual Machine (implies --vm)",
	Tag:          "<CPU #>",
	EnvKeys:      []string{"VM_CPU"},
}

// --vm-ip
var actionVMIPFlag = cmdline.Flag{
	ID:           "actionVMIPFlag",
	Value:        &VMIP,
	DefaultValue: "dhcp",
	Name:         "vm-ip",
	Usage:        "IP Address to assign for container usage. Defaults to DHCP within bridge network.",
	Tag:          "<IP Address>",
	EnvKeys:      []string{"VM_IP"},
}

// --nonet
var actionNONETFlag = cmdline.Flag{
	ID:           "actionNONETFlag",
	Value:        &NoNet,
	DefaultValue: false,
	Name:         "nonet",
	Usage:        "Disable VM network handling",
	EnvKeys:      []string{"VM_NONET"},
}

// hidden flag to handle SINGULARITY_CONTAINLIBS environment variable
var actionContainLibsFlag = cmdline.Flag{
	ID:           "actionContainLibsFlag",
	Value:        &ContainLibsPath,
	DefaultValue: []string{},
	Name:         "containlibs",
	Hidden:       true,
	EnvKeys:      []string{"CONTAINLIBS"},
	ExcludedOS:   []string{cmdline.Darwin},
}

// flag to handle SINGULARITY_ENCRYPTION_PEMPATH environment variable
var pemEncryptFlag = cmdline.Flag{
	ID:           "actionEncryptionPEM",
	Value:        &encryptionPEM,
	DefaultValue: "",
	Name:         "pempath",
	Hidden:       false,
	EnvKeys:      []string{"ENCRYPTION_PEMPATH"},
}

// flag to handle SINGULARITY_ENCRYPTION_PASSPHRASE environment variable
var passEncryptFlag = cmdline.Flag{
	ID:           "actionEncryptionPass",
	Value:        &encryptionPass,
	DefaultValue: "",
	Name:         "passphrase",
	Hidden:       false,
	EnvKeys:      []string{"ENCRYPTION_PASSPHRASE"},
}

// hidden flags to handle docker credentials
var actionDockerUsernameFlag = cmdline.Flag{
	ID:           "actionDockerUsernameFlag",
	Value:        &dockerUsername,
	DefaultValue: "",
	Name:         "docker-username",
	Usage:        "specify a username for docker authentication",
	Hidden:       true,
	EnvKeys:      []string{"DOCKER_USERNAME"},
}

// --docker-password
var actionDockerPasswordFlag = cmdline.Flag{
	ID:           "actionDockerPasswordFlag",
	Value:        &dockerPassword,
	DefaultValue: "",
	Name:         "docker-password",
	Usage:        "specify a password for docker authentication",
	Hidden:       true,
	EnvKeys:      []string{"DOCKER_PASSWORD"},
}

// hidden flag to handle SINGULARITY_TMPDIR environment variable
var actionTmpDirFlag = cmdline.Flag{
	ID:           "actionTmpDirFlag",
	Value:        &tmpDir,
	DefaultValue: "",
	Name:         "tmpdir",
	Usage:        "specify a temporary directory to use for build",
	Hidden:       true,
	EnvKeys:      []string{"TMPDIR"},
}

// --boot
var actionBootFlag = cmdline.Flag{
	ID:           "actionBootFlag",
	Value:        &IsBoot,
	DefaultValue: false,
	Name:         "boot",
	Usage:        "execute /sbin/init to boot container (root only)",
	EnvKeys:      []string{"BOOT"},
	ExcludedOS:   []string{cmdline.Darwin},
}

// -f|--fakeroot
var actionFakerootFlag = cmdline.Flag{
	ID:           "actionFakerootFlag",
	Value:        &IsFakeroot,
	DefaultValue: false,
	Name:         "fakeroot",
	ShortHand:    "f",
	Usage:        "run container in new user namespace as uid 0",
	EnvKeys:      []string{"FAKEROOT"},
	ExcludedOS:   []string{cmdline.Darwin},
}

// -e|--cleanenv
var actionCleanEnvFlag = cmdline.Flag{
	ID:           "actionCleanEnvFlag",
	Value:        &IsCleanEnv,
	DefaultValue: false,
	Name:         "cleanenv",
	ShortHand:    "e",
	Usage:        "clean environment before running container",
	EnvKeys:      []string{"CLEANENV"},
	ExcludedOS:   []string{cmdline.Darwin},
}

// -c|--contain
var actionContainFlag = cmdline.Flag{
	ID:           "actionContainFlag",
	Value:        &IsContained,
	DefaultValue: false,
	Name:         "contain",
	ShortHand:    "c",
	Usage:        "use minimal /dev and empty other directories (e.g. /tmp and $HOME) instead of sharing filesystems from your host",
	EnvKeys:      []string{"CONTAIN"},
	ExcludedOS:   []string{cmdline.Darwin},
}

// -C|--containall
var actionContainAllFlag = cmdline.Flag{
	ID:           "actionContainAllFlag",
	Value:        &IsContainAll,
	DefaultValue: false,
	Name:         "containall",
	ShortHand:    "C",
	Usage:        "contain not only file systems, but also PID, IPC, and environment",
	EnvKeys:      []string{"CONTAINALL"},
	ExcludedOS:   []string{cmdline.Darwin},
}

// --nv
var actionNvidiaFlag = cmdline.Flag{
	ID:           "actionNvidiaFlag",
	Value:        &Nvidia,
	DefaultValue: false,
	Name:         "nv",
	Usage:        "enable experimental Nvidia support",
	EnvKeys:      []string{"NV"},
	ExcludedOS:   []string{cmdline.Darwin},
}

// -w|--writable
var actionWritableFlag = cmdline.Flag{
	ID:           "actionWritableFlag",
	Value:        &IsWritable,
	DefaultValue: false,
	Name:         "writable",
	ShortHand:    "w",
	Usage:        "by default all Singularity containers are available as read only. This option makes the file system accessible as read/write.",
	EnvKeys:      []string{"WRITABLE"},
	ExcludedOS:   []string{cmdline.Darwin},
}

// --writable-tmpfs
var actionWritableTmpfsFlag = cmdline.Flag{
	ID:           "actionWritableTmpfsFlag",
	Value:        &IsWritableTmpfs,
	DefaultValue: false,
	Name:         "writable-tmpfs",
	Usage:        "makes the file system accessible as read-write with non persistent data (with overlay support only)",
	EnvKeys:      []string{"WRITABLE_TMPFS"},
	ExcludedOS:   []string{cmdline.Darwin},
}

// --no-home
var actionNoHomeFlag = cmdline.Flag{
	ID:           "actionNoHomeFlag",
	Value:        &NoHome,
	DefaultValue: false,
	Name:         "no-home",
	Usage:        "do NOT mount users home directory if home is not the current working directory",
	EnvKeys:      []string{"NO_HOME"},
	ExcludedOS:   []string{cmdline.Darwin},
}

// --no-init
var actionNoInitFlag = cmdline.Flag{
	ID:           "actionNoInitFlag",
	Value:        &NoInit,
	DefaultValue: false,
	Name:         "no-init",
	Usage:        "do NOT start shim process with --pid",
	EnvKeys:      []string{"NOSHIMINIT"},
	ExcludedOS:   []string{cmdline.Darwin},
}

// --nohttps
var actionNoHTTPSFlag = cmdline.Flag{
	ID:           "actionNoHTTPSFlag",
	Value:        &noHTTPS,
	DefaultValue: false,
	Name:         "nohttps",
	Usage:        "do NOT use HTTPS, for communicating with local docker registry",
	EnvKeys:      []string{"NOHTTPS"},
}

// --docker-login
var actionDockerLoginFlag = cmdline.Flag{
	ID:           "actionDockerLoginFlag",
	Value:        &dockerLogin,
	DefaultValue: false,
	Name:         "docker-login",
	Usage:        "login to a Docker Repository interactively",
	EnvKeys:      []string{"DOCKER_LOGIN"},
}

// hidden flag to disable nvidia bindings when 'always use nv = yes'
var actionNoNvidiaFlag = cmdline.Flag{
	ID:           "actionNoNvidiaFlag",
	Value:        &NoNvidia,
	DefaultValue: false,
	Name:         "no-nv",
	EnvKeys:      []string{"NV_OFF", "NO_NV"},
	ExcludedOS:   []string{cmdline.Darwin},
}

// --vm
var actionVMFlag = cmdline.Flag{
	ID:           "actionVMFlag",
	Value:        &VM,
	DefaultValue: false,
	Name:         "vm",
	Usage:        "enable VM support",
	EnvKeys:      []string{"VM"},
}

// --vm-err
var actionVMErrFlag = cmdline.Flag{
	ID:           "actionVMErrFlag",
	Value:        &VMErr,
	DefaultValue: false,
	Name:         "vm-err",
	Usage:        "enable attaching stderr from VM",
	EnvKeys:      []string{"VMERROR"},
}

// --syos
// TODO: Keep this in production?
var actionSyOSFlag = cmdline.Flag{
	ID:           "actionSyOSFlag",
	Value:        &IsSyOS,
	DefaultValue: false,
	Name:         "syos",
	Usage:        "execute SyOS shell",
	EnvKeys:      []string{"SYOS"},
}

// -p|--pid
var actionPidNamespaceFlag = cmdline.Flag{
	ID:           "actionPidNamespaceFlag",
	Value:        &PidNamespace,
	DefaultValue: false,
	Name:         "pid",
	ShortHand:    "p",
	Usage:        "run container in a new PID namespace",
	EnvKeys:      []string{"PID", "UNSHARE_PID"},
	ExcludedOS:   []string{cmdline.Darwin},
}

// -i|--ipc
var actionIpcNamespaceFlag = cmdline.Flag{
	ID:           "actionIpcNamespaceFlag",
	Value:        &IpcNamespace,
	DefaultValue: false,
	Name:         "ipc",
	ShortHand:    "i",
	Usage:        "run container in a new IPC namespace",
	EnvKeys:      []string{"IPC", "UNSHARE_IPC"},
	ExcludedOS:   []string{cmdline.Darwin},
}

// -n|--net
var actionNetNamespaceFlag = cmdline.Flag{
	ID:           "actionNetNamespaceFlag",
	Value:        &NetNamespace,
	DefaultValue: false,
	Name:         "net",
	ShortHand:    "n",
	Usage:        "run container in a new network namespace (sets up a bridge network interface by default)",
	EnvKeys:      []string{"NET", "UNSHARE_NET"},
	ExcludedOS:   []string{cmdline.Darwin},
}

// --uts
var actionUtsNamespaceFlag = cmdline.Flag{
	ID:           "actionUtsNamespaceFlag",
	Value:        &UtsNamespace,
	DefaultValue: false,
	Name:         "uts",
	Usage:        "run container in a new UTS namespace",
	EnvKeys:      []string{"UTS", "UNSHARE_UTS"},
	ExcludedOS:   []string{cmdline.Darwin},
}

// -u|--userns
var actionUserNamespaceFlag = cmdline.Flag{
	ID:           "actionUserNamespaceFlag",
	Value:        &UserNamespace,
	DefaultValue: false,
	Name:         "userns",
	ShortHand:    "u",
	Usage:        "run container in a new user namespace, allowing Singularity to run completely unprivileged on recent kernels. This disables some features of Singularity, for example it only works with sandbox images.",
	EnvKeys:      []string{"USERNS", "UNSHARE_USERNS"},
	ExcludedOS:   []string{cmdline.Darwin},
}

// --keep-privs
var actionKeepPrivsFlag = cmdline.Flag{
	ID:           "actionKeepPrivsFlag",
	Value:        &KeepPrivs,
	DefaultValue: false,
	Name:         "keep-privs",
	Usage:        "let root user keep privileges in container (root only)",
	EnvKeys:      []string{"KEEP_PRIVS"},
	ExcludedOS:   []string{cmdline.Darwin},
}

// --no-privs
var actionNoPrivsFlag = cmdline.Flag{
	ID:           "actionNoPrivsFlag",
	Value:        &NoPrivs,
	DefaultValue: false,
	Name:         "no-privs",
	Usage:        "drop all privileges from root user in container)",
	EnvKeys:      []string{"NO_PRIVS"},
	ExcludedOS:   []string{cmdline.Darwin},
}

// --add-caps
var actionAddCapsFlag = cmdline.Flag{
	ID:           "actionAddCapsFlag",
	Value:        &AddCaps,
	DefaultValue: "",
	Name:         "add-caps",
	Usage:        "a comma separated capability list to add",
	EnvKeys:      []string{"ADD_CAPS"},
	ExcludedOS:   []string{cmdline.Darwin},
}

// --drop-caps
var actionDropCapsFlag = cmdline.Flag{
	ID:           "actionDropCapsFlag",
	Value:        &DropCaps,
	DefaultValue: "",
	Name:         "drop-caps",
	Usage:        "a comma separated capability list to drop",
	EnvKeys:      []string{"DROP_CAPS"},
	ExcludedOS:   []string{cmdline.Darwin},
}

// --allow-setuid
var actionAllowSetuidFlag = cmdline.Flag{
	ID:           "actionAllowSetuidFlag",
	Value:        &AllowSUID,
	DefaultValue: false,
	Name:         "allow-setuid",
	Usage:        "allow setuid binaries in container (root only)",
	EnvKeys:      []string{"ALLOW_SETUID"},
	ExcludedOS:   []string{cmdline.Darwin},
}

func init() {
	initializePlugins()

	cmdManager.RegisterCmd(ExecCmd)
	cmdManager.RegisterCmd(ShellCmd)
	cmdManager.RegisterCmd(RunCmd)
	cmdManager.RegisterCmd(TestCmd)

	cmdManager.SetCmdGroup("actions", ExecCmd, ShellCmd, RunCmd, TestCmd)
	actionsCmd := cmdManager.GetCmdGroup("actions")

	if InstanceStartCmd != nil {
		cmdManager.SetCmdGroup("actions_instance", ExecCmd, ShellCmd, RunCmd, TestCmd, InstanceStartCmd)
		cmdManager.RegisterFlagForCmd(&actionBootFlag, InstanceStartCmd)
	} else {
		cmdManager.SetCmdGroup("actions_instance", actionsCmd...)
	}
	actionsInstanceCmd := cmdManager.GetCmdGroup("actions_instance")

	initPlatformDefaults()

	cmdManager.RegisterFlagForCmd(&actionAppFlag, actionsCmd...)
	cmdManager.RegisterFlagForCmd(&actionBindFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionHomeFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionOverlayFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionScratchFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionWorkdirFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionShellFlag, ShellCmd)
	cmdManager.RegisterFlagForCmd(&actionHostnameFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionNetworkFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionNetworkArgsFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionDNSFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionSecurityFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionApplyCgroupsFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionVMRAMFlag, actionsCmd...)
	cmdManager.RegisterFlagForCmd(&actionVMCPUFlag, actionsCmd...)
	cmdManager.RegisterFlagForCmd(&actionVMIPFlag, actionsCmd...)
	cmdManager.RegisterFlagForCmd(&actionNONETFlag, actionsCmd...)
	cmdManager.RegisterFlagForCmd(&actionContainLibsFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionDockerUsernameFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionDockerPasswordFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionFakerootFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionDisableCacheFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionTmpDirFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionCleanEnvFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionContainFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionContainAllFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionNvidiaFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionWritableFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionWritableTmpfsFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionNoHomeFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionNoInitFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionNoHTTPSFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionDockerLoginFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionNoNvidiaFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionVMFlag, actionsCmd...)
	cmdManager.RegisterFlagForCmd(&actionVMErrFlag, actionsCmd...)
	cmdManager.RegisterFlagForCmd(&actionSyOSFlag, ShellCmd)
	cmdManager.RegisterFlagForCmd(&actionPidNamespaceFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionIpcNamespaceFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionNetNamespaceFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionUtsNamespaceFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionUserNamespaceFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionKeepPrivsFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionNoPrivsFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionAddCapsFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionDropCapsFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionAllowSetuidFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&actionPwdFlag, actionsCmd...)
	cmdManager.RegisterFlagForCmd(&pemEncryptFlag, actionsInstanceCmd...)
	cmdManager.RegisterFlagForCmd(&passEncryptFlag, actionsInstanceCmd...)

	for _, cmd := range actionsCmd {
		plugin.AddFlagHooks(cmd.Flags())
	}
}
