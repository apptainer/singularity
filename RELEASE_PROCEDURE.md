# Release Procedure

The release procedure below can be performed by a project member with
"maintainer" or higher privileges on the GitHub repository. It assumes
that you will be working in an up-to-date local clone of the GitHub
repository, where the `upstream` remote points to
`github.com/hpcng/singularity`.

## Prior to Release

1. Set a target date for the release candidate (if required) and
   release. Generally 2 weeks from RC -> release is appropriate for
   new 3.X.0 minor versions.
2. Aim to specifically discuss the release timeline and progress in
   community meetings at least 2 months prior to the scheduled date.
3. Use a GitHub milestone to track issues and PRs that will form part
   of the next release.
4. Ensure that the `CHANGELOG.md` is kept up-to-date on the `master`
   branch, with all relevant changes listed under a "Changes Since
   Last Release" section.
5. Monitor and merge dependabot updates, such that a release is made
   with as up-to-date versions of dependencies as possible. This
   lessens the burden in addressing patch release fixes that require
   dependency updates, as we use several dependencies that move
   quickly.

## Creating the Release Branch and Release Candidate

When a new 3.Y.0 minor version of Singularity is issued the release
process begins by branching, and then issuing a release candidate for
broader testing.

When a new 3.Y.Z patch release is issued, the branch will already be
present, and steps 1-2 should be skipped.

1. From a repository that is up-to-date with master, create a release
   branch e.g. `git checkout upstream/master -b release-3.8`.
2. Push the release branch to GitHub via `git push upstream release-3.8`.
3. Examine the GitHub branch protection rules, to extend them to the
   new release branch if needed.
4. Modify the `README.md`, `INSTALL.md`, `CHANGELOG.md` via PR against
   the release branch, so that they reflect the version to be released.
5. Apply an annotated tag via `git tag -a -m "Singularity v3.8.0
   Release Candidate 1" v3.8.0-rc.1`.
6. Push the tag via `git push upstream v3.8.0-rc.1`.
7. Create a tarball via `./mconfig -v && make dist`.
8. Test intallation from the tarball.
9. Compute the sha256sum of the tarball e.g. `sha256sum *.tar.gz > sha256sums`.
10. Create a GitHub release, marked as a 'pre-release', incorporating
   `CHANGELOG.md` information, and attaching the tarball and
   `sha256sums`.
11. Notify the community about the RC via the Google Group and Slack.

There will often be multiple release candidates issued prior to the
final release of a new 3.Y.0 minor version.

A small 3.Y.Z patch release may not require release candidates where
the code changes are contained, confirmed by the person reporting the
bug(s), and well covered by tests.

## Creating a Final Release

1. Ensure the user and admin documentation is up-to-date for the new
   version, branched, and tagged.
  - [User Docs](https://singularity.hpcng.org/user-docs/master/) can be edited [here](https://github.com/hpcng/singularity-userdocs)
  - [Admin Docs](https://singularity.hpcng.org/admin-docs/master/) can be edited [here](https://github.com/hpcng/singularity-admindocs)
2. Ensure the user and admin documentation has been deployed to the
   singularity.hpcng.org website.
4. Modify the `README.md`, `INSTALL.md`, `CHANGELOG.md` via PR against
   the release branch, so that they reflect the version to be released.
5. Apply an annotated tag via `git tag -a -m "Singularity v3.8.0" v3.8.0`.
6. Push the tag via `git push upstream v3.8.0-rc.1`.
7. Create a tarball via `./mconfig -v && make dist`.
8. Test intallation from the tarball.
9. Compute the sha256sum of the tarball e.g. `sha256sum *.tar.gz > sha256sums`.
10. Create a GitHub release, incorporating `CHANGELOG.md` information,
   and attaching the tarball and `sha256sums`.
11. Notify the community about the RC via the Google Group and Slack.

## After the Release

1. Create and merge a PR from the `release-3.x` branch into `master`,
   so that history from the RC process etc. is captured on `master`.
2. If the release is a new major/minor version, move the prior
   `release-3.x` branch to `vault/release-3.x`.
3. If the release is a new major/minor version, update the
   `.github/dependabot.yml` configuration so that dependabot is tracking
   the new stable release branch.
4. Start scheduling / setting up milestones etc. to track the next release!
