# Release Procedure


## Tagging The Release
_This file uses `v3.0.0` as an example release_

1. Checkout a release branch using `git checkout upstream/master -b release-3.0`
2. Update the `VERSION` [file](../VERSION) to contain `v3.0.0`
3. Update the `CHANGELOG.md` [file](../CHANGELOG.md) section `Changes Since v2.6.0` -> `v3.0.0 - [YYYY.MM.DD]` and add new section `Changes Since v3.0.0` to track new changes
4. Make any necessary changes to the `CHANGELOG.md` [file](../CHANGELOG.md) at this time
5. Commit these changes using `git commit -am "Release v3.0.0"`
6. Tag the release using `git tag -a -m "Singularity v3.0.0" v3.0.0`
7. Push the `release-3.0` branch to `upstream` using `git push upstream release-3.0`, also push the `v3.0.0` tag using `git push upstream v3.0.0`
7. Merge the `upstream/release-3.0` branch into the `upstream/master` branch via a GitHub PR
8. Checkout a new branch _e.g. development_ using `git checkout upstream/master -b development` 
9. Update the `VERSION` [file](../VERSION) to contain `v3.1.0-devel`
10. Commit this change using `git commit -am "Update development version to v3.1.0"` 
11. Push the `development` branch to a fork _e.g. origin_ using `git push origin development`
12. Merge the `origin/development` branch into the `upstream/master` branch via a GitHub PR


## Documentation
Ensure that our documentation is up to date:
  - [User Docs](https://www.sylabs.io/guides/latest/user-guide/) can be edited [here](https://github.com/sylabs/singularity-userdocs)
  - [Admin Docs](https://www.sylabs.io/guides/latest/admin-guide/) can be edited [here](https://github.com/sylabs/singularity-admindocs)


## Announcements
Release announcements should be made on:
  - GitHub [releases page](https://github.com/sylabs/singularity/releases)
  - Singularity mailing list
  - Singularity [Slack channel](https://www.sylabs.io/community/)
  - Blog post on [sylabs.io](https://www.sylabs.io/lab-notes/)
  - Various twitter channels:
    - [@SylabsIO](https://twitter.com/sylabsio)
    - [@SingularityApp](https://twitter.com/singularityapp)
    - etc...
  - LinkedIn
  - etc...
