/*
 * umoci: Umoci Modifies Open Containers' Images
 * Copyright (C) 2018 SUSE LLC.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package casext

import (
	"context"
	"io"

	"github.com/openSUSE/umoci/pkg/hardening"
	ispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// GetVerifiedBlob returns a VerifiedReadCloser for retrieving a blob from the
// image, which the caller must Close() *and* read-to-EOF (checking the error
// code of both). Returns ErrNotExist if the digest is not found, and
// ErrBlobDigestMismatch on a mismatched blob digest. In addition, the reader
// is limited to the descriptor.Size.
func (e Engine) GetVerifiedBlob(ctx context.Context, descriptor ispec.Descriptor) (io.ReadCloser, error) {
	reader, err := e.GetBlob(ctx, descriptor.Digest)
	return &hardening.VerifiedReadCloser{
		Reader:         reader,
		ExpectedDigest: descriptor.Digest,
		ExpectedSize:   descriptor.Size,
	}, err
}
