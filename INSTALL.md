# Installing Singularity development-3.0

Since you are reading this from the Singularity source code, it will be assumed
that you are building/compiling.

## Install system dependencies
You must first install development and libraries to your host.
On Debian-based systems:

```
$ sudo apt-get update && \
sudo apt-get install -y build-essential \
libssl-dev uuid-dev libgpgme11-dev squashfs-tools libseccomp-dev pkg-config
```

On CentOS/RHEL:

```
$ sudo yum groupinstall -y 'Development Tools'
$ sudo yum install -y epel-release
$ sudo yum install -y golang openssl-devel libuuid-devel libseccomp-devel
```
Skip libseccomp-devel on CentOS/RHEL 6.

## Install golang

This is one of several ways to [install and configure golang](https://golang.org/doc/install).  The CentOS/RHEL instructions above already installed it so this method is not needed there.

Visit the [golang download page](https://golang.org/dl/) and pick a
package archive to download.  Copy the link address and download with `wget`.

```
$ export VERSION=1.11 OS=linux ARCH=amd64
$ cd /tmp
$ wget https://dl.google.com/go/go$VERSION.$OS-$ARCH.tar.gz
```

Then extract the archive to `/usr/local` (or use other instructions on go
installation page).

```
$ sudo tar -C /usr/local -xzf go$VERSION.$OS-$ARCH.tar.gz
```

Finally, set up your environment for go

```
$ echo 'export GOPATH=${HOME}/go' >> ~/.bashrc
$ echo 'export PATH=/usr/local/go/bin:${PATH}:${GOPATH}/bin' >> ~/.bashrc
$ source ~/.bashrc
```

## Clone the repo
golang is a bit finicky about where things are placed. Here is the correct way
to build Singularity from source.

```
$ mkdir -p $GOPATH/src/github.com/sylabs
$ cd $GOPATH/src/github.com/sylabs
$ git clone https://github.com/sylabs/singularity.git
$ cd singularity
```

## Compile the Singularity binary
You can build Singularity using the following commands:

```
$ cd $GOPATH/src/github.com/sylabs/singularity
$ ./mconfig
$ cd ./builddir
$ make
$ sudo make install
```

Alternatively, to build an rpm on CentOS/RHEL use the following commands:

```
$ sudo yum install -y rpm-build
$ cd $GOPATH/src/github.com/sylabs/singularity
$ ./mconfig
$ make -C builddir rpm
```

To build a stable version of Singularity, check out a [release tag](https://github.com/sylabs/singularity/tags) before compiling:

```
$ git checkout v3.0.3
```

To build in a different folder and to set the install prefix to a different path:

```
$ ./mconfig -p /usr/local -b ./buildtree
```
