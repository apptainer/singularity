// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package config

import (
	"github.com/singularityware/singularity/src/pkg/buildcfg"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/runtime/engines/common/config"
)

// Name is the name of the runtime.
const Name = "singularity"

// FileConfiguration describes the singularity.conf file options
type FileConfiguration struct {
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

// EngineConfig is the specification of the runtime engine configuration. This is
// parsed from `json:"engineConfig"` within config.CommonEngineConfig
type EngineConfig struct {
	Image            string   `json:"image"`
	WritableImage    bool     `json:"writableImage,omitempty"`
	OverlayImage     string   `json:"overlayImage,omitempty"`
	OverlayFsEnabled bool     `json:"overlayFsEnabled,omitempty"`
	Contain          bool     `json:"container,omitempty"`
	Nv               bool     `json:"nv,omitempty"`
	Workdir          string   `json:"workdir,omitempty"`
	ScratchDir       []string `json:"scratchdir,omitempty"`
	HomeDir          string   `json:"homedir,omitempty"`
	BindPath         []string `json:"bindpath,omitempty"`
	Command          string   `json:"command,omitempty"`
	Shell            string   `json:"shell,omitempty"`
	TmpDir           string   `json:"tmpdir,omitempty"`
	IsInstance       bool     `json:"isInstance,omitempty"`
	BootInstance     bool     `json:"bootInstance,omitempty"`
	RunPrivileged    bool     `json:"runPrivileged,omitempty"`
	AddCaps          string   `json:"addCaps,omitempty"`
	DropCaps         string   `json:"dropCaps,omitempty"`
	Hostname         string   `json:"hostname,omitempty"`
	AllowSUID        bool     `json:"allowSUID,omitempty"`
	KeepPrivs        bool     `json:"keepPrivs,omitempty"`
	NoPrivs          bool     `json:"noPrivs,omitempty"`
	Home             string   `json:"home,omitempty"`

	FileConfig *FileConfiguration `json:"fileConfig"`
}

// NewSingularityConfig returns singularity.EngineConfig with a parsed FileConfig
func NewSingularityConfig() *EngineConfig {
	c := &FileConfiguration{}
	if err := config.Parser(buildcfg.SYSCONFDIR+"/singularity/singularity.conf", c); err != nil {
		sylog.Fatalf("Unable to parse singularity.conf file: %s", err)
	}

	ret := &EngineConfig{
		FileConfig: c,
	}

	return ret
}

// SetImage sets the container image path to be used by container.
func (r *EngineConfig) SetImage(name string) {
	r.Image = name
}

// GetImage retrieves the container image path.
func (r *EngineConfig) GetImage() string {
	return r.Image
}

// SetWritableImage defines the container image as writable or not.
func (r *EngineConfig) SetWritableImage(writable bool) {
	r.WritableImage = writable
}

// GetWritableImage returns if the container image is writable or not.
func (r *EngineConfig) GetWritableImage() bool {
	return r.WritableImage
}

// SetOverlayImage sets the overlay image path to be used on top of container image.
func (r *EngineConfig) SetOverlayImage(name string) {
	r.OverlayImage = name
}

// GetOverlayImage retrieves the overlay image path.
func (r *EngineConfig) GetOverlayImage() string {
	return r.OverlayImage
}

// SetOverlayFsEnabled defines if overlay filesystem is enabled or not.
func (r *EngineConfig) SetOverlayFsEnabled(enabled bool) {
	r.OverlayFsEnabled = enabled
}

// GetOverlayFsEnabled returns if overlay filesystem is enabled or not.
func (r *EngineConfig) GetOverlayFsEnabled() bool {
	return r.OverlayFsEnabled
}

// SetContain sets contain flag.
func (r *EngineConfig) SetContain(contain bool) {
	r.Contain = contain
}

// GetContain returns if contain flag is set or not.
func (r *EngineConfig) GetContain() bool {
	return r.Contain
}

// SetNv sets nv flag to bind cuda libraries into container.
func (r *EngineConfig) SetNv(nv bool) {
	r.Nv = nv
}

// GetNv returns if nv flag is set or not.
func (r *EngineConfig) GetNv() bool {
	return r.Nv
}

// SetWorkdir sets a work directory path.
func (r *EngineConfig) SetWorkdir(name string) {
	r.Workdir = name
}

// GetWorkdir retrieves the work directory path.
func (r *EngineConfig) GetWorkdir() string {
	return r.Workdir
}

// SetScratchDir set a scratch directory path.
func (r *EngineConfig) SetScratchDir(scratchdir []string) {
	r.ScratchDir = scratchdir
}

// GetScratchDir retrieves the scratch directory path.
func (r *EngineConfig) GetScratchDir() []string {
	return r.ScratchDir
}

// SetHomeDir sets the home directory path.
func (r *EngineConfig) SetHomeDir(name string) {
	r.HomeDir = name
}

// GetHomeDir retrieves the home directory path.
func (r *EngineConfig) GetHomeDir() string {
	return r.HomeDir
}

// SetBindPath sets paths to bind into container.
func (r *EngineConfig) SetBindPath(bindpath []string) {
	r.BindPath = bindpath
}

// GetBindPath retrieves bind paths.
func (r *EngineConfig) GetBindPath() []string {
	return r.BindPath
}

// SetCommand sets action command to execute.
func (r *EngineConfig) SetCommand(command string) {
	r.Command = command
}

// GetCommand retrieves action command.
func (r *EngineConfig) GetCommand() string {
	return r.Command
}

// SetShell sets shell to be used by shell command.
func (r *EngineConfig) SetShell(shell string) {
	r.Shell = shell
}

// GetShell retrieves shell for shell command.
func (r *EngineConfig) GetShell() string {
	return r.Shell
}

// SetTmpDir sets temporary directory path.
func (r *EngineConfig) SetTmpDir(name string) {
	r.TmpDir = name
}

// GetTmpDir retrieves temporary directory path.
func (r *EngineConfig) GetTmpDir() string {
	return r.TmpDir
}

// SetInstance sets if container run as instance or not.
func (r *EngineConfig) SetInstance(instance bool) {
	r.IsInstance = instance
}

// GetInstance returns if container run as instance or not.
func (r *EngineConfig) GetInstance() bool {
	return r.IsInstance
}

// SetBootInstance sets boot flag to execute /sbin/init as main instance process.
func (r *EngineConfig) SetBootInstance(boot bool) {
	r.BootInstance = boot
}

// GetBootInstance returns if boot flag is set or not
func (r *EngineConfig) GetBootInstance() bool {
	return r.BootInstance
}

// SetAddCaps sets bounding/effective/permitted/inheritable/ambient capabilities to add.
func (r *EngineConfig) SetAddCaps(caps string) {
	r.AddCaps = caps
}

// GetAddCaps retrieves bounding/effective/permitted/inheritable/ambient capabilities to add.
func (r *EngineConfig) GetAddCaps() string {
	return r.AddCaps
}

// SetDropCaps sets bounding/effective/permitted/inheritable/ambient capabilities to drop.
func (r *EngineConfig) SetDropCaps(caps string) {
	r.DropCaps = caps
}

// GetDropCaps retrieves bounding/effective/permitted/inheritable/ambient capabilities to drop.
func (r *EngineConfig) GetDropCaps() string {
	return r.DropCaps
}

// SetHostname sets hostname to use in container.
func (r *EngineConfig) SetHostname(hostname string) {
	r.Hostname = hostname
}

// GetHostname retrieves hostname to use in container.
func (r *EngineConfig) GetHostname() string {
	return r.Hostname
}

// SetAllowSUID sets allow-suid flag to allow to run setuid binary inside container.
func (r *EngineConfig) SetAllowSUID(allow bool) {
	r.AllowSUID = allow
}

// GetAllowSUID returns if allow-suid is set or not.
func (r *EngineConfig) GetAllowSUID() bool {
	return r.AllowSUID
}

// SetKeepPrivs sets keep-privs flag to allow root to retain all privileges.
func (r *EngineConfig) SetKeepPrivs(keep bool) {
	r.KeepPrivs = keep
}

// GetKeepPrivs returns if keep-privs is set or not
func (r *EngineConfig) GetKeepPrivs() bool {
	return r.KeepPrivs
}

// SetNoPrivs set no-privs flag to force root user to lose all privileges.
func (r *EngineConfig) SetNoPrivs(nopriv bool) {
	r.NoPrivs = nopriv
}

// GetNoPrivs return if no-privs flag is set or not
func (r *EngineConfig) GetNoPrivs() bool {
	return r.NoPrivs
}

// SetHome set user home directory
func (r *EngineConfig) SetHome(home string) {
	r.Home = home
}

// GetHome retrieves user home directory
func (r *EngineConfig) GetHome() string {
	return r.Home
}
