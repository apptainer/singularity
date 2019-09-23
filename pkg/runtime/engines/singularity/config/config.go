// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"github.com/sylabs/singularity/pkg/image"
)

// Name is the name of the runtime.
const Name = "singularity"

const (
	// DefaultLayer is the string representation for the default layer.
	DefaultLayer string = "none"
	// OverlayLayer is the string representation for the overlay layer.
	OverlayLayer = "overlay"
	// UnderlayLayer is the string representation for the underlay layer.
	UnderlayLayer = "underlay"
)

// FileConfig describes the singularity.conf file options
type FileConfig struct {
	AllowSetuid             bool     `default:"yes" authorized:"yes,no" directive:"allow setuid"`
	AllowPidNs              bool     `default:"yes" authorized:"yes,no" directive:"allow pid ns"`
	ConfigPasswd            bool     `default:"yes" authorized:"yes,no" directive:"config passwd"`
	ConfigGroup             bool     `default:"yes" authorized:"yes,no" directive:"config group"`
	ConfigResolvConf        bool     `default:"yes" authorized:"yes,no" directive:"config resolv_conf"`
	MountProc               bool     `default:"yes" authorized:"yes,no" directive:"mount proc"`
	MountSys                bool     `default:"yes" authorized:"yes,no" directive:"mount sys"`
	MountDevPts             bool     `default:"yes" authorized:"yes,no" directive:"mount devpts"`
	MountHome               bool     `default:"yes" authorized:"yes,no" directive:"mount home"`
	MountTmp                bool     `default:"yes" authorized:"yes,no" directive:"mount tmp"`
	MountHostfs             bool     `default:"no" authorized:"yes,no" directive:"mount hostfs"`
	UserBindControl         bool     `default:"yes" authorized:"yes,no" directive:"user bind control"`
	EnableFusemount         bool     `default:"yes" authorized:"yes,no" directive:"enable fusemount"`
	EnableUnderlay          bool     `default:"yes" authorized:"yes,no" directive:"enable underlay"`
	MountSlave              bool     `default:"yes" authorized:"yes,no" directive:"mount slave"`
	AllowContainerSquashfs  bool     `default:"yes" authorized:"yes,no" directive:"allow container squashfs"`
	AllowContainerExtfs     bool     `default:"yes" authorized:"yes,no" directive:"allow container extfs"`
	AllowContainerDir       bool     `default:"yes" authorized:"yes,no" directive:"allow container dir"`
	AlwaysUseNv             bool     `default:"no" authorized:"yes,no" directive:"always use nv"`
	SharedLoopDevices       bool     `default:"no" authorized:"yes,no" directive:"shared loop devices"`
	MaxLoopDevices          uint     `default:"256" directive:"max loop devices"`
	SessiondirMaxSize       uint     `default:"16" directive:"sessiondir max size"`
	MountDev                string   `default:"yes" authorized:"yes,no,minimal" directive:"mount dev"`
	EnableOverlay           string   `default:"try" authorized:"yes,no,try" directive:"enable overlay"`
	BindPath                []string `default:"/etc/localtime,/etc/hosts" directive:"bind path"`
	LimitContainerOwners    []string `directive:"limit container owners"`
	LimitContainerGroups    []string `directive:"limit container groups"`
	LimitContainerPaths     []string `directive:"limit container paths"`
	RootDefaultCapabilities string   `default:"full" authorized:"full,file,no" directive:"root default capabilities"`
	MemoryFSType            string   `default:"tmpfs" authorized:"tmpfs,ramfs" directive:"memory fs type"`
	CniConfPath             string   `directive:"cni configuration path"`
	CniPluginPath           string   `directive:"cni plugin path"`
	MksquashfsPath          string   `directive:"mksquashfs path"`
	CryptsetupPath          string   `directive:"cryptsetup path"`
}

// JSONConfig stores engine specific confguration that is allowed to be set by the user.
type JSONConfig struct {
	ScratchDir        []string      `json:"scratchdir,omitempty"`
	OverlayImage      []string      `json:"overlayImage,omitempty"`
	BindPath          []string      `json:"bindpath,omitempty"`
	NetworkArgs       []string      `json:"networkArgs,omitempty"`
	Security          []string      `json:"security,omitempty"`
	FilesPath         []string      `json:"filesPath,omitempty"`
	LibrariesPath     []string      `json:"librariesPath,omitempty"`
	ImageList         []image.Image `json:"imageList,omitempty"`
	OpenFd            []int         `json:"openFd,omitempty"`
	TargetGID         []int         `json:"targetGID,omitempty"`
	Image             string        `json:"image"`
	Workdir           string        `json:"workdir,omitempty"`
	CgroupsPath       string        `json:"cgroupsPath,omitempty"`
	HomeSource        string        `json:"homedir,omitempty"`
	HomeDest          string        `json:"homeDest,omitempty"`
	Command           string        `json:"command,omitempty"`
	Shell             string        `json:"shell,omitempty"`
	TmpDir            string        `json:"tmpdir,omitempty"`
	AddCaps           string        `json:"addCaps,omitempty"`
	DropCaps          string        `json:"dropCaps,omitempty"`
	Hostname          string        `json:"hostname,omitempty"`
	Network           string        `json:"network,omitempty"`
	DNS               string        `json:"dns,omitempty"`
	Cwd               string        `json:"cwd,omitempty"`
	SessionLayer      string        `json:"sessionLayer,omitempty"`
	EncryptionKey     []byte        `json:"encryptionKey,omitempty"`
	TargetUID         int           `json:"targetUID,omitempty"`
	WritableImage     bool          `json:"writableImage,omitempty"`
	WritableTmpfs     bool          `json:"writableTmpfs,omitempty"`
	Contain           bool          `json:"container,omitempty"`
	Nv                bool          `json:"nv,omitempty"`
	CustomHome        bool          `json:"customHome,omitempty"`
	Instance          bool          `json:"instance,omitempty"`
	InstanceJoin      bool          `json:"instanceJoin,omitempty"`
	BootInstance      bool          `json:"bootInstance,omitempty"`
	RunPrivileged     bool          `json:"runPrivileged,omitempty"`
	AllowSUID         bool          `json:"allowSUID,omitempty"`
	KeepPrivs         bool          `json:"keepPrivs,omitempty"`
	NoPrivs           bool          `json:"noPrivs,omitempty"`
	NoHome            bool          `json:"noHome,omitempty"`
	NoInit            bool          `json:"noInit,omitempty"`
	DeleteImage       bool          `json:"deleteImage,omitempty"`
	Fakeroot          bool          `json:"fakeroot,omitempty"`
	SignalPropagation bool          `json:"signalPropagation,omitempty"`
}

// SetImage sets the container image path to be used by EngineConfig.JSON.
func (e *EngineConfig) SetImage(name string) {
	e.JSON.Image = name
}

// GetImage retrieves the container image path.
func (e *EngineConfig) GetImage() string {
	return e.JSON.Image
}

// SetKey sets the key for the image's system partition.
func (e *EngineConfig) SetEncryptionKey(key []byte) {
	e.JSON.EncryptionKey = key
}

// GetKey retrieves the key for image's system partition.
func (e *EngineConfig) GetEncryptionKey() []byte {
	return e.JSON.EncryptionKey
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

// GetAllowSUID returns true if allow-suid is set and false if not.
func (e *EngineConfig) GetAllowSUID() bool {
	return e.JSON.AllowSUID
}

// SetKeepPrivs sets keep-privs flag to allow root to retain all privileges.
func (e *EngineConfig) SetKeepPrivs(keep bool) {
	e.JSON.KeepPrivs = keep
}

// GetKeepPrivs returns if keep-privs is set or not.
func (e *EngineConfig) GetKeepPrivs() bool {
	return e.JSON.KeepPrivs
}

// SetNoPrivs sets no-privs flag to force root user to lose all privileges.
func (e *EngineConfig) SetNoPrivs(nopriv bool) {
	e.JSON.NoPrivs = nopriv
}

// GetNoPrivs returns if no-privs flag is set or not.
func (e *EngineConfig) GetNoPrivs() bool {
	return e.JSON.NoPrivs
}

// SetNoHome set no-home flag to not mount home user home directory.
func (e *EngineConfig) SetNoHome(val bool) {
	e.JSON.NoHome = val
}

// GetNoHome returns if no-home flag is set or not.
func (e *EngineConfig) GetNoHome() bool {
	return e.JSON.NoHome
}

// SetNoInit set noinit flag to not start shim init process.
func (e *EngineConfig) SetNoInit(val bool) {
	e.JSON.NoInit = val
}

// GetNoInit returns if noinit flag is set or not.
func (e *EngineConfig) GetNoInit() bool {
	return e.JSON.NoInit
}

// SetNetwork sets a list of commas separated networks to configure inside container.
func (e *EngineConfig) SetNetwork(network string) {
	e.JSON.Network = network
}

// GetNetwork retrieves a list of commas separated networks configured in container.
func (e *EngineConfig) GetNetwork() string {
	return e.JSON.Network
}

// SetNetworkArgs sets network arguments to pass to CNI plugins.
func (e *EngineConfig) SetNetworkArgs(args []string) {
	e.JSON.NetworkArgs = args
}

// GetNetworkArgs retrieves network arguments passed to CNI plugins.
func (e *EngineConfig) GetNetworkArgs() []string {
	return e.JSON.NetworkArgs
}

// SetDNS sets a commas separated list of DNS servers to add in resolv.conf.
func (e *EngineConfig) SetDNS(dns string) {
	e.JSON.DNS = dns
}

// GetDNS retrieves list of DNS servers.
func (e *EngineConfig) GetDNS() string {
	return e.JSON.DNS
}

// SetImageList sets image list containing opened images.
func (e *EngineConfig) SetImageList(list []image.Image) {
	e.JSON.ImageList = list
}

// GetImageList returns image list containing opened images.
func (e *EngineConfig) GetImageList() []image.Image {
	return e.JSON.ImageList
}

// SetCwd sets current working directory.
func (e *EngineConfig) SetCwd(path string) {
	e.JSON.Cwd = path
}

// GetCwd returns current working directory.
func (e *EngineConfig) GetCwd() string {
	return e.JSON.Cwd
}

// SetOpenFd sets a list of open file descriptor.
func (e *EngineConfig) SetOpenFd(fds []int) {
	e.JSON.OpenFd = fds
}

// GetOpenFd returns the list of open file descriptor.
func (e *EngineConfig) GetOpenFd() []int {
	return e.JSON.OpenFd
}

// SetWritableTmpfs sets writable tmpfs flag.
func (e *EngineConfig) SetWritableTmpfs(writable bool) {
	e.JSON.WritableTmpfs = writable
}

// GetWritableTmpfs returns if writable tmpfs is set or no.
func (e *EngineConfig) GetWritableTmpfs() bool {
	return e.JSON.WritableTmpfs
}

// SetSecurity sets security feature arguments.
func (e *EngineConfig) SetSecurity(security []string) {
	e.JSON.Security = security
}

// GetSecurity returns security feature arguments.
func (e *EngineConfig) GetSecurity() []string {
	return e.JSON.Security
}

// SetCgroupsPath sets path to cgroups profile.
func (e *EngineConfig) SetCgroupsPath(path string) {
	e.JSON.CgroupsPath = path
}

// GetCgroupsPath returns path to cgroups profile.
func (e *EngineConfig) GetCgroupsPath() string {
	return e.JSON.CgroupsPath
}

// SetTargetUID sets target UID to execute the container process as user ID.
func (e *EngineConfig) SetTargetUID(uid int) {
	e.JSON.TargetUID = uid
}

// GetTargetUID returns the target UID.
func (e *EngineConfig) GetTargetUID() int {
	return e.JSON.TargetUID
}

// SetTargetGID sets target GIDs to execute container process as group IDs.
func (e *EngineConfig) SetTargetGID(gid []int) {
	e.JSON.TargetGID = gid
}

// GetTargetGID returns the target GIDs.
func (e *EngineConfig) GetTargetGID() []int {
	return e.JSON.TargetGID
}

// SetLibrariesPath sets libraries to bind in container
// /.singularity.d/libs directory.
func (e *EngineConfig) SetLibrariesPath(libraries []string) {
	e.JSON.LibrariesPath = libraries
}

// AppendLibrariesPath adds libraries to bind in container
// /.singularity.d/libs directory.
func (e *EngineConfig) AppendLibrariesPath(libraries ...string) {
	e.JSON.LibrariesPath = append(e.JSON.LibrariesPath, libraries...)
}

// GetLibrariesPath returns libraries to bind in container
// /.singularity.d/libs directory.
func (e *EngineConfig) GetLibrariesPath() []string {
	return e.JSON.LibrariesPath
}

// SetFilesPath sets files to bind in container (eg: --nv).
func (e *EngineConfig) SetFilesPath(files []string) {
	e.JSON.FilesPath = files
}

// AppendFilesPath adds files to bind in container (eg: --nv)
func (e *EngineConfig) AppendFilesPath(files ...string) {
	e.JSON.FilesPath = append(e.JSON.FilesPath, files...)
}

// GetFilesPath returns files to bind in container (eg: --nv).
func (e *EngineConfig) GetFilesPath() []string {
	return e.JSON.FilesPath
}

// SetFakeroot sets fakeroot flag.
func (e *EngineConfig) SetFakeroot(fakeroot bool) {
	e.JSON.Fakeroot = fakeroot
}

// GetFakeroot returns if fakeroot is set or not.
func (e *EngineConfig) GetFakeroot() bool {
	return e.JSON.Fakeroot
}

// GetDeleteImage returns if container image must be deleted after use.
func (e *EngineConfig) GetDeleteImage() bool {
	return e.JSON.DeleteImage
}

// SetDeleteImage sets if container image must be deleted after use.
func (e *EngineConfig) SetDeleteImage(delete bool) {
	e.JSON.DeleteImage = delete
}

// SetSignalPropagation sets if engine must propagate signals from
// master process -> container process when PID namespace is disabled
// or from master process -> sinit process -> container
// process when PID namespace is enabled.
func (e *EngineConfig) SetSignalPropagation(propagation bool) {
	e.JSON.SignalPropagation = propagation
}

// GetSignalPropagation returns if engine propagate signals across
// processes (see SetSignalPropagation).
func (e *EngineConfig) GetSignalPropagation() bool {
	return e.JSON.SignalPropagation
}

// GetSessionLayer returns the session layer used to setup the
// container mount points.
func (e *EngineConfig) GetSessionLayer() string {
	return e.JSON.SessionLayer
}

// SetSessionLayer sets the session layer to use to setup the
// container mount points.
func (e *EngineConfig) SetSessionLayer(sessionLayer string) {
	e.JSON.SessionLayer = sessionLayer
}
