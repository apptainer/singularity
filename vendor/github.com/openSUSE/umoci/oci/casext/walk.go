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
	"errors"

	"github.com/apex/log"
	"github.com/openSUSE/umoci/oci/cas"
	"github.com/opencontainers/go-digest"
	ispec "github.com/opencontainers/image-spec/specs-go/v1"
	"golang.org/x/net/context"
)

// childDescriptors is a wrapper around MapDescriptors which just creates a
// slice of all of the arguments, and doesn't modify them.
func childDescriptors(i interface{}) []ispec.Descriptor {
	var children []ispec.Descriptor
	if err := MapDescriptors(i, func(descriptor ispec.Descriptor) ispec.Descriptor {
		children = append(children, descriptor)
		return descriptor
	}); err != nil {
		// If we got an error, this is a bug in MapDescriptors proper.
		log.Fatalf("[internal error] MapDescriptors returned an error inside childDescriptors: %+v", err)
	}
	return children
}

// walkState stores state information about the recursion into a given
// descriptor tree.
type walkState struct {
	// engine is the CAS engine we are operating on.
	engine Engine

	// walkFunc is the WalkFunc provided by the user.
	walkFunc WalkFunc
}

// DescriptorPath is used to describe the path of descriptors (from a top-level
// index) that were traversed when resolving a particular reference name. The
// purpose of this is to allow libraries like github.com/openSUSE/umoci/mutate
// to handle generic manifest updates given an arbitrary descriptor walk. Users
// of ResolveReference that don't care about the descriptor path can just use
// .Descriptor.
type DescriptorPath struct {
	// Walk is the set of descriptors walked to reach Descriptor (inclusive).
	// The order is the same as the order of the walk, with the target being
	// the last entry and the entrypoint from index.json being the first.
	Walk []ispec.Descriptor `json:"descriptor_walk"`
}

// Root returns the first step in the DescriptorPath, which is the point where
// the walk started. This is just shorthand for DescriptorPath.Walk[0]. Root
// will *panic* if DescriptorPath is invalid.
func (d DescriptorPath) Root() ispec.Descriptor {
	if len(d.Walk) < 1 {
		panic("empty DescriptorPath")
	}
	return d.Walk[0]
}

// Descriptor returns the final step in the DescriptorPath, which is the target
// descriptor being referenced by DescriptorPath. This is just shorthand for
// accessing the last entry of DescriptorPath.Walk. Descriptor will *panic* if
// DescriptorPath is invalid.
func (d DescriptorPath) Descriptor() ispec.Descriptor {
	if len(d.Walk) < 1 {
		panic("empty DescriptorPath")
	}
	return d.Walk[len(d.Walk)-1]
}

// ErrSkipDescriptor is a special error returned by WalkFunc which will cause
// Walk to not recurse into the descriptor currently being evaluated by
// WalkFunc.  This interface is roughly equivalent to filepath.SkipDir.
var ErrSkipDescriptor = errors.New("[internal] do not recurse into descriptor")

// WalkFunc is the type of function passed to Walk. It will be a called on each
// descriptor encountered, recursively -- which may involve the function being
// called on the same descriptor multiple times (though because an OCI image is
// a Merkle tree there will never be any loops). If an error is returned by
// WalkFunc, the recursion will halt and the error will bubble up to the
// caller.
//
// TODO: Also provide Blob to WalkFunc so that callers don't need to load blobs
//       more than once. This is quite important for remote CAS implementations.
type WalkFunc func(descriptorPath DescriptorPath) error

func (ws *walkState) recurse(ctx context.Context, descriptorPath DescriptorPath) error {
	log.WithFields(log.Fields{
		"digest": descriptorPath.Descriptor().Digest,
	}).Debugf("-> ws.recurse")
	defer log.WithFields(log.Fields{
		"digest": descriptorPath.Descriptor().Digest,
	}).Debugf("<- ws.recurse")

	// Run walkFunc.
	if err := ws.walkFunc(descriptorPath); err != nil {
		if err == ErrSkipDescriptor {
			return nil
		}
		return err
	}

	// Get blob to recurse into.
	descriptor := descriptorPath.Descriptor()
	blob, err := ws.engine.FromDescriptor(ctx, descriptor)
	if err != nil {
		// Ignore cases where the descriptor points to an object we don't know
		// how to parse.
		if err == cas.ErrUnknownType {
			log.Infof("skipping walk into unknown media-type %v of blob %v", descriptor.MediaType, descriptor.Digest)
			return nil
		}
		return err
	}
	defer blob.Close()

	// Recurse into children.
	for _, child := range childDescriptors(blob.Data) {
		if err := ws.recurse(ctx, DescriptorPath{
			Walk: append(descriptorPath.Walk, child),
		}); err != nil {
			return err
		}
	}

	return nil
}

// Walk preforms a depth-first walk from a given root descriptor, using the
// provided CAS engine to fetch all other necessary descriptors. If an error is
// returned by the provided WalkFunc, walking is terminated and the error is
// returned to the caller.
func (e Engine) Walk(ctx context.Context, root ispec.Descriptor, walkFunc WalkFunc) error {
	ws := &walkState{
		engine:   e,
		walkFunc: walkFunc,
	}
	return ws.recurse(ctx, DescriptorPath{
		Walk: []ispec.Descriptor{root},
	})
}

// Paths returns the set of descriptor paths that can be traversed from the
// provided root descriptor. It is effectively shorthand for Walk(). Note that
// there may be repeated descriptors in the returned slice, due to different
// blobs containing the same (or a similar) descriptor. However, the
// DescriptorPaths should be unique.
func (e Engine) Paths(ctx context.Context, root ispec.Descriptor) ([]DescriptorPath, error) {
	var reachable []DescriptorPath
	err := e.Walk(ctx, root, func(descriptorPath DescriptorPath) error {
		reachable = append(reachable, descriptorPath)
		return nil
	})
	return reachable, err
}

// Reachable returns the set of digests which can be reached using a descriptor
// path from the provided root descriptor. It is effectively a shorthand for
// Walk(). The returned slice will *not* contain any duplicate digest.Digest
// entries. Note that without descriptors, a digest is not particularly
// meaninful (OCI blobs are not self-descriptive).
func (e Engine) Reachable(ctx context.Context, root ispec.Descriptor) ([]digest.Digest, error) {
	seen := map[digest.Digest]struct{}{}

	if err := e.Walk(ctx, root, func(descriptorPath DescriptorPath) error {
		seen[descriptorPath.Descriptor().Digest] = struct{}{}
		return nil
	}); err != nil {
		return nil, err
	}

	var reachable []digest.Digest
	for node := range seen {
		reachable = append(reachable, node)
	}
	return reachable, nil
}
