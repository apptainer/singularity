# CHANGELOG

This is a manually generated log to track changes to the repository for each release. 
Each section should include general headers such as ### Implemented enhancements 
and **Merged pull requests**. All closed issued and bug fixes should be 
represented by the pull requests that fixed them. This log originated with Singularity 2.4
and changes prior to that are (unfortunately) done retrospectively. Critical items to know are:

 - renamed, deprecated, or removed commands
 - defaults that are changed
 - backward incompatible changes (recipe file format? image file format?)
 - migration guidance (how to convert images?)
 - changed behaviour (recipe sections work differently)

## [v2.6.1]

### [Security related fixes](https://cve.mitre.org/cgi-bin/cvename.cgi?name=2018-1929)
 - disables instance features for mount commands, disables instance join for 
   start command, and disables daemon start for action commands

## [v2.6.0]

### Bug fixes
 - Fix image expand functionality by additional losetup/mount -o bind,offset=31     

### Implemented enhancements
 - Allow admin to specify a non-standard location for mksquashfs binary at 
   build time with `--with-mksquashfs` option #1662
 - `--nv` option will use [nvidia-container-cli](https://github.com/NVIDIA/libnvidia-container) if installed #1681
 - [nvliblist.conf](https://github.com/singularityware/singularity/blob/master/etc/nvliblist.conf) now has a section for binaries #1681
 - `--nv` can be made default with all action commands in singularity.conf #1681
 - `--nv` can be controlled by env vars `$SINGULARITY_NV` and 
   `$SINGULARITY_NV_OFF` #1681
 - Refactored travis build and packaging tests #1601
 - Added build and packaging tests for Debian 8/9 and openSUSE 42.3/15.0 #1713
 - Restore shim init process for proper signal handling and child reaping when
   container is initiated in its own PID namespace #1221
 - Add `-i` option to image.create to specify the inode ratio. #1759
 - Bind `/dev/nvidia*` into the container when the `--nv` flag is used in 
    conjuction with the `--contain` flag #1358
 - Add `--no-home` option to not mount user $HOME if it is not the $CWD and
   `mount home = yes` is set. #1761
 - Added support for OAUTH2 Docker registries like Azure Container Registry #1622

### Bug fixes
 - Fix 404 when using Arch Linux bootstrap #1731
 - Fix environment variables clearing while starting instances #1766

## [v2.5.2](https://github.com/singularityware/singularity/releases/tag/2.5.2) (2018-07-03)

### [Security related fixes](https://cve.mitre.org/cgi-bin/cvename.cgi?name=2018-12021)
 - Removed the option to use overlay images with `singularity mount`.  This 
   flaw could allow a malicious user accessing the host system to access
   sensitive information when coupled with persistent ext3 overlay.
 - Fixed a race condition that might allow a malicious user to bypass directory 
   image restrictions, like mounting the host root filesystem as a container 
   image

### Bug fixes
 - Fix an error in malloc allocation #1620
 - Honor debug flag when pulling from docker hub #1556
 - Fix a bug with passwd abort #1580
 - Allow user to override singularity.conf "mount home = no" with --home option
   #1496
 - Improve debugging output #1535
 - Fix some bugs in bind mounting #1525
 - Define PR_(S|G)ET_NO_NEW_PRIVS in user space so that these features will 
   work with kernels that implement them (like Cray systems) #1506
 - Create /dev/fd and standard streams symlinks in /dev when using minimal dev
   mount or when specifying -c/-C/--contain option #1420
 - Fixed * expansion during app runscript creation #1486

## [v2.5.1](https://github.com/singularityware/singularity/releases/tag/2.5.1) (2018-05-03)

### Bug fixes
 - Corrected a permissions error when attempting to run Singularity from a 
   directory on NFS with root_squash enabled  
 - Fixed a bug that closed a socket early, preventing correct container 
   execution on hosts using identity services like SSSD
 - Fixed a regression that broke the debootstrap agent

## [v2.5.0](https://github.com/singularityware/singularity/releases/tag/2.5.0) (2018-04-27)

### Security related fixes

Patches are provided to prevent a malicious user with the ability to log in to 
the host system and use the Singularity container runtime from carrying out any 
of the following actions:

 - Create world writable files in root-owned directories on the host system by 
   manipulating symbolic links and bind mounts 
 - Create folders outside of the container by manipulating symbolic links in 
   conjunction with the `--nv` option or by bypassing check_mounted function 
   with relative symlinks
 - Bypass the `enable overlay = no` option in the `singularity.conf` 
   configuration file by setting an environment variable
 - Exploit buffer overflows in `src/util/daemon.c` and/or 
   `src/lib/image/ext3/init.c` (reported by Erik Sjölund (DBB, Stockholm 
   University, Sweden))
 - Forge of the pid_path to join any Singularity namespace (reported by Erik 
   Sjölund (DBB, Stockholm University, Sweden))

### Implemented enhancements

 - Restore docker-extract aufs whiteout handling that implements correct
   extraction of docker container layers. This adds libarchive-devel as a
   build time dep. At runtime libarchive is needed for whiteout handling. If
   libarchive is not available at runtime will fall back to previous
   extraction method.
 - Changed behavior of SINGULARITYENV_PATH to overwrite container PATH and
   added SINGULARITYENV_PREPEND_PATH and SINGULARITYENV_APPEND_PATH for users
   wanting to prepend or append to the container PATH at runtime

### Bug fixes

 - Support pulls from the NVIDIA cloud docker registry (fix by Justin Riley, 
   Harvard)
 - Close socket file descriptors in fd_cleanup
 - Fix conflict between `--nv` and `--contain` options
 - Throw errors at build and runtime if NO_NEW_PRIVS is not present and working
 - Reset umask to 0022 at start to corrrect several errors
 - Verify docker layers after download with sha256 checksum
 - Do not make excessive requests for auth tokens to docker registries
 - Fixed stripping whitespaces and empty new lines for the app commands (fix by 
   Rafal Gumienny, Biozentrum, Basel)
 - Improved the way that working directory is mounted 
 - Fixed an out of bounds array in src/lib/image/ext3/init.c

## [v2.4.6](https://github.com/singularityware/singularity/releases/tag/2.4.6) (2018-04-04)

 - Fix for check_mounted() to check parent directories #1436
 - Free strdupped temporary variable in joinpath #1438

## [v2.4.5](https://github.com/singularityware/singularity/releases/tag/2.4.5) (2018-03-19)

### Security related fixes
 - Strip authorization header on http redirect to different domain when
   interacting with docker registries.

## [v2.4.4](https://github.com/singularityware/singularity/releases/tag/2.4.4) (2018-03-03)

 - Removed capability to handle docker layer aufs whiteout files correctly as
   it increased potential attack surface on some distros (with apologies to 
   users who requested it).

## [v2.4.3](https://github.com/singularityware/singularity/releases/tag/2.4.3) (2018-03-03)

### Bug Fixes
 - Put /usr/local/{bin,sbin} in front of the default PATH
 - Fixed bug that did not export environment variables for apps with "-" in name
 - Fix permission denied when binding directory located on NFS with root_squash enabled
 - Add capability to support all tar compression formats #1155
 - Handle docker layer aufs whiteout files correctly (requires libarchive).
 - Close file descriptors pointing to a directory #1305
 - Updated output of image.print command #1190
 - Fixed parsing of backslashes in apprun script #1189
 - Fixed parsing of arch keyword from definition file #1217
 - Fixed incompatibility between --pwd and --contain options #1259
 - Updated license information #1267
 - Fix non-root build from docker containers with non-writable file/dir permissions
 - Fix race condition between container exit and cleanupd while removing runtime directory

## [v2.4.2](https://github.com/singularityware/singularity/releases/tag/2.4.2) (2017-12-05)

 - This fixed an issue for support of older distributions and kernels with regards to `setns()`
   functionality.
 - Fixed autofs bug path (lost during merge)
 - Added json format to instance.list with flag --json

## [v2.4.1](https://github.com/singularityware/singularity/releases/tag/2.4.1) (2017-11-22)

### apprun script backslash removal fix
 - Fixed the unwanted removal of backslashes in apprun scripts

### Security related fixes
 - Fixed container path and owner limitations (original merge was lost)
 - Check of overlay upper/work images are symlinks

### Implemented enhancements
 - Users can specify custom shebang in first line of runscript or startscript
 - This changelog was added.
 - Addition of APP[app]_[LABELS,ENV,RUNSCRIPT,META] so apps can internally find one another.
 - Exposing labels for SCI-F in environment

### Bug Fixes
 - Adjusting environment parsing regular expression for Docker to allow for "=" sign in variable
 - Try overlayFS now default option
 - Confirm that localstate directories were properly packaged
 - Fix when running over NFS with root_squash enabled
 - Honor the user name request when pulling from Singularity Hub
 - Allow http_proxy envar for runtime and build
 - Properly require mksquashfs tools for Debian packaging
 - Fix for empty docker namespaces in private repositories
 - Fix Docker environment parsing
 - Revert lolcow easter egg
 - Fix "Duplicate bootstrap definition key" triggered by comments and blank spaces
 - Fix for docker permission error when downloading multiple layers
 - Fix parsing of registry (including port), namespace, tags, and version
 - Add "$@" to any CMD/ENTRYPOINT found when building from Docker
 - Added sqaushfs-tools as a dependency for building deb files
 - Fix terminal echo problem when using PID namespace and killing shell
 - Fix SuSE squashFS package name in RPM spec

## [v2.4](https://github.com/singularityware/singularity/releases/tag/2.4) (2017-10-02)
[Full Changelog](https://github.com/singularityware/singularity/compare/2.3.2...2.4)

### Implemented enhancements

 - a new `build` command was added to replace `create` + `bootstrap` ([build](https://singularityware.github.io/docs-build-container))
 - default image format is squashfs, eliminating the need to specify a size
 - for development build supports `--sandbox` (folder) and `--writable` (ext3)
 - a `localimage` can be used as a build base, including ext3, sandbox, and other squashfs images
 - singularity hub can now be used as a base with the uri `shub://`
 - support has been added for instances (services) including network namespace isolation under the `instances` group of commands.
 - [singularity registry](https://www.github.com/singularityhub/sregistry) is released and published
 - [Standard Container Integration Format](https://singularityware.github.io/docs-apps) apps are added to support internal modularity and organization.
 - [build environment](https://singularityware.github.io/build-environment) is better documented
 - Persistent Overlay 
 - Container checks
 - Tests for instance support
 - Wrapper for create
 - Group instance commands
 - Group image commands
 - Bash completion updates

### Deprecated
 - the `create` command is being deprecated in favor of `image.create`
 - `bootstrap` is being deprecated in favor of `build` (will work through 2.4)
 - `expand` is being deprecated in favor of `image.expand`, and no longer works on images with headers (meaning after they are built).
 - `export` is being deprecated and added to the image command group, `image.export`
 - the `shub://` URI no longer supports an integer to reference a container

## [v2.3.2](https://github.com/singularityware/singularity/releases/tag/2.3.2) (2017-09-15)
[Full Changelog](https://github.com/singularityware/singularity/compare/2.3.1...2.3.2)

### Implemented enhancements
 - Quick fix to support manifest lists when pulling from Docker Hub

## [v2.3.1](https://github.com/singularityware/singularity/releases/tag/2.3.1) (2017-06-26)
[Full Changelog](https://github.com/singularityware/singularity/compare/2.3...2.3.1)

### Security Fix
 - A fix was implemented to address an escalation pathway and various identified bugs and potential race conditions.

## [v2.3](https://github.com/singularityware/singularity/releases/tag/2.3) (2017-05-31)
[Full Changelog](https://github.com/singularityware/singularity/compare/2.2.1...2.3)

### Implemented enhancements
- Lots of backend library changes to accommodate a more flexible API
- Restructured Python backend
- Updated bootstrap backend to make it much more reliable
- Direct support for Singularity-Hub
- Ability to run additional commands without root privileges (e.g. create, import, copy, export, etc..).
- Added ability to pull images from Singularity Hub and Docker
- Containers now have labels, and are inspect'able

## v2.2.1 (2017-02-14)
[Full Changelog](https://github.com/singularityware/singularity/compare/2.2...2.2.1)

### Security Fix
 - a security loophole related to mount devices was fixed (thanks @UMU in Sweden)

### Implemented enhancements
 - Fixed some leaky file descriptors
 - Cleaned up `*printf()` usage
 - Catch if user's group is not properly defined

## v2.2 (2016-10-11)
[Full Changelog](https://github.com/singularityware/singularity/compare/2.1.2...2.2)

### Implemented enhancements
 - A complete rework of the back end source code to allow a much larger feature set, sanity, and facilitate contributions
 - The ability to execute completely unprivileged (does not support Singularity images) (thanks to Brian Bockelman)
 - Container execute by URI support (file, http, https, docker, etc..)
 - Integration with the Docker Registry Remote API (thanks to @vsoch), including stateless containers running ad-hoc, bootstrapping, and importing
 - OverlayFS support - Allows for automatic creation of bind points within containers at runtime (thanks to Amanda Duffy and Jarrod Johnson)
 - Additional container formats supported (directories and archives)
 - New bootstrap definition format to handle much more complicated and intuitive recipes
 - All Singularity 2.x containers continue to be supported with this release.


## v2.1.2 (2016-08-04)
[Full Changelog](https://github.com/singularityware/singularity/compare/2.1.1...2.1.2)

### Bug Fixes
 - Fix for kernel panic on corrupt images
 - Fixes build warning

## v2.1.1 (2016-08-03)
[Full Changelog](https://github.com/singularityware/singularity/compare/2.1...2.1.1)

### Bug Fixes
- Contain option no longer maintains current working directory
- Remove need to obtain a shared lock on the image (was failing on some shared file systems)
- Move creation of a container's /environment to the beginning of the bootstrap (so it can be modified via a bootstrap definition file

## v2.1 (2016-07-28)
[Full Changelog](https://github.com/singularityware/singularity/compare/2.0...2.1)

### Implemented enhancements
- Configuration file for system administrator control over what Singularity features users are allowed to use
- Support for non Gnu LibC based distributions (e.g. Alpine Linux)
- Source file restructuring and refactoring
- Added message(), and enabled very verbose debugging
- Be smarter about when to avoid separation of the PID namespace
- Log container runs to syslog()
- Support custom container environments (via container:/environment)
- Sanitized source files for Flawfinder

### Bug Fixes
- Fix bug with /run and /var directories being read only in some situations
- Fix lots of bootstrap definition issues
- Fixed issue with /dev/pts not being mounted within a container
- Resolved some issues with image file de-looping
- Fixed bugs related to very restrictive umasks set

## v2.0 (2016-06-01)
[Full Changelog](https://github.com/singularityware/singularity/compare/1.x...2.0)

### Implemented enhancements
 - Support for non-root container contexts (user outside container, is same user inside container)
 - Support of “live” container sparse image files
 - Utilizing the operating system’s build and dependency resolution subsystems (e.g. YUM, Apt, etc.)
 - Support for Open MPI 2.1 (pre-release)
 - Updates for usage with non-local file systems
 - Performance optimizations
 - Support for native X11


## v1.x (2016-04-06)

### Implemented enhancements

 - Ability to create Singularity containers based on a package specfile
 - Specfile templates can be generated automatically (singularity specgen …)
 - Support for various automatic dependency resolution
 - Dynamic libraries
 - Perl scripts and modules
 - Python scripts and modules
 - R scripts and modules
 - Basic X11 support
 - Open MPI (v2.1 - which is not yet released)
 - Direct execution of Singularity containers (e.g. ./container.sapp [opts])
 - Access to files in your home directory and a scratch directory
 - Existing IO (pipes, stdio, stderr, and stdin) all maintained through container
 - Singularity internal container cache management
 - Standard networking access (exactly as it does on the host)
 - Singularity containers run within existing resource contexts (CGroups and ulimits are maintained)
 - Support for scalable execution of MPI parallel jobs
 - Singularity containers are portable between Linux distributions

