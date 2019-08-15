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
	"github.com/apex/log"
	"github.com/opencontainers/go-digest"
	ispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

// GC will perform a mark-and-sweep garbage collection of the OCI image
// referenced by the given CAS engine. The root set is taken to be the set of
// references stored in the image, and all blobs not reachable by following a
// descriptor path from the root set will be removed.
//
// GC will only call ListBlobs and ListReferences once, and assumes that there
// is no change in the set of references or blobs after calling those
// functions. In other words, it assumes it is the only user of the image that
// is making modifications. Things will not go well if this assumption is
// challenged.
func (e Engine) GC(ctx context.Context) error {
	// Generate the root set of descriptors.
	var root []ispec.Descriptor

	names, err := e.ListReferences(ctx)
	if err != nil {
		return errors.Wrap(err, "get roots")
	}

	for _, name := range names {
		// TODO: This code is no longer necessary once we have index.json.
		descriptorPaths, err := e.ResolveReference(ctx, name)
		if err != nil {
			return errors.Wrapf(err, "get root %s", name)
		}
		if len(descriptorPaths) == 0 {
			return errors.Errorf("tag not found: %s", name)
		}
		if len(descriptorPaths) != 1 {
			// TODO: Handle this more nicely.
			return errors.Errorf("tag is ambiguous: %s", name)
		}
		descriptor := descriptorPaths[0].Descriptor()
		log.WithFields(log.Fields{
			"name":   name,
			"digest": descriptor.Digest,
		}).Debugf("GC: got reference")
		root = append(root, descriptor)
	}

	// Mark from the root sets.
	black := map[digest.Digest]struct{}{}
	for idx, descriptor := range root {
		log.WithFields(log.Fields{
			"digest": descriptor.Digest,
		}).Debugf("GC: marking from root")

		reachables, err := e.Reachable(ctx, descriptor)
		if err != nil {
			return errors.Wrapf(err, "getting reachables from root %d", idx)
		}
		for _, reachable := range reachables {
			black[reachable] = struct{}{}
		}
	}

	// Sweep all blobs in the white set.
	blobs, err := e.ListBlobs(ctx)
	if err != nil {
		return errors.Wrap(err, "get blob list")
	}

	n := 0
	for _, digest := range blobs {
		if _, ok := black[digest]; ok {
			// Digest is in the black set.
			continue
		}
		log.Infof("garbage collecting blob: %s", digest)

		if err := e.DeleteBlob(ctx, digest); err != nil {
			return errors.Wrapf(err, "remove unmarked blob %s", digest)
		}
		n++
	}

	// Finally, tell CAS to GC it.
	if err := e.Clean(ctx); err != nil {
		return errors.Wrapf(err, "clean engine")
	}

	log.Debugf("garbage collected %d blobs", n)
	return nil
}
