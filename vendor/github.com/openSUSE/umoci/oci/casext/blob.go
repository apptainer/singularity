/*
 * umoci: Umoci Modifies Open Containers' Images
 * Copyright (C) 2016, 2017, 2018 SUSE LLC.
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
	"encoding/json"
	"fmt"
	"io"

	"github.com/openSUSE/umoci/oci/cas"
	"github.com/opencontainers/go-digest"
	ispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

// Blob represents a "parsed" blob in an OCI image's blob store. MediaType
// offers a type-safe way of checking what the type of Data is.
type Blob struct {
	// MediaType is the OCI media type of Data.
	MediaType string

	// Digest is the digest of the parsed image. Note that this does not update
	// if Data is changed (it is the digest that this blob was parsed *from*).
	Digest digest.Digest

	// Data is the "parsed" blob taken from the OCI image's blob store, and is
	// typed according to the media type. The mapping from MIME => type is as
	// follows.
	//
	// ispec.MediaTypeDescriptor => ispec.Descriptor
	// ispec.MediaTypeImageManifest => ispec.Manifest
	// ispec.MediaTypeImageManifestList => ispec.ManifestList
	// ispec.MediaTypeImageLayer => io.ReadCloser
	// ispec.MediaTypeImageLayerGzip => io.ReadCloser
	// ispec.MediaTypeImageLayerNonDistributable => io.ReadCloser
	// ispec.MediaTypeImageLayerNonDistributableGzip => io.ReadCloser
	// ispec.MediaTypeImageConfig => ispec.Image
	// unknown => io.ReadCloser
	Data interface{}
}

func (b *Blob) load(ctx context.Context, engine cas.Engine) error {
	reader, err := engine.GetBlob(ctx, b.Digest)
	if err != nil {
		return errors.Wrap(err, "get blob")
	}

	switch b.MediaType {
	// ispec.MediaTypeDescriptor => ispec.Descriptor
	case ispec.MediaTypeDescriptor:
		defer reader.Close()
		parsed := ispec.Descriptor{}
		if err := json.NewDecoder(reader).Decode(&parsed); err != nil {
			return errors.Wrap(err, "parse MediaTypeDescriptor")
		}
		b.Data = parsed

	// ispec.MediaTypeImageManifest => ispec.Manifest
	case ispec.MediaTypeImageManifest:
		defer reader.Close()
		parsed := ispec.Manifest{}
		if err := json.NewDecoder(reader).Decode(&parsed); err != nil {
			return errors.Wrap(err, "parse MediaTypeImageManifest")
		}
		b.Data = parsed

	// ispec.MediaTypeImageIndex => ispec.Index
	case ispec.MediaTypeImageIndex:
		defer reader.Close()
		parsed := ispec.Index{}
		if err := json.NewDecoder(reader).Decode(&parsed); err != nil {
			return errors.Wrap(err, "parse MediaTypeImageIndex")
		}
		b.Data = parsed

	// ispec.MediaTypeImageConfig => ispec.Image
	case ispec.MediaTypeImageConfig:
		defer reader.Close()
		parsed := ispec.Image{}
		if err := json.NewDecoder(reader).Decode(&parsed); err != nil {
			return errors.Wrap(err, "parse MediaTypeImageConfig")
		}
		b.Data = parsed

	// ispec.MediaTypeImageLayer => io.ReadCloser
	// ispec.MediaTypeImageLayerGzip => io.ReadCloser
	// ispec.MediaTypeImageLayerNonDistributable => io.ReadCloser
	// ispec.MediaTypeImageLayerNonDistributableGzip => io.ReadCloser
	case ispec.MediaTypeImageLayer, ispec.MediaTypeImageLayerNonDistributable,
		ispec.MediaTypeImageLayerGzip, ispec.MediaTypeImageLayerNonDistributableGzip:
		// There isn't anything else we can practically do here.
		b.Data = reader
		return nil

	// unknown => io.ReadCloser()
	default:
		b.Data = reader
		return nil
	}

	if b.Data == nil {
		return fmt.Errorf("[internal error] b.Data was nil after parsing")
	}

	return nil
}

// Close cleans up all of the resources for the opened blob.
func (b *Blob) Close() {
	switch b.MediaType {
	case ispec.MediaTypeImageLayer, ispec.MediaTypeImageLayerNonDistributable,
		ispec.MediaTypeImageLayerGzip, ispec.MediaTypeImageLayerNonDistributableGzip:
		if b.Data != nil {
			b.Data.(io.Closer).Close()
		}
	}
}

// FromDescriptor parses the blob referenced by the given descriptor.
func (e Engine) FromDescriptor(ctx context.Context, descriptor ispec.Descriptor) (*Blob, error) {
	blob := &Blob{
		MediaType: descriptor.MediaType,
		Digest:    descriptor.Digest,
		Data:      nil,
	}

	if err := blob.load(ctx, e); err != nil {
		return nil, errors.Wrap(err, "load")
	}

	return blob, nil
}
