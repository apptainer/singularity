// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

// UploadImageRequest is sent to initiate V2 image upload
type UploadImageRequest struct {
	Size        int64  `json:"filesize"`
	MD5Checksum string `json:"md5sum"`
}

// UploadImageCompleteRequest is sent to complete V2 image upload; it is
// currently unused.
type UploadImageCompleteRequest struct {
}
