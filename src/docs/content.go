// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

//TODO Provide some guidelines for writing these docs

package docs

// Global content for help and man pages
const (

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// main singularity command
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	SingularityUse   string = `singularity [global options...]`
	SingularityShort string = `
Linux container platform optimized for High Performance Computing (HPC) and
Enterprise Performance Computing (EPC)`
	SingularityLong string = `
  Singularity containers provide an application virtualization layer enabling
  mobility of compute via both application and environment portability. With
  Singularity one is capable of building a root file system and running that
  root file system on any other Linux system where Singularity is installed.`
	SingularityExample string = `
  $ singularity help
      Will print a generalized usage summary and available commands.

  $ singularity help <command>
      Additional help for any Singularity subcommand can be seen by appending
      the subcommand name to the above command.`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// build
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	BuildUse   string = `build [local options...] <IMAGE PATH> <BUILD SPEC>`
	BuildShort string = `Build a new Singularity container`
	BuildLong  string = `

  IMAGE PATH:
  
  When Singularity builds the container, output can be one of a few formats:
  
      default:    The compressed Singularity read only image format (default)
      sandbox:    This is a read-write container within a directory structure
      writable:   Legacy writable image format
  
  note: It is a  common workflow to use the "sandbox" mode for development of 
  the  container, and then build it as a default Singularity image for 
  production use. The default format is immutable.
  
  BUILD SPEC:
  
  The build spec target is a definition, local image, archive, or URI that can 
  be used to create a Singularity container. Several different local target 
  formats exist:
  
      def file  : This is a recipe for building a container (examples below)
      directory:  A directory structure containing a (ch)root file system
      image:      A local image on your machine (will convert to squashfs if
                  it is legacy or writable format)
      tar/tar.gz: An archive file which contains the above directory format
                  (must have .tar in the filename!)
  
  Targets can also be remote and defined by a URI of the following formats:
  
      shub://     Build from a Singularity registry (Singularity Hub default)
      docker://   This points to a Docker registry (Docker Hub default)`
	BuildExample string = `

  DEF FILE BASE OS:
  
      Singularity Hub:
          Bootstrap: shub
          From: singularityhub/centos
  
      Docker:
          Bootstrap: docker
          From: tensorflow/tensorflow:latest
          IncludeCmd: yes # Use the CMD as runscript instead of ENTRYPOINT
  
      YUM/RHEL:
          Bootstrap: yum
          OSVersion: 7
          MirrorURL: http://mirror.centos.org/centos-%{OSVERSION}/%{OSVERSION}/os/$basearch/
          Include: yum
  
      Debian/Ubuntu:
          Bootstrap: debootstrap
          OSVersion: trusty
          MirrorURL: http://us.archive.ubuntu.com/ubuntu/
  
      Local Image:
          Bootstrap: localimage
          From: /home/dave/starter.img
  
  DEFFILE SECTIONS:
  
      %setup
          echo "This is a scriptlet that will be executed on the host, as root, after"
          echo "the container has been bootstrapped. To install things into the container"
          echo "reference the file system location with $SINGULARITY_BUILDROOT"
  
      %post
          echo "This scriptlet section will be executed from within the container after"
          echo "the bootstrap/base has been created and setup"
  
      %test
          echo "Define any test commands that should be executed after container has been"
          echo "built. This scriptlet will be executed from within the running container"
          echo "as the root user. Pay attention to the exit/return value of this scriptlet"
          echo "as any non-zero exit code will be assumed as failure"
          exit 0
  
      %runscript
          echo "Define actions for the container to be executed with the run command or"
          echo "when container is executed."
  
      %startscript
          echo "Define actions for container to perform when started as an instance."
  
      %labels
          HELLO MOTO
          KEY VALUE
  
      %files
          /path/on/host/file.txt /path/on/container/file.txt
          relative_file.txt /path/on/container/relative_file.txt
  
      %environment
          LUKE=goodguy
          VADER=badguy
          HAN=someguy
          export HAN VADER LUKE
  
  COMMANDS:
  
      Build a compressed image from a Singularity recipe file:
          $ singularity build /tmp/debian0.simg /path/to/debian.def
  
      Build a base compressed image from Docker Hub:
          $ singularity build /tmp/debian1.simg docker://debian:latest
  
      Build a base sandbox from DockerHub, make changes to it, then build image
          $ singularity build --sandbox /tmp/debian docker://debian:latest
          $ singularity exec --writable /tmp/debian apt-get install python
          $ singularity build /tmp/debian2.simg /tmp/debian`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// keys
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	KeysUse   string = `keys <subcommand>`
	KeysShort string = `Manage OpenPGP key stores`
	KeysLong  string = `
  The 'keys' command  allows you to manage local OpenPGP key stores by create a
  new store and new keys pairs. You can also list available keys from the
  default store. Finally, the keys command offers subcommands to communicate
  with an HKP key server to fetch and upload public keys.`
	KeysExample string = `
  All group commands have their own help output:

  $ singularity help keys newpair
  $ singularity keys list --help`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// keys newpair
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	KeysNewPairUse   string = `newpair`
	KeysNewPairShort string = `Create a new OpenPGP key pair`
	KeysNewPairLong  string = `
	The 'keys newpair' command allows you to create a new key or public/private
	keys to be stored in the default user local key store location
	(e.g., $HOME/.singularity/sypgp).`
	KeysNewPairExample string = `
  $ singularity keys newpair`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// keys list
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	KeysListUse   string = `list`
	KeysListShort string = `List keys from the default key store`
	KeysListLong  string = `
	The 'keys list' command allows you to list public/private key pairs
	from the default user local key store location
	(e.g., $HOME/.singularity/sypgp).`
	KeysListExample string = `
  $ singularity keys list`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// keys search
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	KeysSearchUse   string = `search`
	KeysSearchShort string = `Search for keys matching string argument`
	KeysSearchLong  string = `
	The 'keys search' command allows you to connect to a key server and look
	for public keys marching the string argument passed to the command line.`
	KeysSearchExample string = `
  $ singularity keys search sylabs.io`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// keys pull
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	KeysPullUse   string = `pull`
	KeysPullShort string = `Fetch an OpenPGP public key from a key server`
	KeysPullLong  string = `
	The 'keys pull' command allows you to connect to a key server look for
	and download a public key. Key rings are stored into
	(e.g., $HOME/.singularity/sypgp).`
	KeysPullExample string = `
  $ singularity keys pull D87FE3AF5C1F063FCBCC9B02F812842B5EEE5934`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// keys push
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	KeysPushUse   string = `push`
	KeysPushShort string = `Upload an OpenPGP public key to a key server`
	KeysPushLong  string = `
	The 'keys push' command allows you to connect to a key server and
	upload public keys from the local key store.`
	KeysPushExample string = `
  $ singularity keys push D87FE3AF5C1F063FCBCC9B02F812842B5EEE5934`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// capability
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	CapabilityUse   string = `capability <subcommand>`
	CapabilityShort string = `Manage Linux capabilities on containers`
	CapabilityLong  string = `
  Capabilities allow you to have fine grained control over the permissions that
  your containers need to run. For instance, if you need to `
	CapabilityExample string = `
  All group commands have their own help output:
  
  $ singularity help capability add
  $ singularity capability add --help`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// capability add
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	CapabilityAddUse   string = `add [add options...] <capabilities>`
	CapabilityAddShort string = `Add authorized capabilities for a given user/group`
	CapabilityAddLong  string = `
  Capabilities must be separated by commas and are not case sensitive,
  here accepted values:

  CAP_AUDIT_CONTROL     | AUDIT_CONTROL
  CAP_AUDIT_READ        | AUDIT_READ
  CAP_AUDIT_WRITE       | AUDIT_WRITE
  CAP_BLOCK_SUSPEND     | BLOCK_SUSPEND
  CAP_CHOWN             | CHOWN
  CAP_DAC_OVERRIDE      | DAC_OVERRIDE
  CAP_DAC_READ_SEARCH   | DAC_READ_SEARCH
  CAP_FOWNER            | FOWNER
  CAP_FSETID            | FSETID
  CAP_IPC_LOCK          | IPC_LOCK
  CAP_IPC_OWNER         | IPC_OWNER
  CAP_KILL              | KILL
  CAP_LEASE             | LEASE
  CAP_LINUX_IMMUTABLE   | LINUX_IMMUTABLE
  CAP_MAC_ADMIN         | MAC_ADMIN
  CAP_MAC_OVERRIDE      | MAC_OVERRIDE
  CAP_MKNOD             | MKNOD
  CAP_NET_ADMIN         | NET_ADMIN
  CAP_NET_BIND_SERVICE  | NET_BIND_SERVICE
  CAP_NET_BROADCAST     | NET_BROADCAST
  CAP_NET_RAW           | NET_RAW
  CAP_SETFCAP           | SETFCAP
  CAP_SETGID            | SETGID
  CAP_SETPCAP           | SETPCAP
  CAP_SETUID            | SETUID
  CAP_SYS_ADMIN         | SYS_ADMIN
  CAP_SYS_BOOT          | SYS_BOOT
  CAP_SYS_CHROOT        | SYS_CHROOT
  CAP_SYSLOG            | SYSLOG
  CAP_SYS_MODULE        | SYS_MODULE
  CAP_SYS_NICE          | SYS_NICE
  CAP_SYS_PACCT         | SYS_PACCT
  CAP_SYS_PTRACE        | SYS_PTRACE
  CAP_SYS_RAWIO         | SYS_RAWIO
  CAP_SYS_RESOURCE      | SYS_RESOURCE
  CAP_SYS_TIME          | SYS_TIME
  CAP_SYS_TTY_CONFIG    | SYS_TTY_CONFIG
  CAP_WAKE_ALARM        | WAKE_ALARM

  See "-d" flag example for description of each capabilities`
	CapabilityAddExample string = `
  $ singularity capability add --user nobody AUDIT_READ,chown
  $ singularity capability add --group nobody cap_audit_write

  To print capabilities description:

  $ singularity capability add -d CAP_CHOWN
  $ singularity capability add -d CAP_CHOWN,CAP_SYS_ADMIN`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// capability drop
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	CapabilityDropUse   string = `drop [drop options...] <capabilities>`
	CapabilityDropShort string = `Drop authorized capabilities for a given user/group`
	CapabilityDropLong  string = `
  Capabilities must be separated by commas and are not case sensitive,
  here accepted values:

  CAP_AUDIT_CONTROL     | AUDIT_CONTROL
  CAP_AUDIT_READ        | AUDIT_READ
  CAP_AUDIT_WRITE       | AUDIT_WRITE
  CAP_BLOCK_SUSPEND     | BLOCK_SUSPEND
  CAP_CHOWN             | CHOWN
  CAP_DAC_OVERRIDE      | DAC_OVERRIDE
  CAP_DAC_READ_SEARCH   | DAC_READ_SEARCH
  CAP_FOWNER            | FOWNER
  CAP_FSETID            | FSETID
  CAP_IPC_LOCK          | IPC_LOCK
  CAP_IPC_OWNER         | IPC_OWNER
  CAP_KILL              | KILL
  CAP_LEASE             | LEASE
  CAP_LINUX_IMMUTABLE   | LINUX_IMMUTABLE
  CAP_MAC_ADMIN         | MAC_ADMIN
  CAP_MAC_OVERRIDE      | MAC_OVERRIDE
  CAP_MKNOD             | MKNOD
  CAP_NET_ADMIN         | NET_ADMIN
  CAP_NET_BIND_SERVICE  | NET_BIND_SERVICE
  CAP_NET_BROADCAST     | NET_BROADCAST
  CAP_NET_RAW           | NET_RAW
  CAP_SETFCAP           | SETFCAP
  CAP_SETGID            | SETGID
  CAP_SETPCAP           | SETPCAP
  CAP_SETUID            | SETUID
  CAP_SYS_ADMIN         | SYS_ADMIN
  CAP_SYS_BOOT          | SYS_BOOT
  CAP_SYS_CHROOT        | SYS_CHROOT
  CAP_SYSLOG            | SYSLOG
  CAP_SYS_MODULE        | SYS_MODULE
  CAP_SYS_NICE          | SYS_NICE
  CAP_SYS_PACCT         | SYS_PACCT
  CAP_SYS_PTRACE        | SYS_PTRACE
  CAP_SYS_RAWIO         | SYS_RAWIO
  CAP_SYS_RESOURCE      | SYS_RESOURCE
  CAP_SYS_TIME          | SYS_TIME
  CAP_SYS_TTY_CONFIG    | SYS_TTY_CONFIG
  CAP_WAKE_ALARM        | WAKE_ALARM

  See "-d" flag example for description of each capabilities`
	CapabilityDropExample string = `
  $ singularity capability drop --user nobody AUDIT_READ,CHOWN
  $ singularity capability drop --group nobody audit_write

  To print capabilities description:

  $ singularity capability drop -d CAP_CHOWN
  $ singularity capability drop -d CAP_CHOWN,CAP_SYS_ADMIN`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// capability list
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	CapabilityListUse   string = `list [list options...] <capabilities>`
	CapabilityListShort string = `List authorized capabilities for the given user/group.`
	CapabilityListLong  string = `
  The capability list command allows you to see
  what Linux capabilities are associated with users/groups.`
	CapabilityListExample string = `
  $ singularity capability list --user nobody
  $ singularity capability list --group nobody
  $ singularity capability list --all`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// exec
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	formats string = `

  *.sqsh              SquashFS format.  Native to Singularity 2.4+
  
  *.img               This is the native Singularity image format for all
                      Singularity versions < 2.4.
  
  *.tar\*              Tar archives are exploded to a temporary directory and
                      run within that directory (and cleaned up after). The
                      contents of the archive is a root file system with root
                      being in the current directory. All compression
                      suffixes are supported.
  
  directory/          Container directories that contain a valid root file
                      system.
  
  instance://*        A local running instance of a container. (See the
                      instance command group.)
  
  shub://*            A container hosted on Singularity Hub
  
  docker://*          A container hosted on Docker Hub`
	ExecUse   string = `exec [exec options...] <container> ...`
	ExecShort string = `Execute a command within container`
	ExecLong  string = `
  singularity exec supports the following formats:` + formats
	ExecExamples string = `
  $ singularity exec /tmp/Debian.img cat /etc/debian_version
  $ singularity exec /tmp/Debian.img python ./hello_world.py
  $ cat hello_world.py | singularity exec /tmp/Debian.img python
  $ sudo singularity exec --writable /tmp/Debian.img apt-get update
  $ singularity exec instance://my_instance ps -ef`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// instance
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	InstanceUse   string = `instance <subcommand>`
	InstanceShort string = `Manage containers running in the background`
	InstanceLong  string = `
  Instances allow you to run containers as background processes. This can be
  useful for running services such as web servers or databases.`
	InstanceExample string = `
  All group commands have their own help output:
  
  $ singularity help instance.start
  $ singularity instance.start --help`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// instance list
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	InstanceListUse   string = `list [list options...] <container>`
	InstanceListShort string = `List all running and named Singularity instances`
	InstanceListLong  string = `
  The instance list command allows you to view the Singularity container
  instances that are currently running in the background.`
	InstanceListExample string = `
  $ singularity instance.list
  DAEMON NAME      PID      CONTAINER IMAGE
  test            11963     /home/mibauer/singularity/sinstance/test.img
  
  $ sudo singularity instance.list -u mibauer
  DAEMON NAME      PID      CONTAINER IMAGE
  test            11963     /home/mibauer/singularity/sinstance/test.img
  test2           16219     /home/mibauer/singularity/sinstance/test.img`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// instance start
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	InstanceStartUse   string = `start [start options...] <container path> <instance name>`
	InstanceStartShort string = `Start a named instance of the given container image`
	InstanceStartLong  string = `
  The instance start command allows you to create a new named instance from an
  existing container image that will begin running in the background. If a
  start.sh script is defined in the container metadata the commands in that
  script will be executed with the instance start command as well.
  
  singularity instance start accepts the following container formats` + formats
	InstanceStartExample string = `
  $ singularity instance.start /tmp/my-sql.img mysql
  
  $ singularity shell instance://mysql
  Singularity my-sql.img> pwd
  /home/mibauer/mysql
  Singularity my-sql.img> ps
  PID TTY          TIME CMD
    1 pts/0    00:00:00 sinit
    2 pts/0    00:00:00 bash
    3 pts/0    00:00:00 ps
  Singularity my-sql.img>
  
  $ singularity instance.stop /tmp/my-sql.img mysql
  Stopping /tmp/my-sql.img mysql`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// instance stop
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	InstanceStopUse   string = `stop [stop options...] [instance]`
	InstanceStopShort string = `Stop a named instance of a given container image`
	InstanceStopLong  string = `
  The command singularity instance stop allows you to stop and clean up a named,
  running instance of a given container image.`
	InstanceStopExample string = `
  $ singularity instance.start my-sql.img mysql1
  $ singularity instance.start my-sql.img mysql2
  $ singularity instance.stop mysql*
  Stopping mysql1 instance of my-sql.img (PID=23845)
  Stopping mysql2 instance of my-sql.img (PID=23858)
  
  $ singularity instance.start my-sql.img mysql1
  
  Force instance to shutdown
  $ singularity instance.stop -f mysql1 (may corrupt data)
  
  Send SIGTERM to the instance
  $ singularity instance.stop -s SIGTERM mysql1
  $ singularity instance.stop -s TERM mysql1
  $ singularity instance.stop -s 15 mysql1`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// pull
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	PullUse   string = `pull [pull options...] [library://[user[collection/[<container>:tag]]]]`
	PullShort string = `Pull a container from a URI`
	PullLong  string = `
  SUPPORTED URIs:
  
    library: Pull an image from the currently configured library
    shub: Pull an image using python from Singularity Hub to /home/vagrant/versioned/singularity
    docker: Pull a docker image using python to /home/vagrant/versioned/singularity`
	PullExample string = `
  $ singularity pull docker://ubuntu:latest
  
  $ singularity pull shub://vsoch/singularity-images
  Found image vsoch/singularity-images:mongo
  Downloading image... vsoch-singularity-images-mongo.img
  
  $ singularity pull --name "meatballs.img" shub://vsoch/singularity-images
  $ singularity pull --commit shub://vsoch/singularity-images
  $ singularity pull --hash shub://vsoch/singularity-images`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// push
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	PushUse   string = `push [push options...] <container image> [library://[user[collection/[container[:tag]]]]]`
	PushShort string = `Push a container to a Library URI`
	PushLong  string = `
  The Singularity push command allows you to upload your sif image to a library
  of your choosing`
	PushExample string = `
  $ singularity push /home/user/my.sif library://user/collection/my.sif:latest`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// run
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	RunUse   string = `run [run options...] <container>`
	RunShort string = `Launch a runscript within container`
	RunLong  string = `
  This command will launch a Singularity container and execute a runscript
  if one is defined for that container. The runscript is a metadata file within
  the container that containes shell commands. If the file is present (and
  executable) then this command will execute that file within the container
  automatically. All arguments following the container name will be passed
  directly to the runscript.
  
  singularity run accepts the following container formats:` + formats
	RunExamples string = `
  # Here we see that the runscript prints "Hello world: "
  $ singularity exec /tmp/Debian.img cat /singularity
  #!/bin/sh
  echo "Hello world: "
  
  # It runs with our inputs when we run the image
  $ singularity run /tmp/Debian.img one two three
  Hello world: one two three
  
  # Note that this does the same thing
  $ ./tmp/Debian.img one two three`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// shell
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	ShellUse   string = `shell [shell options...] <container>`
	ShellShort string = `Run a Bourne shell within container`
	ShellLong  string = `
  singularity shell supports the following formats:` + formats
	ShellExamples string = `
  $ singularity shell /tmp/Debian.img
  Singularity/Debian.img> pwd
  /home/gmk/test
  Singularity/Debian.img> exit
  
  $ singularity shell -C /tmp/Debian.img
  Singularity/Debian.img> pwd
  /home/gmk
  Singularity/Debian.img> ls -l
  total 0
  Singularity/Debian.img> exit
  
  $ sudo singularity shell -w /tmp/Debian.img
  $ sudo singularity shell --writable /tmp/Debian.img
  
  $ singularity shell instance://my_instance
  
  $ singularity shell instance://my_instance
  Singularity: Invoking an interactive shell within container...
  Singularity container:~> ps -ef
  UID        PID  PPID  C STIME TTY          TIME CMD
  ubuntu       1     0  0 20:00 ?        00:00:00 /usr/local/bin/singularity/bin/sinit
  ubuntu       2     0  0 20:01 pts/8    00:00:00 /bin/bash --norc
  ubuntu       3     2  0 20:02 pts/8    00:00:00 ps -ef`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// sign
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	SignUse   string = `sign <image path>`
	SignShort string = `Attach cryptographic signature to container`
	SignLong  string = `
  `
	SignExample string = `
  `

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// verify
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	VerifyUse   string = `verify <image path>`
	VerifyShort string = `Verify cryptographic signature on container`
	VerifyLong  string = `
  `
	VerifyExample string = `
  `
)
