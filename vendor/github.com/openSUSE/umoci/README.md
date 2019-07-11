[![umoci](/contrib/logo/umoci-black.png)][umoci-site]

[![Release](https://img.shields.io/github/release/openSUSE/umoci.svg)](https://github.com/openSUSE/umoci/releases/latest)
[![Build Status](https://img.shields.io/travis/openSUSE/umoci/master.svg)](https://travis-ci.org/openSUSE/umoci)
![License: Apache 2.0](https://img.shields.io/github/license/openSUSE/umoci.svg)

[![Go Report Card](https://goreportcard.com/badge/github.com/openSUSE/umoci)](https://goreportcard.com/report/github.com/openSUSE/umoci)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/1084/badge)](https://bestpractices.coreinfrastructure.org/projects/1084)
[![DOI](https://zenodo.org/badge/72283469.svg)](https://zenodo.org/badge/latestdoi/72283469)

**u**moci **m**odifies **O**pen **C**ontainer **i**mages. See [the official
website][umoci-site] for more accessible documentation about how to use umoci.

umoci intends to be a complete manipulation tool for [OCI images][oci-image-spec].
In particular, it should be seen as a more end-user-focused version of the
[`oci-image-tools` provided by the OCI][oci-image-tools]. The hope is that all
of this tooling will eventually be merged with the upstream repository, so that
it is always kept up-to-date by the Open Container Initiative community.

However, currently there is a [lot][disc-1] [of][disc-2] [discussion][disc-3]
about the new tooling going into the OCI image tools, and right now I need
tooling that can abstract all of the internals of the OCI specification into a
single CLI interface. The main purpose of this tool is to serve as example of
what **I** would like to see in an `oci-image` tool.

If you wish to provide feedback or contribute, read the
[`CONTRIBUTING.md`][contributing] for this project to refresh your knowledge
about how to submit good bug reports and patches. Information about how to
submit responsible security disclosures is also provided.

[umoci-site]: https://umo.ci/
[oci-image-spec]: https://github.com/opencontainers/image-spec
[oci-image-tools]: https://github.com/opencontainers/image-tools
[disc-1]: https://github.com/opencontainers/image-spec/pull/411
[disc-2]: https://github.com/opencontainers/image-tools/pull/5
[disc-3]: https://github.com/opencontainers/image-tools/pull/8
[contributing]: /CONTRIBUTING.md

### Releases ###

We regularly publish [new releases][releases], with each release being given a
unique identifying version number (as governed by [Semantic Versioning
(SemVer)][semver]). Information about previous releases including the list of
new features, bug fixes and resolved security issues is available in the
[change log][changelog]. You can get pre-built binaries and corresponding
source code for each release from the [releases page][releases].

[semver]: http://semver.org/
[changelog]: /CHANGELOG.md
[releases]: https://github.com/openSUSE/umoci/releases

### Installation ###

If you wish to build umoci from source, follow these steps to build in with
[golang](https://golang.org).

```bash
GOPATH=$HOME
go get -d github.com/openSUSE/umoci
cd $GOPATH/src/github.com/openSUSE/umoci
make install
```

Your `umoci` binary will be in `$HOME/bin`.

### Usage ###

umoci has a subcommand-based command-line. For more detailed information, see
the generated man pages (which you can build with `make doc`). You can also
read through our [quick start guide][quickstart].

```
% umoci --help
NAME:
   umoci - umoci modifies Open Container images

USAGE:
   umoci [global options] command [command options] [arguments...]

VERSION:
   0.3.1

AUTHOR(S):
   Aleksa Sarai <asarai@suse.com>

COMMANDS:
     raw      advanced internal image tooling
     help, h  Shows a list of commands or help for one command

   image:
     config      modifies the image configuration of an OCI image
     unpack      unpacks a reference into an OCI runtime bundle
     repack      repacks an OCI runtime bundle into a reference
     new         creates a blank tagged OCI image
     tag         creates a new tag in an OCI image
     remove, rm  removes a tag from an OCI image
     stat        displays status information of an image manifest

   layout:
     gc        garbage-collects an OCI image's blobs
     init      create a new OCI layout
     list, ls  lists the set of tags in an OCI image

GLOBAL OPTIONS:
   --verbose      alias for --log=info
   --log value    set the log level (debug, info, [warn], error, fatal) (default: "warn")
   --help, -h     show help
   --version, -v  print the version
```

[quickstart]: https://umo.ci/quick-start/

### License ###

umoci is licensed under the terms of the Apache 2.0 license.

```
umoci: Umoci Modifies Open Containers' Images
Copyright (C) 2016, 2017, 2018 SUSE LLC.
Copyright (C) 2018 Cisco Systems

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```

### Citation ###

If you have used umoci in your research, please cite it like you would any
other useful software. Here is a handy BibTex citation.

```
@misc{umoci,
	title = {umoci - Standalone Tool For Manipulating Container Images},
	author = {Aleksa Sarai et al.},
	year = {2016},
	url = {https://umo.ci/},
	doi = {http://dx.doi.org/10.5281/zenodo.1188474},
}
```

Thank you.
