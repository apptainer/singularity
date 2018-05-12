/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

/*
TODO Provide some guidelines for writing these docs
*/

package docs

// Global content for help and man pages
const (

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
// main singularity command 
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
    SingularityUse string = `singularity [global options...]`

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
    BuildUse string = `build [local options...] <IMAGE PATH> <BUILD SPEC>`

    BuildShort string = `Build a new Singularity container`

    BuildLong string = `

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
// capability
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
    CapabilityUse string = `capability <subcommand>`

    CapabilityShort string = `Manage Linux capabilities on containers`

    CapabilityLong string = `
  Capabilities allow you to have fine grained control over the permissions that
  your containers need to run. For instance, if you need to `

    CapabilityExample string = `
  All group commands have their own help output:
  
  $ singularity help capability add
  $ singularity capability list --help`

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
// capability add
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
    CapabilityAddUse string = `add [add options...] <capabilities>`

    CapabilityAddShort string = `add Linux capabilities to a container`

    CapabilityAddLong string = `
  The capability add command allows you to grant fine grained Linux 
  capabilities to your container at runtime. For instance, `

    CapabilityAddExample string = `
  $ singularity capability.add /tmp/my-sql.img mysql
  
  $ singularity shell capability://mysql
  Singularity my-sql.img> pwd
  /home/mibauer/mysql
  Singularity my-sql.img> ps
  PID TTY          TIME CMD
    1 pts/0    00:00:00 sinit
    2 pts/0    00:00:00 bash
    3 pts/0    00:00:00 ps
  Singularity my-sql.img>
  
  $ singularity capability.stop /tmp/my-sql.img mysql
  Stopping /tmp/my-sql.img mysql`

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
// capability drop
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
    CapabilityDropUse string = `drop [drop options...] <capabilities>`

    CapabilityDropShort string = `remove Linux capabilities from your container`

    CapabilityDropLong string = `
  The capability drop command allows you to remove Linux capabilities from your
  container with fine grained precision. This way you can ensure that your
  container is as secure as it can be given the functions it must carry out. For
  instance, `

    CapabilityDropExample string = `
  $ singularity capability.drop /tmp/my-sql.img mysql
  
  $ singularity shell capability://mysql
  Singularity my-sql.img> pwd
  /home/mibauer/mysql
  Singularity my-sql.img> ps
  PID TTY          TIME CMD
  1 pts/0    00:00:00 sinit
  2 pts/0    00:00:00 bash
  3 pts/0    00:00:00 ps
  Singularity my-sql.img>
  
  $ singularity capability.stop /tmp/my-sql.img mysql
  Stopping /tmp/my-sql.img mysql`

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
// capability drop
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
    CapabilityListUse string = `list [list options...] <capabilities>`

    CapabilityListShort string = `list Linux capabilities on a container`

    CapabilityListLong string = `The capability list command allows you to see
  what Linux capabilities are associated with your container.`

    CapabilityListExample string = `
  $ singularity capability.list /tmp/my-sql.img mysql
  
  $ singularity shell capability://mysql
  Singularity my-sql.img> pwd
  /home/mibauer/mysql
  Singularity my-sql.img> ps
  PID TTY          TIME CMD
    1 pts/0    00:00:00 sinit
    2 pts/0    00:00:00 bash
    3 pts/0    00:00:00 ps
  Singularity my-sql.img>
  
  $ singularity capability.stop /tmp/my-sql.img mysql
  Stopping /tmp/my-sql.img mysql`

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

    ExecUse string = `exec [exec options...] <container> ...`

    ExecShort string = `Execute a command within container`

    ExecLong string = `
  singularity exec supports the following formats:` + formats

    ExecExamples string = `
  $ singularity exec /tmp/Debian.img cat /etc/debian_version
  $ singularity exec /tmp/Debian.img python ./hello_world.py
  $ cat hello_world.py | singularity exec /tmp/Debian.img python
  $ sudo singularity exec --writable /tmp/Debian.img apt-get update
  $ singularity exec instance://my_instance ps -ef`

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
// shell
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
    ShellUse string = `shell [shell options...] <container>`

    ShellShort string = `Run a Bourne shell within container`

    ShellLong string = `
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
// run
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
    RunUse string = `run [run options...] <container>`

    RunShort string = `Launch a runscript within container`

    RunLong string = `
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

)

