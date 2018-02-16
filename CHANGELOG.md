# CHANGELOG

This is a manually generated log to track changes to the repository for each release. 
Each section should include general headers such as ### Implemented enhancements 
and **Merged pull requests**. All closed issued and bug fixes should be 
represented by the pull requests that fixed them. This log originated with Singularity 2.4
and changes prior to that are (unfortunately) done retrospectively. Critical items to know are:

 - renamed, deprecaed, or removed commands
 - defaults that are changed
 - backward incompatible changes (recipe file format? image file format?)
 - migration guidance (how to convert images?)
 - changed behaviour (recipe sections work differently)


## [v2.4.2](https://github.com/singularityware/singularity/tree/release-2.4)

 - This fixed an issue for support of older distributions and kernels with regards to `setns()`
   functionality.
 - Fixed autofs bug path (lost during merge)

## [v2.4.1](https://github.com/singularityware/singularity/tree/release-2.4) (2017-11-22)

### Security related fixes
 - Fixed container path and owner limitations (original merge was lost)
 - Check of overlay upper/work images are symlinks

### Implemented enhancements
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

## [v2.4](https://github.com/singularityware/singularity/tree/v2.4) (2017-10-02)
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

## [v2.3.2](https://github.com/singularityware/singularity/tree/v2.3.2) (2017-09-15)
[Full Changelog](https://github.com/singularityware/singularity/compare/2.3.1...2.3.2)

### Implemented enhancements
 - Quick fix to support manifest lists when pulling from Docker Hub

## [v2.3.1](https://github.com/singularityware/singularity/tree/v2.3.1) (2017-06-26)
[Full Changelog](https://github.com/singularityware/singularity/compare/2.3...2.3.1)

### Security Fix
 - A fix was implemented to address an escalation pathway and various identified bugs and potential race conditions.

## [v2.3](https://github.com/singularityware/singularity/tree/v2.3) (2017-05-31)
[Full Changelog](https://github.com/singularityware/singularity/compare/2.2.1...2.3)

### Implemented enhancements
- Lots of backend library changes to accommodate a more flexible API
- Restructured Python backend
- Updated bootstrap backend to make it much more reliable
- Direct support for Singularity-Hub
- Ability to run additional commands without root privileges (e.g. create, import, copy, export, etc..).
- Added ability to pull images from Singularity Hub and Docker
- Containers now have labels, and are inspect'able

## [v2.2.1](https://github.com/singularityware/singularity/tree/v2.2.1) (2017-02-14)
[Full Changelog](https://github.com/singularityware/singularity/compare/2.2...2.2.1)

### Security Fix
 - a security loophole related to mount devices was fixed (thanks @UMU in Sweden)

### Implemented enhancements
 - Fixed some leaky file descriptors
 - Cleaned up `*printf()` usage
 - Catch if user's group is not properly defined

## [v2.2](https://github.com/singularityware/singularity/tree/v2.2) (2016-10-11)
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


## [v2.1.2](https://github.com/singularityware/singularity/tree/v2.1.2) (2016-08-04)
[Full Changelog](https://github.com/singularityware/singularity/compare/2.1.1...2.1.2)

### Bug Fixes
 - Fix for kernel panic on corrupt images
 - Fixes build warning

## [v2.1.1](https://github.com/singularityware/singularity/tree/v2.1.1) (2016-08-03)
[Full Changelog](https://github.com/singularityware/singularity/compare/2.1...2.1.1)

### Bug Fixes
- Contain option no longer maintains current working directory
- Remove need to obtain a shared lock on the image (was failing on some shared file systems)
- Move creation of a container's /environment to the beginning of the bootstrap (so it can be modified via a bootstrap definition file

## [v2.1](https://github.com/singularityware/singularity/tree/v2.1) (2016-07-28)
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

## [v2.0](https://github.com/singularityware/singularity/tree/v2.0) (2016-06-01)
[Full Changelog](https://github.com/singularityware/singularity/compare/1.x...2.0)

### Implemented enhancements
 - Support for non-root container contexts (user outside container, is same user inside container)
 - Support of “live” container sparse image files
 - Utilizing the operating system’s build and dependency resolution subsystems (e.g. YUM, Apt, etc.)
 - Support for Open MPI 2.1 (pre-release)
 - Updates for usage with non-local file systems
 - Performance optimizations
 - Support for native X11


## [v1.x](https://github.com/singularityware/singularity/tree/v1.x) (2016-04-06)

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
