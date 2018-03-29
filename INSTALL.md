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

And then you can install the golang dependencies like so:

```
$ cd $GOPATH/src/github.com/singularityware/singularity
$ dep ensure -v
```

## Compile the Singularity binary
Now you are ready to build Singularity:

```
$ cd $GOPATH/src/github.com/singularityware/singularity
$ ./compile.sh
```

The binary will appear in `$GOPATH/src/github.com/singularityware/singularity/core/buildtree`
You may copy it wherever you wish.

## To compile and install Singularity from a [release tarball](https://github.com/singularityware/singularity/releases):
Here, the version of Singularity that you want to install is given in <b>&lt;version&gt;</b>.  Please substitute as necessary.  
<pre>
$ version=<b>&lt;version&gt;</b>
$ wget "https://github.com/singularityware/singularity/releases/download/${version}/singularity-${version}.tar.gz"
$ tar -xvzf singularity-${version}.tar.gz
$ cd singularity-${version}
$ ./configure --prefix=/usr/local
$ make
$ sudo make install
</pre>

note: The `sudo` is very important for the `make install`. Failure to do this
will result in a non-functioning or semi-functioning installation.

## To compile and install Singularity (less than 3.0) from a Git clone:
Here, the version of Singularity that you want to install is given in <b>&lt;version&gt;</b>.  Please substitute as necessary.  
<pre>
$ version=<b>&lt;version&gt;</b>
$ git clone https://github.com/singularityware/singularity.git
$ cd singularity
$ git checkout tags/${version} -b ${version}
$ ./autogen.sh
$ ./configure --prefix=/usr/local
$ make
$ sudo make install
</pre>

note: The `sudo` is very important for the `make install`. Failure to do this
will result in a non-functioning or semi-functioning installation.

## To build an RPM of Singularity (less than 3.0) from a Git clone:
Here, the version of Singularity that you want to install is given in <b>&lt;version&gt;</b>.  Please substitute as necessary.  
<pre>
$ version=<b>&lt;version&gt;</b>
$ git clone https://github.com/singularityware/singularity.git
$ cd singularity
$ git checkout tags/${version} -b ${version}
$ ./autogen.sh
$ ./configure
$ make dist
$ rpmbuild -ta singularity-*.tar.gz
</pre>
