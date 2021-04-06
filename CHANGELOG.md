_With the release of `v3.0.0`, we're introducing a new changelog format in an attempt to consolidate the information presented in the changelog. The new changelog is reduced in scope to only documenting functionality changes from version to version. This ensures that the changelog is as useful as it can be. Changes which should be documented include:_
  
  - _Renamed commands_
  - _Deprecated / removed commands_
  - _Changed defaults / behaviors_
  - _Migration guidance_
  - _New features / functionalities_


_The old changelog can be found in the `release-2.6` branch_

# v3.7.3 - [2021-04-06]

## Security Related Fixes

  - [CVE-2021-29136](https://github.com/opencontainers/umoci/security/advisories/GHSA-9m95-8hx6-7p9v):
   A dependency used by Singularity to extract docker/OCI image layers
   can be tricked into modifying host files by creating a malicious
   layer that has a symlink with the name "." (or "/"), when running
   as root. This vulnerability affects a `singularity build` or
   `singularity pull` as root, from a docker or OCI source.


# v3.7.2 - [2021-03-09]

## Bug Fixes

  - Fix progress bar display when source image size is unknown.
  - Fix a memory usage / leak issue when building from an existing
    image file.
  - Fix to allow use of ``--library`` flag to point push/pull at
    default cloud library when another remote is in use.
  - Address false positive loop test errors, and an e2e test registry
    setup issue.


# v3.7.1 - [2021-01-12]

## Bug Fixes

  - Accommodate /sys/fs/selinux mount changes on kernel 5.9+.
  - Fix loop devices file descriptor leak when shared loop devices is
    enabled.
  - Use MaxLoopDevices variable from config file in all appropriate
    locations.
  - Use -buildmode=default (non pie) on ppc64le to prevent crashes
    when using plugins.
  - Remove spurious warning in parseTokenSection()
  - e2e test fixes for new kernels, new unsquashfs version.
  - Show correct web URI for detached builds against alternate remotes.


# v3.7.0 - [2020-11-24]

## New features / functionalities

  - Allow configuration of global custom keyservers, separate from
    remote endpoints.
  - Add a new global keyring, for public keys only (used for ECL).
  - The `remote login` commmand now suports authentication to Docker/OCI
    registries and custom keyservers.
  - New `--exclusive` option for `remote use` allows admin to lock usage
    to a specific remote.
  - A new `Fingerprints:` header in definition files will check that
    a SIF source image can be verified, and is signed with keys
    matching all specified fingerprints.
  - Labels can be set dynamically from a build's `%post` section by
    setting them in the `SINGULARITY_LABELS` environment variable.
  - New `build-arch` label is automatically set to the architecure of
    the host during a container build.
  - New `-D/--description` flag for `singularity push` sets
    description for a library container image.
  - `singularity remote status` shows validity of authentication token if
    set.
  - `singularity push` reports quota usage and URL on successful push
    to a library server that supports this.
  - A new `--no-mount` flag for actions allows a user to disable
    proc/sys/dev/devpts/home/tmp/hostfs/cwd mounts, even if they are
    enabled in `singularity.conf`.

## Changed defaults / behaviours

  - When actions (run/shell/exec...) are used without `--fakeroot` the
    umask from the calling environment will be propagated into the
    container, so that files are created with expected permissions.
    Use the new `--no-umask` flag to return to the previous behaviour
    of setting a default 0022 umask.
  - Container metadata, environment, scripts are recorded in a
    descriptor in builds to SIF files, and `inspect` will use this if
    present.
  - The `--nv` flag for NVIDIA GPU support will not resolve libraries
    reported by `nvidia-container-cli` via the ld cache. Will instead
    respect absolute paths to libraries reported by the tool, and bind
    all versioned symlinks to them.
  - General re-work of the `remote login` flow, adds prompts and token
    verification before replacing an existing authentication token.
  - The Execution Control List (ECL) now verifies container
    fingerprints using the new global keyring. Previously all users
    would need relevant keys in their own keyring.
  - The SIF layer mediatype for ORAS has been changed to
    `application/vnd.sylabs.sif.layer.v1.sif` reflecting the published
    [opencontainers/artifacts](https://github.com/opencontainers/artifacts/blob/master/artifact-authors.md#defining-layermediatypes)
    value.
  - `SINGULARITY_BIND` has been restored as an environment variable
    set within a running container. It now reflects all user binds
    requested by the `-B/--bind` flag, as well as via
    `SINGULARITY_BIND[PATHS]`.
  - `singularity search` now correctly searches for container images
    matching the host architecture by default. A new `--arch` flag
    allows searching for other architectures. A new results format
    gives more detail about container image results, while users and
    collections are no longer returned.

## Bug Fixes

  - Support larger definition files, environments etc. by passing
    engine configuration in the environment vs. via socket buffer.
  - Ensure `docker-daemon:` and other source operations respect
    `SINGULARITY_TMPDIR` for all temporary files.
  - Support double quoted filenames in the `%files` section of build
    definitions.
  - Correct `cache list` sizes to show KiB with powers of 1024,
    matching `du` etc.
  - Don't fail on `enable fusemount=no` when no fuse mounts are
    needed.
  - Pull OCI images to the correct requested location when the cache
    is disabled.
  - Ensure `Singularity>` prompt is set when container has no
    environment script, or singularity is called through a wrapper
    script.
  - Avoid build failures in `yum/dnf` operations against the 'setup'
    package on `RHEL/CentOS/Fedora` by ensuring staged `/etc/` files
    do not match distro default content.
  - Failed binds to `/etc/hosts` and `/etc/localtime` in a container
    run with `--contain` are no longer fatal errors.
  - Don't initialize the cache for actions where it is not required.
  - Increase embedded shell interpreter timeout, to allow slow-running
    environment scripts to complete.
  - Correct buffer handling for key import to allow import from STDIN. 
  - Reset environment to avoid `LD_LIBRARY_PATH` issues when resolving
    dependencies for the `unsquashfs` sandbox.
  - Fall back to `/sbin/ldconfig` if `ldconfig` on `PATH` fails while
    resolving GPU libraries. Fixes problems on systems using Nix /
    Guix.
  - Address issues caused by error code changes in `unsquashfs`
    version 4.4.
  - Ensure `/dev/kfd` is bound into container for ROCm when `--rocm`
    is used with `--contain`.
  - Tolerate comments on `%files` sections in build definition files.
  - Fix a loop device file descriptor leak.

## Known Issues

  - A change in Linux kernel 5.9 causes `--fakeroot` builds to fail with a
    `/sys/fs/selinux` remount error. This will be addressed in Singularity
    v3.7.1.


# v3.6.4 - [2020-10-13]

## Security related fixes

Singularity 3.6.4 addresses the following security issue.

  - [CVE-2020-15229](https://github.com/hpcng/singularity/security/advisories/GHSA-7gcp-w6ww-2xv9):
    Due to insecure handling of path traversal and the lack of path
    sanitization within unsquashfs (a distribution provided utility
    used by Singularity), it is possible to overwrite/create files on
    the host filesystem during the extraction of a crafted squashfs
    filesystem. Affects unprivileged execution of SIF / SquashFS
    images, and image builds from SIF / SquashFS images.

## Bug Fixes

  - Update scs-library-client to support `library://` backends using an
    3rd party S3 object store that does not strictly conform to v4
    signature spec.


# v3.6.3 - [2020-09-15]

## Security related fixes

Singularity 3.6.3 addresses the following security issues.

  - [CVE-2020-25039](https://github.com/hpcng/singularity/security/advisories/GHSA-w6v2-qchm-grj7):
    When a Singularity action command (run, shell, exec) is run with
    the fakeroot or user namespace option, Singularity will extract a
    container image to a temporary sandbox directory. Due to insecure
    permissions on the temporary directory it is possible for any user
    with access to the system to read the contents of the
    image. Additionally, if the image contains a world-writable file
    or directory, it is possible for a user to inject arbitrary
    content into the running container.

  - [CVE-2020-25040](https://github.com/hpcng/singularity/security/advisories/GHSA-jv9c-w74q-6762):
    When a Singularity command that results in a container build
    operation is executed, it is possible for a user with access to
    the system to read the contents of the image during the
    build. Additionally, if the image contains a world-writable file
    or directory, it is possible for a user to inject arbitrary
    content into the running build, which in certain circumstances may
    enable arbitrary code execution during the build and/or when the
    built container is run.

  ## Change defaults / behaviours

  - The value for maximum number of loop devices in the config file is now used everywhere
    instead of redefining this value

## Bug Fixes

  - Add CAP_MKNOD in capability bounding set of RPC to fix issue with
    cryptsetup when decrypting image from within a docker container.
  - Fix decryption issue when using both IPC and PID namespaces.
  - Fix unsupported builtins panic from shell interpreter and add umask
    support for definition file scripts.
  - Do not load keyring in prepare_linux if ECL not enabled.
  - Ensure sandbox option overrides remote build destination.


# v3.6.2 - [2020-08-25]

## New features / functionalities

  - Add --force option to `singularity delete` for non-interactive
    workflows.

## Change defaults / behaviours

  - Default to current architecture for `singularity delete`.

## Bug Fixes

  - Respect current remote for `singularity delete` command.
  - Allow `rw` as a (noop) bind option.
  - Fix capability handling regression in overlay mount.
  - Fix LD_LIBRARY_PATH environment override regression with
    `--nv/--rocm`.
  - Fix environment variable duplication within singularity engine.
  - Use `-user-xattrs` for unsquashfs to avoid error with rootless
    extraction using unsquashfs 3.4 (Ubuntu 20.04).
  - Correct `--no-home` message for 3.6 CWD behavior.
  - Don't fail if parent of cache dir not accessible.
  - Fix tests for Go 1.15 Ctty handling.
  - Fix additional issues with test images on ARM64. 
  - Fix FUSE e2e tests to use container ssh_config.


# v3.6.1 - [2020-07-21]

## New features / functionalities

  - Support compilation with `FORTIFY_SOURCE=2` and build in `pie`
    mode with `fstack-protector` enabled (#5433).

## Bug Fixes

  - Provide advisory message r.e. need for `upper` and `work` to
    exist in overlay images.
  - Use squashfs mem and processor limits in squashfs gzip check.
  - Ensure build destination path is not an empty string - do
    not overwrite CWD.
  - Don't unset PATH when interpreting legacy /environment files.


# v3.6.0 - [2020-07-14]

## Security related fixes

Singularity 3.6.0 introduces a new signature format for SIF images,
and changes to the signing / verification code to address:

  - [CVE-2020-13845](https://cve.mitre.org/cgi-bin/cvename.cgi?name=2020-13845)
    In Singularity 3.x versions below 3.6.0, issues allow the ECL to
    be bypassed by a malicious user.
  - [CVE-2020-13846](https://cve.mitre.org/cgi-bin/cvename.cgi?name=2020-13846)
    In Singularity 3.5 the `--all / -a` option to `singularity verify`
    returns success even when some objects in a SIF container are not
    signed, or cannot be verified.
  - [CVE-2020-13847](https://cve.mitre.org/cgi-bin/cvename.cgi?name=2020-13847)
    In Singularity 3.x versions below 3.6.0, Singularity's sign and
    verify commands do not sign metadata found in the global header or
    data object descriptors of a SIF file, allowing an attacker to
    cause unexpected behavior. A signed container may verify
    successfully, even when it has been modified in ways that could be
    exploited to cause malicious behavior.

Please see the published security advisories at
https://github.com/hpcng/singularity/security/advisories for full
detail of these security issues.

Note that the new signature format is necessarily incompatible with
Singularity < 3.6.0 - e.g. Singularity 3.5.3 cannot verify containers
signed by 3.6.0.

We thank Tru Huynh for a report that led to the review of, and changes to,
the signature implementation.

## New features / functionalities
  - Singularity now supports the execution of minimal Docker/OCI
    containers that do not contain `/bin/sh`, e.g. `docker://hello-world`.
  - A new cache structure is used that is concurrency safe on a filesystem that
    supports atomic rename. *If you downgrade to Singularity 3.5 or older after
    using 3.6 you will need to run `singularity cache clean`.*
  - A plugin system rework adds new hook points that will allow the
    development of plugins that modify behavior of the runtime. An image driver
    concept is introduced for plugins to support new ways of handling image and
    overlay mounts. *Plugins built for <=3.5 are not compatible with 3.6*.
  - The `--bind` flag can now bind directories from a SIF or ext3 image into a
    container.
  - The `--fusemount` feature to mount filesystems to a container via FUSE
    drivers is now a supported feature (previously an experimental hidden flag).
    This permits users to mount e.g. `sshfs` and `cvmfs` filesystems to the
    container at runtime.
  - A new `-c/--config` flag allows an alternative `singularity.conf` to be
    specified by the `root` user, or all users in an unprivileged installation.
  - A new `--env` flag allows container environment variables to be set via the
    Singularity command line.
  - A new `--env-file` flag allows container environment variables to be set from
    a specified file.
  - A new `--days` flag for `cache clean` allows removal of items older than a
    specified number of days. Replaces the `--name` flag which is not generally
    useful as the cache entries are stored by hash, not a friendly name.
  - A new '--legacy-insecure' flag to `verify` allows verification of SIF signatures
    in the old, insecure format.
  - A new '-l / --logs' flag for `instance list` that shows the paths
    to instance STDERR / STDOUT log files.
  - The `--json` output of `instance list` now include paths to STDERR
    / STDOUT log files.

## Changed defaults / behaviours
  - New signature format (see security fixes above).
  - Environment variables prefixed with `SINGULARITYENV_` always take
    precedence over variables without `SINGULARITYENV_` prefix.
  - The `%post` build section inherits environment variables from the base image.
  - `%files from ...` will now follow symlinks for sources that are directly
    specified, or directly resolved from a glob pattern. It will not follow
    symlinks found through directory traversal. This mirrors Docker multi-stage
    COPY behaviour.
  - Restored the CWD mount behaviour of v2, implying that CWD path is not recreated
    inside container and any symlinks in the CWD path are not resolved anymore to
    determine the destination path inside container.
  - The `%test` build section is executed the same manner as `singularity test image`.
  - `--fusemount` with the `container:` default directive will foreground the FUSE
     process. Use `container-daemon:` for previous behavior.
  - Fixed spacing of `singularity instance list` to be dynamically changing based off of
    input lengths instead of fixed number of spaces to account for long instance names. 

## Deprecated / removed commands
  - Removed `--name` flag for `cache clean`; replaced with `--days`.
  - Deprecate `-a / --all` option to `sign/verify` as new signature
    behavior makes this the default.

## Bug Fixes
  - Don't try to mount `$HOME` when it is `/` (e.g. `nobody` user).
  - Process `%appinstall` sections in order when building from a definition file.
  - Ensure `SINGULARITY_CONTAINER`, `SINGULARITY_ENVIRONMENT` and the custom
    shell prompt are set inside a container.
  - Honor insecure registry settings from `/etc/containers/registries.conf`.
  - Fix `http_proxy` env var handling in `yum` bootstrap builds.
  - Disable log colorization when output location is not a terminal.
  - Check encryption keys are usable before beginning an encrypted build.
  - Allow app names with non-alphanumeric characters.
  - Use the `base` metapackage for arch bootstrap builds - arch no longer has a
    `base` group.
  - Ensure library client messages are logged with `--debug`.
  - Do not mount `$HOME` with `--fakeroot --contain`.
  - Fall back to underlay automatically when using a sandbox on GPFS.
  - Fix Ctrl-Z handling - propagation of signal.


# v3.5.3 - [2020.02.18]

## Changed defaults / behaviours

The following minor behaviour changes have been made in 3.5.3 to allow
correct operation on CRAY CLE6, and correct an issue with multi-stage
image builds that was blocking use by build systems such as Spack:

  - Container action scripts are no longer bound in from `etc/actions.d` on the
    host. They are created dynamically and inserted at container startup.
  - `%files from ...` will no longer follow symlinks when copying between
    stages in a multi stage build, as symlinks should be copied so that they
    resolve identically in later stages. Copying `%files` from the host will
    still maintain previous behavior of following links.

## Bug Fixes

  - Bind additional CUDA 10.2 libs when using the `--nv` option without
    `nvidia-container-cli`.
  - Fix an NVIDIA persistenced socket bind error with `--writable`.
  - Add detection of ceph to allow workarounds that avoid issues with
    sandboxes on ceph filesystems.
  - Ensure setgid is inherited during make install.
  - Ensure the root directory of a build has owner write permissions,
    regardless of the permissions in the bootstrap source.
  - Fix a regression in `%post` and `%test` to honor the `-c` option.
  - Fix an issue running `%post` when a container doesn't have
    `/etc/resolv.conf` or `/etc/hosts` files.
  - Fix an issue with UID detection on RHEL6 when running instances.
  - Fix a logic error when a sandbox image is in an overlay incompatible
    location, and both overlay and underlay are disabled globally.
  - Fix an issue causing user namespace to always be used when `allow-setuid=no`
    was configured in a setuid installation.
  - Always allow key IDs and fingerprints to be specified with or without a `0x`
    prefix when using `singularity keys` 
  - Fix an issue preventing joining an instance started with `--boot`.
  - Provide a useful error message if an invalid library:// path is provided.
  - Bring in multi-part upload client functionality that will address large
    image upload / proxied upload issues with a future update to Sylabs cloud.

In addition, numerous improvements have been made to the test suites, allowing
them to pass cleanly on a range of kernel versions and distributions that are
not covered by the open-source CI runs.


# v3.5.2 - [2019.12.17]

## [Security related fix](https://cve.mitre.org/cgi-bin/cvename.cgi?name=2019-19724)
  - 700 permissions are enforced on `$HOME/.singularity` and `SINGULARITY_CACHEDIR`
  directories (CVE-2019-19724). Many thanks to Stuart Barkley for reporting this issue.

## Bug Fixes

  - Fixes an issue preventing use of `.docker/config` for docker registry
    authentication.
  - Fixes the `run-help` command in the unprivileged workflow.
  - Fixes a regression in the `inspect` command to support older image formats.
  - Adds a workaround for an EL6 kernel bug regarding shared bind mounts.
  - Fixes caching of http(s) sources with conflicting filenames.
  - Fixes a fakeroot sandbox build error on certain filesystems, e.g. lustre, GPFS.
  - Fixes a fakeroot build failure to a sandbox in $HOME.
  - Fixes a fakeroot build failure from a bad def file section script location.
  - Fixes container execution errors when CWD is a symlink.
  - Provides a useful warning r.e. possible fakeroot build issues when seccomp
    support is not available.
  - Fixes an issue where the `--disable-cache` option was not being honored.

 - Deprecated `--groupid` flag for `sign` and `verify`; replaced with `--group-id`.
 - Removed useless flag `--url` for `sign`.

# v3.5.1 - [2019.12.05]

## New features / functionalities

A single feature has been added in the bugfix release, with specific
functionality:

  - A new option `allow container encrypted` can be set to `no` in
      `singularity.conf` to prevent execution of encrypted containers.

## Bug Fixes

This point release addresses the following issues:

  - Fixes a disk space leak when building from docker-archive.
  - Makes container process SIGABRT return the expected code.
  - Fixes the `inspect` command in unprivileged workflow.
  - Sets an appropriate default umask during build stages, to avoid issues with
      very restrictive user umasks.
  - Fixes an issue with build script content being consumed from STDIN.
  - Corrects the behaviour of underlay with non-empty / symlinked CWD and absolute
    symlink binds targets.
  - Fixes execution of containers when binding BTRFS filesystems.
  - Fixes build / check failures for MIPS & PPC64.
  - Ensures file ownership maintained when building image from sandbox.
  - Fixes a squashfs mount error on kernel 5.4.0 and above.
  - Fixes an underlay fallback problem, which prevented use of sandboxes on
    lustre filesystems.

# v3.5.0 - [2019.11.13]

## New features / functionalities

  - New support for AMD GPUs via `--rocm` option added to bind ROCm devices and
    libraries into containers.
  - Plugins can now modify Singularity behaviour with two mutators: CLI and
    Runtime.
  - Introduced the `config global` command to edit `singularity.conf` settings
    from the CLI.
  - Introduced the `config fakeroot` command to setup `subuid` and `subgid`
    mappings for `--fakeroot` from the Singularity CLI.
      
## Changed defaults / behaviours

  - Go 1.13 adopted.
  - Vendored modules removed from the Git tree, will be included in release tarballs.
  - Singularity will now fail with an error if a requested bind mount cannot be
      made.
    - This is beneficial to fail fast in workflows where a task may fail a long
         way downstream if a bind mount is unavailable.
    - Any unavailable bind mount sources must be removed from
        `singularity.conf`.
  - Docker/OCI image extraction now faithfully respects layer
    permissions.
    - This may lead to sandboxes that cannot be removed without
    modifying permissions.
    - `--fix-perms` option added to preserve old behaviour when
    building sandboxes.
    - Discussion issue for this change at: https://github.com/sylabs/singularity/issues/4671
  - `Singularity>` prompt is always set when entering shell in a container.
  - The current `umask` will be honored when building a SIF file.
  - `instance exec` processes acquire cgroups set on `instance start`
  - `--fakeroot` supports uid/subgid ranges >65536
  - `singularity version` now reports semver compliant version
      information.

## Deprecated / removed commands

  - Deprecated `--id` flag for `sign` and `verify`; replaced with `--sif-id`.

# v3.4.2 - [2019.10.08]

  - This point release addresses the following issues:
    - Sets workable permissions on OCI -> sandbox rootless builds
    - Fallback correctly to user namespace for non setuid installation
    - Correctly handle the starter-suid binary for non-root installs
    - Creates CACHEDIR if it doesn't exist
    - Set apex loglevel for umoci to match singularity loglevel

# v3.4.1 - [2019.09.17]

  - This point release addresses the following issues:
    - Fixes an issue where a PID namespace was always being used
    - Fixes compilation on non 64-bit architectures
    - Allows fakeroot builds for zypper, pacstrap, and debootstrap
    - Correctly detects seccomp on OpenSUSE
    - Honors GO_MODFLAGS properly in the mconfig generated makefile
    - Passes the Mac hostname to the VM in MacOS Singularity builds
    - Handles temporary EAGAIN failures when setting up loop devices on recent kernels
    - Fixes excessive memory usage in singularity push

# v3.4.0 - [2019.08.30]

## New features / functionalities
  
  - New support for building and running encrypted containers with RSA keys and passphrases
    - `--pem-path` option added to the `build` and action commands for RSA based encrypted containers
    - `--passphrase` option added to `build` and action commands for passphrase based encrypted containers
    - `SINGULARITY_ENCRYPTION_PEM_PATH` and `SINGULARITY_ENCRYPTION_PASSPHRASE` environment variables added to serve same functions as above
    - `--encrypt` option added to `build` command to build an encrypted container when environment variables contain a secret
  - New `--disable-cache` flag prevents caching of downloaded containers
  - Added support for multi-line variables in singularity def-files
  - Added support for 'indexed' def-file variables (like arrays)
  - Added support for SUSE SLE Products
  - Added the def-file variables:
      product, user, regcode, productpgp, registerurl, modules,	otherurl (indexed)
  - Support multiple-architecture tags in the SCS library
  - Added a `--dry-run` flag to `cache clean`
  - Added a `SINGULARITY_SYPGPDIR` environment variable to specify the location of PGP key data
  - Added a `--nonet` option to the action commands to disable networking when running with the `--vm` option
  - Added a `--long-list` flag to the `key search` command to preserve 
  - Added experimental, hidden `--fusemount` flag to pass a command to mount a libfuse3 based file system within the container

## Changed defaults / behaviors

  - Runtime now properly honors `SINGULARITY_DISABLE_CACHE` environment variable
  - `remote add` command now automatically attempts to login and a `--no-login` flag is added to disable this behavior
  - Using the `pull` command to download an unsigned container no longer produces an error code
  - `cache clean` command now prompts user before cleaning when run without `--force` option and is more verbose
  - Shortened the default output of the `key search` command

## Deprecated / removed commands

  - The `--allow-unsigned` flag to `pull` has been deprecated and will be removed in the future

# v3.3.0 - [2019.06.17]

## Changed defaults / behaviors

  - Remote login and status commands will now use the default remote if a remote name is not supplied
  - Added Singularity hub (`shub`) cache support when using the `pull` command
  - Clean cache in a safer way by only deleting the cache subdirectories
  - Improvements to the `cache clean` command 

## New features / functionalities

  - new `oras` URI for pushing and pulling SIF files to and from supported OCI registries
  - added the `--fakeroot` option to `build`, `exec`, `run`, `shell`, `test`, and `instance start` commands to run container in a new user namespace as uid 0
  - added the `fakeroot` network type for use with the `--network` option
  - `sif` command to allow for the inspection and manipulation of SIF files with the following subcommands
    - `add`      Add a data object to a SIF file
    - `del`      Delete a specified object descriptor and data from SIF file
    - `dump`     Extract and output data objects from SIF files
    - `header`   Display SIF global headers
    - `info`     Display detailed information of object descriptors
    - `list`     List object descriptors from SIF files
    - `new`      Create a new empty SIF image file
    - `setprim`  Set primary system partition

# v3.2.1 - [2019.05.28]

  - This point release fixes the following bugs:
    - Allows users to join instances with non-suid workflow
    - Removes false warning when seccomp is disabled on the host
    - Fixes an issue in the terminal when piping output to commands
    - Binds NVIDIA persistenced socket when `--nv` is invoked

# v3.2.0 - [2019.05.14]

## [Security related fix](https://cve.mitre.org/cgi-bin/cvename.cgi?name=2019-11328)
  - Instance files are now stored in user's home directory for privacy and many checks have been added to ensure that a user can't manipulate files to change `starter-suid` behavior when instances are joined (many thanks to Matthias Gerstner from the SUSE security team for finding and securely reporting this vulnerability) 

## New features / functionalities
  - Introduced a new basic framework for creating and managing plugins
  - Added the ability to create containers through multi-stage builds
    - Definitions now require `Bootstrap` be the first parameter of header
  - Created the concept of a Sylabs Cloud "remote" endpoint and added the ability for users and admins to set them through CLI and conf files 
  - Added caching for images from Singularity Hub
  - Made it possible to compile Singularity outside of `$GOPATH`
  - Added a json partition to SIF files for OCI configuration when building from an OCI source
  - Full integration with Singularity desktop for MacOS code base

## New Commands
 - Introduced the `plugin` command group for creating and managing plugins
    - `compile`   Compile a singularity plugin
    - `disable`   disable an installed singularity plugin
    - `enable`    Enable an installed singularity plugin
    - `inspect`   Inspect a singularity plugin (either an installed one or an image)
    - `install`   Install a singularity plugin
    - `list`      List installed singularity plugins
    - `uninstall` Uninstall removes the named plugin from the system

  - Introduced the `remote` command group to support management of Singularity endpoints:
    - `add`       Create a new Sylabs Cloud remote endpoint
    - `list`      List all remote endpoints that are configured
    - `login`     Log into a remote endpoint using an authentication token
    - `remove`    Remove an existing Sylabs Cloud remote endpoint
    - `status`    Check the status of the services at an endpoint
    - `use`       Set a remote endpoint to be used by default

  - Added to the `key` command group to improve PGP key management:
    - ` export`   Export a public or private key into a specific file
    - ` import`   Import a local key into the local keyring
    - ` remove`   Remove a local public key

  - Added the `Stage: <name>` keyword to the definition file header and the `from <stage name>` option/argument pair to the `%files` section to support multistage builds

## Deprecated / removed commands
  - The `--token/-t` option has been deprecated in favor of the `singularity remote` command group

## Changed defaults / behaviors
  - Ask to confirm password on a newly generated PGP key
  - Prompt to push a key to the KeyStore when generated
  - Refuse to push an unsigned container unless overridden with `--allow-unauthenticated/-U` option
  - Warn and prompt when pulling an unsigned container without the `--allow-unauthenticated/-U` option
  - `Bootstrap` must now be the first field of every header because of parser requirements for multi-stage builds

# v3.1.1 - [2019.04.02]

## New Commands
  - New hidden `buildcfg` command to display compile-time parameters 
  - Added support for `LDFLAGS`, `CFLAGS`, `CGO_` variables in build system
  - Added `--nocolor` flag to Singularity client to disable color in logging

## Removed Commands
  - `singularity capability <add/drop> --desc` has been removed
  - `singularity capability list <--all/--group/--user>` flags have all been removed 

## New features / functionalities
  - The `--builder` flag to the `build` command implicitly sets `--remote`
  - Repeated binds no longer cause Singularity to exit and fail, just warn instead
  - Corrected typos and improved docstrings throughout
  - Removed warning when CWD does not exist on the host system
  - Added support to spec file for RPM building on SLES 11

# v3.1.0 - [2019.02.22]

## New Commands
  - Introduced the `oci` command group to support a new OCI compliant variant of the Singularity runtime:
    - `attach` Attach console to a running container process
    - `create` Create a container from a bundle directory
    - `delete` Delete container
    - `exec`   Execute a command within container
    - `kill`   Kill a container
    - `mount`  Mount create an OCI bundle from SIF image
    - `pause`  Suspends all processes inside the container
    - `resume` Resumes all processes previously paused inside the container
    - `run`    Create/start/attach/delete a container from a bundle directory
    - `start`  Start container process
    - `state`  Query state of a container
    - `umount` Umount delete bundle
    - `update` Update container cgroups resources
  - Added `cache` command group to inspect and manage cached files
    - `clean` Clean your local Singularity cache
    - `list`  List your local Singularity cache

## New features / functionalities
  - Can now build CLI on darwin for limited functionality on Mac
  - Added the `scratch` bootstrap agent to build from anything
  - Reintroduced support for zypper bootstrap agent
  - Added the ability to overwrite a new `singularity.conf` when building from RPM if desired
  - Fixed several regressions and omissions in [SCIF](https://sci-f.github.io/) support
  - Added caching for containers pulled/built from the [Container Library](https://cloud.sylabs.io/library)
  - Changed `keys` command group to `key` (retained hidden `keys` command for backward compatibility)  
  - Created an `RPMPREFIX` variable to allow RPMs to be installed in custom locations
  - Greatly expanded CI unit and end-to-end testing

# v3.0.3 - [2019.01.21]
  
  - Bind paths in `singularity.conf` are properly parsed and applied at runtime
  - Singularity runtime will properly fail if `singularity.conf` file is not owned by the root user
  - Several improvements to RPM packaging including using golang from epel, improved support for Fedora, and avoiding overwriting conf file on new RPM install
  - Unprivileged `--contain` option now properly mounts `devpts` on older kernels
  - Uppercase proxy environment variables are now rightly respected
  - Add http/https protocols for singularity run/pull commands
  - Update to SIF 1.0.2
  - Add _noPrompt_ parameter to `pkg/signing/Verify` function to enable silent verification

# v3.0.2 - [2019.01.04]

  - Added the `--docker-login` flag to enable interactive authentication with docker registries
  - Added support for pulling directly from HTTP and HTTPS
  - Made minor improvements to RPM packaging and added basic support for alpine packaging
  - The `$SINGULARITY_NOHTTPS`,`$SINGULARITY_TMPDIR`, and `$SINGULARITY_DOCKER_USERNAME`/`$SINGULARITY_DOCKER_PASSWORD` environment variables are now correctly respected
  - Pulling from a private shub registry now works as expected
  - Running a container with `--network="none"` no longer incorrectly fails with an error message
  - Commands now correctly return 1 when incorrectly executed without arguments
  - Progress bars no longer incorrectly display when running with `--quiet` or `--silent`
  - Contents of `91-environment.sh` file are now displayed if appropriate when running `inspect --environment`

# v3.0.1 - [2018.10.31]

  - Improved RPM packaging procedure via makeit
  - Enhanced general stability of runtime

# v3.0.0 - [2018.10.08]

  - Singularity is now written primarily in Go to bring better integration with the existing container ecosystem
  - Added support for new URIs (`build` & `run/exec/shell/start`):
    - `library://` - Supports the [Sylabs.io Cloud Library](https://cloud.sylabs.io/library)
    - `docker-daemon:` - Supports images managed by the locally running docker daemon
    - `docker-archive:` - Supports archived docker images
    - `oci:` - Supports oci images
    - `oci-archive:` - Supports archived oci images
  - Handling of `docker` & `oci` URIs/images now utilizes [containers/image](https://github.com/containers/image) to parse and convert those image types in a supported way
  - Replaced `singularity instance.*` command group with `singularity instance *`
  - The command `singularity help` now only provides help regarding the usage of the `singularity` command. To display an image's `help` message, use `singularity run-help <image path>` instead
 
## Removed Deprecated Commands
  - Removed deprecated `singularity image.*` command group
  - Removed deprecated `singularity create` command
  - Removed deprecated `singularity bootstrap` command
  - Removed deprecated `singularity mount` command
  - Removed deprecated `singularity check` command

## New Commands
  - Added `singularity run-help <image path>` command to output an image's `help` message
  - Added `singularity sign <image path>` command to allow a user to cryptographically sign a SIF image
  - Added `singularity verify <image path>` command to allow a user to verify a SIF image's cryptographic signatures
  - Added `singularity keys` command to allow the management of `OpenPGP` key stores
  - Added `singularity capability` command to allow fine grained control over the capabilities of running containers
  - Added `singularity push` command to push images to the [Sylabs.io Cloud Library](https://cloud.sylabs.io/library)

## Changed Commands

### Action Command Group (`run/shell/exec/instance start`)
  - Added flags:
    - `--add-caps <string>`: Run the contained process with the specified capability set (requires root)
    - `--allow-setuid`: Allows setuid binaries to be mounted into the container (requires root)
    - `--apply-cgroups <path>`: Apply cgroups configuration from file to contained processes (requires root)
    - `--dns <string>`: Adds the comma separated list of DNS servers to the containers `resolv.conf` file
    - `--drop-caps <string>`: Drop the specified capabilities from the container (requires root)
    - `--fakeroot`: Run the container in a user namespace as `uid=0`. Requires a recent kernel to function properly
    - `--hostname <string>`: Set the hostname of the container
    - `--keep-privs`: Keep root user privilege inside the container (requires root)
    - `--network <string>`: Specify a list of comma separated network types ([CNI Plugins](https://github.com/containernetworking/cni)) to be present inside the container, each with its own dedicated interface in the container
    - `--network-args <string>`: Specify arguments to pass to CNI network plugins (set by `--network`)
    - `--no-privs`: Drop all privileges from root user inside the container (requires root)
    - `--security <string>`: Configure security features such as SELinux, Apparmor, Seccomp...
    - `--writable-tmpfs`: Run container with a `tmpfs` overlay
  - The command `singularity instance start` now supports the `--boot` flag to boot the container via `/sbin/init`
  - Changes to image mounting behavior:
    - All image formats are mounted as read only by default
    - `--writable` only works on images which can be mounted in read/write [applicable to: `sandbox` and legacy `ext3` images]
    - `--writable-tmpfs` runs the container with a writable `tmpfs`-based overlay [applicable to: all image formats]
    - `--overlay <string>` now specifies a list of `ext3`/`sandbox` images which are set as the containers overlay [applicable to: all image formats] 

### Build Command:
  - All images are now built as [Singularity Image Format (SIF)](https://www.sylabs.io/2018/03/sif-containing-your-containers/) images by default
  - When building to a path that already exists, `singularity build` will now prompt the user if they wish to overwrite the file existing at the specified location
  - The `-w|--writable` flag has been removed
  - The `-F|--force` flag now overrides the interactive prompt and will always attempt to overwrite the file existing at the specified location
  - The `-u|--update` flag has been added to support the workflow of running a definition file on top of an existing container [implies `--sandbox`, only supports `sandbox` image types]
  - The `singularity build` command now supports the following flags for integration with the [Sylabs.io Cloud Library](https://cloud.sylabs.io/library):
    - `-r|--remote`: Build the image remotely on the Sylabs Remote Builder (currently unavailable)
    - `-d|--detached`: Detach from the `stdout` of the remote build [requires `--remote`]
    - `--builder <string>`: Specifies the URL of the remote builder to access
    - `--library <string>`: Specifies the URL of the [Sylabs.io Cloud Library](https://cloud.sylabs.io/library) to push the built image to when the build command destination is in the form `library://<reference>`
  - The `bootstrap` keyword in the definition file now supports the following values:
    - `library`
    - `docker-daemon`
    - `docker-archive`
    - `oci`
    - `oci-archive`
  - The `from` keyword in the definition file now correctly parses a `docker` URI which includes the `registry` and/or `namespace` components
  - The `registry` and `namespace` keywords in the definition file are no longer supported. Instead, those values may all go into the `from` keyword
  - Building from a tar archive of a `sandbox` no longer works
