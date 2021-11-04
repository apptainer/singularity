// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

/*
Package image provides underlying data types for Singularity image
formats. Additionally, all image types will satisfy the ImageFormat{}
interface. This interface will expose all things necessary to use
a Singularity image, whether through OCI or directly.

type ImageFormat interface {
    Root() *spec.Root - Root() returns the OCI compliant root of the
                        Image. This function may perform some action,
                        such as extracting the filesystem to a dir.

}
*/
package image
