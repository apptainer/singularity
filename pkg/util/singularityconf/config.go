// Copyright (c) 2019-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularityconf

// currentConfig corresponds to the current configuration, may
// be useful for packages requiring to share the same configuration.
var currentConfig *File

// SetCurrentConfig sets the provided configuration as the current
// configuration.
func SetCurrentConfig(config *File) {
	currentConfig = config
}

// GetCurrentConfig returns the current configuration if any.
func GetCurrentConfig() *File {
	return currentConfig
}

// File describes the singularity.conf file options
type File struct {
	BindPath                   []string `default:"/etc/localtime,/etc/hosts" directive:"bind path"`
	LimitContainerOwners       []string `directive:"limit container owners"`
	LimitContainerGroups       []string `directive:"limit container groups"`
	LimitContainerPaths        []string `directive:"limit container paths"`
	MountDev                   string   `default:"yes" authorized:"yes,no,minimal" directive:"mount dev"`
	EnableOverlay              string   `default:"try" authorized:"yes,no,try,driver" directive:"enable overlay"`
	RootDefaultCapabilities    string   `default:"full" authorized:"full,file,no" directive:"root default capabilities"`
	MemoryFSType               string   `default:"tmpfs" authorized:"tmpfs,ramfs" directive:"memory fs type"`
	CniConfPath                string   `directive:"cni configuration path"`
	CniPluginPath              string   `directive:"cni plugin path"`
	MksquashfsPath             string   `directive:"mksquashfs path"`
	MksquashfsMem              string   `directive:"mksquashfs mem"`
	CryptsetupPath             string   `directive:"cryptsetup path"`
	ImageDriver                string   `directive:"image driver"`
	DefaultKeyserver           string   `directive:"default keyserver"`
	MaxLoopDevices             uint     `default:"256" directive:"max loop devices"`
	SessiondirMaxSize          uint     `default:"16" directive:"sessiondir max size"`
	MksquashfsProcs            uint     `default:"0" directive:"mksquashfs procs"`
	AllowSetuid                bool     `default:"yes" authorized:"yes,no" directive:"allow setuid"`
	AllowPidNs                 bool     `default:"yes" authorized:"yes,no" directive:"allow pid ns"`
	ConfigPasswd               bool     `default:"yes" authorized:"yes,no" directive:"config passwd"`
	ConfigGroup                bool     `default:"yes" authorized:"yes,no" directive:"config group"`
	ConfigResolvConf           bool     `default:"yes" authorized:"yes,no" directive:"config resolv_conf"`
	MountProc                  bool     `default:"yes" authorized:"yes,no" directive:"mount proc"`
	MountSys                   bool     `default:"yes" authorized:"yes,no" directive:"mount sys"`
	MountDevPts                bool     `default:"yes" authorized:"yes,no" directive:"mount devpts"`
	MountHome                  bool     `default:"yes" authorized:"yes,no" directive:"mount home"`
	MountTmp                   bool     `default:"yes" authorized:"yes,no" directive:"mount tmp"`
	MountHostfs                bool     `default:"no" authorized:"yes,no" directive:"mount hostfs"`
	UserBindControl            bool     `default:"yes" authorized:"yes,no" directive:"user bind control"`
	EnableFusemount            bool     `default:"yes" authorized:"yes,no" directive:"enable fusemount"`
	EnableUnderlay             bool     `default:"yes" authorized:"yes,no" directive:"enable underlay"`
	MountSlave                 bool     `default:"yes" authorized:"yes,no" directive:"mount slave"`
	AllowContainerSquashfs     bool     `default:"yes" authorized:"yes,no" directive:"allow container squashfs"`
	AllowContainerExtfs        bool     `default:"yes" authorized:"yes,no" directive:"allow container extfs"`
	AllowContainerDir          bool     `default:"yes" authorized:"yes,no" directive:"allow container dir"`
	AllowContainerEncrypted    bool     `default:"yes" authorized:"yes,no" directive:"allow container encrypted"`
	AlwaysUseNv                bool     `default:"no" authorized:"yes,no" directive:"always use nv"`
	AlwaysUseRocm              bool     `default:"no" authorized:"yes,no" directive:"always use rocm"`
	SharedLoopDevices          bool     `default:"no" authorized:"yes,no" directive:"shared loop devices"`
	KeyserverVerifyDefaultOnly bool     `default:"yes" authorized:"yes,no" directive:"keyserver verify default only"`
}

const TemplateAsset = `# SINGULARITY.CONF
# This is the global configuration file for Singularity. This file controls
# what the container is allowed to do on a particular host, and as a result
# this file must be owned by root.

# ALLOW SETUID: [BOOL]
# DEFAULT: yes
# Should we allow users to utilize the setuid program flow within Singularity?
# note1: This is the default mode, and to utilize all features, this option
# must be enabled.  For example, without this option loop mounts of image 
# files will not work; only sandbox image directories, which do not need loop
# mounts, will work (subject to note 2).
# note2: If this option is disabled, it will rely on unprivileged user
# namespaces which have not been integrated equally between different Linux
# distributions.
allow setuid = {{ if eq .AllowSetuid true }}yes{{ else }}no{{ end }}

# MAX LOOP DEVICES: [INT]
# DEFAULT: 256
# Set the maximum number of loop devices that Singularity should ever attempt
# to utilize.
max loop devices = {{ .MaxLoopDevices }}

# ALLOW PID NS: [BOOL]
# DEFAULT: yes
# Should we allow users to request the PID namespace? Note that for some HPC
# resources, the PID namespace may confuse the resource manager and break how
# some MPI implementations utilize shared memory. (note, on some older
# systems, the PID namespace is always used)
allow pid ns = {{ if eq .AllowPidNs true }}yes{{ else }}no{{ end }}

# CONFIG PASSWD: [BOOL]
# DEFAULT: yes
# If /etc/passwd exists within the container, this will automatically append
# an entry for the calling user.
config passwd = {{ if eq .ConfigPasswd true }}yes{{ else }}no{{ end }}

# CONFIG GROUP: [BOOL]
# DEFAULT: yes
# If /etc/group exists within the container, this will automatically append
# group entries for the calling user.
config group = {{ if eq .ConfigGroup true }}yes{{ else }}no{{ end }}

# CONFIG RESOLV_CONF: [BOOL]
# DEFAULT: yes
# If there is a bind point within the container, use the host's
# /etc/resolv.conf.
config resolv_conf = {{ if eq .ConfigResolvConf true }}yes{{ else }}no{{ end }}

# MOUNT PROC: [BOOL]
# DEFAULT: yes
# Should we automatically bind mount /proc within the container?
mount proc = {{ if eq .MountProc true }}yes{{ else }}no{{ end }}

# MOUNT SYS: [BOOL]
# DEFAULT: yes
# Should we automatically bind mount /sys within the container?
mount sys = {{ if eq .MountSys true }}yes{{ else }}no{{ end }}

# MOUNT DEV: [yes/no/minimal]
# DEFAULT: yes
# Should we automatically bind mount /dev within the container? If 'minimal'
# is chosen, then only 'null', 'zero', 'random', 'urandom', and 'shm' will
# be included (the same effect as the --contain options)
mount dev = {{ .MountDev }}

# MOUNT DEVPTS: [BOOL]
# DEFAULT: yes
# Should we mount a new instance of devpts if there is a 'minimal'
# /dev, or -C is passed?  Note, this requires that your kernel was
# configured with CONFIG_DEVPTS_MULTIPLE_INSTANCES=y, or that you're
# running kernel 4.7 or newer.
mount devpts = {{ if eq .MountDevPts true }}yes{{ else }}no{{ end }}

# MOUNT HOME: [BOOL]
# DEFAULT: yes
# Should we automatically determine the calling user's home directory and
# attempt to mount it's base path into the container? If the --contain option
# is used, the home directory will be created within the session directory or
# can be overridden with the SINGULARITY_HOME or SINGULARITY_WORKDIR
# environment variables (or their corresponding command line options).
mount home = {{ if eq .MountHome true }}yes{{ else }}no{{ end }}

# MOUNT TMP: [BOOL]
# DEFAULT: yes
# Should we automatically bind mount /tmp and /var/tmp into the container? If
# the --contain option is used, both tmp locations will be created in the
# session directory or can be specified via the  SINGULARITY_WORKDIR
# environment variable (or the --workingdir command line option).
mount tmp = {{ if eq .MountTmp true }}yes{{ else }}no{{ end }}

# MOUNT HOSTFS: [BOOL]
# DEFAULT: no
# Probe for all mounted file systems that are mounted on the host, and bind
# those into the container?
mount hostfs = {{ if eq .MountHostfs true }}yes{{ else }}no{{ end }}

# BIND PATH: [STRING]
# DEFAULT: Undefined
# Define a list of files/directories that should be made available from within
# the container. The file or directory must exist within the container on
# which to attach to. you can specify a different source and destination
# path (respectively) with a colon; otherwise source and dest are the same.
# NOTE: these are ignored if singularity is invoked with --contain except
# for /etc/hosts and /etc/localtime. When invoked with --contain and --net,
# /etc/hosts would contain a default generated content for localhost resolution.
#bind path = /etc/singularity/default-nsswitch.conf:/etc/nsswitch.conf
#bind path = /opt
#bind path = /scratch
{{ range $path := .BindPath }}
{{- if ne $path "" -}}
bind path = {{$path}}
{{ end -}}
{{ end }}
# USER BIND CONTROL: [BOOL]
# DEFAULT: yes
# Allow users to influence and/or define bind points at runtime? This will allow
# users to specify bind points, scratch and tmp locations. (note: User bind
# control is only allowed if the host also supports PR_SET_NO_NEW_PRIVS)
user bind control = {{ if eq .UserBindControl true }}yes{{ else }}no{{ end }}

# ENABLE FUSEMOUNT: [BOOL]
# DEFAULT: yes
# Allow users to mount fuse filesystems inside containers with the --fusemount
# command line option.
enable fusemount = {{ if eq .EnableFusemount true }}yes{{ else }}no{{ end }}

# ENABLE OVERLAY: [yes/no/try/driver]
# DEFAULT: try
# Enabling this option will make it possible to specify bind paths to locations
# that do not currently exist within the container.  If 'try' is chosen,
# overlayfs will be tried but if it is unavailable it will be silently ignored.
# If 'driver' is chosen, overlayfs is handled by the image driver.
enable overlay = {{ .EnableOverlay }}

# ENABLE UNDERLAY: [yes/no]
# DEFAULT: yes
# Enabling this option will make it possible to specify bind paths to locations
# that do not currently exist within the container even if overlay is not
# working.  If overlay is available, it will be tried first.
enable underlay = {{ if eq .EnableUnderlay true }}yes{{ else }}no{{ end }}

# MOUNT SLAVE: [BOOL]
# DEFAULT: yes
# Should we automatically propagate file-system changes from the host?
# This should be set to 'yes' when autofs mounts in the system should
# show up in the container.
mount slave = {{ if eq .MountSlave true }}yes{{ else }}no{{ end }}

# SESSIONDIR MAXSIZE: [STRING]
# DEFAULT: 16
# This specifies how large the default sessiondir should be (in MB) and it will
# only affect users who use the "--contain" options and don't also specify a
# location to do default read/writes to (e.g. "--workdir" or "--home").
sessiondir max size = {{ .SessiondirMaxSize }}

# LIMIT CONTAINER OWNERS: [STRING]
# DEFAULT: NULL
# Only allow containers to be used that are owned by a given user. If this
# configuration is undefined (commented or set to NULL), all containers are
# allowed to be used. This feature only applies when Singularity is running in
# SUID mode and the user is non-root.
#limit container owners = gmk, singularity, nobody
{{ range $index, $owner := .LimitContainerOwners }}
{{- if eq $index 0 }}limit container owners = {{ else }}, {{ end }}{{$owner}}
{{- end }}

# LIMIT CONTAINER GROUPS: [STRING]
# DEFAULT: NULL
# Only allow containers to be used that are owned by a given group. If this
# configuration is undefined (commented or set to NULL), all containers are
# allowed to be used. This feature only applies when Singularity is running in
# SUID mode and the user is non-root.
#limit container groups = group1, singularity, nobody
{{ range $index, $group := .LimitContainerGroups }}
{{- if eq $index 0 }}limit container groups = {{ else }}, {{ end }}{{$group}}
{{- end }}

# LIMIT CONTAINER PATHS: [STRING]
# DEFAULT: NULL
# Only allow containers to be used that are located within an allowed path
# prefix. If this configuration is undefined (commented or set to NULL),
# containers will be allowed to run from anywhere on the file system. This
# feature only applies when Singularity is running in SUID mode and the user is
# non-root.
#limit container paths = /scratch, /tmp, /global
{{ range $index, $path := .LimitContainerPaths }}
{{- if eq $index 0 }}limit container paths = {{ else }}, {{ end }}{{$path}}
{{- end }}

# ALLOW CONTAINER ${TYPE}: [BOOL]
# DEFAULT: yes
# This feature limits what kind of containers that Singularity will allow
# users to use (note this does not apply for root).
allow container squashfs = {{ if eq .AllowContainerSquashfs true }}yes{{ else }}no{{ end }}
allow container extfs = {{ if eq .AllowContainerExtfs true }}yes{{ else }}no{{ end }}
allow container dir = {{ if eq .AllowContainerDir true }}yes{{ else }}no{{ end }}
allow container encrypted = {{ if eq .AllowContainerEncrypted true }}yes{{ else }}no{{ end }}

# ALWAYS USE NV ${TYPE}: [BOOL]
# DEFAULT: no
# This feature allows an administrator to determine that every action command
# should be executed implicitly with the --nv option (useful for GPU only 
# environments). 
always use nv = {{ if eq .AlwaysUseNv true }}yes{{ else }}no{{ end }}

# ALWAYS USE ROCM ${TYPE}: [BOOL]
# DEFAULT: no
# This feature allows an administrator to determine that every action command
# should be executed implicitly with the --rocm option (useful for GPU only
# environments).
always use rocm = {{ if eq .AlwaysUseRocm true }}yes{{ else }}no{{ end }}

# ROOT DEFAULT CAPABILITIES: [full/file/no]
# DEFAULT: full
# Define default root capability set kept during runtime
# - full: keep all capabilities (same as --keep-privs)
# - file: keep capabilities configured in ${prefix}/etc/singularity/capabilities/user.root
# - no: no capabilities (same as --no-privs)
root default capabilities = {{ .RootDefaultCapabilities }}

# MEMORY FS TYPE: [tmpfs/ramfs]
# DEFAULT: tmpfs
# This feature allow to choose temporary filesystem type used by Singularity.
# Cray CLE 5 and 6 up to CLE 6.0.UP05 there is an issue (kernel panic) when Singularity
# use tmpfs, so on affected version it's recommended to set this value to ramfs to avoid
# kernel panic
memory fs type = {{ .MemoryFSType }}

# CNI CONFIGURATION PATH: [STRING]
# DEFAULT: Undefined
# Defines path from where CNI configuration files are stored
#cni configuration path =
{{ if ne .CniConfPath "" }}cni configuration path = {{ .CniConfPath }}{{ end }}
# CNI PLUGIN PATH: [STRING]
# DEFAULT: Undefined
# Defines path from where CNI executable plugins are stored
#cni plugin path =
{{ if ne .CniPluginPath "" }}cni plugin path = {{ .CniPluginPath }}{{ end }}
# MKSQUASHFS PATH: [STRING]
# DEFAULT: Undefined
# This allows the administrator to specify the location for mksquashfs if it is not
# installed in a standard system location
# mksquashfs path =
{{ if ne .MksquashfsPath "" }}mksquashfs path = {{ .MksquashfsPath }}{{ end }}

# MKSQUASHFS PROCS: [UINT]
# DEFAULT: 0 (All CPUs)
# This allows the administrator to specify the number of CPUs for mksquashfs 
# to use when building an image.  The fewer processors the longer it takes.
# To enable it to use all available CPU's set this to 0.
# mksquashfs procs = 0
mksquashfs procs = {{ .MksquashfsProcs }}

# MKSQUASHFS MEM: [STRING]
# DEFAULT: Unlimited
# This allows the administrator to set the maximum amount of memory for mkswapfs
# to use when building an image.  e.g. 1G for 1gb or 500M for 500mb. Restricting memory
# can have a major impact on the time it takes mksquashfs to create the image.
# NOTE: This fuctionality did not exist in squashfs-tools prior to version 4.3
# If using an earlier version you should not set this.
# mksquashfs mem = 1G
{{ if ne .MksquashfsMem "" }}mksquashfs mem = {{ .MksquashfsMem }}{{ end }}

# CRYPTSETUP PATH: [STRING]
# DEFAULT: Undefined
# This allows the administrator to specify the location of cryptsetup if
# they wish to use custom location for this installation. If this value
# is undefined, at runtime singularity falls back to the value that was
# recorded at build time.
# cryptsetup path =
{{ if ne .CryptsetupPath "" }}cryptsetup path = {{ .CryptsetupPath }}{{ end }}
# SHARED LOOP DEVICES: [BOOL]
# DEFAULT: no
# Allow to share same images associated with loop devices to minimize loop
# usage and optimize kernel cache (useful for MPI)
shared loop devices = {{ if eq .SharedLoopDevices true }}yes{{ else }}no{{ end }}

# IMAGE DRIVER: [STRING]
# DEFAULT: Undefined
# This option specifies the name of an image driver provided by a plugin that
# will be used to handle image mounts. If the 'enable overlay' option is set
# to 'driver' the driver name specified here will also be used to handle
# overlay mounts.
# If the driver name specified has not been registered via a plugin installation
# the run-time will abort.
image driver = {{ .ImageDriver }}

# DEFAULT KEYSERVER: [STRING]
# DEFAULT: Undefined
# This allows the administrator to set the keyserver used by default to pull, push
# and search public keys, it's also used as the default keyserver to query during
# signature verification. When this value is empty the default keyserver used is
# https://keys.sylabs.io.
# This option may be useful for environments with their own public key server.
# NOTE: this value is ignored when user specify --url with the relevant commands.
# default keyserver = https://keys.example.com
{{ if ne .DefaultKeyserver "" }}default keyserver = {{ .DefaultKeyserver }}{{ end }}

# KEYSERVER VERIFY DEFAULT ONLY: [BOOL]
# DEFAULT: yes
# This allows the administrator to control during the image verification whether the
# 'default keyserver' is the only keyserver to used or not. When set to 'no' the verify
# command will also reach the keyserver associated with the active remote endpoint of users
# as the secondary keyserver to use if no key is found for the 'default keyserver'.
# NOTE: this directive has no effect if the 'default keyserver' is empty or if --url
# is specified by user.
keyserver verify default only = {{ if eq .KeyserverVerifyDefaultOnly true }}yes{{ else }}no{{ end }}
`
