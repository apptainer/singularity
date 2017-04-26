# Installing Singularity

Since you are reading this from the Singualrity source code, it will be
assumed that you are building/compiling. To start off with you must first
install the development tools and libraries to your host. Assuming a Red
Hat compatible system (apply similar to Debian derivitives):

```
$ sudo yum groupinstall "Development Tools"
```

## To compile and install Singularity from source:

```
$ git clone https://github.com/singularityware/singularity.git
$ cd singularity
$ ./autogen.sh
$ ./configure --prefix=/path/to/singularity
$ make
$ sudo make install
```

note: The `sudo` is very important for the `make install`. Failure to do this
will result in a non-functioning or semi-functioning installation.

## To build an RPM of Singularity

```
$ git clone https://github.com/singularityware/singularity.git
$ cd singularity
$ ./autogen.sh
$ ./configure
$ make dist
$ rpmbuild -ta singularity-*.tar.gz
```

