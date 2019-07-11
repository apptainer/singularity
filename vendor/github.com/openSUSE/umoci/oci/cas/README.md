### `umoci/oci/cas` ###

This is a reimplemented version of the currently in-flight [`image-tools` CAS
PR][cas-pr], which combines the `cas` and `refs` interfaces into a single
`Engine` that represents the image. In addition, I've implemented more
auto-detection and creature comforts.

When the PR is merged, these changes will probably go upstream as well.

[cas-pr]: https://github.com/opencontainers/image-tools/pull/5
