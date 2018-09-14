// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"github.com/singularityware/singularity/src/pkg/buildcfg"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/runtime/engines/config"
	"github.com/singularityware/singularity/src/runtime/engines/config/oci"
)

// Name is the name of the runtime.
const Name = "singularity"

// FileConfig describes the singularity.conf file options
type FileConfig struct {
	AllowSetuid             bool     `default:"yes" authorized:"yes,no" directive:"allow setuid"`
	MaxLoopDevices          uint     `default:"256" directive:"max loop devices"`
	AllowPidNs              bool     `default:"yes" authorized:"yes,no" directive:"allow pid ns"`
	ConfigPasswd            bool     `default:"yes" authorized:"yes,no" directive:"config passwd"`
	ConfigGroup             bool     `default:"yes" authorized:"yes,no" directive:"config group"`
	ConfigResolvConf        bool     `default:"yes" authorized:"yes,no" directive:"config resolv_conf"`
	MountProc               bool     `default:"yes" authorized:"yes,no" directive:"mount proc"`
	MountSys                bool     `default:"yes" authorized:"yes,no" directive:"mount sys"`
	MountDev                string   `default:"yes" authorized:"yes,no,minimal" directive:"mount dev"`
	MountDevPts             bool     `default:"yes" authorized:"yes,no" directive:"mount devpts"`
	MountHome               bool     `default:"yes" authorized:"yes,no" directive:"mount home"`
	MountTmp                bool     `default:"yes" authorized:"yes,no" directive:"mount tmp"`
	MountHostfs             bool     `default:"no" authorized:"yes,no" directive:"mount hostfs"`
	BindPath                []string `default:"/etc/localtime,/etc/hosts" directive:"bind path"`
	UserBindControl         bool     `default:"yes" authorized:"yes,no" directive:"user bind control"`
	EnableOverlay           string   `default:"try" authorized:"yes,no,try" directive:"enable overlay"`
	EnableUnderlay          bool     `default:"yes" authorized:"yes,no" directive:"enable underlay"`
	MountSlave              bool     `default:"yes" authorized:"yes,no" directive:"mount slave"`
	SessiondirMaxSize       uint     `default:"16" directive:"sessiondir max size"`
	LimitContainerOwners    []string `directive:"limit container owners"`
	LimitContainerGroups    []string `directive:"limit container groups"`
	LimitContainerPaths     []string `directive:"limit container paths"`
	AllowContainerSquashfs  bool     `default:"yes" authorized:"yes,no" directive:"allow container squashfs"`
	AllowContainerExtfs     bool     `default:"yes" authorized:"yes,no" directive:"allow container extfs"`
	AllowContainerDir       bool     `default:"yes" authorized:"yes,no" directive:"allow container dir"`
	AutofsBugPath           []string `directive:"autofs bug path"`
	AlwaysUseNv             bool     `default:"no" authorized:"yes,no" directive:"always use nv"`
	RootDefaultCapabilities string   `default:"full" authorized:"full,file,no" directive:"root default capabilities"`
	AllowRootCapabilities   bool     `default:"yes" authorized:"yes,no" directive:"allow root capabilities"`
	AllowUserCapabilities   bool     `default:"no" authorized:"yes,no" directive:"allow user capabilities"`
	MemoryFSType            string   `default:"tmpfs" authorized:"tmpfs,ramfs" directive:"memory fs type"`
}

// JSONConfig stores engine specific confguration that is allowed to be set by the user
type JSONConfig struct {
	Image            string   `json:"image"`
	WritableImage    bool     `json:"writableImage,omitempty"`
	OverlayImage     []string `json:"overlayImage,omitempty"`
	OverlayFsEnabled bool     `json:"overlayFsEnabled,omitempty"`
	Contain          bool     `json:"container,omitempty"`
	Nv               bool     `json:"nv,omitempty"`
	Workdir          string   `json:"workdir,omitempty"`
	ScratchDir       []string `json:"scratchdir,omitempty"`
	HomeSource       string   `json:"homedir,omitempty"`
	HomeDest         string   `json:"homeDest,omitempty"`
	CustomHome       bool     `json:"customHome,omitempty"`
	BindPath         []string `json:"bindpath,omitempty"`
	Command          string   `json:"command,omitempty"`
	Shell            string   `json:"shell,omitempty"`
	TmpDir           string   `json:"tmpdir,omitempty"`
	Instance         bool     `json:"instance,omitempty"`
	InstanceJoin     bool     `json:"instanceJoin,omitempty"`
	BootInstance     bool     `json:"bootInstance,omitempty"`
	RunPrivileged    bool     `json:"runPrivileged,omitempty"`
	AddCaps          string   `json:"addCaps,omitempty"`
	DropCaps         string   `json:"dropCaps,omitempty"`
	Hostname         string   `json:"hostname,omitempty"`
	AllowSUID        bool     `json:"allowSUID,omitempty"`
	KeepPrivs        bool     `json:"keepPrivs,omitempty"`
	NoPrivs          bool     `json:"noPrivs,omitempty"`
	NoHome           bool     `json:"noHome,omitempty"`
	NoInit           bool     `json:"noInit,omitempty"`
}

// EngineConfig stores both the JSONConfig and the FileConfig
type EngineConfig struct {
	JSON      *JSONConfig `json:"jsonConfig"`
	OciConfig *oci.Config `json:"ociConfig"`
	File      *FileConfig `json:"-"`
}

// NewConfig returns singularity.EngineConfig with a parsed FileConfig
func NewConfig() *EngineConfig {
	c := &FileConfig{}
	if err := config.Parser(buildcfg.SYSCONFDIR+"/singularity/singularity.conf", c); err != nil {
		sylog.Fatalf("Unable to parse singularity.conf file: %s", err)
	}

	ret := &EngineConfig{
		JSON:      &JSONConfig{},
		OciConfig: &oci.Config{},
		File:      c,
	}

	return ret
}

// SetImage sets the container image path to be used by EngineConfig.JSON.
func (e *EngineConfig) SetImage(name string) {
	e.JSON.Image = name
}

// GetImage retrieves the container image path.
func (e *EngineConfig) GetImage() string {
	return e.JSON.Image
}

// SetWritableImage defines the container image as writable or not.
func (e *EngineConfig) SetWritableImage(writable bool) {
	e.JSON.WritableImage = writable
}

// GetWritableImage returns if the container image is writable or not.
func (e *EngineConfig) GetWritableImage() bool {
	return e.JSON.WritableImage
}

// SetOverlayImage sets the overlay image path to be used on top of container image.
func (e *EngineConfig) SetOverlayImage(paths []string) {
	e.JSON.OverlayImage = paths
}

// GetOverlayImage retrieves the overlay image path.
func (e *EngineConfig) GetOverlayImage() []string {
	return e.JSON.OverlayImage
}

// SetOverlayFsEnabled defines if overlay filesystem is enabled or not.
func (e *EngineConfig) SetOverlayFsEnabled(enabled bool) {
	e.JSON.OverlayFsEnabled = enabled
}

// GetOverlayFsEnabled returns if overlay filesystem is enabled or not.
func (e *EngineConfig) GetOverlayFsEnabled() bool {
	return e.JSON.OverlayFsEnabled
}

// SetContain sets contain flag.
func (e *EngineConfig) SetContain(contain bool) {
	e.JSON.Contain = contain
}

// GetContain returns if contain flag is set or not.
func (e *EngineConfig) GetContain() bool {
	return e.JSON.Contain
}

// SetNv sets nv flag to bind cuda libraries into containee.JSON.
func (e *EngineConfig) SetNv(nv bool) {
	e.JSON.Nv = nv
}

// GetNv returns if nv flag is set or not.
func (e *EngineConfig) GetNv() bool {
	return e.JSON.Nv
}

// SetWorkdir sets a work directory path.
func (e *EngineConfig) SetWorkdir(name string) {
	e.JSON.Workdir = name
}

// GetWorkdir retrieves the work directory path.
func (e *EngineConfig) GetWorkdir() string {
	return e.JSON.Workdir
}

// SetScratchDir set a scratch directory path.
func (e *EngineConfig) SetScratchDir(scratchdir []string) {
	e.JSON.ScratchDir = scratchdir
}

// GetScratchDir retrieves the scratch directory path.
func (e *EngineConfig) GetScratchDir() []string {
	return e.JSON.ScratchDir
}

// SetHomeSource sets the source home directory path.
func (e *EngineConfig) SetHomeSource(source string) {
	e.JSON.HomeSource = source
}

// GetHomeSource retrieves the source home directory path.
func (e *EngineConfig) GetHomeSource() string {
	return e.JSON.HomeSource
}

// SetHomeDest sets the container home directory path.
func (e *EngineConfig) SetHomeDest(dest string) {
	e.JSON.HomeDest = dest
}

// GetHomeDest retrieves the container home directory path.
func (e *EngineConfig) GetHomeDest() string {
	return e.JSON.HomeDest
}

// SetCustomHome sets if home path is a custom path or not.
func (e *EngineConfig) SetCustomHome(custom bool) {
	e.JSON.CustomHome = custom
}

// GetCustomHome retrieves if home path is a custom path.
func (e *EngineConfig) GetCustomHome() bool {
	return e.JSON.CustomHome
}

// SetBindPath sets paths to bind into containee.JSON.
func (e *EngineConfig) SetBindPath(bindpath []string) {
	e.JSON.BindPath = bindpath
}

// GetBindPath retrieves bind paths.
func (e *EngineConfig) GetBindPath() []string {
	return e.JSON.BindPath
}

// SetCommand sets action command to execute.
func (e *EngineConfig) SetCommand(command string) {
	e.JSON.Command = command
}

// GetCommand retrieves action command.
func (e *EngineConfig) GetCommand() string {
	return e.JSON.Command
}

// SetShell sets shell to be used by shell command.
func (e *EngineConfig) SetShell(shell string) {
	e.JSON.Shell = shell
}

// GetShell retrieves shell for shell command.
func (e *EngineConfig) GetShell() string {
	return e.JSON.Shell
}

// SetTmpDir sets temporary directory path.
func (e *EngineConfig) SetTmpDir(name string) {
	e.JSON.TmpDir = name
}

// GetTmpDir retrieves temporary directory path.
func (e *EngineConfig) GetTmpDir() string {
	return e.JSON.TmpDir
}

// SetInstance sets if container run as instance or not.
func (e *EngineConfig) SetInstance(instance bool) {
	e.JSON.Instance = instance
}

// GetInstance returns if container run as instance or not.
func (e *EngineConfig) GetInstance() bool {
	return e.JSON.Instance
}

// SetInstanceJoin sets if process joins an instance or not.
func (e *EngineConfig) SetInstanceJoin(join bool) {
	e.JSON.InstanceJoin = join
}

// GetInstanceJoin returns if process joins an instance or not.
func (e *EngineConfig) GetInstanceJoin() bool {
	return e.JSON.InstanceJoin
}

// SetBootInstance sets boot flag to execute /sbin/init as main instance process.
func (e *EngineConfig) SetBootInstance(boot bool) {
	e.JSON.BootInstance = boot
}

// GetBootInstance returns if boot flag is set or not
func (e *EngineConfig) GetBootInstance() bool {
	return e.JSON.BootInstance
}

// SetAddCaps sets bounding/effective/permitted/inheritable/ambient capabilities to add.
func (e *EngineConfig) SetAddCaps(caps string) {
	e.JSON.AddCaps = caps
}

// GetAddCaps retrieves bounding/effective/permitted/inheritable/ambient capabilities to add.
func (e *EngineConfig) GetAddCaps() string {
	return e.JSON.AddCaps
}

// SetDropCaps sets bounding/effective/permitted/inheritable/ambient capabilities to drop.
func (e *EngineConfig) SetDropCaps(caps string) {
	e.JSON.DropCaps = caps
}

// GetDropCaps retrieves bounding/effective/permitted/inheritable/ambient capabilities to drop.
func (e *EngineConfig) GetDropCaps() string {
	return e.JSON.DropCaps
}

// SetHostname sets hostname to use in containee.JSON.
func (e *EngineConfig) SetHostname(hostname string) {
	e.JSON.Hostname = hostname
}

// GetHostname retrieves hostname to use in containee.JSON.
func (e *EngineConfig) GetHostname() string {
	return e.JSON.Hostname
}

// SetAllowSUID sets allow-suid flag to allow to run setuid binary inside containee.JSON.
func (e *EngineConfig) SetAllowSUID(allow bool) {
	e.JSON.AllowSUID = allow
}

// GetAllowSUID returns if allow-suid is set or not.
func (e *EngineConfig) GetAllowSUID() bool {
	return e.JSON.AllowSUID
}

// SetKeepPrivs sets keep-privs flag to allow root to retain all privileges.
func (e *EngineConfig) SetKeepPrivs(keep bool) {
	e.JSON.KeepPrivs = keep
}

// GetKeepPrivs returns if keep-privs is set or not
func (e *EngineConfig) GetKeepPrivs() bool {
	return e.JSON.KeepPrivs
}

// SetNoPrivs set no-privs flag to force root user to lose all privileges.
func (e *EngineConfig) SetNoPrivs(nopriv bool) {
	e.JSON.NoPrivs = nopriv
}

// GetNoPrivs return if no-privs flag is set or not
func (e *EngineConfig) GetNoPrivs() bool {
	return e.JSON.NoPrivs
}

// SetNoHome set no-home flag to not mount home user home directory
func (e *EngineConfig) SetNoHome(val bool) {
	e.JSON.NoHome = val
}

// GetNoHome returns if no-home flag is set or not
func (e *EngineConfig) GetNoHome() bool {
	return e.JSON.NoHome
}

// SetNoInit set noinit flag to not start shim init process
func (e *EngineConfig) SetNoInit(val bool) {
	e.JSON.NoInit = val
}

// GetNoInit returns if noinit flag is set or not
func (e *EngineConfig) GetNoInit() bool {
	return e.JSON.NoInit
}
