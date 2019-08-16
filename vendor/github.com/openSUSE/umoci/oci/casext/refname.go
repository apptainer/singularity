/*
 * umoci: Umoci Modifies Open Containers' Images
 * Copyright (C) 2017, 2018 SUSE LLC.
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
	"regexp"

	"github.com/apex/log"
	ispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

// isKnownMediaType returns whether a media type is known by the spec. This
// probably should be moved somewhere else to avoid going out of date.
func isKnownMediaType(mediaType string) bool {
	return mediaType == ispec.MediaTypeDescriptor ||
		mediaType == ispec.MediaTypeImageManifest ||
		mediaType == ispec.MediaTypeImageIndex ||
		mediaType == ispec.MediaTypeImageLayer ||
		mediaType == ispec.MediaTypeImageLayerGzip ||
		mediaType == ispec.MediaTypeImageLayerNonDistributable ||
		mediaType == ispec.MediaTypeImageLayerNonDistributableGzip ||
		mediaType == ispec.MediaTypeImageConfig
}

// refnameRegex is a regex that only matches reference names that are valid
// according to the OCI specification. See IsValidReferenceName for the EBNF.
var refnameRegex = regexp.MustCompile(`^([A-Za-z0-9]+(([-._:@+]|--)[A-Za-z0-9]+)*)(/([A-Za-z0-9]+(([-._:@+]|--)[A-Za-z0-9]+)*))*$`)

// IsValidReferenceName returns whether the provided annotation value for
// "org.opencontainers.image.ref.name" is actually valid according to the
// OCI specification. This only matches against the MUST requirement, not the
// SHOULD requirement. The EBNF defined in the specification is:
//
//   refname   ::= component ("/" component)*
//   component ::= alphanum (separator alphanum)*
//   alphanum  ::= [A-Za-z0-9]+
//   separator ::= [-._:@+] | "--"
func IsValidReferenceName(refname string) bool {
	return refnameRegex.MatchString(refname)
}

// ResolveReference will attempt to resolve all possible descriptor paths to
// Manifests (or any unknown blobs) that match a particular reference name (if
// descriptors are stored in non-standard blobs, Resolve will be unable to find
// them but will return the top-most unknown descriptor).
// ResolveReference assumes that "reference name" refers to the value of the
// "org.opencontainers.image.ref.name" descriptor annotation. It is recommended
// that if the returned slice of descriptors is greater than zero that the user
// be consulted to resolve the conflict (due to ambiguity in resolution paths).
//
// TODO: How are we meant to implement other restrictions such as the
//       architecture and feature flags? The API will need to change.
func (e Engine) ResolveReference(ctx context.Context, refname string) ([]DescriptorPath, error) {
	// XXX: It should be possible to override this somehow, in case we are
	//      dealing with an image that abuses the image specification in some
	//      way.
	if !IsValidReferenceName(refname) {
		return nil, errors.Errorf("refusing to resolve invalid reference %q", refname)
	}

	index, err := e.GetIndex(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get top-level index")
	}

	// Set of root links that match the given refname.
	var roots []ispec.Descriptor

	// We only consider the case where AnnotationRefName is defined on the
	// top-level of the index tree. While this isn't codified in the spec (at
	// the time of writing -- 1.0.0-rc5) there are some discussions to add this
	// restriction in 1.0.0-rc6.
	for _, descriptor := range index.Manifests {
		// XXX: What should we do if refname == "".
		if descriptor.Annotations[ispec.AnnotationRefName] == refname {
			roots = append(roots, descriptor)
		}
	}

	// The resolved set of descriptors.
	var resolutions []DescriptorPath
	for _, root := range roots {
		// Find all manifests or other blobs that are reachable from the given
		// descriptor.
		if err := e.Walk(ctx, root, func(descriptorPath DescriptorPath) error {
			descriptor := descriptorPath.Descriptor()

			// It is very important that we do not ignore unknown media types
			// here. We only recurse into mediaTypes that are *known* and are
			// also not ispec.MediaTypeImageManifest.
			if isKnownMediaType(descriptor.MediaType) && descriptor.MediaType != ispec.MediaTypeImageManifest {
				return nil
			}

			// Add the resolution and do not recurse any deeper.
			resolutions = append(resolutions, descriptorPath)
			return ErrSkipDescriptor
		}); err != nil {
			return nil, errors.Wrapf(err, "walk %s", root.Digest)
		}
	}

	log.WithFields(log.Fields{
		"refs": resolutions,
	}).Debugf("casext.ResolveReference(%s) got these descriptors", refname)
	return resolutions, nil
}

// XXX: Should the *Reference set of interfaces support DescriptorPath? While
//      it might seem like it doesn't make sense, a DescriptorPath entirely
//      removes ambiguity with regards to which root needs to be operated on.
//      If a user has that information we should provide them a way to use it.

// UpdateReference replaces an existing entry for refname with the given
// descriptor. If there are multiple descriptors that match the refname they
// are all replaced with the given descriptor.
func (e Engine) UpdateReference(ctx context.Context, refname string, descriptor ispec.Descriptor) error {
	// XXX: It should be possible to override this somehow, in case we are
	//      dealing with an image that abuses the image specification in some
	//      way.
	if !IsValidReferenceName(refname) {
		return errors.Errorf("refusing to update invalid reference %q", refname)
	}

	// Get index to modify.
	index, err := e.GetIndex(ctx)
	if err != nil {
		return errors.Wrap(err, "get top-level index")
	}

	// TODO: Handle refname = "".
	var newIndex []ispec.Descriptor
	for _, descriptor := range index.Manifests {
		if descriptor.Annotations[ispec.AnnotationRefName] != refname {
			newIndex = append(newIndex, descriptor)
		}
	}
	if len(newIndex)-len(index.Manifests) > 1 {
		// Warn users if the operation is going to remove more than one references.
		log.Warn("multiple references match the given reference name -- all of them have been replaced due to this ambiguity")
	}

	// Append the descriptor.
	if descriptor.Annotations == nil {
		descriptor.Annotations = map[string]string{}
	}
	descriptor.Annotations[ispec.AnnotationRefName] = refname
	newIndex = append(newIndex, descriptor)

	// Commit to image.
	index.Manifests = newIndex
	if err := e.PutIndex(ctx, index); err != nil {
		return errors.Wrap(err, "replace index")
	}
	return nil
}

// DeleteReference removes all entries in the index that match the given
// refname.
func (e Engine) DeleteReference(ctx context.Context, refname string) error {
	// XXX: It should be possible to override this somehow, in case we are
	//      dealing with an image that abuses the image specification in some
	//      way.
	if !IsValidReferenceName(refname) {
		return errors.Errorf("refusing to delete invalid reference %q", refname)
	}

	// Get index to modify.
	index, err := e.GetIndex(ctx)
	if err != nil {
		return errors.Wrap(err, "get top-level index")
	}

	// TODO: Handle refname = "".
	var newIndex []ispec.Descriptor
	for _, descriptor := range index.Manifests {
		if descriptor.Annotations[ispec.AnnotationRefName] != refname {
			newIndex = append(newIndex, descriptor)
		}
	}
	if len(newIndex)-len(index.Manifests) > 1 {
		// Warn users if the operation is going to remove more than one references.
		log.Warn("multiple references match the given reference name -- all of them have been deleted due to this ambiguity")
	}

	// Commit to image.
	index.Manifests = newIndex
	if err := e.PutIndex(ctx, index); err != nil {
		return errors.Wrap(err, "replace index")
	}
	return nil
}

// ListReferences returns all of the ref.name entries that are specified in the
// top-level index. Note that the list may contain duplicates, due to the
// nature of references in the image-spec.
func (e Engine) ListReferences(ctx context.Context) ([]string, error) {
	// Get index.
	index, err := e.GetIndex(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get top-level index")
	}

	var refs []string
	for _, descriptor := range index.Manifests {
		ref, ok := descriptor.Annotations[ispec.AnnotationRefName]
		if ok {
			refs = append(refs, ref)
		}
	}
	return refs, nil
}
