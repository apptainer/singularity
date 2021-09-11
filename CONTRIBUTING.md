# Contributing

## Contributor's Agreement

You are under no obligation whatsoever to provide any bug fixes, patches, or
upgrades to the features, functionality or performance of the source code
("Enhancements") to anyone; however, if you choose to make your Enhancements
available either publicly, or directly to the project, without imposing a
separate written license agreement for such Enhancements, then you hereby grant
the following license: a non-exclusive, royalty-free perpetual license to
install, use, modify, prepare derivative works, incorporate into other computer
software, distribute, and sublicense such enhancements or derivative works
thereof, in binary and source code form.

## Getting Started

When contributing to Singularity, it is important to properly communicate the
gist of the contribution. If it is a simple code or editorial fix, simply
explaining this within the GitHub Pull Request (PR) will suffice. But if this is
a larger fix or Enhancement, you are advised to first discuss the change with
the project leader or developers.

Please note we have a [code of conduct](CODE_OF_CONDUCT.md). Please follow it in
all your interactions with the project members and users.

## Pull Requests (PRs)

1. Essential bug fix PRs should be sent to both master and release branches.
1. Small bug fix and feature enhancement PRs should be sent to master only.
1. Follow the existing code style precedent, especially for C. For Go, you
   will mostly conform to the style and form enforced by the "go fmt" and
   "golint" tools for proper formatting.
1. For any new functionality, please write appropriate go tests that will run as
   part of the Continuous Integration (github workflow actions) system.
1. Make sure that the project's default copyright and header have been included
   in any new source files.
1. Make sure your code passes linting, by running `make check` before submitting
   the PR. We use `golangci-lint` as our linter. You may need to address linting
   errors by:
   - Running `gofumpt -w .` to format all `.go` files. We use
     [gofumpt](https://github.com/mvdan/gofumpt) instead of `gofmt` as it adds
     additional formatting rules which are helpful for clarity.
   - Leaving a function comment on **every** new exported function and package
     that your PR has introduced. To learn about how to properly comment Go
     code, read
     [this post on golang.org](https://golang.org/doc/effective_go.html#commentary)
1. Make sure you have locally tested using `make -C builddir test` and that all
   tests succeed before submitting the PR.
1. If possible, run `make -C builddir testall` locally, after setting the
   environment variables `E2E_DOCKER_USERNAME` and `E2E_DOCKER_PASSWORD`
   appropriately for an authorized Docker Hub account. This is required as
   Singularity's end-to-end tests perform many tests that build from or execute
   docker images. Our CI is authorized to run these tests if you cannot.
1. Ask yourself is the code human understandable? This can be accomplished via a
   clear code style as well as documentation and/or comments.
1. The pull request will be reviewed by others, and finally merged when all
   requirements are met.
1. The `CHANGELOG.md` must be updated for any of the following changes:
   - Renamed commands
   - Deprecated / removed commands
   - Changed defaults / behaviors
   - Backwards incompatible changes
   - New features / functionalities
1. PRs which introduce a new Go dependency to the project via `go get` and
   additions to `go.mod` should explain why the dependency is required.

## Documentation

There are a few places where documentation for the Singularity project lives.
The [changelog](CHANGELOG.md) is where PRs should include documentation if
necessary. When a new release is tagged, the
[user-docs](https://singularity.hpcng.org/user-docs/master/) and
[admin-docs](https://singularity.hpcng.org/admin-docs/master/) will be updated
using the contents of the `CHANGELOG.md` file as reference.

1. The [changelog](CHANGELOG.md) is a place to document **functional**
   differences between versions of Singularity. PRs which require
   documentation must update this file. This should be a document which can be
   used to explain what the new features of each version of Singularity are,
   and should **not** read like a commit log. Once a release is tagged (*e.g.
   v3.0.0*), a new top level section will be made titled **Changes Since
   vX.Y.Z** (*e.g. Changes Since v3.0.0*) where new changes will now be
   documented, leaving the previous section immutable.
1. The [README](README.md) is a place to document critical information for new
   users of Singularity. It should typically not change, but in the case where
   a change is necessary a PR may update it.
1. The [user-docs](https://singularity.hpcng.org/user-docs/master/) should
   document anything pertinent to the usage of Singularity.
1. The [admin-docs](https://singularity.hpcng.org/admin-docs/master/)
   document anything that is pertinent to a system administrator who manages a
   system with Singularity installed.
1. If necessary, changes to the message displayed when running
   `singularity help *` can be made by editing `docs/content.go`.
