// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Package oci provides transparent caching of oci-like refs
package oci

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/containers/image/copy"
	"github.com/containers/image/oci/layout"
	"github.com/containers/image/signature"
	"github.com/containers/image/transports"
	"github.com/containers/image/types"
	"github.com/singularityware/singularity/src/pkg/client/cache"
	"github.com/singularityware/singularity/src/pkg/sylog"
)

// ImageReference wraps containers/image ImageReference type
type ImageReference struct {
	source types.ImageReference
	types.ImageReference
}

// ConvertReference converts a source reference into a cache.ImageReference to cache its blobs
func ConvertReference(src types.ImageReference) (types.ImageReference, error) {
	// Our cache dir is an OCI directory. We are using this as a 'blob pool'
	// storing all incoming containers under unique tags, which are a hash of
	// their source URI.
	cacheTag := fmt.Sprintf("%x", sha256.Sum256([]byte(transports.ImageName(src))))

	c, err := layout.ParseReference(cache.Oci() + ":" + cacheTag)
	if err != nil {
		return nil, err
	}

	return &ImageReference{
		source:         src,
		ImageReference: c,
	}, nil

}

// NewImageSource wraps the cache's oci-layout ref to first download the real source image to the cache
func (t *ImageReference) NewImageSource(ctx context.Context, sys *types.SystemContext) (types.ImageSource, error) {
	policy := &signature.Policy{Default: []signature.PolicyRequirement{signature.NewPRInsecureAcceptAnything()}}
	policyCtx, err := signature.NewPolicyContext(policy)

	// First we are fetching into the cache
	err = copy.Image(context.Background(), policyCtx, t.ImageReference, t.source, &copy.Options{
		ReportWriter: sylog.Writer(),
	})
	if err != nil {
		return nil, err
	}
	return t.ImageReference.NewImageSource(ctx, sys)
}
