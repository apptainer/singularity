// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"encoding/json"

	"github.com/singularityware/singularity/src/pkg/buildcfg"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/runtime/engines/common/config"
)

// Name is the name of the runtime.
const Name = "singularity"

// FileConfiguration describes the singularity.conf file options
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
}

type EngineConfig struct {
	JSON *JSONConfig `json:"jsonConfig"`
	File *FileConfig `json:"-"`
}

func (e *EngineConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.JSON)
}

func (e *EngineConfig) UnmarshalJSON(b []byte) error {
	c := &FileConfig{}
	if err := config.Parser(buildcfg.SYSCONFDIR+"/singularity/singularity.conf", c); err != nil {
		sylog.Fatalf("Unable to parse singularity.conf file: %s", err)
	}

	e.File = c
	return json.Unmarshal(b, e.JSON)
}

// NewSingularityConfig returns singularity.EngineConfig with a parsed FileConfig
func NewConfig() *EngineConfig {
	c := &FileConfig{}
	if err := config.Parser(buildcfg.SYSCONFDIR+"/singularity/singularity.conf", c); err != nil {
		sylog.Fatalf("Unable to parse singularity.conf file: %s", err)
	}

	ret := &EngineConfig{
		JSON: &JSONConfig{},
		File: c,
	}

	return ret
}

// SetImage sets the container image path to be used by container.JSON.
func (r *EngineConfig) SetImage(name string) {
	r.JSON.Image = name
}

// GetImage retrieves the container image path.
func (r *EngineConfig) GetImage() string {
	return r.JSON.Image
}

// SetWritableImage defines the container image as writable or not.
func (r *EngineConfig) SetWritableImage(writable bool) {
	r.JSON.WritableImage = writable
}

// GetWritableImage returns if the container image is writable or not.
func (r *EngineConfig) GetWritableImage() bool {
	return r.JSON.WritableImage
}

// SetOverlayImage sets the overlay image path to be used on top of container image.
func (r *EngineConfig) SetOverlayImage(name string) {
	r.JSON.OverlayImage = name
}

// GetOverlayImage retrieves the overlay image path.
func (r *EngineConfig) GetOverlayImage() string {
	return r.JSON.OverlayImage
}

// SetOverlayFsEnabled defines if overlay filesystem is enabled or not.
func (r *EngineConfig) SetOverlayFsEnabled(enabled bool) {
	r.JSON.OverlayFsEnabled = enabled
}

// GetOverlayFsEnabled returns if overlay filesystem is enabled or not.
func (r *EngineConfig) GetOverlayFsEnabled() bool {
	return r.JSON.OverlayFsEnabled
}

// SetContain sets contain flag.
func (r *EngineConfig) SetContain(contain bool) {
	r.JSON.Contain = contain
}

// GetContain returns if contain flag is set or not.
func (r *EngineConfig) GetContain() bool {
	return r.JSON.Contain
}

// SetNv sets nv flag to bind cuda libraries into container.JSON.
func (r *EngineConfig) SetNv(nv bool) {
	r.JSON.Nv = nv
}

// GetNv returns if nv flag is set or not.
func (r *EngineConfig) GetNv() bool {
	return r.JSON.Nv
}

// SetWorkdir sets a work directory path.
func (r *EngineConfig) SetWorkdir(name string) {
	r.JSON.Workdir = name
}

// GetWorkdir retrieves the work directory path.
func (r *EngineConfig) GetWorkdir() string {
	return r.JSON.Workdir
}

// SetScratchDir set a scratch directory path.
func (r *EngineConfig) SetScratchDir(scratchdir []string) {
	r.JSON.ScratchDir = scratchdir
}

// GetScratchDir retrieves the scratch directory path.
func (r *EngineConfig) GetScratchDir() []string {
	return r.JSON.ScratchDir
}

// SetHomeDir sets the home directory path.
func (r *EngineConfig) SetHomeDir(name string) {
	r.JSON.HomeDir = name
}

// GetHomeDir retrieves the home directory path.
func (r *EngineConfig) GetHomeDir() string {
	return r.JSON.HomeDir
}

// SetBindPath sets paths to bind into container.JSON.
func (r *EngineConfig) SetBindPath(bindpath []string) {
	r.JSON.BindPath = bindpath
}

// GetBindPath retrieves bind paths.
func (r *EngineConfig) GetBindPath() []string {
	return r.JSON.BindPath
}

// SetCommand sets action command to execute.
func (r *EngineConfig) SetCommand(command string) {
	r.JSON.Command = command
}

// GetCommand retrieves action command.
func (r *EngineConfig) GetCommand() string {
	return r.JSON.Command
}

// SetShell sets shell to be used by shell command.
func (r *EngineConfig) SetShell(shell string) {
	r.JSON.Shell = shell
}

// GetShell retrieves shell for shell command.
func (r *EngineConfig) GetShell() string {
	return r.JSON.Shell
}

// SetTmpDir sets temporary directory path.
func (r *EngineConfig) SetTmpDir(name string) {
	r.JSON.TmpDir = name
}

// GetTmpDir retrieves temporary directory path.
func (r *EngineConfig) GetTmpDir() string {
	return r.JSON.TmpDir
}

// SetInstance sets if container run as instance or not.
func (r *EngineConfig) SetInstance(instance bool) {
	r.JSON.IsInstance = instance
}

// GetInstance returns if container run as instance or not.
func (r *EngineConfig) GetInstance() bool {
	return r.JSON.IsInstance
}

// SetBootInstance sets boot flag to execute /sbin/init as main instance process.
func (r *EngineConfig) SetBootInstance(boot bool) {
	r.JSON.BootInstance = boot
}

// GetBootInstance returns if boot flag is set or not
func (r *EngineConfig) GetBootInstance() bool {
	return r.JSON.BootInstance
}

// SetAddCaps sets bounding/effective/permitted/inheritable/ambient capabilities to add.
func (r *EngineConfig) SetAddCaps(caps string) {
	r.JSON.AddCaps = caps
}

// GetAddCaps retrieves bounding/effective/permitted/inheritable/ambient capabilities to add.
func (r *EngineConfig) GetAddCaps() string {
	return r.JSON.AddCaps
}

// SetDropCaps sets bounding/effective/permitted/inheritable/ambient capabilities to drop.
func (r *EngineConfig) SetDropCaps(caps string) {
	r.JSON.DropCaps = caps
}

// GetDropCaps retrieves bounding/effective/permitted/inheritable/ambient capabilities to drop.
func (r *EngineConfig) GetDropCaps() string {
	return r.JSON.DropCaps
}

// SetHostname sets hostname to use in container.JSON.
func (r *EngineConfig) SetHostname(hostname string) {
	r.JSON.Hostname = hostname
}

// GetHostname retrieves hostname to use in container.JSON.
func (r *EngineConfig) GetHostname() string {
	return r.JSON.Hostname
}

// SetAllowSUID sets allow-suid flag to allow to run setuid binary inside container.JSON.
func (r *EngineConfig) SetAllowSUID(allow bool) {
	r.JSON.AllowSUID = allow
}

// GetAllowSUID returns if allow-suid is set or not.
func (r *EngineConfig) GetAllowSUID() bool {
	return r.JSON.AllowSUID
}

// SetKeepPrivs sets keep-privs flag to allow root to retain all privileges.
func (r *EngineConfig) SetKeepPrivs(keep bool) {
	r.JSON.KeepPrivs = keep
}

// GetKeepPrivs returns if keep-privs is set or not
func (r *EngineConfig) GetKeepPrivs() bool {
	return r.JSON.KeepPrivs
}

// SetNoPrivs set no-privs flag to force root user to lose all privileges.
func (r *EngineConfig) SetNoPrivs(nopriv bool) {
	r.JSON.NoPrivs = nopriv
}

// GetNoPrivs return if no-privs flag is set or not
func (r *EngineConfig) GetNoPrivs() bool {
	return r.JSON.NoPrivs
}

// SetHome set user home directory
func (r *EngineConfig) SetHome(home string) {
	r.JSON.Home = home
}

// GetHome retrieves user home directory
func (r *EngineConfig) GetHome() string {
	return r.JSON.Home
}
