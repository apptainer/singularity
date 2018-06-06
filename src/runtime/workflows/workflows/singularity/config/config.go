// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package config

import (
	"fmt"

	"github.com/singularityware/singularity/src/pkg/buildcfg"

	config "github.com/singularityware/singularity/src/pkg/workflows/config"
	oci "github.com/singularityware/singularity/src/pkg/workflows/oci/config"
)

// Name is the name of the runtime.
const Name = "singularity"

// Configuration describes the runtime configuration.
type Configuration struct {
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

// RuntimeEngineSpec is the specification of the runtime engine configuration.
type RuntimeEngineSpec struct {
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

// RuntimeEngineConfig is the configuration of the runtime engine.
type RuntimeEngineConfig struct {
	config.RuntimeConfig
	RuntimeEngineSpec RuntimeEngineSpec `json:"runtimeConfig"`
	FileConfig        *Configuration
}

// NewSingularityConfig returns a new Singularity configuration.
func NewSingularityConfig(containerID string) (*oci.RuntimeOciConfig, *RuntimeEngineConfig) {
	c := &Configuration{}
	if err := config.Parser(buildcfg.SYSCONFDIR+"/singularity/singularity.conf", c); err != nil {
		fmt.Println(err)
	}
	runtimecfg := &RuntimeEngineConfig{FileConfig: c}
	cfg := &runtimecfg.RuntimeConfig
	runtimecfg.RuntimeSpec.ID = containerID
	runtimecfg.RuntimeSpec.RuntimeName = Name
	runtimecfg.RuntimeSpec.RuntimeOciSpec = &cfg.OciConfig.RuntimeOciSpec
	runtimecfg.RuntimeSpec.RuntimeEngineSpec = &runtimecfg.RuntimeEngineSpec
	oci.DefaultRuntimeOciConfig(&cfg.OciConfig)
	return &cfg.OciConfig, runtimecfg
}

// SetImage sets the container image path to be used by container.
func (r *RuntimeEngineConfig) SetImage(name string) {
	r.RuntimeEngineSpec.Image = name
}

// GetImage retrieves the container image path.
func (r *RuntimeEngineConfig) GetImage() string {
	return r.RuntimeEngineSpec.Image
}

// SetWritableImage defines the container image as writable or not.
func (r *RuntimeEngineConfig) SetWritableImage(writable bool) {
	r.RuntimeEngineSpec.WritableImage = writable
}

// GetWritableImage returns if the container image is writable or not.
func (r *RuntimeEngineConfig) GetWritableImage() bool {
	return r.RuntimeEngineSpec.WritableImage
}

// SetOverlayImage sets the overlay image path to be used on top of container image.
func (r *RuntimeEngineConfig) SetOverlayImage(name string) {
	r.RuntimeEngineSpec.OverlayImage = name
}

// GetOverlayImage retrieves the overlay image path.
func (r *RuntimeEngineConfig) GetOverlayImage() string {
	return r.RuntimeEngineSpec.OverlayImage
}

// SetOverlayFsEnabled defines if overlay filesystem is enabled or not.
func (r *RuntimeEngineConfig) SetOverlayFsEnabled(enabled bool) {
	r.RuntimeEngineSpec.OverlayFsEnabled = enabled
}

// GetOverlayFsEnabled returns if overlay filesystem is enabled or not.
func (r *RuntimeEngineConfig) GetOverlayFsEnabled() bool {
	return r.RuntimeEngineSpec.OverlayFsEnabled
}

// SetContain sets contain flag.
func (r *RuntimeEngineConfig) SetContain(contain bool) {
	r.RuntimeEngineSpec.Contain = contain
}

// GetContain returns if contain flag is set or not.
func (r *RuntimeEngineConfig) GetContain() bool {
	return r.RuntimeEngineSpec.Contain
}

// SetNv sets nv flag to bind cuda libraries into container.
func (r *RuntimeEngineConfig) SetNv(nv bool) {
	r.RuntimeEngineSpec.Nv = nv
}

// GetNv returns if nv flag is set or not.
func (r *RuntimeEngineConfig) GetNv() bool {
	return r.RuntimeEngineSpec.Nv
}

// SetWorkdir sets a work directory path.
func (r *RuntimeEngineConfig) SetWorkdir(name string) {
	r.RuntimeEngineSpec.Workdir = name
}

// GetWorkdir retrieves the work directory path.
func (r *RuntimeEngineConfig) GetWorkdir() string {
	return r.RuntimeEngineSpec.Workdir
}

// SetScratchDir set a scratch directory path.
func (r *RuntimeEngineConfig) SetScratchDir(scratchdir []string) {
	r.RuntimeEngineSpec.ScratchDir = scratchdir
}

// GetScratchDir retrieves the scratch directory path.
func (r *RuntimeEngineConfig) GetScratchDir() []string {
	return r.RuntimeEngineSpec.ScratchDir
}

// SetHomeDir sets the home directory path.
func (r *RuntimeEngineConfig) SetHomeDir(name string) {
	r.RuntimeEngineSpec.HomeDir = name
}

// GetHomeDir retrieves the home directory path.
func (r *RuntimeEngineConfig) GetHomeDir() string {
	return r.RuntimeEngineSpec.HomeDir
}

// SetBindPath sets paths to bind into container.
func (r *RuntimeEngineConfig) SetBindPath(bindpath []string) {
	r.RuntimeEngineSpec.BindPath = bindpath
}

// GetBindPath retrieves bind paths.
func (r *RuntimeEngineConfig) GetBindPath() []string {
	return r.RuntimeEngineSpec.BindPath
}

// SetCommand sets action command to execute.
func (r *RuntimeEngineConfig) SetCommand(command string) {
	r.RuntimeEngineSpec.Command = command
}

// GetCommand retrieves action command.
func (r *RuntimeEngineConfig) GetCommand() string {
	return r.RuntimeEngineSpec.Command
}

// SetShell sets shell to be used by shell command.
func (r *RuntimeEngineConfig) SetShell(shell string) {
	r.RuntimeEngineSpec.Shell = shell
}

// GetShell retrieves shell for shell command.
func (r *RuntimeEngineConfig) GetShell() string {
	return r.RuntimeEngineSpec.Shell
}

// SetTmpDir sets temporary directory path.
func (r *RuntimeEngineConfig) SetTmpDir(name string) {
	r.RuntimeEngineSpec.TmpDir = name
}

// GetTmpDir retrieves temporary directory path.
func (r *RuntimeEngineConfig) GetTmpDir() string {
	return r.RuntimeEngineSpec.TmpDir
}

// SetInstance sets if container run as instance or not.
func (r *RuntimeEngineConfig) SetInstance(instance bool) {
	r.RuntimeEngineSpec.IsInstance = instance
}

// GetInstance returns if container run as instance or not.
func (r *RuntimeEngineConfig) GetInstance() bool {
	return r.RuntimeEngineSpec.IsInstance
}

// SetBootInstance sets boot flag to execute /sbin/init as main instance process.
func (r *RuntimeEngineConfig) SetBootInstance(boot bool) {
	r.RuntimeEngineSpec.BootInstance = boot
}

// GetBootInstance returns if boot flag is set or not
func (r *RuntimeEngineConfig) GetBootInstance() bool {
	return r.RuntimeEngineSpec.BootInstance
}

// SetAddCaps sets bounding/effective/permitted/inheritable/ambient capabilities to add.
func (r *RuntimeEngineConfig) SetAddCaps(caps string) {
	r.RuntimeEngineSpec.AddCaps = caps
}

// GetAddCaps retrieves bounding/effective/permitted/inheritable/ambient capabilities to add.
func (r *RuntimeEngineConfig) GetAddCaps() string {
	return r.RuntimeEngineSpec.AddCaps
}

// SetDropCaps sets bounding/effective/permitted/inheritable/ambient capabilities to drop.
func (r *RuntimeEngineConfig) SetDropCaps(caps string) {
	r.RuntimeEngineSpec.DropCaps = caps
}

// GetDropCaps retrieves bounding/effective/permitted/inheritable/ambient capabilities to drop.
func (r *RuntimeEngineConfig) GetDropCaps() string {
	return r.RuntimeEngineSpec.DropCaps
}

// SetHostname sets hostname to use in container.
func (r *RuntimeEngineConfig) SetHostname(hostname string) {
	r.RuntimeEngineSpec.Hostname = hostname
}

// GetHostname retrieves hostname to use in container.
func (r *RuntimeEngineConfig) GetHostname() string {
	return r.RuntimeEngineSpec.Hostname
}

// SetAllowSUID sets allow-suid flag to allow to run setuid binary inside container.
func (r *RuntimeEngineConfig) SetAllowSUID(allow bool) {
	r.RuntimeEngineSpec.AllowSUID = allow
}

// GetAllowSUID returns if allow-suid is set or not.
func (r *RuntimeEngineConfig) GetAllowSUID() bool {
	return r.RuntimeEngineSpec.AllowSUID
}

// SetKeepPrivs sets keep-privs flag to allow root to retain all privileges.
func (r *RuntimeEngineConfig) SetKeepPrivs(keep bool) {
	r.RuntimeEngineSpec.KeepPrivs = keep
}

// GetKeepPrivs returns if keep-privs is set or not
func (r *RuntimeEngineConfig) GetKeepPrivs() bool {
	return r.RuntimeEngineSpec.KeepPrivs
}

// SetNoPrivs set no-privs flag to force root user to lose all privileges.
func (r *RuntimeEngineConfig) SetNoPrivs(nopriv bool) {
	r.RuntimeEngineSpec.NoPrivs = nopriv
}

// GetNoPrivs return if no-privs flag is set or not
func (r *RuntimeEngineConfig) GetNoPrivs() bool {
	return r.RuntimeEngineSpec.NoPrivs
}

// SetHome set user home directory
func (r *RuntimeEngineConfig) SetHome(home string) {
	r.RuntimeEngineSpec.Home = home
}

// GetHome retrieves user home directory
func (r *RuntimeEngineConfig) GetHome() string {
	return r.RuntimeEngineSpec.Home
}
