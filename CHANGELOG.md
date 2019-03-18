_With the release of `v3.0.0`, we're introducing a new changelog format in an attempt to consolidate the information presented in the changelog. The new changelog is reduced in scope to only documenting functionality changes from version to version. This ensures that the changelog is as useful as it can be. Changes which should be documented include:_
  
  - _Renamed commands_
  - _Deprecated / removed commands_
  - _Changed defaults / behaviors_
  - _Migration guidance_
  - _New features / functionalities_


_The old changelog can be found in the `release-2.6` branch_

# Changes Since v3.1.0

## New Commands
- Introduced the `remote` command group to support management of SCS endpoints:
    - `add`     Create a new SCS remote endpoint
    - `list`    List all remote endpoints that are configured
    - `login`   Log into a remote endpoint using an authentication token
    - `remove`  Remove an existing SCS remote endpoint
    - `status`  Check the status of the services at an endpoint
    - `use`     Set a remote endpoint to be used by default

## New features / functionalities
  - Fixed usage docstrings for OCI - was missing "oci" in command examples.
  - Added --nocolor flag to Singularity client to disable color in logging
  - Repeated binds does not exit and fail, just issues warning

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
