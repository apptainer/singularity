# Installing Singularity

Since you are reading this from the Singularity source code, it will be assumed
that you are building/compiling from source.

Singularity packages are available for various Linux distributions, but may not
always be up-to-date with the latest source release version.

For full instructions on installation, including building RPMs,
installing pre-built EPEL packages etc. please check the
[installation section of the admin guide](https://apptainer.org/admin-docs/master/installation.html).

## Install system dependencies

You must first install development tools and libraries to your host.
On Debian-based systems:

```
$ sudo apt-get update && \
  sudo apt-get install -y build-essential \
  libseccomp-dev pkg-config squashfs-tools cryptsetup
```

On CentOS/RHEL:

```
$ sudo yum groupinstall -y 'Development Tools' && \
  sudo yum install -y epel-release && \
  sudo yum install -y golang libseccomp-devel \
  squashfs-tools cryptsetup
```

## Install Golang

This is one of several ways to [install and configure golang](https://golang.org/doc/install).
The CentOS/RHEL instructions above already installed it so this method is not needed there.

First, download the Golang archive to `/tmp`, then extract the archive to `/usr/local`.

_**NOTE:** if you are updating Go from a older version, make sure you remove `/usr/local/go` before
reinstalling it._

```
$ export GOVERSION=1.16.9 OS=linux ARCH=amd64  # change this as you need

$ wget -O /tmp/go${GOVERSION}.${OS}-${ARCH}.tar.gz https://dl.google.com/go/go${GOVERSION}.${OS}-${ARCH}.tar.gz && \
  sudo tar -C /usr/local -xzf /tmp/go${GOVERSION}.${OS}-${ARCH}.tar.gz
```

Finally, set up your environment for Go:

```
$ echo 'export GOPATH=${HOME}/go' >> ~/.bashrc && \
  echo 'export PATH=/usr/local/go/bin:${PATH}:${GOPATH}/bin' >> ~/.bashrc && \
  source ~/.bashrc
```

## Install golangci-lint

This is an optional (but highly recommended!) step. To ensure
consistency and to catch certain kinds of issues early, we provide a
configuration file for `golangci-lint`. Every pull request must pass the
checks specified there, and these will be run automatically before
attempting to merge the code. If you are modifying Singularity and
contributing your changes to the repository, it's faster to run these
checks locally before uploading your pull request.

In order to download and install the latest version of `golangci-lint`,
you can run:

```
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
```

## Clone the repo

Golang is a bit finicky about where things are placed. Here is the correct way
to build Singularity from source:

```
$ mkdir -p ${GOPATH}/src/github.com/apptainer && \
  cd ${GOPATH}/src/github.com/apptainer && \
  git clone https://github.com/apptainer/singularity.git && \
  cd singularity
```

To build a stable version of Singularity, check out a [release tag](https://github.com/apptainer/singularity/tags) before compiling:

```
$ git checkout v3.8.6
```

## Compiling Singularity

You can build Singularity using the following commands:

```
$ cd ${GOPATH}/src/github.com/apptainer/singularity && \
  ./mconfig && \
  cd ./builddir && \
  make && \
  sudo make install
```

And that's it! Now you can check your Singularity version by running:

```
$ singularity version
```
To build in a different folder and to set the install prefix to a different path:

```
$ ./mconfig -b ./buildtree -p /usr/local
```

## Install from the RPM

*NOTE: You should only attempt to build the RPM on a CentOS/RHEL system.*

To build the rpm, in addition to the
[dependencies](#install-system-dependencies),
install `rpm-build`, `wget`, and `golang`:

```
$ sudo yum install -y rpm-build wget golang
```

The rpm build will use the OS distribution or EPEL version of Go,
or it will use a different installation of Go, whichever is first in $PATH.
If the first `go` found in $PATH is too old,
then the rpm build uses that older version to compile the newer go
toolchain from source.
This mechanism is necessary for rpm build systems that do not allow
downloading anything from the internet.
In order to make use of this mechanism, use the `mconfig --only-rpm` option
to skip the minimum version check.
`mconfig` will then create a `.spec` file that looks for a go source
tarball in the rpm build's current directory.
Download the tarball like this:

```
wget https://dl.google.com/go/go$(scripts/get-min-go-version).src.tar.gz
```

Download the latest
[release tarball](https://github.com/apptainer/singularity/releases)
and use it to install the RPM like this:

```
$ export VERSION=3.8.6  # this is the singularity version, change as you need

$ wget https://github.com/apptainer/singularity/releases/download/v${VERSION}/singularity-${VERSION}.tar.gz && \
    rpmbuild -tb singularity-${VERSION}.tar.gz && \
    sudo rpm -ivh ~/rpmbuild/RPMS/x86_64/singularity-${VERSION}-1.el7.x86_64.rpm && \
    rm -rf ~/rpmbuild singularity-${VERSION}*.tar.gz

Alternatively, to build an RPM from the latest master you can
[clone the repo as detailed above](#clone-the-repo).
Then use the `rpm` make target to build Singularity as an rpm package,
for example like this if you already have a new enough golang first
in your PATH:

```
VERSION=3.8.6  # this is the singularity version, change as you need
./mconfig
make -C builddir rpm
sudo rpm -ivh ~/rpmbuild/RPMS/x86_64/singularity-${VERSION}*.x86_64.rpm 
```

Alternatively, to build an RPM from the latest master you can 
[clone the repo as detailed above](#clone-the-repo).  Then create your own
tarball and use it to install Singularity:

```
$ cd $GOPATH/src/github.com/apptainer/singularity && \
  ./mconfig && \
  make -C builddir rpm && \
  sudo rpm -ivh ~/rpmbuild/RPMS/x86_64/singularity-3.8.6*.x86_64.rpm # or whatever version you built
```

To build an rpm with an alternative install prefix set RPMPREFIX on the
make step, for example:

```
$ make -C builddir rpm RPMPREFIX=/usr/local
```

For more information on installing/updating/uninstalling the RPM, check out our 
[admin docs](https://apptainer.org/admin-docs/master/admin_quickstart.html).

## Debian Package

Information on how to build a Debian package can be found in
[dist/debian/DEBIAN_PACKAGE.md](dist/debian/DEBIAN_PACKAGE.md).
