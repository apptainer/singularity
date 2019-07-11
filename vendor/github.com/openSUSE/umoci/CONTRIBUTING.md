## Contribution Guidelines ##

If you're reading this, you're likely interested in contributing to this
project. That's great! The intention of this document is to describe the basic
requirements and rules-of-thumb for contributions.

### Security Issues ###

If you are reporting a security issue, do not create an issue or file a pull
request on GitHub. Instead, disclose the issue responsibly by sending an email
to <mailto:cyphar@cyphar.com>. If you feel it is necessary you may also encrypt
your email with [Pretty Good Privacy (PGP)][pgp] using the PGP key
[`6FA1B3E3F9A18CDCBE6A2CF54A7BE7BF70DE9B9F`][pgp-key]. *In future, the above
email will be replaced with a mailing list as part of our ongoing effort to
reduce the bus factor of this project.*

[pgp]: https://en.wikipedia.org/wiki/Pretty_Good_Privacy
[pgp-key]: http://pgp.mit.edu/pks/lookup?op=vindex&search=0x6FA1B3E3F9A18CDCBE6A2CF54A7BE7BF70DE9B9F

### Issues ###

If you have found a bug in this project or have a question, first make sure
that the issue you are facing has not already been reported by another user. If
the issue you are facing has already been reported and you have more
information to provide, feel free to add a follow-up comment (but avoid adding
"me too" style comments as it distracts from discussion). If you couldn't find
an existing report for your issue, feel free to [open a new issue][issue-new].
If you do not wish to use proprietary software to submit an issue, you may send
an email to <mailto:cyphar@cyphar.com> and I will submit an issue on your
behalf.

When reporting an issue, please provide the following information (to the best
of your ability) so we can debug your issue far more easily:

* The version of this project you are using. If you are not using the latest
  version of this project, please try to reproduce your issue on the latest
  version.

* A (short) description of what you are trying to accomplish so as to avoid the
  [XY problem][xy-problem].

* A minimal example of the bug with a contrast between what you expect to
  happen versus what actually happened.

[issue-new]: https://github.com/openSUSE/umoci/issues/new
[xy-problem]: http://xyproblem.info/

### Submitting Changes ###

In order to submit a change, you may [create a pull request][pr-new].  If you
do not wish to use proprietary software to submit an pull request, you may send
an email to <mailto:cyphar@cyphar.com> and I will submit a pull request on your
behalf.

All changes should be based off the latest commit of the master branch of this
project. In order for a change to be merged into this project, it must fulfil
all of the following requirements (note that many of these only apply for major
changes):

* All changes must pass the automated testing and continuous integration. This
  means they must build successfully without errors, must not produce errors
  from static analysis and must not break existing functionality. You can run
  all of these tests on your local machine if you wish by reading through
  `.travis.yml` and running the listed commands.

* All changes must be formatted using the Go style conventions, which ensures
  that code remains consistent. You can automatically format your code in any
  given `file.go` using `go fmt -s -w file.go`.

* Any significant changes (such as those that implement a feature or fix a bug)
  must include an entry in the top-level [`CHANGELOG.md`][changelog] (see the
  file for more details) that describes the change and links to the pull
  request that implemented it (as well as issues that are being resolved).

* Any feature change or bug fix should include one or more corresponding test
  cases to ensure that the code is operating as intended. Significant features
  warrant the addition of significant numbers of both integration and unit
  tests.

* Any feature change should include a corresponding change to the project
  documentation describing the feature and how it should be used.

If you miss any of the above things, don't worry we'll remind you and provide
help if you need any. In addition to the above requirements, your code will be
reviewed by the maintainer(s) of this project, using the looks-good-to-me
system (LGTM). All patches must have the approval of at least two maintainers
that did not author a change before they are merged (the only exception to this
is related to the approval of security patches -- which must be approved in
private instead -- and cases where there are not enough maintainers to fulfil
this requirement).

Each commit should be self-contained and minimal (and should build and pass the
tests individually), and commit messages should follow the Linux kernel style
of commit messages. For more information see [&sect; 2 and 3 of
`submitting-patches.rst` from the Linux kernel source][lk-commit].

In addition, all commits must include a `Signed-off-by:` line in their
description. This indicates that you certify [the following statement, known as
the Developer Certificate of Origin][dco]). You can automatically add this line
to your commits by using `git commit -s --amend`.

```
Developer Certificate of Origin
Version 1.1

Copyright (C) 2004, 2006 The Linux Foundation and its contributors.
1 Letterman Drive
Suite D4700
San Francisco, CA, 94129

Everyone is permitted to copy and distribute verbatim copies of this
license document, but changing it is not allowed.


Developer's Certificate of Origin 1.1

By making a contribution to this project, I certify that:

(a) The contribution was created in whole or in part by me and I
    have the right to submit it under the open source license
    indicated in the file; or

(b) The contribution is based upon previous work that, to the best
    of my knowledge, is covered under an appropriate open source
    license and I have the right under that license to submit that
    work with modifications, whether created in whole or in part
    by me, under the same open source license (unless I am
    permitted to submit under a different license), as indicated
    in the file; or

(c) The contribution was provided directly to me by some other
    person who certified (a), (b) or (c) and I have not modified
    it.

(d) I understand and agree that this project and the contribution
    are public and that a record of the contribution (including all
    personal information I submit with it, including my sign-off) is
    maintained indefinitely and may be redistributed consistent with
    this project or the open source license(s) involved.
```

[pr-new]: https://github.com/openSUSE/umoci/compare
[changelog]: /CHANGELOG.md
[lk-commit]: https://www.kernel.org/doc/Documentation/process/submitting-patches.rst
[dco]: https://developercertificate.org/
