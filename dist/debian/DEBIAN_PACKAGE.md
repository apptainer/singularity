# Creating the Debian package

## Preparation

As long as the debian directory is in the sub-directory you need to link
or copy it in the top directory.

In the top directory do this:

```sh
rm -rf debian
cp -r dist/debian .
```

Make sure all the dependencies are met. See the `INSTALL.md` for this.
Otherwise `debuild` will complain and quit.

## Configuration

To do some configuration for the build, some environment variables can
be used.

Due to the fact, that `debuild` filters out some variables, all the
configuration variables need to be prefixed by `DEB_`

### mconfig

See `mconfig --help` for details about the configuration options.

`export DEB_NOSUID=1`    adds --without-suid

`export DEB_NONETWORK=1` adds --without-network

`export DEB_NOSECCOMP=1` adds --without-seccomp

`export DEB_NOALL=1`     adds all of the above

To select a specific profile for `mconfig`.

For real production environment us this configuration:

```sh
export DEB_SC_PROFILE=release-stripped
```

or if debugging is needed use this.

```sh
export DEB_SC_PROFILE=debug
```

In case a different build directory is needed:

```sh
export DEB_SC_BUILDDIR=builddir
```

### debchange

One way to update the changelog would be that the developer of singularity
update the Debian changelog on every commit. As this is double work, because
of the CHANGELOG.md in the top directory, the changelog is automatically
updated with the version of the source which is currently checked out.
Which means you can easily build Debian packages for all the different tagged
versions of the software. See `INSTALL.md` on how to checkout a specific
version.

Be aware, that `debchange` will complain about a lower version as the top in
the current changelog. Which means you have to cleanup the changelog if needed.
If you did not change anything in the debian directory manually, it might be easiest
to [start from scratch](#Preparation).
Be aware, that the Debian install directory as you see it now, might not be available
in older versions (branches, tags). Make sure you have a clean copy of the debian
directory before you switch to (checkout) an older version.

Usually `debchange` is configured by the environment variables
`DEBFULLNAME` and `EMAIL`. As `debuild` creates a clean environment it
filters out most of the environment variables. To set `DEBFULLNAME` for
the `debchange` command in the makefile, you have to set `DEB_FULLNAME`.
If these variables are not set, `debchange` will try to find appropriate
values from the system configuration. Usually by using the login name
and the domain-name.

```sh
export DEB_FULLNAME="Your Name"
export EMAIL="you@example.org"
```

## Building

As usual for creating a Debian package you can use `dpkg-buildpackage`
or `debuild` which is a kind of wrapper for the first and includes the start
of `lintian`, too.

```sh
dpkg-buildpackage --build=binary --no-sign
lintian --verbose --display-info --show-overrides
```

or all in one

```sh
debuild --build=binary --no-sign --lintian-opts --display-info --show-overrides
```

After successful build the Debian package can be found in the parent directory.

To clean up the temporary files created by `debuild` use the command:

```sh
dh clean
```

To cleanup the copy of the debian directory, make sure you saved your
changes (if any) and remove it.

```sh
rm -rf debian
```

For details on Debian package building see the man-page of `debuild` and
`dpkg-buildpackage` and `lintian`

## Debian Repository

In the current version this is by far not ready for using it in official
Debian Repositories.

This might change in future. I updated the old debian directory to make
it just work, for people needing it.

Any help is welcome to provide a Debian installer which can be used for
building a Debian package,
that can be used in official Debian Repositories.
