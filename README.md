# Singularity - Enabling Mobility of Compute

Singularity is a container platform focused on supporting "Mobility of
Compute".

Mobility of Compute encapsulates the development to compute model where
developers can work in an envrionment of their choosing and creation and
when the developer needs additional compute resources, this environment
can easily be copied and executed on other platforms. Additionally as
the primary use case for Singularity is targetted towards computational
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
own environment will love Singularities flexibility. Singularity does not
provide a pathway for escalation of privledge (as do other container
platforms which are thus not applicable for multi-tennant resources) so
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

Additionally relevent file systems on your host are automatically shared
within the context of your container. This can be demonstrated as
follows:

    [gmk@centos7-x64 demo]$ pwd
    /home/gmk/demo
    [gmk@centos7-x64 demo]$ echo "world" > hello
    [gmk@centos7-x64 demo]$ singularity shell /tmp/Centos-7.img 
    gmk@Centos-7.img demo> cat hello
    world
    gmk@Centos-7.img demo> 

Once the developer has completed their environment the image file can be
compressed and copied to any other system that has Singularity installed.
If you do not have root on that system, you will not be able to make any
changes to the image once on that system. But you will be able to use
the container and access the data and files outside the container as
easily as you would on your development system or virtual machine.

# Webpage
We are working on documentation and web pages now, but checkout the work
in progress here:

http://gmkurtzer.github.io/singularity
