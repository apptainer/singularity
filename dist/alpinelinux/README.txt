Porting singularity to the lightweight musl libc.

Building requirements:
$ sudo apk update && sudo apk upgrade
$ sudo apk add alpine-sdk autoconf automake libtool linux-headers
$ mkdir singularity-build
$ cd singularity-build

fetch the APKBUILD 

$ apbuild -r

if you get something like:
~/singularity-build$ abuild -r
No private key found. Use 'abuild-keygen' to generate the keys.
Then you can either:
  * set the PACKAGER_PRIVKEY in /home/tru/.abuild/abuild.conf
    ('abuild-keygen -a' does this for you)
  * set the PACKAGER_PRIVKEY in /etc/abuild.conf
  * specify the key with the -k option to abuild-sign

>>> ERROR: singularity: all failed
you need to properly setup your alpine build environment :P

~/singularity-build$ abuild-keygen -a
>>> Generating public/private rsa key pair for abuild
Enter file in which to save the key [/home/tru/.abuild/tru-5805cc3f.rsa]: 
Generating RSA private key, 2048 bit long modulus
..................................................................................................+++
.................+++
e is 65537 (0x10001)
writing RSA key
>>> 
>>> You'll need to install /home/tru/.abuild/tru-5805cc3f.rsa.pub into 
>>> /etc/apk/keys to be able to install packages and repositories signed with
>>> /home/tru/.abuild/tru-5805cc3f.rsa
>>> 
>>> Please remember to make a safe backup of your private key:
>>> /home/tru/.abuild/tru-5805cc3f.rsa
>>> 


~/singularity-build$ abuild -r
>>> singularity: Checking sanity of /home/tru/singularity-build/APKBUILD...
>>> WARNING: singularity: depends_dev found but no development subpackage found
>>> singularity: Analyzing dependencies...
abuild-apk: User tru is not a member of group abuild

>>> ERROR: singularity: all failed
>>> singularity: Uninstalling dependencies...
abuild-apk: User tru is not a member of group abuild

Just add yourself to the abuild group, logout and login.

~/singularity-build$ abuild -r
>>> singularity: Checking sanity of /home/tru/singularity-build/APKBUILD...
>>> WARNING: singularity: depends_dev found but no development subpackage found
>>> singularity: Analyzing dependencies...
WARNING: Ignoring /home/tru/packages//tru/x86_64/APKINDEX.tar.gz: No such file or directory
(1/1) Installing .makedepends-singularity (0)
OK: 249 MiB in 84 packages
>>> singularity: Cleaning temporary build dirs...
  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100   134    0   134    0     0    284      0 --:--:-- --:--:-- --:--:--   292
100 47328    0 47328    0     0  45085      0 --:--:--  0:00:01 --:--:--  125k
>>> singularity: Checking sha512sums...
singularity-2.1.2.tar.gz: OK
>>> singularity: Unpacking /var/cache/distfiles/singularity-2.1.2.tar.gz...
+autoreconf -i -f
libtoolize: putting auxiliary files in '.'.
libtoolize: copying file './ltmain.sh'
libtoolize: putting macros in AC_CONFIG_MACRO_DIRS, '.'.
...
>>> singularity-doc*: Preparing subpackage singularity-doc...
fatal: Not a git repository (or any of the parent directories): .git
fatal: Not a git repository (or any of the parent directories): .git
>>> singularity*: Running postcheck for singularity-doc
>>> singularity*: Running split function examples...
>>> singularity-examples*: Preparing subpackage singularity-examples...
fatal: Not a git repository (or any of the parent directories): .git
fatal: Not a git repository (or any of the parent directories): .git
>>> singularity*: Running postcheck for singularity-examples
>>> WARNING: singularity*: Found /usr/share/doc but package name doesn't end with -doc
>>> singularity*: Running postcheck for singularity
>>> singularity*: Preparing package singularity...
>>> singularity*: Stripping binaries
fatal: Not a git repository (or any of the parent directories): .git
fatal: Not a git repository (or any of the parent directories): .git
>>> singularity-doc*: Scanning shared objects
>>> singularity-examples*: Scanning shared objects
>>> singularity*: Scanning shared objects
>>> singularity-doc*: Tracing dependencies...
>>> singularity-doc*: Package size: 40.0 KB
>>> singularity-doc*: Compressing data...
>>> singularity-doc*: Create checksum...
>>> singularity-doc*: Create singularity-doc-2.1.2-r0.apk
>>> singularity-examples*: Tracing dependencies...
>>> singularity-examples*: Package size: 48.0 KB
>>> singularity-examples*: Compressing data...
>>> singularity-examples*: Create checksum...
>>> singularity-examples*: Create singularity-examples-2.1.2-r0.apk
>>> singularity*: Tracing dependencies...
	so:libc.musl-x86_64.so.1
>>> singularity*: Package size: 364.0 KB
>>> singularity*: Compressing data...
>>> singularity*: Create checksum...
>>> singularity*: Create singularity-2.1.2-r0.apk
>>> singularity: Cleaning up srcdir
>>> singularity: Cleaning up pkgdir
>>> singularity: Uninstalling dependencies...
>>> singularity: Updating the cached abuild repository index...
fatal: Not a git repository (or any of the parent directories): .git
>>> singularity: Signing the index...

You should have a your freshly cooked packages in ~/packages.

$ find ~/packages/
/home/tru/packages/
/home/tru/packages/tru
/home/tru/packages/tru/x86_64
/home/tru/packages/tru/x86_64/singularity-2.1.2-r0.apk
/home/tru/packages/tru/x86_64/singularity-doc-2.1.2-r0.apk
/home/tru/packages/tru/x86_64/singularity-examples-2.1.2-r0.apk
/home/tru/packages/tru/x86_64/APKINDEX.tar.gz

