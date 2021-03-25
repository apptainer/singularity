// Copyright (c) 2019-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/sylabs/singularity/internal/pkg/runtime/engine/config/oci"
	"github.com/sylabs/singularity/pkg/image"
	"github.com/sylabs/singularity/pkg/util/singularityconf"
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

// EngineConfig stores the JSONConfig, the OciConfig and the File configuration.
type EngineConfig struct {
	JSON      *JSONConfig `json:"jsonConfig"`
	OciConfig *oci.Config `json:"ociConfig"`

	// File is not passed across stage but stay here for
	// convenient use by runtime code and plugins.
	File *singularityconf.File `json:"-"`
}

// NewConfig returns singularity.EngineConfig.
func NewConfig() *EngineConfig {
	ret := &EngineConfig{
		JSON:      new(JSONConfig),
		OciConfig: new(oci.Config),
		File:      new(singularityconf.File),
	}
	return ret
}

// FuseMount stores the FUSE-related information required or provided by
// plugins implementing options to add FUSE filesystems in the
// container.
type FuseMount struct {
	Program       []string  `json:"program,omitempty"`       // the FUSE driver program and all required arguments
	MountPoint    string    `json:"mountPoint,omitempty"`    // the mount point for the FUSE filesystem
	Fd            int       `json:"fd,omitempty"`            // /dev/fuse file descriptor
	FromContainer bool      `json:"fromContainer,omitempty"` // is FUSE driver program is run from container or from host
	Daemon        bool      `json:"daemon,omitempty"`        // is FUSE driver program is run in daemon/background mode
	Cmd           *exec.Cmd `json:"-"`                       // holds the process exec command when FUSE driver run in foreground mode
}

// BindOption represents a bind option with its associated
// value if any.
type BindOption struct {
	Value string `json:"value,omitempty"`
}

// BindPath stores bind path.
type BindPath struct {
	Source      string                 `json:"source"`
	Destination string                 `json:"destination"`
	Options     map[string]*BindOption `json:"options"`
}

// ImageSrc returns the value of option image-src or an empty
// string if the option wasn't set.
func (b *BindPath) ImageSrc() string {
	if b.Options != nil && b.Options["image-src"] != nil {
		src := b.Options["image-src"].Value
		if src == "" {
			return "/"
		}
		return src
	}
	return ""
}

// ImageSrc returns the value of option id or an empty
// string if the option wasn't set.
func (b *BindPath) ID() string {
	if b.Options != nil && b.Options["id"] != nil {
		return b.Options["id"].Value
	}
	return ""
}

// Readonly returns the option ro was set or not.
func (b *BindPath) Readonly() bool {
	return b.Options != nil && b.Options["ro"] != nil
}

// JSONConfig stores engine specific confguration that is allowed to be set by the user.
type JSONConfig struct {
	ScratchDir        []string          `json:"scratchdir,omitempty"`
	OverlayImage      []string          `json:"overlayImage,omitempty"`
	NetworkArgs       []string          `json:"networkArgs,omitempty"`
	Security          []string          `json:"security,omitempty"`
	FilesPath         []string          `json:"filesPath,omitempty"`
	LibrariesPath     []string          `json:"librariesPath,omitempty"`
	FuseMount         []FuseMount       `json:"fuseMount,omitempty"`
	ImageList         []image.Image     `json:"imageList,omitempty"`
	BindPath          []BindPath        `json:"bindpath,omitempty"`
	SingularityEnv    map[string]string `json:"singularityEnv,omitempty"`
	UnixSocketPair    [2]int            `json:"unixSocketPair,omitempty"`
	OpenFd            []int             `json:"openFd,omitempty"`
	TargetGID         []int             `json:"targetGID,omitempty"`
	Image             string            `json:"image"`
	Workdir           string            `json:"workdir,omitempty"`
	CgroupsPath       string            `json:"cgroupsPath,omitempty"`
	HomeSource        string            `json:"homedir,omitempty"`
	HomeDest          string            `json:"homeDest,omitempty"`
	Command           string            `json:"command,omitempty"`
	Shell             string            `json:"shell,omitempty"`
	TmpDir            string            `json:"tmpdir,omitempty"`
	AddCaps           string            `json:"addCaps,omitempty"`
	DropCaps          string            `json:"dropCaps,omitempty"`
	Hostname          string            `json:"hostname,omitempty"`
	Network           string            `json:"network,omitempty"`
	DNS               string            `json:"dns,omitempty"`
	Cwd               string            `json:"cwd,omitempty"`
	SessionLayer      string            `json:"sessionLayer,omitempty"`
	ConfigurationFile string            `json:"configurationFile,omitempty"`
	EncryptionKey     []byte            `json:"encryptionKey,omitempty"`
	TargetUID         int               `json:"targetUID,omitempty"`
	WritableImage     bool              `json:"writableImage,omitempty"`
	WritableTmpfs     bool              `json:"writableTmpfs,omitempty"`
	Contain           bool              `json:"container,omitempty"`
	Nv                bool              `json:"nv,omitempty"`
	Rocm              bool              `json:"rocm,omitempty"`
	CustomHome        bool              `json:"customHome,omitempty"`
	Instance          bool              `json:"instance,omitempty"`
	InstanceJoin      bool              `json:"instanceJoin,omitempty"`
	BootInstance      bool              `json:"bootInstance,omitempty"`
	RunPrivileged     bool              `json:"runPrivileged,omitempty"`
	AllowSUID         bool              `json:"allowSUID,omitempty"`
	KeepPrivs         bool              `json:"keepPrivs,omitempty"`
	NoPrivs           bool              `json:"noPrivs,omitempty"`
	NoProc            bool              `json:"noProc,omitempty"`
	NoSys             bool              `json:"noSys,omitempty"`
	NoDev             bool              `json:"noDev,omitempty"`
	NoDevPts          bool              `json:"noDevPts,omitempty"`
	NoHome            bool              `json:"noHome,omitempty"`
	NoTmp             bool              `json:"noTmp,omitempty"`
	NoHostfs          bool              `json:"noHostfs,omitempty"`
	NoCwd             bool              `json:"noCwd,omitempty"`
	NoInit            bool              `json:"noInit,omitempty"`
	Fakeroot          bool              `json:"fakeroot,omitempty"`
	SignalPropagation bool              `json:"signalPropagation,omitempty"`
	RestoreUmask      bool              `json:"restoreUmask,omitempty"`
	DeleteTempDir     string            `json:"deleteTempDir,omitempty"`
	Umask             int               `json:"umask,omitempty"`
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

// SetRocm sets rocm flag to bind rocm libraries into containee.JSON.
func (e *EngineConfig) SetRocm(rocm bool) {
	e.JSON.Rocm = rocm
}

// GetRocm returns if rocm flag is set or not.
func (e *EngineConfig) GetRocm() bool {
	return e.JSON.Rocm
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

// ParseBindPath parses a string and returns all encountered
// bind paths as array.
func ParseBindPath(bindpaths string) ([]BindPath, error) {
	var bind string
	var binds []BindPath
	var elem int

	var validOptions = map[string]bool{
		"ro":        true,
		"rw":        true,
		"image-src": false,
		"id":        false,
	}

	// there is a better regular expression to handle
	// that directly without all the logic below ...
	// we need to parse various syntax:
	// source1
	// source1:destination1
	// source1:destination1:option1
	// source1:destination1:option1,option2
	// source1,source2
	// source1:destination1:option1,source2
	re := regexp.MustCompile(`([^,^:]+:?)`)

	// with the regex above we get string array:
	// - source1 -> [source1]
	// - source1:destination1 -> [source1:, destination1]
	// - source1:destination1:option1 -> [source1:, destination1:, option1]
	// - source1:destination1:option1,option2 -> [source1:, destination1:, option1, option2]
	for _, m := range re.FindAllString(bindpaths, -1) {
		s := strings.TrimSpace(m)
		isColon := bind != "" && bind[len(bind)-1] == ':'

		// options are taken only if the bind has a source
		// and a destination
		if elem == 2 {
			isOption := false

			for option, flag := range validOptions {
				if flag {
					if s == option {
						isOption = true
						break
					}
				} else {
					if strings.HasPrefix(s, option+"=") {
						isOption = true
						break
					}
				}
			}
			if isOption {
				if !isColon {
					bind += ","
				}
				bind += s
				continue
			}
		} else if elem > 2 {
			return nil, fmt.Errorf("wrong bind syntax: %s", bind)
		}

		elem++

		if bind != "" {
			if isColon {
				bind += s
				continue
			}
			bp, err := newBindPath(bind, validOptions)
			if err != nil {
				return nil, fmt.Errorf("while getting bind path: %s", err)
			}
			binds = append(binds, bp)
			elem = 1
		}
		// new bind path
		bind = s
	}

	if bind != "" {
		bp, err := newBindPath(bind, validOptions)
		if err != nil {
			return nil, fmt.Errorf("while getting bind path: %s", err)
		}
		binds = append(binds, bp)
	}

	return binds, nil
}

// newBindPath returns BindPath record based on the provided bind
// string argument and ensures that the options are valid.
func newBindPath(bind string, validOptions map[string]bool) (BindPath, error) {
	var bp BindPath

	splitted := strings.SplitN(bind, ":", 3)

	bp.Source = splitted[0]
	if bp.Source == "" {
		return bp, fmt.Errorf("empty bind source for bind path %q", bind)
	}

	bp.Destination = bp.Source

	if len(splitted) > 1 {
		bp.Destination = splitted[1]
	}

	if len(splitted) > 2 {
		bp.Options = make(map[string]*BindOption)

		for _, value := range strings.Split(splitted[2], ",") {
			valid := false
			for optName, optFlag := range validOptions {
				if optFlag && optName == value {
					bp.Options[optName] = &BindOption{}
					valid = true
					break
				} else if strings.HasPrefix(value, optName+"=") {
					bp.Options[optName] = &BindOption{Value: value[len(optName+"="):]}
					valid = true
					break
				}
			}
			if !valid {
				return bp, fmt.Errorf("%s is not a valid bind option", value)
			}
		}
	}

	return bp, nil
}

// SetBindPath sets the paths to bind into container.
func (e *EngineConfig) SetBindPath(bindpath []BindPath) {
	e.JSON.BindPath = bindpath
}

// GetBindPath retrieves the bind paths.
func (e *EngineConfig) GetBindPath() []BindPath {
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

// SetNoProc set flag to not mount proc directory.
func (e *EngineConfig) SetNoProc(val bool) {
	e.JSON.NoProc = val
}

// GetNoProc returns if no-proc flag is set or not.
func (e *EngineConfig) GetNoProc() bool {
	return e.JSON.NoProc
}

// SetNoSys set flag to not mount sys directory.
func (e *EngineConfig) SetNoSys(val bool) {
	e.JSON.NoSys = val
}

// GetNoSys returns if no-sys flag is set or not.
func (e *EngineConfig) GetNoSys() bool {
	return e.JSON.NoSys
}

// SetNoDev set flag to not mount dev directory.
func (e *EngineConfig) SetNoDev(val bool) {
	e.JSON.NoDev = val
}

// GetNoDev returns if no-dev flag is set or not.
func (e *EngineConfig) GetNoDev() bool {
	return e.JSON.NoDev
}

// SetNoDevPts set flag to not mount dev directory.
func (e *EngineConfig) SetNoDevPts(val bool) {
	e.JSON.NoDevPts = val
}

// GetNoDevPts returns if no-devpts flag is set or not.
func (e *EngineConfig) GetNoDevPts() bool {
	return e.JSON.NoDevPts
}

// SetNoHome set flag to not mount user home directory.
func (e *EngineConfig) SetNoHome(val bool) {
	e.JSON.NoHome = val
}

// GetNoHome returns if no-home flag is set or not.
func (e *EngineConfig) GetNoHome() bool {
	return e.JSON.NoHome
}

// SetNoTmp set flag to not mount tmp directories
func (e *EngineConfig) SetNoTmp(val bool) {
	e.JSON.NoTmp = val
}

// GetNoTmp returns if no-tmo flag is set or not.
func (e *EngineConfig) GetNoTmp() bool {
	return e.JSON.NoTmp
}

// SetNoHostFs set flag to not mount all host mounts.
func (e *EngineConfig) SetNoHostfs(val bool) {
	e.JSON.NoHostfs = val
}

// SetNoHostfs returns if no-hostfs flag is set or not.
func (e *EngineConfig) GetNoHostfs() bool {
	return e.JSON.NoHostfs
}

// SetNoCwd set flag to not mount CWD
func (e *EngineConfig) SetNoCwd(val bool) {
	e.JSON.NoCwd = val
}

// SetNoCwd returns if no-cwd flag is set or not.
func (e *EngineConfig) GetNoCwd() bool {
	return e.JSON.NoCwd
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

// GetDeleteTempDir returns the path of the temporary directory containing the root filesystem
// which must be deleted after use. If no deletion is required, the empty string is returned.
func (e *EngineConfig) GetDeleteTempDir() string {
	return e.JSON.DeleteTempDir
}

// SetDeleteTempDir sets dir as the path of the temporary directory containing the root filesystem,
// which must be deleted after use.
func (e *EngineConfig) SetDeleteTempDir(dir string) {
	e.JSON.DeleteTempDir = dir
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

// SetFuseMount takes a list of fuse mount options and sets
// fuse mount configuration accordingly.
func (e *EngineConfig) SetFuseMount(mount []string) error {
	e.JSON.FuseMount = make([]FuseMount, len(mount))

	for i, mountspec := range mount {
		words := strings.Fields(mountspec)

		if len(words) == 0 {
			continue
		} else if len(words) == 1 {
			return fmt.Errorf("no whitespace separators found in command %q", words[0])
		}

		prefix := strings.SplitN(words[0], ":", 2)[0]

		words[0] = strings.Replace(words[0], prefix+":", "", 1)

		e.JSON.FuseMount[i].Fd = -1
		e.JSON.FuseMount[i].MountPoint = words[len(words)-1]
		e.JSON.FuseMount[i].Program = words[0 : len(words)-1]

		switch prefix {
		case "container":
			e.JSON.FuseMount[i].FromContainer = true
		case "container-daemon":
			e.JSON.FuseMount[i].FromContainer = true
			e.JSON.FuseMount[i].Daemon = true
		case "host":
			e.JSON.FuseMount[i].FromContainer = false
		case "host-daemon":
			e.JSON.FuseMount[i].FromContainer = false
			e.JSON.FuseMount[i].Daemon = true
		default:
			return fmt.Errorf("fusemount spec begin with an unknown prefix %s", prefix)
		}
	}

	return nil
}

// GetFuseMount returns the list of fuse mount after processing
// by SetFuseMount.
func (e *EngineConfig) GetFuseMount() []FuseMount {
	return e.JSON.FuseMount
}

// SetUnixSocketPair sets a unix socketpair used to pass file
// descriptors between RPC and master process, actually used
// to pass /dev/fuse file descriptors.
func (e *EngineConfig) SetUnixSocketPair(fds [2]int) {
	e.JSON.UnixSocketPair = fds
}

// GetUnixSocketPair returns the unix socketpair previously set
// in stage one by the engine.
func (e *EngineConfig) GetUnixSocketPair() [2]int {
	return e.JSON.UnixSocketPair
}

// SetSingularityEnv sets singularity environment variables
// as a key/value string map.
func (e *EngineConfig) SetSingularityEnv(senv map[string]string) {
	e.JSON.SingularityEnv = senv
}

// GetSingularityEnv returns singularity environment variables
// as a key/value string map.
func (e *EngineConfig) GetSingularityEnv() map[string]string {
	return e.JSON.SingularityEnv
}

// SetConfigurationFile sets the singularity configuration file to
// use instead of the default one.
func (e *EngineConfig) SetConfigurationFile(filename string) {
	e.JSON.ConfigurationFile = filename
}

// GetConfigurationFile returns the singularity configuration file to use.
func (e *EngineConfig) GetConfigurationFile() string {
	return e.JSON.ConfigurationFile
}

// SetRestoreUmask returns whether to restore Umask for the container launched process.
func (e *EngineConfig) SetRestoreUmask(restoreUmask bool) {
	e.JSON.RestoreUmask = restoreUmask
}

// GetRestoreUmask returns the umask to be used in the container launched process.
func (e *EngineConfig) GetRestoreUmask() bool {
	return e.JSON.RestoreUmask
}

// SetUmask sets the umask to be used in the container launched process.
func (e *EngineConfig) SetUmask(umask int) {
	e.JSON.Umask = umask
}

// SetUmask returns the umask to be used in the container launched process.
func (e *EngineConfig) GetUmask() int {
	return e.JSON.Umask
}
