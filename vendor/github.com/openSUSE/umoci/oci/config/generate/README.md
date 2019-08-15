### `umoci/oci/config/generate` ###

This intends to be a library like `runtime-tools/generate` which allows you to
generate modifications to an OCI image configuration blob (of type
[`application/vnd.oci.image.config.v1+json`][oci-image-config]). It's a bit of
a shame that this is necessary, but it shouldn't be *that bad* to implement

The hope is that this library (or some form of it) will become an upstream
library so I don't have to maintain this for any extended period of time.

[oci-image-config]: https://github.com/opencontainers/image-spec/blob/master/config.md
