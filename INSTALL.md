# Installing Singularity

Since you are reading this from the Singularity source code, it will be assumed
that you are building/compiling from source.

Singularity packages are available for various Linux distributions, but may not
always be up-to-date with the latest source release version.

For full instructions on installation, including building RPMs,
installing pre-built EPEL packages etc. please check the
[installation section of the admin guide](https://singularity.hpcng.org/admin-docs/master/installation.html).

## Install system dependencies

You must first install development tools and libraries to your host.

On Debian-based systems, including Ubuntu:

```sh
# Ensure repositories are up-to-date
sudo apt-get update
# Install debian packages for dependencies
sudo apt-get install -y \
    build-essential \
    libseccomp-dev \
    pkg-config \
    squashfs-tools \
    cryptsetup \
    curl wget git
```

On CentOS/RHEL:

```sh
# Install basic tools for compiling
sudo yum groupinstall -y 'Development Tools'
# Ensure EPEL repository is available
sudo yum install -y epel-release
# Install RPM packages for dependencies
sudo yum install -y \
    libseccomp-devel \
    squashfs-tools \
    cryptsetup \
    wget git
```

## Install Go

Singularity is written in Go, and may require a newer version of Go than is
available in the repositories of your distribution. We recommend installing the
latest version of Go from the [official binaries](https://golang.org/dl/).

First, download the Go tar.gz archive to `/tmp`, then extract the archive to
`/usr/local`.

_**NOTE:** if you are updating Go from a older version, make sure you remove
`/usr/local/go` before reinstalling it._

```sh
export GOVERSION=1.17.3 OS=linux ARCH=amd64  # change this as you need

wget -O /tmp/go${GOVERSION}.${OS}-${ARCH}.tar.gz \
  https://dl.google.com/go/go${GOVERSION}.${OS}-${ARCH}.tar.gz
sudo tar -C /usr/local -xzf /tmp/go${GOVERSION}.${OS}-${ARCH}.tar.gz
```

Finally, add `/usr/local/go/bin` to the `PATH` environment variable:

```sh
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

## Install golangci-lint

If you will be making changes to the source code, and submitting PRs, you should
install `golangci-lint`, which is the linting tool used in the Singularity
project to ensure code consistency.

Every pull request must pass the `golangci-lint` checks, and these will be run
automatically before attempting to merge the code. If you are modifying
Singularity and contributing your changes to the repository, it's faster to run
these checks locally before uploading your pull request.

In order to download and install the latest version of `golangci-lint`, you can
run:

<!-- markdownlint-disable MD013 -->

```sh
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.43.0
```

<!-- markdownlint-enable MD013 -->

Add `$(go env GOPATH)` to the `PATH` environment variable:

```sh
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.bashrc
source ~/.bashrc
```

## Clone the repo

With the adoption of Go modules you no longer need to clone the Singularity
repository to a specific location.

Clone the repository with `git` in a location of your choice:

```sh
git clone https://github.com/hpcng/singularity.git
cd singularity
```

By default your clone will be on the `master` branch which is where development
of Singularity happens.
To build a specific version of Singularity, check out a
[release tag](https://github.com/hpcng/singularity/tags) before compiling,
for example:

```sh
git checkout v3.8.4
```

## Compiling Singularity

You can configure, build, and install Singularity using the following commands:

```sh
./mconfig
cd ./builddir
make
sudo make install
```

And that's it! Now you can check your Singularity version by running:

```sh
singularity --version
```

The `mconfig` command accepts options that can modify the build and installation
of Singularity. For example, to build in a different folder and to set the
install prefix to a different path:

```sh
./mconfig -b ./buildtree -p /usr/local
```

See the output of `./mconfig -h` for available options.

## Building & Installing from an RPM

On a RHEL / CentOS / Fedora machine you can build a Singularity into an rpm
package, and install it from the rpm. This is useful if you need to install
Singularity across multiple machines, or wish to manage all software via
`yum/dnf`.

To build the rpm, in addition to the
[dependencies](#install-system-dependencies),
install `rpm-build`, `wget`, and `golang`:

```sh
sudo yum install -y rpm-build wget golang
```

The rpm build can use the distribution or EPEL version of Go, even
though as of this writing that version is older than the default
minimum version of Go that Singularity requires.
This is because the rpm applies a source code patch to lower the minimum
required.

To build from a release source tarball do these commands:

<!-- markdownlint-disable MD013 -->

```sh
export VERSION=3.8.4  # this is the singularity version, change as you need

# Fetch the source
wget https://github.com/hpcng/singularity/releases/download/v${VERSION}/singularity-${VERSION}.tar.gz
# Build the rpm from the source tar.gz
rpmbuild -tb singularity-${VERSION}.tar.gz
# Install Singularity using the resulting rpm
sudo rpm -ivh ~/rpmbuild/RPMS/x86_64/singularity-${VERSION}-1.el7.x86_64.rpm
# (Optionally) Remove the build tree and source to save space
rm -rf ~/rpmbuild singularity-${VERSION}*.tar.gz
```

<!-- markdownlint-enable MD013 -->

Alternatively, to build an RPM from the latest master you can
[clone the repo as detailed above](#clone-the-repo).
Create the build configuration using the `--only-rpm` option of
`mconfig` if you're using the system's too-old golang installation,
to lower the minimum required version.
Then use the `rpm` make target to build Singularity as an rpm package:

<!-- markdownlint-disable MD013 -->

```sh
./mconfig --only-rpm
make -C builddir rpm
sudo rpm -ivh ~/rpmbuild/RPMS/x86_64/singularity-3.8.4*.x86_64.rpm # or whatever version you built
```

<!-- markdownlint-enable MD013 -->

By default, the rpm will be built so that Singularity is installed under
`/usr/local`.

To build an rpm with an alternative install prefix set RPMPREFIX on the make
step, for example:

```sh
make -C builddir rpm RPMPREFIX=/opt/singularity
```

For more information on installing/updating/uninstalling the RPM, check out our
[admin docs](https://singularity.hpcng.org/admin-docs/master/admin_quickstart.html).

## Debian Package

Additional information on how to build a Debian package can be found in [dist/debian/DEBIAN_PACKAGE.md](dist/debian/DEBIAN_PACKAGE.md).
