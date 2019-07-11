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
	"io"
	"io/ioutil"

	ispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

// Blob represents a "parsed" blob in an OCI image's blob store. MediaType
// offers a type-safe way of checking what the type of Data is.
type Blob struct {
	// Descriptor is the {mediatype,digest,length} 3-tuple. Note that this
	// isn't updated if the Data is modified.
	Descriptor ispec.Descriptor

	// Data is the "parsed" blob taken from the OCI image's blob store, and is
	// typed according to the media type. The mapping from MIME => type is as
	// follows.
	//
	// ispec.MediaTypeDescriptor => ispec.Descriptor
	// ispec.MediaTypeImageManifest => ispec.Manifest
	// ispec.MediaTypeImageIndex => ispec.Index
	// ispec.MediaTypeImageLayer => io.ReadCloser
	// ispec.MediaTypeImageLayerGzip => io.ReadCloser
	// ispec.MediaTypeImageLayerNonDistributable => io.ReadCloser
	// ispec.MediaTypeImageLayerNonDistributableGzip => io.ReadCloser
	// ispec.MediaTypeImageConfig => ispec.Image
	// unknown => io.ReadCloser
	Data interface{}
}

func (b *Blob) isParseable() bool {
	return b.Descriptor.MediaType == ispec.MediaTypeDescriptor ||
		b.Descriptor.MediaType == ispec.MediaTypeImageManifest ||
		b.Descriptor.MediaType == ispec.MediaTypeImageIndex ||
		b.Descriptor.MediaType == ispec.MediaTypeImageConfig
}

func (b *Blob) load(ctx context.Context, engine Engine) (Err error) {
	reader, err := engine.GetVerifiedBlob(ctx, b.Descriptor)
	if err != nil {
		return errors.Wrap(err, "get blob")
	}

	if b.isParseable() {
		defer func() {
			if _, err := io.Copy(ioutil.Discard, reader); Err == nil {
				Err = errors.Wrapf(err, "discard trailing %q blob", b.Descriptor.MediaType)
			}
			if err := reader.Close(); Err == nil {
				Err = errors.Wrapf(err, "close %q blob", b.Descriptor.MediaType)
			}
		}()
	}

	switch b.Descriptor.MediaType {
	// ispec.MediaTypeDescriptor => ispec.Descriptor
	case ispec.MediaTypeDescriptor:
		parsed := ispec.Descriptor{}
		if err := json.NewDecoder(reader).Decode(&parsed); err != nil {
			return errors.Wrap(err, "parse MediaTypeDescriptor")
		}
		b.Data = parsed

	// ispec.MediaTypeImageManifest => ispec.Manifest
	case ispec.MediaTypeImageManifest:
		parsed := ispec.Manifest{}
		if err := json.NewDecoder(reader).Decode(&parsed); err != nil {
			return errors.Wrap(err, "parse MediaTypeImageManifest")
		}
		b.Data = parsed

	// ispec.MediaTypeImageIndex => ispec.Index
	case ispec.MediaTypeImageIndex:
		parsed := ispec.Index{}
		if err := json.NewDecoder(reader).Decode(&parsed); err != nil {
			return errors.Wrap(err, "parse MediaTypeImageIndex")
		}
		b.Data = parsed

	// ispec.MediaTypeImageConfig => ispec.Image
	case ispec.MediaTypeImageConfig:
		parsed := ispec.Image{}
		if err := json.NewDecoder(reader).Decode(&parsed); err != nil {
			return errors.Wrap(err, "parse MediaTypeImageConfig")
		}
		b.Data = parsed

	// unknown => io.ReadCloser()
	default:
		fallthrough
	// ispec.MediaTypeImageLayer => io.ReadCloser
	// ispec.MediaTypeImageLayerGzip => io.ReadCloser
	// ispec.MediaTypeImageLayerNonDistributable => io.ReadCloser
	// ispec.MediaTypeImageLayerNonDistributableGzip => io.ReadCloser
	case ispec.MediaTypeImageLayer, ispec.MediaTypeImageLayerNonDistributable,
		ispec.MediaTypeImageLayerGzip, ispec.MediaTypeImageLayerNonDistributableGzip:
		// There isn't anything else we can practically do here.
		b.Data = reader
	}

	if b.Data == nil {
		return errors.Errorf("[internal error] b.Data was nil after parsing")
	}
	return nil
}

// Close cleans up all of the resources for the opened blob.
func (b *Blob) Close() error {
	switch b.Descriptor.MediaType {
	case ispec.MediaTypeImageLayer, ispec.MediaTypeImageLayerNonDistributable,
		ispec.MediaTypeImageLayerGzip, ispec.MediaTypeImageLayerNonDistributableGzip:
		if b.Data != nil {
			return b.Data.(io.Closer).Close()
		}
	}
	return nil
}

// FromDescriptor parses the blob referenced by the given descriptor.
func (e Engine) FromDescriptor(ctx context.Context, descriptor ispec.Descriptor) (*Blob, error) {
	blob := &Blob{
		Descriptor: descriptor,
		Data:       nil,
	}

	if err := blob.load(ctx, e); err != nil {
		return nil, errors.Wrap(err, "load")
	}
	return blob, nil
}
