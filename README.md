[![Build Status](https://travis-ci.org/singularityware/singularity.svg?branch=master)](https://travis-ci.org/singularityware/singularity)

- [Guidelines for Contributing](CONTRIBUTING.md)
- [Pull Request Template](.github/PULL_REQUEST_TEMPLATE.md)
- [Project License](LICENSE.md)
- [Documentation](http://singularity.lbl.gov/)
- [Citation](http://journals.plos.org/plosone/article?id=10.1371/journal.pone.0177459)

# Singularity - Enabling users to have full control of their environment.

Starting a Singularity container "swaps" out the host operating system
environment for one the user controls!

Let's say you are running Ubuntu on your workstation or server, but you
have an application which only runs on Red Hat Enterprise Linux 6.3.
Singularity can instantly virtualize the operating system, without
having root access, and allow you to run that application in its native
environment!

# About

Singularity is a container platform focused on supporting "Mobility of
Compute" 

Mobility of Compute encapsulates the development to compute model where
developers can work in an environment of their choosing and creation and
when the developer needs additional compute resources, this environment
can easily be copied and executed on other platforms. Additionally as
the primary use case for Singularity is targeted towards computational
portability, many of the barriers to entry of other container solutions
do not apply to Singularity making it an ideal solution for users (both
computational and non-computational) and HPC centers.

## The Container
Singularity utilizes container images, which means when you enter and
work within the Singularity container, you are physically located inside
of this image. The image grows and shrinks in real time as you install
or delete files within the container. If you want to copy a container,
you copy the image.

Using a single image for the container format, has added advantages
especially within the context of HPC with large parallel file systems
because all metadata operations within the container occur within the
container image (and not on the metadata server!).

## Mobility of Compute
With Singularity, developers who like to be able to easily control their
own environment will love Singularity's flexibility. Singularity does not
provide a pathway for escalation of privilege (as do other container
platforms which are thus not applicable for multi-tenant resources) so
you must be able to become root on the host system (or virtual machine)
in order to modify the container.

A Singularity container can be launched in a variety of different ways
depending on what you wanted to do with it. A simple method might be to
launch an interactive shell within the container image as follows:

    [gmk@centos7-x64 demo]$ singularity shell /tmp/Centos-7.img 
    gmk@Centos-7.img demo> echo "Hello from within the container"
    Hello from within the container
    gmk@Centos-7.img demo> whoami
    gmk
    gmk@Centos-7.img demo> 

And if you wanted to do the same thing as root:

    [gmk@centos7-x64 demo]$ sudo singularity shell -w /tmp/Centos-7.img 
    root@Centos-7.img demo> whoami
    root
    root@Centos-7.img demo> 

*note: By default, Singularity launches the container image in read
only mode (so it can be easily launched in parallel). The -w option
used above tells Singularity to mount the image in read/write mode such
that root can now make changes to the container.*

Additionally relevant file systems on your host are automatically shared
within the context of your container. This can be demonstrated as
follows:

    [gmk@centos7-x64 demo]$ pwd
    /home/gmk/demo
    [gmk@centos7-x64 demo]$ echo "world" > hello
    [gmk@centos7-x64 demo]$ singularity shell /tmp/Centos-7.img 
    gmk@Centos-7.img demo> pwd
    /home/gmk/demo
    gmk@Centos-7.img demo> cat hello
    world

Once the developer has completed their environment the image file can be
compressed and copied to any other system that has Singularity installed.
If you do not have root on that system, you will not be able to make any
changes to the image once on that system. But you will be able to use
the container and access the data and files outside the container as
easily as you would on your development system or virtual machine.

## Portability of Singularity container images
Singularity images are highly portable between Linux distributions (as
long as the binary format is the same). You can generate your image on
Debian or CentOS, and run it on Mint or Slackware.

Within a particular container one can include their programs, data,
scripts and pipelines and thus portable to any other architecture
compatible Linux system or distribution.

## Bootstrapping new images
Generally when bootstrapping an image from scratch you must build it from
a compatible host. This is because you must use the distribution specific
tools it comes with (e.g. Red Hat does not provide Debian's debootstrap).
But once the image has been bootstrapped and includes the necessary bits
to be self hosting (e.g. YUM on CentOS and apt-get on Debian/Ubuntu) then
the process of managing the container can be implemented from within the
container.

The process of building a bootstrap starts with a definition
specification. The definition file describes how you want the operating
system to be built, what should go inside it and any additional
modifications necessary.

Here is an example of a very simple bootstrap definition file for CentOS:

    BootStrap: yum
    OSVersion: 7
    MirrorURL: http://mirror.centos.org/centos-%{OSVERSION}/%{OSVERSION}/os/$basearch/
    Include: yum

Once you have created your bootstrap definition, you can build your
Singularity container image by first creating a blank image, and then
bootstrapping using your definition file:

    [gmk@centos7-x64 demo]$ sudo singularity create /tmp/Centos-7.img
    [gmk@centos7-x64 demo]$ sudo singularity bootstrap /tmp/Centos-7.img centos.def

From there we can immediately start using the container:

    [gmk@centos7-x64 demo]$ singularity exec /tmp/Centos-7.img cat /etc/redhat-release 
    CentOS Linux release 7.2.1511 (Core) 
    [gmk@centos7-x64 demo]$ singularity exec /tmp/Centos-7.img python --version
    Python 2.7.5
    [gmk@centos7-x64 demo]$ singularity exec /tmp/Centos-7.img python hello.py 
    hello world
    [gmk@centos7-x64 demo]$ 

And if I do this same process again, while changing the **OSVersion**
variable in the bootstrap definition to **6** (where previously it was
automatically ascertained by querying the RPM database), we can
essentially build a CentOS-6 image in exactly the same manner as
above. Doing so reveals this:

    [gmk@centos7-x64 demo]$ singularity exec /tmp/Centos-6.img cat /etc/redhat-release 
    CentOS release 6.7 (Final)
    [gmk@centos7-x64 demo]$ singularity exec /tmp/Centos-6.img python --version
    Python 2.6.6
    [gmk@centos7-x64 demo]$ 

And as expected, the Python version we now see is what comes from by 
default in CentOS-6.


# Cite as:

```
Kurtzer GM, Sochat V, Bauer MW (2017) Singularity: Scientific containers for mobility of compute. PLoS ONE 12(5): e0177459. https://doi.org/10.1371/journal.pone.0177459
```

We also have a Zenodo citation:

```
Kurtzer, Gregory M.. (2016). Singularity 2.1.2 - Linux application and environment
containers for science. 10.5281/zenodo.60736

http://dx.doi.org/10.5281/zenodo.60736
```

# Webpage
We have full documentation at [http://singularity.lbl.gov/](http://singularity.lbl.gov/), and [welcome contributions](http://www.github.com/singularityware/singularityware.github.io).
