// Copyright (c) 2017-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// TODO Provide some guidelines for writing these docs

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
  Singularity one is capable of building a root file system that runs on any 
  other Linux system where Singularity is installed.`
	SingularityExample string = `
  $ singularity help <command> [<subcommand>]
  $ singularity help build
  $ singularity help instance start`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// build
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	BuildUse   string = `build [local options...] <IMAGE PATH> <BUILD SPEC>`
	BuildShort string = `Build a Singularity image`
	BuildLong  string = `

  IMAGE PATH:

  When Singularity builds the container, output can be one of a few formats:

      default:    The compressed Singularity read only image format (default)
      sandbox:    This is a read-write container within a directory structure

  note: It is a common workflow to use the "sandbox" mode for development of the
  container, and then build it as a default Singularity image for production 
  use. The default format is immutable.

  BUILD SPEC:

  The build spec target is a definition (def) file, local image, or URI that can 
  be used to create a Singularity container. Several different local target 
  formats exist:

      def file  : This is a recipe for building a container (examples below)
      directory:  A directory structure containing a (ch)root file system
      image:      A local image on your machine (will convert to sif if
                  it is legacy format)

  Targets can also be remote and defined by a URI of the following formats:

      library://  an image library (default https://cloud.sylabs.io/library)
      docker://   a Docker/OCI registry (default Docker Hub)
      shub://     a Singularity registry (default Singularity Hub)
      oras://     an OCI registry that holds SIF files using ORAS`

	BuildExample string = `

  DEF FILE BASE OS:

      Library:
          Bootstrap: library
          From: debian:9

      Docker:
          Bootstrap: docker
          From: tensorflow/tensorflow:latest
          IncludeCmd: yes # Use the CMD as runscript instead of ENTRYPOINT

      Singularity Hub:
          Bootstrap: shub
          From: singularityhub/centos

      YUM/RHEL:
          Bootstrap: yum
          OSVersion: 7
          MirrorURL: http://mirror.centos.org/centos-%{OSVERSION}/%{OSVERSION}/os/x86_64/
          Include: yum

      Debian/Ubuntu:
          Bootstrap: debootstrap
          OSVersion: trusty
          MirrorURL: http://us.archive.ubuntu.com/ubuntu/

      Local Image:
          Bootstrap: localimage
          From: /home/dave/starter.img

      Scratch:
          Bootstrap: scratch # Populate the container with a minimal rootfs in %setup

  DEFFILE SECTIONS:

      %pre
          echo "This is a scriptlet that will be executed on the host, as root before"
          echo "the container has been bootstrapped. This section is not commonly used."

      %setup
          echo "This is a scriptlet that will be executed on the host, as root, after"
          echo "the container has been bootstrapped. To install things into the container"
          echo "reference the file system location with $SINGULARITY_ROOTFS."

      %post
          echo "This scriptlet section will be executed from within the container after"
          echo "the bootstrap/base has been created and setup."

      %test
          echo "Define any test commands that should be executed after container has been"
          echo "built. This scriptlet will be executed from within the running container"
          echo "as the root user. Pay attention to the exit/return value of this scriptlet"
          echo "as any non-zero exit code will be assumed as failure."
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

      %help
          This is a text file to be displayed with the run-help command.

  COMMANDS:

      Build a sif file from a Singularity recipe file:
          $ singularity build /tmp/debian0.sif /path/to/debian.def

      Build a sif image from the Library:
          $ singularity build /tmp/debian1.sif library://debian:latest

      Build a base sandbox from DockerHub, make changes to it, then build sif
          $ singularity build --sandbox /tmp/debian docker://debian:latest
          $ singularity exec --writable /tmp/debian apt-get install python
          $ singularity build /tmp/debian2.sif /tmp/debian`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// Cache
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	CacheUse   string = `cache`
	CacheShort string = `Manage the local cache`
	CacheLong  string = `
  Manage your local Singularity cache. You can list/clean using the specific 
  types.`
	CacheExample string = `
  All group commands have their own help output:

  $ singularity cache
  $ singularity cache --help`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// Cache clean
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	CacheCleanUse   string = `clean [clean options...]`
	CacheCleanShort string = `Clean your local Singularity cache`
	CacheCleanLong  string = `
  This will clean your local cache (stored at $HOME/.singularity/cache if
  SINGULARITY_CACHEDIR is not set). By default the entire cache is cleaned, use
  --days and --type flags to override this behavior. Note: if you use Singularity
  as root, cache will be stored in '/root/.singularity/.cache', to clean that
  cache, you will need to run 'cache clean' as root, or with 'sudo'.`
	CacheCleanExample string = `
  All group commands have their own help output:

  $ singularity help cache clean --days 30
  $ singularity help cache clean --type=library,oci
  $ singularity cache clean --help`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// Cache List
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	CacheListUse   string = `list [list options...]`
	CacheListShort string = `List your local Singularity cache`
	CacheListLong  string = `
  This will list your local cache (stored at $HOME/.singularity/cache if
  SINGULARITY_CACHEDIR is not set).`
	CacheListExample string = `
  All group commands have their own help output:

  $ singularity help cache list
  $ singularity help cache list --type=library,oci
  $ singularity cache list --help`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// key
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	KeyUse   string = `key [key options...]`
	KeyShort string = `Manage OpenPGP keys`
	KeyLong  string = `
  Manage your trusted, public and private keys in your local or in the global keyring
  (local keyring: '~/.singularity/sypgp' if 'SINGULARITY_SYPGPDIR' is not set,
  global keyring: '%[1]s/singularity/global-pgp-public')`
	KeyExample string = `
  All group commands have their own help output:

  $ singularity help key newpair
  $ singularity key list --help`

	// keys is for the hidden 'keys' command
	KeysUse string = `keys [keys options...]`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// key import
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	KeyImportUse   string = `import [import options...] <input-key>`
	KeyImportShort string = `Import a local key into the local or global keyring`
	KeyImportLong  string = `
  The 'key import' command allows you to add a key to your local or global keyring
  from a specific file.`
	KeyImportExample string = `
  $ singularity key import ./my-key.asc

  # Import into global keyring (root user only)
  $ singularity key import --global ./my-key.asc`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// key export
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	KeyExportUse   string = `export [export options...] <output-file>`
	KeyExportShort string = `Export a public or private key into a specific file`
	KeyExportLong  string = `
  The 'key export' command allows you to export a key and save it to a file.`
	KeyExportExample string = `
  Exporting a private key:
  
  $ singularity key export --secret ./private.asc

  Exporting a public key:
  
  $ singularity key export ./public.asc`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// key newpair
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	KeyNewPairUse   string = `newpair`
	KeyNewPairShort string = `Create a new key pair`
	KeyNewPairLong  string = `
  The 'key newpair' command allows you to create a new key or public/private
  keys to be stored in the default user local keyring location (e.g., 
  $HOME/.singularity/sypgp).`
	KeyNewPairExample string = `
  $ singularity key newpair
  $ singularity key newpair --password=psk --name=your-name --comment="key comment" --email=mail@email.com --push=false`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// key list
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	KeyListUse   string = `list`
	KeyListShort string = `List keys in your local or in the global keyring`
	KeyListLong  string = `
  List your local keys in your keyring. Will list public (trusted) keys
  by default.`
	KeyListExample string = `
  $ singularity key list
  $ singularity key list --secret

  # list global public keys
  $ singularity key list --global`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// key search
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	KeySearchUse   string = `search [search options...] <search_string>`
	KeySearchShort string = `Search for keys on a key server`
	KeySearchLong  string = `
  The 'key search' command allows you to connect to a key server and look for
  public keys matching the argument passed to the command line. You can  
  search by name, email, or fingerprint / key ID. (Maximum 100 search entities)`
	KeySearchExample string = `
  $ singularity key search sylabs.io

  # search by fingerprint:
  $ singularity key search 8883491F4268F173C6E5DC49EDECE4F3F38D871E

  # search by key ID:
  $ singularity key search F38D871E`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// key pull
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	KeyPullUse   string = `pull [pull options...] <fingerprint>`
	KeyPullShort string = `Download a public key from a key server`
	KeyPullLong  string = `
  The 'key pull' command allows you to retrieve public key material from a
  remote key server, and add it to your keyring. Note that Singularity consults
  your keyring when running commands such as 'singularity verify', and thus
  adding a key to your keyring implies a level of trust. Because of this, it is
  recommended that you verify the fingerprint of the key with its owner prior
  to running this command.`
	KeyPullExample string = `
  $ singularity key pull 8883491F4268F173C6E5DC49EDECE4F3F38D871E`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// key push
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	KeyPushUse   string = `push [push options...] <fingerprint>`
	KeyPushShort string = `Upload a public key to a key server`
	KeyPushLong  string = `
  The 'key push' command allows you to connect to a key server and upload public
  keys from the local or the global keyring.`
	KeyPushExample string = `
  $ singularity key push 8883491F4268F173C6E5DC49EDECE4F3F38D871E`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// key remove
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	KeyRemoveUse   string = `remove <fingerprint>`
	KeyRemoveShort string = `Remove a local public key from your local or the global keyring`
	KeyRemoveLong  string = `
  The 'key remove' command will remove a local public key from
  the local or the global keyring.`
	KeyRemoveExample string = `
  $ singularity key remove D87FE3AF5C1F063FCBCC9B02F812842B5EEE5934`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// delete
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	DeleteUse   string = `delete [arch] <imageRef>`
	DeleteShort string = `Deletes requested image from the library`
	DeleteLong  string = `
  The 'delete' command allows you to delete an image from a remote library.`
	DeleteExample string = `
  $ singularity delete --arch=amd64 library://username/project/image:1.0`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// capability
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	CapabilityUse   string = `capability`
	CapabilityShort string = `Manage Linux capabilities for users and groups`
	CapabilityLong  string = `
  Capabilities allow you to have fine grained control over the permissions that
  your containers need to run.

  NOTE: capability add/drop commands require root to run. Granting capabilities 
  to users allows them to escalate privilege inside the container and will
  likely give them a route to privilege escalation on the host system as well.
  Do not add capabilities to users who should not have root on the host system.`
	CapabilityExample string = `
  All group commands have their own help output:

  $ singularity help capability add
  $ singularity capability add --help`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// capability add
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	CapabilityAddUse   string = `add [add options...] <capabilities>`
	CapabilityAddShort string = `Add capabilities to a user or group (requires root)`
	CapabilityAddLong  string = `
  Add Linux capabilities to a user or group. NOTE: This command requires root to 
  run.

  The capabilities argument must be separated by commas and is not case 
  sensitive.

  To see available capabilities, type "singularity capability avail" or refer to
  capabilities manual "man 7 capabilities".`
	CapabilityAddExample string = `
  $ sudo singularity capability add --user nobody AUDIT_READ,chown
  $ sudo singularity capability add --group nobody cap_audit_write

  To add all capabilities to a user:

  $ sudo singularity capability add --user nobody all`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// capability drop
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	CapabilityDropUse   string = `drop [drop options...] <capabilities>`
	CapabilityDropShort string = `Remove capabilities from a user or group (requires root)`
	CapabilityDropLong  string = `
  Remove Linux capabilities from a user/group. NOTE: This command requires root 
  to run.

  The capabilities argument must be separated by commas and is not case 
  sensitive.

  To see available capabilities, type "singularity capability avail" or refer to
  capabilities manual "man 7 capabilities"`
	CapabilityDropExample string = `
  $ sudo singularity capability drop --user nobody AUDIT_READ,CHOWN
  $ sudo singularity capability drop --group nobody audit_write

  To drop all capabilities for a user:

  $ sudo singularity capability drop --user nobody all`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// capability list
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	CapabilityListUse   string = `list [user/group]`
	CapabilityListShort string = `Show capabilities for a given user or group`
	CapabilityListLong  string = `
  Show the capabilities for a user or group.`
	CapabilityListExample string = `
  To list capabilities set for user or group nobody:

  $ singularity capability list nobody

  To list capabilities for all users/groups:

  $ singularity capability list`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// capability avail
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	CapabilityAvailUse   string = `avail [capabilities]`
	CapabilityAvailShort string = `Show description for available capabilities`
	CapabilityAvailLong  string = `
  Show description for available Linux capabilities.`
	CapabilityAvailExample string = `
  Show description for all available capabilities:

  $ singularity capability avail

  Show CAP_CHOWN description:

  $ singularity capability avail CAP_CHOWN

  Show CAP_CHOWN/CAP_NET_RAW description:

  $ singularity capability avail CAP_CHOWN,CAP_NET_RAW`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// exec
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	formats string = `

  *.sif               Singularity Image Format (SIF). Native to Singularity 3.0+
  
  *.sqsh              SquashFS format.  Native to Singularity 2.4+

  *.img               ext3 format. Native to Singularity versions < 2.4.

  directory/          sandbox format. Directory containing a valid root file 
                      system and optionally Singularity meta-data.

  instance://*        A local running instance of a container. (See the instance
                      command group.)

  library://*         A SIF container hosted on a Library
                      (default https://cloud.sylabs.io/library)

  docker://*          A Docker/OCI container hosted on Docker Hub or another
                      OCI registry.

  shub://*            A container hosted on Singularity Hub.

  oras://*            A SIF container hosted on an OCI registry that supports
                      the OCI Registry As Storage (ORAS) specification.`
	ExecUse   string = `exec [exec options...] <container> <command>`
	ExecShort string = `Run a command within a container`
	ExecLong  string = `
  singularity exec supports the following formats:` + formats
	ExecExamples string = `
  $ singularity exec /tmp/debian.sif cat /etc/debian_version
  $ singularity exec /tmp/debian.sif python ./hello_world.py
  $ cat hello_world.py | singularity exec /tmp/debian.sif python
  $ sudo singularity exec --writable /tmp/debian.sif apt-get update
  $ singularity exec instance://my_instance ps -ef
  $ singularity exec library://centos cat /etc/os-release`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// instance
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	InstanceUse   string = `instance`
	InstanceShort string = `Manage containers running as services`
	InstanceLong  string = `
  Instances allow you to run containers as background processes. This can be
  useful for running services such as web servers or databases.`
	InstanceExample string = `
  All group commands have their own help output:

  $ singularity help instance start
  $ singularity instance start --help`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// instance list
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	InstanceListUse   string = `list [list options...] [<instance name glob>]`
	InstanceListShort string = `List all running and named Singularity instances`
	InstanceListLong  string = `
  The instance list command allows you to view the Singularity container
  instances that are currently running in the background.`
	InstanceListExample string = `
  $ singularity instance list
  INSTANCE NAME      PID       IMAGE
  test               11963     /home/mibauer/singularity/sinstance/test.sif
  test2              11964     /home/mibauer/singularity/sinstance/test.sif
  lolcow             11965     /home/mibauer/singularity/sinstance/lolcow.sif

  $ singularity instance list 'test*'
  INSTANCE NAME      PID       IMAGE
  test               11963     /home/mibauer/singularity/sinstance/test.sif
  test2              11964     /home/mibauer/singularity/sinstance/test.sif

  $ sudo singularity instance list -u mibauer
  INSTANCE NAME      PID       IMAGE
  test               11963     /home/mibauer/singularity/sinstance/test.sif
  test2              16219     /home/mibauer/singularity/sinstance/test.sif`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// instance start
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	InstanceStartUse   string = `start [start options...] <container path> <instance name> [startscript args...]`
	InstanceStartShort string = `Start a named instance of the given container image`
	InstanceStartLong  string = `
  The instance start command allows you to create a new named instance from an
  existing container image that will begin running in the background. If a
  startscript is defined in the container metadata the commands in that script
  will be executed with the instance start command as well. You can optionally
  pass arguments to startscript

  singularity instance start accepts the following container formats` + formats
	InstanceStartExample string = `
  $ singularity instance start /tmp/my-sql.sif mysql

  $ singularity shell instance://mysql
  Singularity my-sql.sif> pwd
  /home/mibauer/mysql
  Singularity my-sql.sif> ps
  PID TTY          TIME CMD
    1 pts/0    00:00:00 sinit
    2 pts/0    00:00:00 bash
    3 pts/0    00:00:00 ps
  Singularity my-sql.sif>

  $ singularity instance stop /tmp/my-sql.sif mysql
  Stopping /tmp/my-sql.sif mysql`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// instance stop
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	InstanceStopUse   string = `stop [stop options...] [instance]`
	InstanceStopShort string = `Stop a named instance of a given container image`
	InstanceStopLong  string = `
  The command singularity instance stop allows you to stop and clean up a named,
  running instance of a given container image.`
	InstanceStopExample string = `
  $ singularity instance start my-sql.sif mysql1
  $ singularity instance start my-sql.sif mysql2
  $ singularity instance stop mysql*
  Stopping mysql1 instance of my-sql.sif (PID=23845)
  Stopping mysql2 instance of my-sql.sif (PID=23858)

  $ singularity instance start my-sql.sif mysql1

  Force instance to shutdown
  $ singularity instance stop -f mysql1 (may corrupt data)

  Send SIGTERM to the instance
  $ singularity instance stop -s SIGTERM mysql1
  $ singularity instance stop -s TERM mysql1
  $ singularity instance stop -s 15 mysql1`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// pull
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	PullUse   string = `pull [pull options...] [output file] <URI>`
	PullShort string = `Pull an image from a URI`
	PullLong  string = `
  The 'pull' command allows you to download or build a container from a given
  URI. Supported URIs include:

  library: Pull an image from the currently configured library
      library://user/collection/container[:tag]

  docker: Pull a Docker/OCI image from Docker Hub, or another OCI registry.
      docker://user/image:tag
    
  shub: Pull an image from Singularity Hub
      shub://user/image:tag

  oras: Pull a SIF image from an OCI registry that supports ORAS.
      oras://registry/namespace/image:tag

  http, https: Pull an image using the http(s?) protocol
      https://library.sylabs.io/v1/imagefile/library/default/alpine:latest`
	PullExample string = `
  From Sylabs cloud library
  $ singularity pull alpine.sif library://alpine:latest

  From Docker
  $ singularity pull tensorflow.sif docker://tensorflow/tensorflow:latest

  From Shub
  $ singularity pull singularity-images.sif shub://vsoch/singularity-images

  From supporting OCI registry (e.g. Azure Container Registry)
  $ singularity pull image.sif oras://<username>.azurecr.io/namespace/image:tag`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// push
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	PushUse   string = `push [push options...] <image> <URI>`
	PushShort string = `Upload image to the provided URI`
	PushLong  string = `
  The 'push' command allows you to upload a SIF container to a given
  URI.  Supported URIs include:

  library:
      library://user/collection/container[:tag]

  oras:
      oras://registry/namespace/repo:tag


  NOTE: It's always good practice to sign your containers before
  pushing them to the library. An auth token is required to push to the library,
  so you may need to configure it first with 'singularity remote'.`
	PushExample string = `
  To Library
  $ singularity push /home/user/my.sif library://user/collection/my.sif:latest

  To supported OCI registry
  $ singularity push /home/user/my.sif oras://registry/namespace/image:tag`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// search
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	SearchUse   string = `search [search options...] <search_query>`
	SearchShort string = `Search a Container Library for images`
	SearchLong  string = `
  Search a Container Library for container images matching the search query.
  (default cloud.sylabs.io). You can specify an alternate architecture, and/or limit
  the results to only signed images.`
	SearchExample string = `
  $ singularity search lolcow
  $ singularity search --arch arm64 alpine
  $ singularity search --signed tensorflow`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// run
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	RunUse   string = `run [run options...] <container>`
	RunShort string = `Run the user-defined default command within a container`
	RunLong  string = `
  This command will launch a Singularity container and execute a runscript
  if one is defined for that container. The runscript is a metadata file within
  the container that contains shell commands. If the file is present (and
  executable) then this command will execute that file within the container
  automatically. All arguments following the container name will be passed
  directly to the runscript.

  singularity run accepts the following container formats:` + formats
	RunExamples string = `
  # Here we see that the runscript prints "Hello world: "
  $ singularity exec /tmp/debian.sif cat /singularity
  #!/bin/sh
  echo "Hello world: "

  # It runs with our inputs when we run the image
  $ singularity run /tmp/debian.sif one two three
  Hello world: one two three

  # Note that this does the same thing
  $ ./tmp/debian.sif one two three`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// shell
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	ShellUse   string = `shell [shell options...] <container>`
	ShellShort string = `Run a shell within a container`
	ShellLong  string = `
  singularity shell supports the following formats:` + formats
	ShellExamples string = `
  $ singularity shell /tmp/Debian.sif
  Singularity/Debian.sif> pwd
  /home/gmk/test
  Singularity/Debian.sif> exit

  $ singularity shell -C /tmp/Debian.sif
  Singularity/Debian.sif> pwd
  /home/gmk
  Singularity/Debian.sif> ls -l
  total 0
  Singularity/Debian.sif> exit

  $ sudo singularity shell -w /tmp/Debian.sif
  $ sudo singularity shell --writable /tmp/Debian.sif

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
	SignUse   string = `sign [sign options...] <image path>`
	SignShort string = `Attach digital signature(s) to an image`
	SignLong  string = `
  The sign command allows a user to add one or more digital signatures to a SIF
  image. By default, one digital signature is added for each object group in
  the file.
  
  To generate a key pair, see 'singularity help key newpair'`
	SignExample string = `
  $ singularity sign container.sif`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// verify
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	VerifyUse   string = `verify [verify options...] <image path>`
	VerifyShort string = `Verify cryptographic signatures attached to an image`
	VerifyLong  string = `
  The verify command allows a user to verify cryptographic signatures on SIF 
  container files. There may be multiple signatures for data objects and 
  multiple data objects signed. By default the command searches for the primary 
  partition signature. If found, a list of all verification blocks applied on 
  the primary partition is gathered so that data integrity (hashing) and 
  signature verification is done for all those blocks.`
	VerifyExample string = `
  $ singularity verify container.sif`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// Run-help
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	RunHelpUse   string = `run-help <image path>`
	RunHelpShort string = `Show the user-defined help for an image`
	RunHelpLong  string = `
  The help text is from the '%help' section of the definition file. If you are 
  using the '--apps' option, the help text is instead from that app's '%apphelp' 
  section.`
	RunHelpExample string = `
  $ cat my_container.def
  Bootstrap: docker
  From: busybox

  %help
      Some help for this container

  %apphelp foo
      Some help for application 'foo' in this container

  $ sudo singularity build my_container.sif my_container.def
  Using container recipe deffile: my_container.def
  [...snip...]
  Cleaning up...

  $ singularity run-help my_container.sif

    Some help for this container

  $ singularity run-help --app foo my_container.sif

    Some help for application in this container`
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// Inspect
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	InspectUse   string = `inspect [inspect options...] <image path>`
	InspectShort string = `Show metadata for an image`
	InspectLong  string = `
  Inspect will show you labels, environment variables, apps and scripts associated 
  with the image determined by the flags you pass. By default, they will be shown in 
  plain text. If you would like to list them in json format, you should use the --json flag.
  `
	InspectExample string = `
  $ singularity inspect ubuntu.sif
  
  If you want to list the applications (apps) installed in a container (located at
  /scif/apps) you should run inspect command with --list-apps <container-image> flag.
  ( See https://sci-f.github.io for more information on SCIF apps)

  The following environment variables are available to you when called 
  from the shell inside the container. The top variables are relevant 
  to the active app (--app <app>) and the bottom available for all 
  apps regardless of the active app. Both sets of variables are also available during development (at build time).

  ACTIVE APP ENVIRONMENT:
      SCIF_APPNAME       the name for the active application
      SCIF_APPROOT       the installation folder for the application created at /scif/apps/<app>
      SCIF_APPMETA       the application metadata folder
      SCIF_APPDATA       the data folder created for the application at /scif/data/<app>
        SCIF_APPINPUT    expected input folder within data base folder
        SCIF_APPOUTPUT   the output data folder within data base folder

      SCIF_APPENV        points to the application's custom environment.sh file in its metadata folder
      SCIF_APPLABELS     is the application's labels.json in the metadata folder
      SCIF_APPBIN        is the bin folder for the app, which is automatically added to the $PATH when the app is active
      SCIF_APPLIB        is the application's library folder that is added to the LD_LIBRARY_PATH
      SCIF_APPRUN        is the runscript
      SCIF_APPHELP       is the help file for the runscript
      SCIF_APPTEST       is the testing script (test.sh) associated with the application
      SCIF_APPNAME       the name for the active application
      SCIF_APPFILES      the files section associated with the application that are added to


  GLOBAL APP ENVIRONMENT:
    
      SCIF_DATA             scif defined data base for all apps (/scif/data)
      SCIF_APPS             scif defined install bases for all apps (/scif/apps)
      SCIF_APPROOT_<app>    root for application <app>
      SCIF_APPDATA_<app>    data root for application <app>

  To list all your apps:

  $ singularity inspect --list-apps ubuntu.sif 

  To list only labels in the json format from an image:

  $ singularity inspect --json --labels ubuntu.sif

  To verify you own a single application on your container image, use the --app <appname> flag:

  $ singularity inspect --app <appname> ubuntu.sif`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// Test
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	RunTestUse   string = `test [exec options...] <image path>`
	RunTestShort string = `Run the user-defined tests within a container`
	RunTestLong  string = `
  The 'test' command allows you to execute a testscript (if available) inside of
  a given container 

  NOTE:
      For instances if there is a daemon process running inside the container,
      then subsequent container commands will all run within the same 
      namespaces. This means that the --writable and --contain options will not 
      be honored as the namespaces have already been configured by the 
      'singularity start' command.
`
	RunTestExample string = `
  Set the '%test' section with a definition file like so:
  %test
      echo "hello from test" "$@"

  $ singularity test /tmp/debian.sif command
      hello from test command

  For additional help, please visit our public documentation pages which are
  found at:

      https://www.sylabs.io/docs/`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// OCI
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	OciUse   string = `oci`
	OciShort string = `Manage OCI containers`
	OciLong  string = `
  Allow you to manage containers from OCI bundle directories.

  NOTE: all oci commands requires to run as root`
	OciExample string = `
  All group commands have their own help output:

  $ singularity oci create -b ~/bundle mycontainer
  $ singularity oci start mycontainer`

	OciCreateUse   string = `create -b <bundle_path> [create options...] <container_ID>`
	OciCreateShort string = `Create a container from a bundle directory (root user only)`
	OciCreateLong  string = `
  Create invoke create operation to create a container instance from an OCI 
  bundle directory`
	OciCreateExample string = `
  $ singularity oci create -b ~/bundle mycontainer`

	OciStartUse   string = `start <container_ID>`
	OciStartShort string = `Start container process (root user only)`
	OciStartLong  string = `
  Start invoke start operation to start a previously created container 
  identified by container ID.`
	OciStartExample string = `
  $ singularity oci start mycontainer`

	OciStateUse   string = `state <container_ID>`
	OciStateShort string = `Query state of a container (root user only)`
	OciStateLong  string = `
  State invoke state operation to query state of a created/running/stopped 
  container identified by container ID.`
	OciStateExample string = `
  $ singularity oci state mycontainer`

	OciKillUse   string = `kill [kill options...] <container_ID>`
	OciKillShort string = `Kill a container (root user only)`
	OciKillLong  string = `
  Kill invoke kill operation to kill processes running within container 
  identified by container ID.`
	OciKillExample string = `
  $ singularity oci kill mycontainer INT
  $ singularity oci kill mycontainer -s INT`

	OciDeleteUse   string = `delete <container_ID>`
	OciDeleteShort string = `Delete container (root user only)`
	OciDeleteLong  string = `
  Delete invoke delete operation to delete resources that were created for 
  container identified by container ID.`
	OciDeleteExample string = `
  $ singularity oci delete mycontainer`

	OciAttachUse   string = `attach <container_ID>`
	OciAttachShort string = `Attach console to a running container process (root user only)`
	OciAttachLong  string = `
  Attach will attach console to a running container process running within 
  container identified by container ID.`
	OciAttachExample string = `
  $ singularity oci attach mycontainer`

	OciExecUse   string = `exec <container_ID> <command> <args>`
	OciExecShort string = `Execute a command within container (root user only)`
	OciExecLong  string = `
  Exec will execute the provided command/arguments within container identified 
  by container ID.`
	OciExecExample string = `
  $ singularity oci exec mycontainer id`

	OciRunUse   string = `run -b <bundle_path> [run options...] <container_ID>`
	OciRunShort string = `Create/start/attach/delete a container from a bundle directory (root user only)`
	OciRunLong  string = `
  Run will invoke equivalent of create/start/attach/delete commands in a row.`
	OciRunExample string = `
  $ singularity oci run -b ~/bundle mycontainer

  is equivalent to :

  $ singularity oci create -b ~/bundle mycontainer
  $ singularity oci start mycontainer
  $ singularity oci attach mycontainer
  $ singularity oci delete mycontainer`

	OciUpdateUse   string = `update [update options...] <container_ID>`
	OciUpdateShort string = `Update container cgroups resources (root user only)`
	OciUpdateLong  string = `
  Update will update cgroups resources for the specified container ID. Container 
  must be in a RUNNING or CREATED state.`
	OciUpdateExample string = `
  $ singularity oci update --from-file /tmp/cgroups-update.json mycontainer

  or to update from stdin :

  $ cat /tmp/cgroups-update.json | singularity oci update --from-file - mycontainer`

	OciPauseUse   string = `pause <container_ID>`
	OciPauseShort string = `Suspends all processes inside the container (root user only)`
	OciPauseLong  string = `
  Pause will suspend all processes for the specified container ID.`
	OciPauseExample string = `
  $ singularity oci pause mycontainer`

	OciResumeUse   string = `resume <container_ID>`
	OciResumeShort string = `Resumes all processes previously paused inside the container (root user only)`
	OciResumeLong  string = `
  Resume will resume all processes previously paused for the specified container 
  ID.`
	OciResumeExample string = `
  $ singularity oci resume mycontainer`

	OciMountUse   string = `mount <sif_image> <bundle_path>`
	OciMountShort string = `Mount create an OCI bundle from SIF image (root user only)`
	OciMountLong  string = `
  Mount will mount and create an OCI bundle from a SIF image.`
	OciMountExample string = `
  $ singularity oci mount /tmp/example.sif /var/lib/singularity/bundles/example`

	OciUmountUse   string = `umount <bundle_path>`
	OciUmountShort string = `Umount delete bundle (root user only)`
	OciUmountLong  string = `
  Umount will umount an OCI bundle previously mounted with singularity oci 
  mount.`
	OciUmountExample string = `
  $ singularity oci umount /var/lib/singularity/bundles/example`

	ConfigUse   string = `config`
	ConfigShort string = `Manage various singularity configuration (root user only)`
	ConfigLong  string = `
  The config command allows root user to manage various configuration like fakeroot
  user mapping entries.`
	ConfigExample string = `
  All config commands have their own help output:

  $ singularity help config fakeroot
  $ singularity config fakeroot --help`

	ConfigFakerootUse   string = `fakeroot <option> <user>`
	ConfigFakerootShort string = `Manage fakeroot user mappings entries (root user only)`
	ConfigFakerootLong  string = `
  The config fakeroot command allow a root user to add/remove/enable/disable fakeroot
  user mappings.`
	ConfigFakerootExample string = `
  To add a fakeroot user mapping for vagrant user:
  $ singularity config fakeroot --add vagrant

  To remove a fakeroot user mapping for vagrant user:
  $ singularity config fakeroot --remove vagrant

  To disable a fakeroot user mapping for vagrant user:
  $ singularity config fakeroot --disable vagrant

  To enable a fakeroot user mapping for vagrant user:
  $ singularity config fakeroot --enable vagrant`

	ConfigGlobalUse   string = `global <option> <directive> [value,...]`
	ConfigGlobalShort string = `Edit singularity.conf from command line (root user only or unprivileged installation)`
	ConfigGlobalLong  string = `
  The config global command allow administrators to set/unset/get/reset configuration
  directives of singularity.conf from command line.`
	ConfigGlobalExample string = `
  To add a path to "bind path" directive:
  $ singularity config global --set "bind path" /etc/resolv.conf

  To remove a path from "bind path" directive:
  $ singularity config global --unset "bind path" /etc/resolv.conf

  To set "bind path" to the default value:
  $ singularity config global --reset "bind path"

  To get "bind path" directive value:
  $ singularity config global --get "bind path"

  To display the resulting configuration instead of writing it to file:
  $ singularity config global --dry-run --set "bind path" /etc/resolv.conf`

	OverlayUse   string = `overlay`
	OverlayShort string = `Manage an EXT3 writable overlay image`
	OverlayLong  string = `
  The overlay command allows management of EXT3 writable overlay images.`
	OverlayExample string = `
  All overlay commands have their own help output:

  $ singularity help overlay create
  $ singularity overlay create --help`

	OverlayCreateUse   string = `create <options> image`
	OverlayCreateShort string = `Create EXT3 writable overlay image`
	OverlayCreateLong  string = `
  The overlay create command allows to create EXT3 writable overlay image either
  as a single EXT3 image or by adding it automatically to an existing SIF image.`
	OverlayCreateExample string = `
  To create and add a writable overlay to an existing SIF image:
  $ singularity overlay create --size 1024 /tmp/image.sif

  To create a single EXT3 writable overlay image:
  $ singularity overlay create --size 1024 /tmp/my_overlay.img`
)

// Documentation for sif/siftool command.
const (
	SIFUse   string = `sif`
	SIFAlias string = `siftool`
	SIFShort string = `Manipulate Singularity Image Format (SIF) images`
	SIFLong  string = `
  A set of commands are provided to display elements such as the SIF global
  header, the data object descriptors and to dump data objects. It is also
  possible to modify a SIF file via this tool via the add/del commands.`
	SIFExample string = `
  All sif commands have their own help output:

  $ singularity help sif list
  $ singularity sif list --help`
)
