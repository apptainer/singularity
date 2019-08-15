# How to do releases:

* Create a changeset with an update to `version.go`
 - this commit will be tagged
 - add another commit putting it back with '-dev' appended
* gpg sign the commit with an incremented version, like 'vX.Y.Z'
* Push the tag
* Create a "release" from the tag on github
 - include the binaries from `make build.arches`
 - write about notable changes, and their contributors
 - PRs merged for the release
