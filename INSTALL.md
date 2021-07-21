# Installing Singularity

Since you are reading this from the Singularity source code, it will be assumed
that you are building/compiling from source.

Singularity packages are available for various Linux distributions, but may not
always be up-to-date with the latest source release version.

For full instructions on installation, including building RPMs,
installing pre-built EPEL packages etc. please check the
[installation section of the admin guide](https://singularity.hpcng.org/admin-docs/master/installation.html).

## Install system dependencies

You must first install development tools and libraries to your host. On
Debian-based systems:

```sh
sudo apt-get update && \
  sudo apt-get install -y build-essential \
    libseccomp-dev pkg-config squashfs-tools cryptsetup
```

On CentOS/RHEL:

```sh
sudo yum groupinstall -y 'Development Tools' && \
  sudo yum install -y epel-release && \
  sudo yum install -y golang libseccomp-devel \
    squashfs-tools cryptsetup
```

## Install Golang

This is one of several ways to
[install and configure golang](https://golang.org/doc/install). The CentOS/RHEL
instructions above already installed it so this method is not needed there.

First, download the Golang archive to `/tmp`, then extract the archive to
`/usr/local`.

_**NOTE:** if you are updating Go from a older version, make sure you remove
`/usr/local/go` before reinstalling it._

```sh
export VERSION=1.16.7 OS=linux ARCH=amd64  # change this as you need

wget -O /tmp/go${VERSION}.${OS}-${ARCH}.tar.gz \
  https://dl.google.com/go/go${VERSION}.${OS}-${ARCH}.tar.gz && \
sudo tar -C /usr/local -xzf /tmp/go${VERSION}.${OS}-${ARCH}.tar.gz
```

Finally, set up your environment for Go:

```sh
echo 'export GOPATH=${HOME}/go' >> ~/.bashrc && \
  echo 'export PATH=/usr/local/go/bin:${PATH}:${GOPATH}/bin' >> ~/.bashrc && \
  source ~/.bashrc
```

## Install golangci-lint

This is an optional (but highly recommended!) step. To ensure consistency and to
catch certain kinds of issues early, we provide a configuration file for
`golangci-lint`. Every pull request must pass the checks specified there, and
these will be run automatically before attempting to merge the code. If you are
modifying Singularity and contributing your changes to the repository, it's
faster to run these checks locally before uploading your pull request.

In order to download and install the latest version of `golangci-lint`, you can
run:

<!-- markdownlint-disable MD013 -->
```sh
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
```
<!-- markdownlint-enable MD013 -->

## Clone the repo

Golang is a bit finicky about where things are placed. Here is the correct way
to build Singularity from source:

```sh
mkdir -p ${GOPATH}/src/github.com/hpcng && \
  cd ${GOPATH}/src/github.com/hpcng && \
  git clone https://github.com/hpcng/singularity.git && \
  cd singularity
```

To build a specific version of Singularity, check out a
[release tag](https://github.com/hpcng/singularity/tags) before compiling:

```sh
git checkout v3.8.2
```

## Compiling Singularity

You can build Singularity using the following commands:

```sh
cd ${GOPATH}/src/github.com/hpcng/singularity && \
  ./mconfig && \
  cd ./builddir && \
  make && \
  sudo make install
```

And that's it! Now you can check your Singularity version by running:

```sh
singularity version
```

To build in a different folder and to set the install prefix to a different
path:

```sh
./mconfig -b ./buildtree -p /usr/local
```

## Install from the RPM

*NOTE: You should only attempt to build the RPM on a CentOS/RHEL system.*

To build the RPM, you first need to install `rpm-build` and `wget`:

```sh
sudo yum -y update && sudo yum install -y rpm-build wget
```

Make sure you have also
[installed the system dependencies](#install-system-dependencies)
as shown above.  Then download the latest
[release tarball](https://github.com/hpcng/singularity/releases)
and use it to install the RPM like this:

<!-- markdownlint-disable MD013 -->
```sh
export VERSION=3.8.2  # this is the singularity version, change as you need

wget https://github.com/hpcng/singularity/releases/download/v${VERSION}/singularity-${VERSION}.tar.gz && \
    rpmbuild -tb singularity-${VERSION}.tar.gz && \
    sudo rpm -ivh ~/rpmbuild/RPMS/x86_64/singularity-${VERSION}-1.el7.x86_64.rpm && \
    rm -rf ~/rpmbuild singularity-${VERSION}*.tar.gz
```
<!-- markdownlint-enable MD013 -->

Alternatively, to build an RPM from the latest master you can
[clone the repo as detailed above](#clone-the-repo). Then create your own
tarball and use it to install Singularity:

<!-- markdownlint-disable MD013 -->
```sh
cd $GOPATH/src/github.com/hpcng/singularity && \
  ./mconfig && \
  make -C builddir rpm && \
  sudo rpm -ivh ~/rpmbuild/RPMS/x86_64/singularity-3.8.2*.x86_64.rpm # or whatever version you built
```
<!-- markdownlint-enable MD013 -->

To build an rpm with an alternative install prefix set RPMPREFIX on the make
step, for example:

```sh
make -C builddir rpm RPMPREFIX=/usr/local
```

For more information on installing/updating/uninstalling the RPM, check out our
[admin docs](https://singularity.hpcng.org/admin-docs/master/admin_quickstart.html).

## Debian Package

Additional information on how to build a Debian package can be found in [dist/debian/DEBIAN_PACKAGE.md](dist/debian/DEBIAN_PACKAGE.md).
