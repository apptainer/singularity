# Singularity User Guide
This document will cover the usage of Singularity, working with containers, and all of the user facing features. There is a separate "Singularity Administration Guide" which targets system administrators, so if you are a service provider, or an interested user, it is encouraged that you read that document as well.


## Welcome to Singularity!
Singularity is a container solution created by necessity.

Over the past decade and a half, virtualization has gone from an engineering tool to an infrastructure necessity and the evolution of enabling technologies has flourished. Most recently, we have seen the introduction of the latest ... "containers". People's general conception of containers carry the heredity of it's lineage. The use cases of virtual machines have not only guided the understanding of what a container is, but it has also influenced many of the features that people expect from any virtualization solution. This is both a good and a bad thing...

For the industry at the forefront of the virtualization front this is a good thing. The enterprise and web enabled cloud requirements are very much in alignment with the feature set of virtual machines, and thus the predeceasing container technologies, but this does not bode as well for the scientific world and specifically the high performance computation (HPC) use case. While there are many overlapping features of these two fields, they differ in ways that make a shared implementation generally incompatible. While some have been able to leverage custom built resources that can operate on a lower performance scale, a proper integration is difficult and perhaps impossible with today's technology.

Scientists are a resourceful bunch and many of the features which exist both purposefully and incidentally via commonly used container technologies are not only desired, they are required for scientific use cases. This is the necessity which drove the creation of Singularity and articulated it's 4 primary functions:

1. **Mobility Of Compute**

	Mobility of compute is defined as the ability to define, create and maintain a workflow and be confident that the workflow can be executed on different hosts, operating systems (as long as it is Linux) and service providers. Being able to contain the entire software stack, from data files to library stack, and portably move it from system to system is true mobility.

	Singularity achieves this by utilizing a distributable image format that contains the entire container and stack into a single file. This file can be copied, shared, archived, and thus standard UNIX file permissions also apply. Additionally containers are portable (even across different C library versions and implementations) which makes sharing and copying an image as easy as `cp` or `scp` or `ftp`.

2. **Reproducibility**

	As mentioned above, Singularity containers utilize a single file which is the complete representation of all the files within the container. The same features which facilitate mobility also facilitate reproducibility. Once a contained workflow has been defined, the container image can be snapshotted, archived, and locked down such that it can be used later and you can be confident that the code within the container has not changed. The container is not subject to any external influence from the host operating system (aside from the kernel).

3. **User Freedom**

	System integrators, administrators, and engineers spend a lot of effort maintaining the operating systems on the resources they are reasonable for, and as a result tend to take a cautious approach on their systems. As a result, you may see hosts installed with a production, mission critical operating system that is "old" and may not have a lot of packages available for it. Or you may see software or libraries that are too old or incompatible with the software you need to run, or maybe just haven't installed the software stack you need due to complexities with building, specific software knowledge, incompatibilities or conflicts with other installed programs.

	Singularity can give the user the freedom they need to install the applications, versions, and dependencies for their workflows without impacting the system in any way. Users can define their own working environment and literally copy that environment image (single file) to a shared resource, and run their workflow inside that image.

4. **Support On Existing Traditional HPC**

	There are a lot of container systems presently available which are designed either for the enterprise, a replacement for virtual machines, cloud focused, or requires kernel features which are either not stable yet, or not available on your distribution of choice (or both).

	Replicating a virtual machine cloud like environment within an existing HPC resource is not a reasonable task, but this is the direction one would need to take to integrate OpenStack or Docker into traditional HPC. The use cases do not overlap nicely, nor can the solutions be forcibly wed.

	The goal of Singularity is to support existing and traditional HPC resources as easily as installing a single package onto the host operating system. Some configuration maybe required via a single configuration file, but the defaults are tuned to be generally applicable for shared environments.

	Singularity can run on host Linux distributions from RHEL6 (RHEL5 for versions lower then 2.2) and similar vintages, and the contained images have been tested as far back as Linux 2.2 (approximately 14 years old). Singularity natively supports IniniBand, Lustre, and works seamlessly with all resource managers (e.g. SLURM, Torque, SGE, etc.) because it works like running any other command on the system.


## A High Level View of Singularity

### Security and privilege escalation
*A user inside a Singularity container is the same user as outside the container*

This is one of Singularities defining characteristics. It allows a user (that may already have shell access to a particular host) to simply run a command inside of a container image as themselves. Here is a scenario to help articulate this:

> %SERVER is a shared multi-tenant resource to a number of users and as a result it is a large expensive resource far exceeding the resources of my personal workstation. But because it is a shared system, no users have root access and it is a controlled environment managed by a staff of system administrators. To keep the system secure, only the system administrators are granted root access and they control the state of the operating system. If a user is able to escalate to root (even within a container) on %SERVER, they can do bad things to the network, cause denial of service to the host (as well as other hosts on the same network), and may have unrestricted access to file systems reachable by the container.

To mitigate security concerns like this, Singularity limits one's ability to escalate permission inside a container. For example, if I do not have root access on the target system, I should not be able to escalate my privileges within the container to root either. This is semi-antagonistic to Singularity's 3rd tenant; allowing the users to have freedom of their own environments. Because if a user has the freedom to create and manipulate their own container environment, surely they know how to escalate their privileges to root within that container. Possible means could be setting the root user's password, or enabling themselves to have sudo access. For these reasons, Singularity prevents user context escalation within the container, and thus makes it possible to run user supplied containers on shared infrastructures.

But this mitigation dictates the Singularity workflow. If user's need to be root in order to make changes to their containers, then they need to have an endpoint (a local workstation, laptop, or server) where they have root access. Considering almost everybody at least has a laptop, this is not an unreasonable or unmanageable mitigation, but it must be defined and articulated.


### The Singularity container image
Singularity makes use of a container image file, which physically contains the container. This file is a physical representation of the container environment itself. If you obtain an interactive shell within a Singularity container, you are literally running within that file.

This simplifies management of files to the element of least surprise, basic file permission. If you either own a container image, or have read access to that container image, you can start a shell inside that image. If you wish to disable or limit access to a shared image, you simply change the permission ACLs to that file.

There are numerous benefits for using a single file image for the entire container and is summarized here:

- Copying or branching an entire container is as simple as `cp`
- Permission/access to the container is managed via standard file system permissions
- Large scale performance (especially over parallel file systems) is very efficient
- No caching of the image contents to run (especially nice on clusters)
- Container is a sparse file so it only consumes the disk space actually used
- Changes are implemented in real time (image grows and shrinks as needed)
- Images can serve as stand-alone programs, and can be executed like any other program on the host

#### Other container formats supported
In addition to the default Singularity container image, the following other formats are supported:

- **directory**: Standard Unix directories containing a root container image
- **tar.gz**: Zlib compressed tar archives
- **tar.bz2**: Bzip2 compressed tar archives
- **tar**: Uncompressed tar archives
- **cpio.gz**: Zlib compressed CPIO archives
- **cpio**: Uncompressed CPIO archives

*note: the suffix for the formats (except directory) are necessary as that is how Singularity identifies the image type.*

#### Supported URIs
Singularity also supports several different mechanisms for obtaining the images using a standard URI format

- **http://** Singularity will use Curl to download the image locally, and then run from the local image
- **https://** Same as above using encryption
- **docker://** Singularity can pull Docker images from a Docker registry, and will run them non-persistently (e.g. changes are not persisted as they can not be saved upstream)


### Name-spaces and isolation
When asked, "What namespaces does Singularity virtualize?", the most appropriate response from a Singularity use case is "As few as possible!". This is because the goals of Singularity are mobility, reproducibility and freedom, not full isolation (as you would expect from industry driven container technologies). Singularity only separates the needed namespaces in order to satisfy our primary goals.

So considering that, and considering that the user inside the container is the same user outside the container, allows us to blur the lines between what is contained and what is on the host. When this is done properly, using Singularity feels more like running in a parallel universe, where there are two timelines. One timeline, is the one we are familiar with, where the system administrators installed their operating system of choice. But on this alternate timeline, we bribed the system administrators and they installed our favorite operating system, and gave us full control but configured the rest of the system identically. And Singularity gives us the power to pick between these two timelines.

Or in summary, Singularity allows us to virtually swap out the underlying operating system for one that we defined without affecting anything else on the system and still having all of the host resources available to us.

It can also be described as ssh'ing into another identical host running a different operating system. One moment you are on Centos-6 and the next minute you are on the latest version of Ubuntu that has Tensorflow installed, or Debian with the latest OpenFoam, or a custom workflow that you installed.

Additionally what name-spaces are selected for virtualization can be dynamic or conditional. For example, the PID namespace is not separated from the host by default, but if you want to separate it, you can with a command line (or environment variable) setting. You can also decide you want to contain a process so it can not reach out to the host file system if you don't know if you trust the image. But by default, you are allowed to interface with all of the resources, devices and network inside the container as you are outside the container.


### Compatibility with standard work-flows, pipes and IO
Singularity does its best to abstract the complications of running an application in a different environment then what is expected on the host. For example, applications or scripts within a Singularity container can easily be part of a pipeline that is being executed on the host. Singularity containers can also be executed from a batch script or other program (e.g. an HPC system's resource manager) naively.

Some usage examples of Singularity can be seen as follows:

```bash
$ singularity exec /tmp/Demo.img xterm
$ singularity exec /tmp/Demo.img python script.py
$ singularity exec /tmp/Demo.img python < /path/to/python/script.py
$ cat /path/to/python/script.py | singularity exec /tmp/Demo.img python
```

You can even run MPI executables within the container as simply as:

```bash
$ mpirun -np X singularity exec /path/to/container.img /usr/bin/mpi_program_inside_container (mpi program args)
```

### The Singularity Process Flow
When executing container commands, the Singularity process flow can be generalized as follows:

1. Singularity application is invoked
2. Global options are parsed and activated
3. The Singularity command (subcommand) process is activated
4. Subcommand options are parsed
5. The appropriate sanity checks are made
6. Environment variables are set
7. The Singularity Execution binary is called (`sexec`)
8. Sexec determines if it is running privileged and calls the `SUID` code if necessary
9. Namespaces are created depending on configuration and process requirements
10. The Singularity image is checked, parsed, and mounted in the `CLONE_NEWNS` namespace
11. Bind mount points are setup
12. The namespace `CLONE_FS` is used to virtualize a new root file system
13. Singularity calls `execvp()` and Singularity process itself is replaced by the process inside the container
14. When the process inside the container exists, all namespaces collapse with that process, leaving a clean system

All of the above steps take approximately 15-25 thousandths of a second to run, which is fast enough to seem instantaneous. 


## The Singularity Usage Workflow
The security model of Singularity (as described above, "*A user inside a Singularity container is the same user as outside the container*") defines the Singularity workflow. There are generally two classifications of actions you would implement on a container; modification (which encompasses creation, bootstrapping, installing, admin) and using the container.

Modification of containers (new or existing) generally require root administrative privileges just like these actions would require on any system, container, or virtual machine. This means that a user must have a system that they have root access on. This could be a server, workstation, or even a laptop. If you are using OS X or Windows on your laptop, it is recommended to setup Vagrant, and run Singularity from there (there are recipes for this which can be found at http://singularity.lbl.gov/). Once you have Singularity installed on your endpoint of choice, this is where you will do the bulk of your container development.

This workflow can be described visually as follows:

![Singularity Workflow](workflow-overview.png)

One the left side, you have your laptop, workstation, or a server that you control. Here you will create your containers, modify and update your containers as you need. Once you have the container with the necessary applications, libraries and data inside it can be easily shared to other hosts and executed without have root access. But if you need to make changes again to your container, you must go back to an endpoint or system that you have root on, make the necessary changes, and then re-upload the container to the computation resource you wish to execute it on.

## Quick Start Installation
If you already have Singularity installed, or if you are using Singularity from your distribution provider and the version they have included version 2.2 or newer, you may skip this section. Otherwise, it is recommended that you install or upgrade the version of Singularity you have on your system. The following commands will get you going, and install Singularity to `/usr/local`. If you have an earlier version of Singularity installed, you should first remove it before continuing with the following installation commands.

```bash
$ mkdir ~/git
$ cd ~/git
$ git clone https://github.com/gmkurtzer/singularity.git
$ cd singularity
$ ./autogen.sh
$ ./configure --prefix=/usr/local --sysconfdir=/etc
$ make
$ sudo make install
```

You should note that the installation prefix is `/usr/local` but the configuration directory is `/etc`. This is done such that the configuration file is in the traditionally found location. If you omit that configure parameter, the configuration file will be found within `/usr/local/etc`.


## Overview of the Singularity Interface
Singularity is a command line driven interface that is designed to interact with containers and applications inside the container in as a transparent manner as possible. This means you can not only run programs inside a container as if they were on your host directly, but also redirect IO, pipes, arguments, files, shell redirects and sockets directly to the applications inside the container. 

Once you have Singularity installed, you should inspect the output of the `--help` option as follows:

```
$ singularity --help
USAGE: singularity [global options...] <command> [command options...] ...

GLOBAL OPTIONS:
    -d --debug    Print debugging information
    -h --help     Display usage summary
    -q --quiet    Only print errors
       --version  Show application version
    -v --verbose  Increase verbosity +1
    -x --sh-debug Print shell wrapper debugging information

GENERAL COMMANDS:
    help          Show additional help for a command

CONTAINER USAGE COMMANDS:
    exec          Execute a command within container
    run           Launch a runscript within container
    shell         Run a Bourne shell within container
    test          Execute any test code defined within container

CONTAINER MANAGEMENT COMMANDS (requires root):
    bootstrap     Bootstrap a new Singularity image from scratch
    copy          Copy files from your host into the container
    create        Create a new container image
    expand        Grow the container image
    export        Export the contents of a container via a tar pipe
    import        Import/add container contents via a tar pipe
    mount         Mount a Singularity container image

For any additional help or support visit the Singularity
website: http://singularity.lbl.gov/
```

Specifically notice the first line marked "USAGE". Here you will see the basic Singularity command usage, and notice the placement of the options. Option placement is very important in Singularity to ensure that the right options are being parsed at the right time. As you will see later in the guide, if you were to run a command inside the container called `foo -v`, then Singularity must be aware that the option `-v` that you are passing to the command `foo` is not intended to be parsed or interfered with by Singularity. So the placement of the options is very critical. In this example, you may pass the `-v` option twice, once in the Singularity global options and once for the command that you are executing inside the container. The final command may look like:

```bash
$ singularity -v exec container.img foo -v
```

The take home message here is that option placement is exceedingly important. The algorithm that Singularity uses for option parsing for both global options as well as subcommand options is as follows:

1. Read in the current option name
2. If the option is recognized do what is needed, move to next option (goto #1)
3. If the paramater is prefixed with a `-` (hyphen) but is not recognized, error out
4. If the next option is not prefixed with a `-` (hyphen), then assume we are done with option parsing

This means that options will continue to be parsed until no more options are listed.

*note: Options that require data (e.g. `--bind <path>`) must be separated by white space, not an equals sign!*

As the above "USAGE" describes, Singularity will parse the command as follows:

1. Singularity command (`singularity`)
2. Global options
3. Singularity subcommand (`shell` or `exec`)
4. Subcommand options
5. Any additional input is passed to the subcommand

You can get additional help on any of the Singularity subcommands by using any one of the following command syntaxes:

```bash
$ singularity help <subcommand>
$ singularity --help <subcommand>
$ singularity -h <subcommand>
$ singularity <subcommand> --help
$ singularity <subcommand -h
```

## Invoking a Non-Persistent Container
At this point, you can easily test Singularity by downloading and running a non-persistent container. As mentioned earlier, Singularity has the ability to interface with the main Docker Registry, so let's start off by pulling a container down from the main Docker Registry and launching a shell inside of a given container:

```bash
$ cat /etc/redhat-release 
CentOS Linux release 7.2.1511 (Core) 
$ singularity shell docker://ubuntu:latest
library/ubuntu:latest
Downloading layer: sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4
Downloading layer: sha256:9f03ce1741bf604c84258a4c4f1dc98cc35aebdd76c14ed4ffeb6bc3584c1f9b
Downloading layer: sha256:61e032b8f2cb04e7a2d4efa83eb6837c6b92bd1553cbe46cffa76121091d8301
Downloading layer: sha256:50de990d7957c304603ac78d094f3acf634c1261a3a5a89229fa81d18cdb7945
Downloading layer: sha256:3a80a22fea63572c387efb1943e6095587f9ea8343af129934d4c81e593374a4
Downloading layer: sha256:cad964aed91d2ace084302c587dfc502b5869c5b1d15a1f0e458a45e3cadfaa6
Singularity: Invoking an interactive shell within container...

Singularity.ubuntu:latest> cat /etc/lsb-release
DISTRIB_ID=Ubuntu
DISTRIB_RELEASE=16.04
DISTRIB_CODENAME=xenial
DISTRIB_DESCRIPTION="Ubuntu 16.04.1 LTS"
Singularity.ubuntu:latest> which apt-get
/usr/bin/apt-get
Singularity.ubuntu:latest> exit
[gmk@centos7-x64 ~]$
```

In this example, you can see we started off on a Centos-7.2 host operating system, ran Singularity as a non-root user and used a URI which tells Singularity to pull a given container from the main Docker Registry and execute a shell within it. In this example, we are not telling Singularity to use a local image, which means that any changes we make will be non-persistent (e.g. the container is removed automatically as soon as the shell is exited).

You may select other images that are currently hosted on the main Docker Hub Library.

You now have a properly functioning Singularity installation on your system. 

## Creating a New Singularity Image
The primary use cases of Singularity revolve around the idea of mobility, portability, reproducibility, and archival of containers. These features are realized via Singularity via the Singularity image file. As explained earlier, Singularity images are single files which can be copied, shared, and easily archived along with relevant data. This means that the all of the computational components can be easily replicated, utilized and extended on by other researchers.

The first part of building your reproducible container is to first create the raw Singularity image file:

```bash
$ sudo singularity create /tmp/container.img
Creating a new image with a maximum size of 768MiB...
Executing image create helper
Formatting image with ext3 file system
Done.
```

Think of this as an empty bucket of a given size, and you can fill that bucket up to the specified size. By default the size in Singularity v2.2 is 768MiB (but this has changed from 512 - 1024 in different versions). You can override the default size by specifying the `--size` option in MiB as follows:

```bash
$ sudo singularity create --size 2048 /tmp/container.img
Creating a new image with a maximum size of 2048MiB...
Executing image create helper
Formatting image with ext3 file system
Done.
```

Notice that the permissions of the generated file. While the `umask` is adhered to, you should find that the file is executable. While at this point there is nothing to execute within that image, once this image has within it a proper container file system, you can define what this image will do when it is executed directly. 


## The Bootstrap Definition
The process of *bootstrapping* a Singularity container is equivalent to describing a recipe for the container creation. There are several recipe formats that Singularity supports, but only the primary format of version 2.2 will be documented here.

There are multiple sections of the Singularity bootstrap definition file:

1. **The header**: The Header describes the core operating system to bootstrap within the container. Here you will configure the base operating system features that you need within your container. Examples of this include, what distribution of Linux, what version, what packages must be part of a core install.
2. **The scriptlets**: The reset of the definition is comprised of mini-scripts (or scriptlets). These scripts are executed within the core operating system of the container (or externally) and are responsible for building the container.

### The header fields:

#### Bootstrap

#### OSVersion

#### MirrorURL

#### Conditional fields


### The scriptlets:

#### %inside


#### %outside


#### %runscript


#### %test



## Bootstrapping an Image


## Making Changes to an Existing Container

include increasing size of container image

## Using Your Container Image
When using your containers, Shell, exec, run, test...

### Executing a container directly

### Options and runtime features

### Alternative image formats

#### Directories
#### Archives
#### Dockers

## Creating a Container Workflow


## Best Practices for Bootstrapping
install things into OS locations (e.g. not /home, or /tmp)
don't count on non OS standard paths