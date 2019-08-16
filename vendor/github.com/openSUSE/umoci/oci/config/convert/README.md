### `umoci/oci/config/convert` ###

One fairly important aspect of creating a runtime bundle is the configuration
of the container. While an image configuration and runtime configuration are
defined on different levels (images are far more platform agnostic than runtime
bundles), conversion from an image to a runtime configuration is defined as
part of the OCI specification (thanks to this reference implementation).

This package implements a fairly unopinionated implementation of that
conversion, allowing consumers to easily add their own extensions in the
runtime configuration generation.
