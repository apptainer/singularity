# Installing Singularity

Since you are reading this from the Singualrity source code, it will be
assumed that you are building/compiling. To start off with you must first
install the development tools and libraries to your host. Assuming a Red
Hat compatible system (apply similar to Debian derivitives):

```
$ sudo yum groupinstall "Development Tools"
```


## To compile and install Singularity from a released tarball:
Assuming a 2.3.1 released tarball...
```
$ version=2.3.1
$ wget "https://github.com/singularityware/singularity/releases/download/${version}/singularity-${version}.tar.gz"
$ tar -xvzf singularity-${version}.tar.gz
$ cd singularity-${version}
$ ./configure --prefix=/usr/local
$ make
$ sudo make install
```

note: The `sudo` is very important for the `make install`. Failure to do this
will result in a non-functioning or semi-functioning installation.

## To compile and install Singularity from a Git clone:

```
$ git clone https://github.com/singularityware/singularity.git
$ cd singularity
$ git checkout tags/2.3.1 -b 2.3.1
$ ./autogen.sh
$ ./configure --prefix=/usr/local
$ make
$ sudo make install
```

note: The `sudo` is very important for the `make install`. Failure to do this
will result in a non-functioning or semi-functioning installation.

## To compile and install Singularity from an existing Git clone:

```
$ cd singularity
$ git fetch --tags origin
$ git checkout tags/2.3.1 -b 2.3.1
$ ./autogen.sh
$ ./configure --prefix=/usr/local
$ make
$ sudo make install
```

note: The `sudo` is very important for the `make install`. Failure to do this
will result in a non-functioning or semi-functioning installation.

## To build an RPM of Singularity from a Git clone:

```
$ git clone https://github.com/singularityware/singularity.git
$ cd singularity
$ git checkout tags/2.3.1 -b 2.3.1
$ ./autogen.sh
$ ./configure
$ make dist
$ rpmbuild -ta singularity-*.tar.gz
```

