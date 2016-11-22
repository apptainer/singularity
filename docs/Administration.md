# Singularity Administration Guide
This document will cover installation and administration points of Singularity for multi-tenant HPC resources and will not cover usage of the command line tools, container usage, or example use cases.


## Installation
There are two common ways to install Singularity, from source code and via binary packages. This document will explain the process of installation from source, and it will depend on your build host to have the appropriate development tools and packages installed. For Red Hat and derivitives, you should install the following `yum` group to ensure you have an appropriately setup build server:

```bash
$ sudo yum groupinstall "Development Tools"
```

### Downloading the Source
You can download the source code either from the latest stable tarball release or via the GitHub master repository. Here is an example downloading and preparing the latest development code from GitHub:

```bash
$ mkdir ~/git
$ cd ~/git
$ git clone https://github.com/singularityware/singularity.git
$ cd singularity
$ ./autogen.sh
```

Once you have downloaded the source, the following installation procedures will assume you are running from the root of the source directory.

### Source Installation
The following example demonstrates how to install Singularity into `/usr/local`. You can install Singularity into any directory of your choosing, but you must ensure that the location you select supports programs running as `SUID`. It is common for people to disable `SUID` with the mount option `nosuid` for various network mounted file systems. To ensure proper support, it is easiest to make sure you install Singularity to a local file system.

Assuming that `/usr/local` is a local file system:

```bash
$ ./configure --prefix=/usr/local --sysconfdir=/etc
$ make
$ sudo make install
```

***NOTE: The `make install` above must be run as root to have Singularity properly installed. Failure to install as root will cause Singularity to not function properly or have limited functionality when run by a non-root user.***

### Building an RPM directly from the source
Singularity includes all of the necessary bits to properly create an RPM package directly from the source tree, and you can create an RPM by doing the following:

```bash
$ ./configure
$ make dist
$ rpmbuild -ta singularity-*.tar.gz
```

Near the bottom of the build output you will see several lines like:

```
...
Wrote: /home/gmk/rpmbuild/SRPMS/singularity-2.2-0.1.el7.centos.src.rpm
Wrote: /home/gmk/rpmbuild/RPMS/x86_64/singularity-2.2-0.1.el7.centos.x86_64.rpm
Wrote: /home/gmk/rpmbuild/RPMS/x86_64/singularity-devel-2.2-0.1.el7.centos.x86_64.rpm
Wrote: /home/gmk/rpmbuild/RPMS/x86_64/singularity-debuginfo-2.2-0.1.el7.centos.x86_64.rpm
...
```

You will want to identify the appropriate path to the binary RPM that you wish to install, in the above example the package we want to install is `singularity-2.2-0.1.el7.centos.x86_64.rpm`, and you should install it with the following command:

```bash
$ sudo yum install /home/gmk/rpmbuild/RPMS/x86_64/singularity-2.2-0.1.el7.centos.x86_64.rpm
```

*Note: If you want to have the binary RPM install the files to an alternative location, you should define the environment variable 'PREFIX' (below) to suit your needs, and use the following command to build:*

```bash
$ PREFIX=/opt/singularity
$ rpmbuild -ta --define="_prefix $PREFIX" --define "_sysconfdir $PREFIX/etc" --define "_defaultdocdir $PREFIX/share" singularity-*.tar.gz
```

### Building a DEB directly from source

To build a deb package for Debian/Ubuntu/LinuxMint invoke the following commands:

```bash
$ fakeroot dpkg-buildpackage -b -us -uc # sudo will ask for a password to run the tests
$ sudo dpkg -i ../singularity-container_2.2-1_amd64.deb
```
 
Note that the tests will fail if singularity is not already installed on your system. This is the case when you run this procedure for the first time.
In that case run the following sequence:

```bash
$ echo "echo SKIPPING TESTS THEYRE BROKEN" > ./test.sh
$ fakeroot dpkg-buildpackage -nc -b -us -uc # this will continue the previous build without an initial 'make clean'
```

## Security
Once Singularity is installed in it's default configuration you may find that there is a SETUID component installed at `$PREFIX/libexec/singularity/sexec-suid`. The purpose of this is to do the require privilege escalation necessary for Singularity to operate properly. There are a few aspects of Singularity's functionality that require escalated privileges:

1. Mounting (and looping) the Singularity container image
2. Creation of the necessary namespaces in the kernel
3. Binding host paths into the container

In general, it is impossible to implement a container system that employs the features that Singularity offers without requiring extended privileges, but if this is a concern to you, the SUID components can be disabled via either the configuration file, changing the physical permissions on the sexec-suid file, or just removing that file. Depending on the kernel you are using and what Singularity features you employ this may (or may not) be an option for you. But first a warning...

Many people (ignorantly) claim that the 'user namespace' will solve all of the implementation problems with unprivileged containers. While it does solve some, it is currently feature limited. With time this may change, but even on kernels that have a reasonable feature list implemented, it is known to be very buggy and cause kernel panics. Additionally very few distribution vendors are shipping supported kernels that include this feature. For example, Red Hat considers this a "technology preview" and is only available via a system modification, while other kernels enable it and have been trying to keep up with the bugs it has caused. But, even in it's most stable form, the user namespace does not completely alleviate the necessity of privilege escalation unless you also give up the desire to support images (#1 above).

### How do other container solutions do it?
Docker and the like implement a root owned daemon to control the bring up, teardown, and functions of the containers. Users have the ability to control the daemon via a socket (either a UNIX domain socket or network socket). Allowing users to control a root owned daemon process which has the ability to assign network addresses, bind file systems, spawn other scripts and tools, is a large problem to solve and one of the reasons why Docker is not typically used on multi-tenant HPC resources.

### Security mitigations
SUID programs are common targets for attackers because they provide a direct mechanism to gain privileged command execution. These are some of the baseline security mitigations for Singularity:

1. Keep the escalated bits within the code as simple and transparent so it can be easily audit-able
2. Check the return value of every system call, command, and check and bomb out early if anything looks weird
3. Make sure that proper permissions of files and directories are owned by root (e.g. the config must be owned by root to work)
4. Don't trust any non-root inputs (like config values) unless they have been checked and/or sanitized
5. As much IO as possible is done via the calling user (not root)
6. Put as much system administrator control into the configuration file as possible
7. Drop permissions before running any non trusted code pathways
8. Limit all user actions within the container to that single user (disable escalation of privileges within a container)
9. Even though the user owns the image, it utilizes a POSIX like file system inside so files inside the container owned by root can only be modified by root

Additionally Singularity offers a very comprehensive auditing mechanism within it's debugging output by printing UID, PID, and location of every call it is making. For example:

```
$ singularity --debug shell /tmp/Centos7.img
...
DEBUG   [U=1000,P=33160]   privilege.c:152:singularity_priv_escalate(): Temporarily escalating privileges (U=1000)
VERBOSE [U=0,P=33160]      tmp.c:79:singularity_mount_tmp()           : Mounting directory: /tmp
DEBUG   [U=0,P=33160]      privilege.c:179:singularity_priv_drop()    : Dropping privileges to UID=1000, GID=1000
DEBUG   [U=1000,P=33160]   privilege.c:191:singularity_priv_drop()    : Confirming we have correct UID/GID
...
```

In the above output you can see that we are starting as UID 1000 (U=1000) and PID 33160 and we are escalating privileges. Once privileges have been increased, Singularity can properly mount /tmp and then it immediately drops permissions back to the calling user.

For comparison, the below output is when being called with the user namespace. Notice that I am not able to use the Singularity image format, and instead I am referencing a raw directory which contains the contents of the Singularity image:

```
$ singularity --debug shell -u /tmp/Centos7/
...
DEBUG   [U=1000,P=111121]  privilege.c:142:singularity_priv_escalate(): Not escalating privileges, user namespace enabled
VERBOSE [U=1000,P=111121]  tmp.c:80:singularity_mount_tmp()           : Mounting directory: /tmp
DEBUG   [U=1000,P=111121]  privilege.c:169:singularity_priv_drop()    : Not dropping privileges, user namespace enabled
...
```

## The Configuration File
When Singularity is running via the SUID pathway, the configuration **must** be owned by the root user otherwise Singularity will error out. This ensures that the system administrators have direct say as to what functions the users can utilize when running as root. If Singularity is installed as a non-root user, the SUID components are not installed, and the configuration file can be owned by the user (but again, this will limit functionality).

The Configuration file can be found at `$SYSCONFDIR/singularity/singularity.conf` and is generally self documenting but there are several things to pay special attention to:


### ALLOW SETUID (boolean, default='yes')
This parameter toggles the global ability to execute the SETUID (SUID) portion of the code if it exists. As mentioned earlier, if the SUID features are disabled, various Singularity features will not function (e.g. mounting of the Singularity image file format).

You can however disable SUID support ***iff*** (if and only if) you do not need to use the default Singularity image file format and if your kernel supports user namespaces and you choose to use user namespaces.

*note: as of the time of this writing, the user namespace is rather buggy*


### ALLOW PID NS (boolean, default='yes')
While the PID namespace is a *neat* feature, it does not have much practical usage in an HPC context so it is recommended to disable this if you are running on an HPC system where a resource manager is involved as it has been known to cause confusion on some kernels with enforcement of user limits.

Even if the PID namespace is enabled by the system administrator here, it is not implemented by default when running containers. The user will have to specify they wish to implement un-sharing of the PID namespace as it must fork a child process.


### ENABLE OVERLAY (boolean, default='yes')
The overlay file system creates a writable substrate to create bind points if necessary. This feature is very useful when implementing bind points within containers where the bind point may not already exist so it helps with portability of containers. Enabling this option has been known to cause some kernels to panic as this feature maybe present within a kernel, but has not proved to be stable as of the time of this writing (e.g. the Red Hat 7.2 kernel).

### CONFIG PASSWD,GROUP,RESOLV_CONF (boolean, default='yes')
All of these options essentially do the same thing for different files within the container. This feature updates the described file (`/etc/passwd`, `/etc/group`, and `/etc/resolv.conf` respectively) to be updated dynamically as the container is executed. It uses binds and modifies temporary files such that the original files are not manipulated.

### MOUNT PROC,SYS,DEV,HOME,TMP (boolean, default='yes')
These configuration options control the mounting of these file systems within the container and of course can be overridden by the system administrator (e.g. the system admin decides not to include the /dev tree inside the container). In most useful cases, these are all best to leave enabled.

### MOUNT HOSTFS (boolean, default='no')
This feature will parse the host's mounted file systems and attempt to replicate all mount points within the container. This maybe a desirable feature for the lazy, but it is generally better to statically define what bind points you wish to encapsulate within the container by hand (using the below "bind path" feature).

### BIND PATH (string)
With this configuration directive, you can specify any number of bind points that you want to extend from the host system into the container. Bind points on the host file system must be either real files or directories (no special files supported at this time). If the overlayFS is not supported on your host, or if `enable overlay = no` in this configuration file, a bind point must exist for the file or directory within the container.

The syntax for this consists of a bind path source and an optional bind path destination separated by a colon. If not bind path destination is specified the bind path source is used also as the destination.


### USER BIND CONTROL (boolean, default='yes')
In addition to the system bind points as specified within this configuration file, you may also allow users to define their own bind points inside the container. This feature is used via multiple command line arguments (e.g. `--bind`, `--scratch`, and `--home`) so disabling user bind control will also disable those command line options.

Singularity will automatically disable this feature if the host does not support the prctl option `PR_SET_NO_NEW_PRIVS`.


## Logging
In order to facilitate monitoring and auditing, Singularity will syslog() every action and error that takes place to the `LOCAL0` syslog facility. You can define what to do with those logs in your syslog configuration.

## Loop Devices
Singularity images have `ext3` file systems embedded within them, and thus to mount them, we need to convert the raw file system image (with variable offset) to a block device. To do this, Singularity utilizes the `/dev/loop*` block devices on the host system and manages the devices programmatically within Singularity itself. Singularity also uses the `LO_FLAGS_AUTOCLEAR` loop device `ioctl()` flag which tells the kernel to automatically free the loop device when there are no more open file descriptors to the device itself.

Earlier versions of Singularity managed the loop devices via a background watchdog process, but since version 2.2 we leverage the `LO_FLAGS_AUTOCLEAR` functionality and we forego the watchdog process. Unfortunately, this means that some older Linux distributions are no longer supported (e.g. RHEL <= 5).

Given that loop devices are consumable (there are a limited number of them on a system), Singularity attempts to be smart in how loop devices are allocated. For example, if a given user executes a specific container it will bind that image to the next available loop device automatically. If that same user executes another command on the same container, it will use the loop device that has already been allocated instead of binding to another loop device. Most Linux distributions only support 8 loop devices by default, so if you find that you have a lot of different users running Singularity containers, you may need to increase the number of loop devices that your system supports by doing the following:

Edit or create the file `/etc/modprobe.d/loop.conf` and add the following line:

```
options loop max_loop=128

```

After making this change, you should be able to reboot your system or unload/reload the loop device as root using the following commands:

```bash
# modprobe -r loop
# modprobe loop
```

## Troubleshooting
This section will help you debug (from the system administrator's perspective) Singularity.

### Not installed correctly, or installed to a non-compatible location
Singularity must be installed by root into a location that allows for `SUID` programs to be executed (as described above in the installation section of this manual). If you fail to do that, you may have user's reporting one of the following error conditions:

```
ERROR  : Singularity must be executed in privileged mode to use images
ABORT  : Retval = 255
```
```
ERROR  : User namespace not supported, and program not running privileged.
ABORT  : Retval = 255
```
```
ABORT  : This program must be SUID root
ABORT  : Retval = 255
```
If one of these errors is reported, it is best to check the installation of Singularity and ensure that it was properly installed by the root user onto a local file system.
