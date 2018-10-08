# Release Procedure


## Tagging The Release
_This file uses `v3.0.0` as an example release_

1. Checkout a release branch using `git checkout upstream/master -b release-3.0`
2. Update the `CHANGELOG.md` [file](../CHANGELOG.md) section `Changes Since v2.6.0` -> `v3.0.0 - [YYYY.MM.DD]` and add new section `Changes Since v3.0.0` to track new changes
3. Make any other necessary changes to the `CHANGELOG.md` [file](../CHANGELOG.md) at this time
4. Commit these changes using `git commit -am "Release v3.0.0"`
5. Tag the release using `git tag -a -m "Singularity v3.0.0" v3.0.0`
6. Push the `release-3.0` branch to `upstream` using `git push upstream release-3.0`, also push the `v3.0.0` tag using `git push upstream v3.0.0`
7. Merge the `upstream/release-3.0` branch into the `upstream/master` branch via a GitHub PR


## Documentation
Ensure that our documentation is up to date:
  - [User Docs](https://www.sylabs.io/guides/3.0/user-guide/) can be edited [here](https://github.com/sylabs/singularity-userdocs)
  - [Admin Docs](https://www.sylabs.io/guides/3.0/admin-guide/) can be edited [here](https://github.com/sylabs/singularity-admindocs)


## Announcements
Release announcements should be made on:
  - GitHub [releases page](https://github.com/sylabs/singularity/releases)
    - Run `make -C builddir/ dist` and attach the generated tarball as an asset to the release
    - **NOTE:** The GitHub release MUST contain a line about the proper installation procedure when installing from the GitHub generated tarballs. Namely, that you must build using `./mconfig [-V version]`
  - Singularity mailing list
  - Singularity [Slack channel](https://www.sylabs.io/join-the-community/)
  - Blog post on [sylabs.io](https://www.sylabs.io/category/labnotes/)
  - Various twitter channels:
    - [@SylabsIO](https://twitter.com/sylabsio)
    - [@SingularityApp](https://twitter.com/singularityapp)
    - etc...
  - LinkedIn
  - etc...
