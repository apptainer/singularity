# go-mtree

[![Build Status](https://travis-ci.org/vbatts/go-mtree.svg?branch=master)](https://travis-ci.org/vbatts/go-mtree) [![Go Report Card](https://goreportcard.com/badge/github.com/vbatts/go-mtree)](https://goreportcard.com/report/github.com/vbatts/go-mtree)

`mtree` is a filesystem hierarchy validation tooling and format.
This is a library and simple cli tool for [mtree(8)][mtree(8)] support.

While the traditional `mtree` cli utility is primarily on BSDs (FreeBSD,
openBSD, etc), even broader support for the `mtree` specification format is
provided with libarchive ([libarchive-formats(5)][libarchive-formats(5)]).

There is also an [mtree port for Linux][archiecobbs/mtree-port] though it is
not widely packaged for Linux distributions.


## Format

The format of hierarchy specification is consistent with the `# mtree v2.0`
format.  Both the BSD `mtree` and libarchive ought to be interoperable with it
with only one definite caveat.  On Linux, extended attributes (`xattr`) on
files are often a critical aspect of the file, holding ACLs, capabilities, etc.
While FreeBSD filesystem do support `extattr`, this feature has not made its
way into their `mtree`.

This implementation of mtree supports a few non-upstream "keyword"s, such as:
`xattr` and `tar_time`. If you include these keywords, the FreeBSD `mtree`
will fail, as they are unknown keywords to that implementation.

To have `go-mtree` produce specifications that will be 
strictly compatible with the BSD `mtree`, use the `-bsd-keywords` flag when
creating a manifest. This will make sure that only the keywords supported by
BSD `mtree` are used in the program.


### Typical form

With the standard keywords, plus say `sha256digest`, the hierarchy
specification looks like:

```mtree
# .
/set type=file nlink=1 mode=0664 uid=1000 gid=100
. size=4096 type=dir mode=0755 nlink=6 time=1459370393.273231538
    LICENSE size=1502 mode=0644 time=1458851690.0 sha256digest=ef4e53d83096be56dc38dbf9bc8ba9e3068bec1ec37c179033d1e8f99a1c2a95
    README.md size=2820 mode=0644 time=1459370256.316148361 sha256digest=d9b955134d99f84b17c0a711ce507515cc93cd7080a9dcd50400e3d993d876ac

[...]
```

See the directory presently in, and the files present. Along with each
path, is provided the keywords and the unique values for each path. Any common
keyword and values are established in the `/set` command.


### Extended attributes form

```mtree
# .
/set type=file nlink=1 mode=0664 uid=1000 gid=1000
. size=4096 type=dir mode=0775 nlink=6 time=1459370191.11179595 xattr.security.selinux=dW5jb25maW5lZF91Om9iamVjdF9yOnVzZXJfaG9tZV90OnMwAA==
    LICENSE size=1502 time=1458851690.583562292 xattr.security.selinux=dW5jb25maW5lZF91Om9iamVjdF9yOnVzZXJfaG9tZV90OnMwAA==
    README.md size=2366 mode=0644 time=1459369604.0 xattr.security.selinux=dW5jb25maW5lZF91Om9iamVjdF9yOnVzZXJfaG9tZV90OnMwAA==

[...]
```

See the keyword prefixed with `xattr.` followed by the extended attribute's
namespace and keyword. This setup is consistent for use with Linux extended
attributes as well as FreeBSD extended attributes.

Since extended attributes are an unordered hashmap, this approach allows for
checking each `<namespace>.<key>` individually.

The value is the [base64 encoded][base64] of the value of the particular
extended attribute. Since the values themselves could be raw bytes, this
approach avoids issues with encoding.

### Tar form

```mtree
# .
/set type=file mode=0664 uid=1000 gid=1000
. type=dir mode=0775 tar_time=1468430408.000000000

# samedir
samedir type=dir mode=0775 tar_time=1468000972.000000000
    file2 size=0 tar_time=1467999782.000000000
    file1 size=0 tar_time=1467999781.000000000
    
[...]
```

While `go-mtree` serves mainly as a library for upstream `mtree` support,
`go-mtree` is also compatible with [tar archives][tar] (which is not an upstream feature).
This means that we can now create and validate a manifest by specifying a tar file.
More interestingly, this also means that we can create a manifest from an archive, and then
validate this manifest against a filesystem hierarchy that's on disk, and vice versa.

Notice that for the output of creating a validation manifest from a tar file, the default behavior
for evaluating a notion of time is to use the `tar_time` keyword. In the 
"filesystem hierarchy" format of mtree, `time` is being evaluated with 
nanosecond precision. However, GNU tar truncates a file's modification time
to 1-second precision. That is, if a file's full modification time is 
123456789.123456789, the "tar time" equivalent would be 123456789.000000000.
This way, if you validate a manifest created using a tar file against an
actual root directory, there will be no complaints from `go-mtree` so long as the
1-second precision time of a file in the root directory is the same.


## Usage

To use the Go programming language library, see [the docs][godoc].

To use the command line tool, first [build it](#Building), then the following.


### Create a manifest

This will also include the sha512 digest of the files.

```bash
gomtree -c -K sha512digest -p . > /tmp/root.mtree
```

With a tar file: 

```bash
gomtree -c -K sha512digest -T sometarfile.tar > /tmp/tar.mtree
```

### Validate a manifest

```bash
gomtree -p . -f /tmp/root.mtree
```

With a tar file:

```bash
gomtree -T sometarfile.tar -f /tmp/root.mtree
```

### See the supported keywords

```bash
gomtree -list-keywords
Available keywords:
 uname
 sha1
 sha1digest
 sha256digest
 xattrs (not upstream)
 link (default)
 nlink (default)
 md5digest
 rmd160digest
 mode (default)
 cksum
 md5
 rmd160
 type (default)
 time (default)
 uid (default)
 gid (default)
 sha256
 sha384
 sha512
 xattr (not upstream)
 tar_time (not upstream)
 size (default)
 ripemd160digest
 sha384digest
 sha512digest
```


## Building

Either:

```bash
go get github.com/vbatts/go-mtree/cmd/gomtree
```

or

```bash
git clone git://github.com/vbatts/go-mtree.git $GOPATH/src/github.com/vbatts/go-mtree
cd $GOPATH/src/github.com/vbatts/go-mtree
go build ./cmd/gomtree
```

## Testing

On Linux:
```bash
cd $GOPATH/src/github.com/vbatts/go-mtree
make
```

On FreeBSD:
```bash
cd $GOPATH/src/github.com/vbatts/go-mtree
gmake
```


[mtree(8)]: https://www.freebsd.org/cgi/man.cgi?mtree(8)
[libarchive-formats(5)]: https://www.freebsd.org/cgi/man.cgi?query=libarchive-formats&sektion=5&n=1
[archiecobbs/mtree-port]: https://github.com/archiecobbs/mtree-port
[godoc]: https://godoc.org/github.com/vbatts/go-mtree
[tar]: http://man7.org/linux/man-pages/man1/tar.1.html
[base64]: https://tools.ietf.org/html/rfc4648
