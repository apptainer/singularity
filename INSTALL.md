# Installing Singularity development-3.0

Since you are reading this from the Singularity source code, it will be assumed
that you are building/compiling.

For full instructions on installation, check out our
[installation guide](https://www.sylabs.io/guides/3.0/user-guide/installation.html).

## Install system dependencies

You must first install development and libraries to your host.
On Debian-based systems:

```
$ sudo apt-get update && \
  sudo apt-get install -y build-essential \
  libssl-dev uuid-dev libgpgme11-dev libseccomp-dev pkg-config squashfs-tools
```

On CentOS/RHEL:

```
$ sudo yum groupinstall -y 'Development Tools' && \
  sudo yum install -y epel-release && \
  sudo yum install -y golang openssl-devel libuuid-devel libseccomp-devel
```

On CentOS/RHEL 6 or less, you may skip `libseccomp-devel`.

## Install Golang

This is one of several ways to [install and configure golang](https://golang.org/doc/install).  The CentOS/RHEL instructions above already installed it so this method is not needed there.

First, download the Golang archive to `/tmp/`, then extract the archive to `/usr/local`: (or use other instructions on Go
[installation page](https://golang.org/doc/install))

```
$ export VERSION=1.11.4 OS=linux ARCH=amd64  # change this as you need

$ wget -O /tmp/go${VERSION}.${OS}-${ARCH}.tar.gz https://dl.google.com/go/go${VERSION}.${OS}-${ARCH}.tar.gz && \
  sudo tar -C /usr/local -xzf /tmp/go${VERSION}.${OS}-${ARCH}.tar.gz
```

Finally, set up your environment for Go:

```
$ echo 'export GOPATH=${HOME}/go' >> ~/.bashrc && \
  echo 'export PATH=/usr/local/go/bin:${PATH}:${GOPATH}/bin' >> ~/.bashrc && \
  source ~/.bashrc
```

## Clone the repo

Golang is a bit finicky about where things are placed. Here is the correct way
to build Singularity from source:

```
$ mkdir -p ${GOPATH}/src/github.com/sylabs && \
  cd ${GOPATH}/src/github.com/sylabs && \
  git clone https://github.com/sylabs/singularity.git && \
  cd singularity
```

To build a stable version of Singularity, check out a [release tag](https://github.com/sylabs/singularity/tags) before compiling:

```
$ git checkout v3.0.2
```

## Compiling Singularity

You can build Singularity using the following commands:

```
$ cd ${GOPATH}/src/github.com/sylabs/singularity && \
  ./mconfig && \
  cd ./builddir && \
  make && \
  sudo make install
```

And thats it! Now you can check your Singularity version by running:

```
$ singularity version
```

<br>

Alternatively, to build an RPM on CentOS/RHEL use the following commands:

```
$ sudo yum install -y rpm-build && \
 cd $GOPATH/src/github.com/sylabs/singularity && \
  ./mconfig && \
  make -C builddir rpm
```

To build a stable version of Singularity, check out a [release tag](https://github.com/sylabs/singularity/tags) before compiling:

```
$ sudo yum install -y rpm-build wget

$ cd ${GOPATH}/src/github.com/sylabs/singularity && \
  ./mconfig && \
  make -C builddir rpm
```

Golang doesn't have to be installed to build an RPM because the RPM build installs golang
and all dependencies, but it is still recommended for a complete development environment.

To build in a different folder and to set the install prefix to a different path:

```
$ ./mconfig -p /usr/local -b ./buildtree
```

## Install from the RPM

To build the RPM, you first need to download the latest [relese tarball](https://github.com/sylabs/singularity/releases).
Since we are building from the RPM, you don't need to install Golang, but you do need to [install the other dependencies](#install-system-dependencies).

```
$ export VERSION=3.0.2  # this is the singularity version, change as you need

$ wget https://github.com/sylabs/singularity/releases/download/v${VERSION}/singularity-${VERSION}.tar.gz && \
    rpmbuild -tb singularity-${VERSION}.tar.gz && \
    sudo rpm -ivh ~/rpmbuild/RPMS/x86_64/singularity-${VERSION}-1.el7.x86_64.rpm && \
    rm -rf ~/rpmbuild singularity-${VERSION}*.tar.gz
```

*NOTE: You should only atempt to build the RPM on a CentOS/RHEL system.*

For more infoation on installing/updating/uninstalling the RPM, check out our [admin docs](https://www.sylabs.io/guides/3.0/admin-guide/admin_quickstart.html).
