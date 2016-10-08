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
When smugly asked, "What namespaces does Singularity virtualize?", the most enjoyed response is "As few as possible!". This is because the goals of Singularity are mobility, reproducibility and freedom, not full isolation (as you would expect from industry driven container technologies). Singularity only separates the needed namespaces in order to satisfy our primary goals.

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


## The Singularity Workflow
create, bootstrap, setup image on endpoint, copy to host to run

## Overview of the Singularity Interface

description of help output

description of argument processing logic

how to get more help on commands

## Creating and Bootstrapping a Container


## Bootstrap Definition File Format


## Making Changes to an Existing Container

include increasing size of container image

## Using Your Container Image
Shell, exec, run, test...

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