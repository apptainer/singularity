### `umoci/oci/layer` ###

This is my own implementation of the [currently under development
`oci-create-layer` functions][create-layer]. The reason for implementing this
myself is that we use [`mtree` specifications][mtree] which are not the same
method that `oci-create-layer` uses. While the two implementations could be
combined (since this implementation is more general), in order to speed things
up I just decided to implement it myself.

This also implements `oci-create-runtime-bundle`, since it's under layer
management. The real difference is that we've split up the API (and based it on
CAS) so we have more control when generating the bundle.

I'm hoping that this will be merged upstream, but since it's just a whiteout
tar archive generator there isn't a *huge* requirement that this is kept up to
date. Though, it should be noted that [the whiteout format may change in the
future][whiteout-disc].

[create-layer]: https://github.com/opencontainers/image-tools/pull/8
[mtree]: https://github.com/vbatts/go-mtree
[whiteout-disc]: https://github.com/opencontainers/image-spec/issues/24
