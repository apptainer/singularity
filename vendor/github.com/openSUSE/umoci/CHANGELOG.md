# Change Log
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

## [0.4.2] - 2018-09-11
## Added
- umoci now has an exposed Go API. At the moment it's unclear whether it will
  be changed significantly, but at the least now users can use
  umoci-as-a-library in a fairly sane way. openSUSE/umoci#245
- Added `umoci unpack --keep-dirlinks` (in the same vein as rsync's flag with
  the same name) which allows layers that contain entries which have a symlink
  as a path component. openSUSE/umoci#246
- `umoci insert` now supports whiteouts in two significant ways. You can use
  `--whiteout` to "insert" a deletion of a given path, while you can use
  `--opaque` to replace a directory by adding an opaque whiteout (the default
  behaviour causes the old and new directories to be merged).
  openSUSE/umoci#257

## Fixed
- Docker has changed how they handle whiteouts for non-existent files. The
  specification is loose on this (and in umoci we've always been liberal with
  whiteout generation -- to avoid cases where someone was confused we didn't
  have a whiteout for every entry). But now that they have deviated from the
  spec, in the interest of playing nice, we can just follow their new
  restriction (even though it is not supported by the spec). This also makes
  our layers *slightly* smaller. openSUSE/umoci#254
- `umoci unpack` now no longer erases `system.nfs4_acl` and also has some more
  sophisticated handling of forbidden xattrs. openSUSE/umoci#252
  openSUSE/umoci#248
- `umoci unpack` now appears to work correctly on SELinux-enabled systems
  (previously we had various issues where `umoci` wouldn't like it when it was
  trying to ensure the filesystem was reproducibly generated and SELinux xattrs
  would act strangely). To fix this, now `umoci unpack` will only cause errors
  if it has been asked to change a forbidden xattr to a value different than
  it's current on-disk value. openSUSE/umoci#235 openSUSE/umoci#259

## [0.4.1] - 2018-08-16
### Added
- The number of possible tags that are now valid with `umoci` subcommands has
  increased significantly due to an expansion in the specification of the
  format of the `ref.name` annotation. To quote the specification, the
  following is the EBNF of valid `refname` values. openSUSE/umoci#234
  ```
  refname   ::= component ("/" component)*
  component ::= alphanum (separator alphanum)*
  alphanum  ::= [A-Za-z0-9]+
  separator ::= [-._:@+] | "--"
  ```
- A new `umoci insert` subcommand which adds a given file to a path inside the
  container. openSUSE/umoci#237
- A new `umoci raw unpack` subcommand in order to allow users to unpack images
  without needing a configuration or any of the manifest generation.
  openSUSE/umoci#239
- `umoci` how has a logo. Thanks to [Max Bailey][maxbailey] for contributing
  this to the project. openSUSE/umoci#165 openSUSE/umoci#249

### Fixed
- `umoci unpack` now handles out-of-order regular whiteouts correctly (though
  this ordering is not recommended by the spec -- nor is it required). This is
  an extension of openSUSE/umoci#229 that was missed during review.
  openSUSE/umoci#232
- `umoci unpack` and `umoci repack` now make use of a far more optimised `gzip`
  compression library. In some benchmarks this has resulted in `umoci repack`
  speedups of up to 3x (though of course, you should do your own benchmarks).
  `umoci unpack` unfortunately doesn't have as significant of a performance
  improvement, due to the nature of `gzip` decompression (in future we may
  switch to `zlib` wrappers). openSUSE/umoci#225 openSUSE/umoci#233

[maxbailey]: http://www.maxbailey.me/

## [0.4.0] - 2018-03-10
### Added
- `umoci repack` now supports `--refresh-bundle` which will update the
  OCI bundle's metadata (mtree and umoci-specific manifests) after packing the
  image tag. This means that the bundle can be used as a base layer for
  future diffs without needing to unpack the image again. openSUSE/umoci#196
- Added a website, and reworked the documentation to be better structured. You
  can visit the website at [`umo.ci`][umo.ci]. openSUSE/umoci#188
- Added support for the `user.rootlesscontainers` specification, which allows
  for persistent on-disk emulation of `chown(2)` inside rootless containers.
  This implementation is interoperable with [@AkihiroSuda's `PRoot`
  fork][as-proot-fork] (though we do not test its interoperability at the
  moment) as both tools use [the same protobuf
  specification][rootlesscontainers-proto]. openSUSE/umoci#227
- `umoci unpack` now has support for opaque whiteouts (whiteouts which remove
  all children of a directory in the lower layer), though `umoci repack` does
  not currently have support for generating them. While this is technically a
  spec requirement, through testing we've never encountered an actual user of
  these whiteouts. openSUSE/umoci#224 openSUSE/umoci#229
- `umoci unpack` will now use some rootless tricks inside user namespaces for
  operations that are known to fail (such as `mknod(2)`) while other operations
  will be carried out as normal (such as `lchown(2)`). It should be noted that
  the `/proc/self/uid_map` checking we do can be tricked into not detecting
  user namespaces, but you would need to be trying to break it on purpose.
  openSUSE/umoci#171 openSUSE/umoci#230

### Fixed
- Fix a bug in our "parent directory restore" code, which is responsible for
  ensuring that the mtime and other similar properties of a directory are not
  modified by extraction inside said directory. The bug would manifest as
  xattrs not being restored properly in certain edge-cases (which we
  incidentally hit in a test-case). openSUSE/umoci#161 openSUSE/umoci#162
- `umoci unpack` will now "clean up" the bundle generated if an error occurs
  during unpacking. Previously this didn't happen, which made cleaning up the
  responsibility of the caller (which was quite difficult if you were
  unprivileged). This is a breaking change, but is in the error path so it's
  not critical. openSUSE/umoci#174 openSUSE/umoci#187
- `umoci gc` now will no longer remove unknown files and directories that
  aren't `flock(2)`ed, thus ensuring that any possible OCI image-spec
  extensions or other users of an image being operated on will no longer
  break.  openSUSE/umoci#198
- `umoci unpack --rootless` will now correctly handle regular file unpacking
  when overwriting a file that `umoci` doesn't have write access to. In
  addition, the semantics of pre-existing hardlinks to a clobbered file are
  clarified (the hard-links will not refer to the new layer's inode).
  openSUSE/umoci#222 openSUSE/umoci#223

[as-proot-fork]: https://github.com/AkihiroSuda/runrootless
[rootlesscontainers-proto]: https://rootlesscontaine.rs/proto/rootlesscontainers.proto
[umo.ci]: https://umo.ci/

## [0.3.1] - 2017-10-04
### Fixed
- Fix several minor bugs in `hack/release.sh` that caused the release artefacts
  to not match the intended style, as well as making it more generic so other
  projects can use it. openSUSE/umoci#155 openSUSE/umoci#163
- A recent configuration issue caused `go vet` and `go lint` to not run as part
  of our CI jobs. This means that some of the information submitted as part of
  [CII best practices badging][cii] was not accurate. This has been corrected,
  and after review we concluded that only stylistic issues were discovered by
  static analysis. openSUSE/umoci#158
- 32-bit unit test builds were broken in a refactor in [0.3.0]. This has been
  fixed, and we've added tests to our CI to ensure that something like this
  won't go unnoticed in the future. openSUSE/umoci#157
- `umoci unpack` would not correctly preserve set{uid,gid} bits. While this
  would not cause issues when building an image (as we only create a manifest
  of the final extracted rootfs), it would cause issues for other users of
  `umoci`. openSUSE/umoci#166 openSUSE/umoci#169
- Updated to [v0.4.1 of `go-mtree`][gomtree-v0.4.1], which fixes several minor
  bugs with manifest generation. openSUSE/umoci#176
- `umoci unpack` would not handle "weird" tar archive layers previously (it
  would error out with DiffID errors). While this wouldn't cause issues for
  layers generated using Go's `archive/tar` implementation, it would cause
  issues for GNU gzip and other such tools. openSUSE/umoci#178
  openSUSE/umoci#179

### Changed
- `umoci unpack`'s mapping options (`--uid-map` and `--gid-map`) have had an
  interface change, to better match the [`user_namespaces(7)`][user_namespaces]
  interfaces. Note that this is a **breaking change**, but the workaround is to
  switch to the trivially different (but now more consistent) format.
  openSUSE/umoci#167

### Security
- `umoci unpack` used to create the bundle and rootfs with world
  read-and-execute permissions by default. This could potentially result in an
  unsafe rootfs (containing dangerous setuid binaries for instance) being
  accessible by an unprivileged user. This has been fixed by always setting the
  mode of the bundle to `0700`, which requires a user to explicitly work around
  this basic protection. This scenario was documented in our security
  documentation previously, but has now been fixed. openSUSE/umoci#181
  openSUSE/umoci#182

[cii]: https://bestpractices.coreinfrastructure.org/projects/1084
[gomtree-v0.4.1]: https://github.com/vbatts/go-mtree/releases/tag/v0.4.1
[user_namespaces]: http://man7.org/linux/man-pages/man7/user_namespaces.7.html

## [0.3.0] - 2017-07-20
### Added
- `umoci` now passes all of the requirements for the [CII best practices bading
  program][cii]. openSUSE/umoci#134
- `umoci` also now has more extensive architecture, quick-start and roadmap
  documentation. openSUSE/umoci#134
- `umoci` now supports [`1.0.0` of the OCI image
  specification][ispec-v1.0.0] and [`1.0.0` of the OCI runtime
  specification][rspec-v1.0.0], which are the first milestone release. Note
  that there are still some remaining UX issues with `--image` and other parts
  of `umoci` which may be subject to change in future versions. In particular,
  this update of the specification now means that images may have ambiguous
  tags. `umoci` will warn you if an operation may have an ambiguous result, but
  we plan to improve this functionality far more in the future.
  openSUSE/umoci#133 openSUSE/umoci#142
- `umoci` also now supports more complicated descriptor walk structures, and
  also handles mutation of such structures more sanely. At the moment, this
  functionality has not been used "in the wild" and `umoci` doesn't have the UX
  to create such structures (yet) but these will be implemented in future
  versions. openSUSE/umoci#145
- `umoci repack` now supports `--mask-path` to ignore changes in the rootfs
  that are in a child of at least one of the provided masks when generating new
  layers. openSUSE/umoci#127

### Changed
- Error messages from `github.com/openSUSE/umoci/oci/cas/drivers/dir` actually
  make sense now. openSUSE/umoci#121
- `umoci unpack` now generates `config.json` blobs according to the [still
  proposed][ispec-pr492] OCI image specification conversion document.
  openSUSE/umoci#120
- `umoci repack` also now automatically adding `Config.Volumes` from the image
  configuration to the set of masked paths.  This matches recently added
  [recommendations by the spec][ispec-pr694], but is a backwards-incompatible
  change because the new default is that `Config.Volumes` **will** be masked.
  If you wish to retain the old semantics, use `--no-mask-volumes` (though make
  sure to be aware of the reasoning behind `Config.Volume` masking).
  openSUSE/umoci#127
- `umoci` now uses [`SecureJoin`][securejoin] rather than a patched version of
  `FollowSymlinkInScope`. The two implementations are roughly equivalent, but
  `SecureJoin` has a nicer API and is maintained as a separate project.
- Switched to using `golang.org/x/sys/unix` over `syscall` where possible,
  which makes the codebase significantly cleaner. openSUSE/umoci#141

[cii]: https://bestpractices.coreinfrastructure.org/projects/1084
[rspec-v1.0.0]: https://github.com/opencontainers/runtime-spec/releases/tag/v1.0.0
[ispec-v1.0.0]: https://github.com/opencontainers/image-spec/releases/tag/v1.0.0
[ispec-pr492]: https://github.com/opencontainers/image-spec/pull/492
[ispec-pr694]: https://github.com/opencontainers/image-spec/pull/694
[securejoin]: https://github.com/cyphar/filepath-securejoin

## [0.2.1] - 2017-04-12
### Added
- `hack/release.sh` automates the process of generating all of the published
  artefacts for releases. The new script also generates signed source code
  archives. openSUSE/umoci#116

### Changed
- `umoci` now outputs configurations that are compliant with [`v1.0.0-rc5` of
  the OCI runtime-spec][rspec-v1.0.0-rc5]. This means that now you can use runc
  v1.0.0-rc3 with `umoci` (and rootless containers should work out of the box
  if you use a development build of runc). openSUSE/umoci#114
- `umoci unpack` no longer adds a dummy linux.seccomp entry, and instead just
  sets it to null. openSUSE/umoci#114

[rspec-v1.0.0-rc5]: https://github.com/opencontainers/runtime-spec/releases/tag/v1.0.0-rc5

## [0.2.0] - 2017-04-11
### Added
- `umoci` now has some automated scripts for generated RPMs that are used in
  openSUSE to automatically submit packages to OBS. openSUSE/umoci#101
- `--clear=config.{cmd,entrypoint}` is now supported. While this interface is a
  bit weird (`cmd` and `entrypoint` aren't treated atomically) this makes the
  UX more consistent while we come up with a better `cmd` and `entrypoint` UX.
  openSUSE/umoci#107
- New subcommand: `umoci raw runtime-config`. It generates the runtime-spec
  config.json for a particular image without also unpacking the root
  filesystem, allowing for users of `umoci` that are regularly parsing
  `config.json` without caring about the root filesystem to be more efficient.
  However, a downside of this approach is that some image-spec fields
  (`Config.User`) require a root filesystem in order to make sense, which is
  why this command is hidden under the `umoci-raw(1)` subcommand (to make sure
  only users that understand what they're doing use it). openSUSE/umoci#110

### Changed
- `umoci`'s `oci/cas` and `oci/config` libraries have been massively refactored
  and rewritten, to allow for third-parties to use the OCI libraries. The plan
  is for these to eventually become part of an OCI project. openSUSE/umoci#90
- The `oci/cas` interface has been modifed to switch from `*ispec.Descriptor`
  to `ispec.Descriptor`. This is a breaking, but fairly insignificant, change.
  openSUSE/umoci#89

### Fixed
- `umoci` now uses an updated version of `go-mtree`, which has a complete
  rewrite of `Vis` and `Unvis`. The rewrite ensures that unicode handling is
  handled in a far more consistent and sane way. openSUSE/umoci#88
- `umoci` used to set `process.user.additionalGids` to the "normal value" when
  unpacking an image in rootless mode, causing issues when trying to actually
  run said bundle with runC. openSUSE/umoci#109

## [0.1.0] - 2017-02-11
### Added
- `CHANGELOG.md` has now been added. openSUSE/umoci#76

### Changed
- `umoci` now supports `v1.0.0-rc4` images, which has made fairly minimal
  changes to the schema (mainly related to `mediaType`s). While this change
  **is** backwards compatible (several fields were removed from the schema, but
  the specification allows for "additional fields"), tools using older versions
  of the specification may fail to operate on newer OCI images. There was no UX
  change associated with this update.

### Fixed
- `umoci tag` would fail to clobber existing tags, which was in contrast to how
  the rest of the tag clobbering commands operated. This has been fixed and is
  now consistent with the other commands. openSUSE/umoci#78
- `umoci repack` now can correctly handle unicode-encoded filenames, allowing
  the creation of containers that have oddly named files. This required fixes
  to go-mtree (where the issue was). openSUSE/umoci#80

## [0.0.0] - 2017-02-07
### Added
- Unit tests are massively expanded, as well as the integration tests.
  openSUSE/umoci#68 openSUSE/umoci#69
- Full coverage profiles (unit+integration) are generated to get all
  information about how much code is tested. openSUSE/umoci#68
  openSUSE/umoci#69

### Fixed
- Static compilation now works properly. openSUSE/umoci#64
- 32-bit architecture builds are fixed. openSUSE/umoci#70

### Changed
- Unit tests can now be run inside `%check` of an `rpmbuild` script, allowing
  for proper testing. openSUSE/umoci#65.
- The logging output has been cleaned up to be much nicer for end-users to
  read. openSUSE/umoci#73
- Project has been moved to an openSUSE project. openSUSE/umoci#75

## [0.0.0-rc3] - 2016-12-19
### Added
- `unpack`, `repack`: `xattr` support which also handles `security.selinux.*`
  difficulties. openSUSE/umoci#49 openSUSE/umoci#52
- `config`, `unpack`: Ensure that environment variables are not duplicated in
  the extracted or stored configurations. openSUSE/umoci#30
- Add support for read-only CAS operations for read-only filesystems.
  openSUSE/umoci#47
- Add some helpful output about `--rootless` if `umoci` fails with `EPERM`.
- Enable stack traces with errors if the `--debug` flag was given to `umoci`.
  This requires a patch to `pkg/errors`.

### Changed
- `gc`: Garbage collection now also garbage collects temporary directories.
  openSUSE/umoci#17
- Clean-ups to vendoring of `go-mtree` so that it's much more
  upstream-friendly.

## [0.0.0-rc2] - 2016-12-12
### Added
- `unpack`, `repack`: Support for rootless unpacking and repacking.
  openSUSE/umoci#26
- `unpack`, `repack`: UID and GID mapping when unpacking and repacking.
  openSUSE/umoci#26
- `tag`, `rm`, `ls`: Tag modification commands such as `umoci tag`, `umoci rm`
  and `umoci ls`. openSUSE/umoci#6 openSUSE/umoci#27
- `stat`: Output information about an image. Currently only shows the history
  information. Only the **JSON** output is stable. openSUSE/umoci#38
- `init`, `new`: New commands have been created to allow for image creation
  from scratch. openSUSE/umoci#5 openSUSE/umoci#42
- `gc`: Garbage collection of images. openSUSE/umoci#6
- Full integration and unit testing, with OCI validation to ensure that we
  always create valid images. openSUSE/umoci#12

### Changed
- `unpack`, `repack`: Create history entries automatically (with options to
  modify the entries). openSUSE/umoci#36
- `unpack`: Store information about its source to ensure consistency when doing
  a `repack`. openSUSE/umoci#14
- The `--image` and `--from` arguments have been combined into a single
  `<path>[:<tag>]` argument for `--image`. openSUSE/umoci#39
- `unpack`: Configuration annotations are now extracted, though there are still
  some discussions happening upstream about the correct way of doing this.
  openSUSE/umoci#43

### Fixed
- `repack`: Errors encountered during generation of delta layers are now
  correctly propagated. openSUSE/umoci#33
- `unpack`: Hardlinks are now extracted as real hardlinks. openSUSE/umoci#25

### Security
- `unpack`, `repack`: Symlinks are now correctly resolved inside the unpacked
  rootfs. openSUSE/umoci#27

## 0.0.0-rc1 - 2016-11-10
### Added
- Proof of concept with major functionality implemented.
  + `unpack`
  + `repack`
  + `config`

[Unreleased]: https://github.com/openSUSE/umoci/compare/v0.4.2...HEAD
[0.4.2]: https://github.com/openSUSE/umoci/compare/v0.4.1...v0.4.2
[0.4.1]: https://github.com/openSUSE/umoci/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/openSUSE/umoci/compare/v0.3.1...v0.4.0
[0.3.1]: https://github.com/openSUSE/umoci/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/openSUSE/umoci/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/openSUSE/umoci/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/openSUSE/umoci/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/openSUSE/umoci/compare/v0.0.0...v0.1.0
[0.0.0]: https://github.com/openSUSE/umoci/compare/v0.0.0-rc3...v0.0.0
[0.0.0-rc3]: https://github.com/openSUSE/umoci/compare/v0.0.0-rc2...v0.0.0-rc3
[0.0.0-rc2]: https://github.com/openSUSE/umoci/compare/v0.0.0-rc1...v0.0.0-rc2
