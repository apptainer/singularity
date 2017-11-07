# Installing Singularity

Since you are reading this from the Singualrity source code, it will be
assumed that you are building/compiling. You must first
install the development tools and libraries to your host. Assuming a Red
Hat compatible system (apply similar to Debian derivitives):

```
$ sudo yum groupinstall "Development Tools"
```


## To compile and install Singularity from a [released tarball](https://github.com/singularityware/singularity/releases):
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

## To compile and install Singularity from a Git clone:
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

## To build an RPM of Singularity from a Git clone:
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
