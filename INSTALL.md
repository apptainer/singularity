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
will result in a non-functioning or semi-functioning installation. This is required
due to Singularity requiring the `SUID` bit to be set, even if the user installs
into a directory in which she has full priviledges.


### Using nixpkgs to build

If you happen to be using [Nix](https://nixos.org/nix/), you can typically install the latest
version of Singularity in the usual way. For instance `nix-env -i singularity` to put it in your
user environment, or just `nix-shell -p singularity` to try it out in a shell.

But if you want to test a custom version of Singularity or hack on it, you can pull in all build dependencies 
specified in the 
[Singularity derivation](https://github.com/NixOS/nixpkgs/blob/master/pkgs/applications/virtualization/singularity/default.nix) 
by doing:

```
nix-shell '<nixpkgs>' -A singularity --pure
```

(Note the `--pure` is optional and only recommended  if you want to perform the build in isolation of your
environment, which *is* recommended before doing a pull request, for example.)

Now you can follow the standard build instructions above in the shell that has been launched.


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
