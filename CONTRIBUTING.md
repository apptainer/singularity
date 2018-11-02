# Contributor's Agreement

You are under no obligation whatsoever to provide any bug fixes, patches,
or upgrades to the features, functionality or performance of the source
code ("Enhancements") to anyone; however, if you choose to make your
Enhancements available either publicly, or directly to the project,
without imposing a separate written license agreement for such
Enhancements, then you hereby grant the following license: a non-exclusive,
royalty-free perpetual license to install, use, modify, prepare derivative
works, incorporate into other computer software, distribute, and sublicense
such enhancements or derivative works thereof, in binary and source code
form.


# Contributing

When contributing to Singularity, it is important to properly communicate the
gist of the contribution. If it is a simple code or editorial fix, simply
explaining this within the GitHub Pull Request (PR) will suffice. But if this
is a larger fix or Enhancement, you are advised to first discuss the change
with the project leader or developers.

Please note we have a code of conduct, described below. Please follow it in
all your interactions with the project members and users.

## Pull Requests (PRs)

1. Essential bug fix PRs should be sent to both master and release branches.
2. Small bug fix and feature enhancement PRs should be sent to master only.
3. Follow the existing code style precedent, especially for C. For Golang, you
   will mostly conform to the style and form enforced by the "go fmt" and
   "golint" tools for proper formatting.
4. Ensure any install or build dependencies are removed before doing a build
   to test your PR locally.
5. For any new functionality, please write appropriate go tests that will run
   as part of the Continuous Integration (Circle CI and Travis) systems.
6. Make sure that the project's default copyright and header have been included 
   in any new source files.
7. Make sure you have locally tested using `make test` and that all tests succeed
   before submitting the PR.
8. To conform to the Golang standards and idioms, make sure you have done the 
   following:
    - Run `go fmt ./...` to format all `.go` files. We use `go1.11`'s 
      formatting as our standard
    - Left a function comment on **every** new exported function and package that
      your PR has introduced. To learn about how to properly comment Golang code, read [this post on golang.org](https://golang.org/doc/effective_go.html?#commentary)
9. Is the code human understandable? This can be accomplished via a clear code
   style as well as documentation and/or comments.
10. The pull request will be reviewed by others, and finally merged when all
   requirements are met.
11. The `CHANGELOG.md` must be updated for any of the following changes:
    - Renamed commands
    - Deprecated / removed commands
    - Changed defaults / behaviors
    - Backwards incompatible changes
    - New features / functionalities
12. PRs which introduce a new Golang dependency to the project via `dep` must include a justification for introducing the dependency. Ideally, newly introduced dependencies should also be pinned to a specific version.

## Documentation
There are a few places where documentation for the Singularity project lives. The [changelog](CHANGELOG.md) is where PRs should include documentation if necessary. When a new release is tagged, the [user-docs](https://www.sylabs.io/guides/3.0/user-guide/) and [admin-docs](https://www.sylabs.io/guides/3.0/admin-guide/) will be updated using the contents of the `CHANGELOG.md` file as reference.

1. The [changelog](CHANGELOG.md) is a place to document **functional** differences between versions of Singularity. PRs which require documentation must update this file. This should be a document which can be used to explain what the new features of each version of Singularity are, and should **not** read like a commit log. Once a release is tagged (*e.g. v3.0.0*), a new top level section will be made titled **Changes Since vX.Y.Z** (*e.g. Changes Since v3.0.0*) where new changes will now be documented, leaving the previous section immutable.
2. The [README](README.md) is a place to document critical information for new users of Singularity. It should typically not change, but in the case where a change is necessary a PR may update it.
3. The [user-docs](https://www.github.com/sylabs/singularity-userdocs) should document anything pertinent to the usage of Singularity.
4. The [admin-docs](https://www.github.com/sylabs/singularity-admindocs) document anything that is pertinent to a system administrator who manages a system with Singularity installed.
5. If necessary, changes to the message displayed when running `singularity help *` can be made by editing `src/docs/content.go`.


# Code of Conduct

## Our Pledge

In the interest of fostering an open and welcoming environment, we as
contributors and maintainers pledge to making participation in our project and
our community a harassment-free experience for everyone, regardless of age, body
size, disability, ethnicity, gender identity and expression, level of experience,
nationality, personal appearance, race, religion, or sexual identity and
orientation.

## Our Standards

Examples of behavior that contributes to creating a positive environment
include:

* Using welcoming and inclusive language
* Being respectful of differing viewpoints and experiences
* Gracefully accepting constructive criticism
* Focusing on what is best for the community
* Showing empathy towards other community members

Examples of unacceptable behavior by participants include:

* The use of sexualized language or imagery and unwelcome sexual attention or
  advances
* Trolling, insulting/derogatory comments, and personal or political attacks
* Public or private harassment
* Publishing others' private information, such as a physical or electronic
  address, without explicit permission
* Other conduct which could reasonably be considered inappropriate in a
  professional setting

### Our Responsibilities

Project maintainers are responsible for clarifying the standards of acceptable
behavior and are expected to take appropriate and fair corrective action in
response to any instances of unacceptable behavior.

Project maintainers have the right and responsibility to remove, edit, or
reject comments, commits, code, wiki edits, issues, and other contributions
that are not aligned to this Code of Conduct, or to ban temporarily or
permanently any contributor for other behaviors that they deem inappropriate,
threatening, offensive, or harmful.

## Scope

This Code of Conduct applies both within project spaces and in public spaces
when an individual is representing the project or its community. Examples of
representing a project or community include using an official project e-mail
address, posting via an official social media account, or acting as an appointed
representative at an online or offline event. Representation of a project may be
further defined and clarified by project maintainers.

## Enforcement

Instances of abusive, harassing, or otherwise unacceptable behavior may be
reported by contacting the project leader (gmkurtzer@gmail.com). All
complaints will be reviewed and investigated and will result in a response
that is deemed necessary and appropriate to the circumstances. The project
team is obligated to maintain confidentiality with regard to the reporter of
an incident. Further details of specific enforcement policies may be posted
separately.

Project maintainers, contributors and users who do not follow or enforce the
Code of Conduct in good faith may face temporary or permanent repercussions 
with their involvement in the project as determined by the project's leader(s).

## Attribution

This Code of Conduct is adapted from the [Contributor Covenant][homepage], version 1.4,
available at [http://contributor-covenant.org/version/1/4][version]

[homepage]: http://contributor-covenant.org
[version]: http://contributor-covenant.org/version/1/4/
