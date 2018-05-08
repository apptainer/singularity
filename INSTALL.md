# Installing Singularity development-3.0

Since you are reading this from the Singualrity source code, it will be assumed 
that you are building/compiling. 

## Install system dependencies 
You must first install development and libraries to your host. Assuming Ubuntu 
(apply similar to RHEL derivatives):

```
$ sudo apt-get update && sudo apt-get install -y build-essential libssl-dev uuid-dev
```

## Install golang
This is one of several ways to [install and configure golang](https://golang.org/doc/install).

```
$ sudo apt-get update && sudo apt-get install -y golang
$ echo 'export GOPATH=$HOME/go' >> ~/.bashrc
$ echo 'export PATH=${PATH}:${GOPATH}/bin' >> ~/.bashrc
$ source ~/.bashrc
```

## Clone the repo
golang is a bit finicky about where things are placed. Here is the correct way
to build Singularity from source.

```
$ mkdir -p $GOPATH/src/github.com/singularityware
$ cd $GOPATH/src/github.com/singularityware
$ git clone https://github.com/singularityware/singularity.git
$ cd singularity
$ git fetch
$ git checkout development-3.0
```

## Install golang dependencies 
Dependencies are managed using [`dep`](https://github.com/golang/dep). You can 
use `go get` to install it like so:

```
$ go get -u -v github.com/golang/dep/cmd/dep
```

And then you can install the golang dependencies as part of the build later on or like so:

```
$ cd $GOPATH/src/github.com/singularityware/singularity
$ dep ensure -v
```

## Compile the Singularity binary
Now you are ready to build Singularity:

```
$ cd $GOPATH/src/github.com/singularityware/singularity
$ ./mconfig
$ cd ./builddir
$ make dep
$ make
$ sudo make install
```

To build in a different folder and to set the install prefix to a different path:

```
$ ./mconfig -p /usr/local -b ./buildtree
```
