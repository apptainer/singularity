// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Package oci provides transparent caching of oci-like refs
package oci

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"strings"

	"github.com/containers/image/copy"
	"github.com/containers/image/oci/layout"
	"github.com/containers/image/signature"
	"github.com/containers/image/transports"
	"github.com/containers/image/types"
	"github.com/pkg/errors"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

// ImageReference wraps containers/image ImageReference type
type ImageReference struct {
	source types.ImageReference
	types.ImageReference
}

// ConvertReference converts a source reference into a cache.ImageReference to cache its blobs
func ConvertReference(ctx context.Context, imgCache *cache.Handle, src types.ImageReference, sys *types.SystemContext) (types.ImageReference, error) {
	if imgCache == nil {
		return nil, fmt.Errorf("undefined image cache")
	}

	// Our cache dir is an OCI directory. We are using this as a 'blob pool'
	// storing all incoming containers under unique tags, which are a hash of
	// their source URI.
	cacheTag, err := calculateRefHash(ctx, src, sys)
	if err != nil {
		return nil, err
	}

	c, err := layout.ParseReference(imgCache.OciBlob + ":" + cacheTag)
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
	return t.newImageSource(ctx, sys, sylog.Writer())
}

func (t *ImageReference) newImageSource(ctx context.Context, sys *types.SystemContext, w io.Writer) (types.ImageSource, error) {
	policy := &signature.Policy{Default: []signature.PolicyRequirement{signature.NewPRInsecureAcceptAnything()}}
	policyCtx, err := signature.NewPolicyContext(policy)
	if err != nil {
		return nil, err
	}

	// First we are fetching into the cache
	_, err = copy.Image(ctx, policyCtx, t.ImageReference, t.source, &copy.Options{
		ReportWriter: w,
		SourceCtx:    sys,
	})
	if err != nil {
		return nil, err
	}
	return t.ImageReference.NewImageSource(ctx, sys)
}

// ParseImageName parses a uri (e.g. docker://ubuntu) into it's transport:reference
// combination and then returns the proper reference
func ParseImageName(ctx context.Context, imgCache *cache.Handle, uri string, sys *types.SystemContext) (types.ImageReference, error) {
	ref, err := parseURI(uri)
	if err != nil {
		return nil, fmt.Errorf("unable to parse image name %v: %v", uri, err)
	}

	return ConvertReference(ctx, imgCache, ref, sys)
}

func parseURI(uri string) (types.ImageReference, error) {
	sylog.Debugf("Parsing %s into reference", uri)

	split := strings.SplitN(uri, ":", 2)
	if len(split) != 2 {
		return nil, fmt.Errorf("%s not in transport:reference pair", uri)
	}

	transport := transports.Get(split[0])
	if transport == nil {
		return nil, fmt.Errorf("%s not a registered transport", split[0])
	}

	return transport.ParseReference(split[1])
}

// ImageSHA calculates the SHA of a uri's manifest
func ImageSHA(ctx context.Context, uri string, sys *types.SystemContext) (string, error) {
	ref, err := parseURI(uri)
	if err != nil {
		return "", fmt.Errorf("unable to parse image name %v: %v", uri, err)
	}

	return calculateRefHash(ctx, ref, sys)
}

func calculateRefHash(ctx context.Context, ref types.ImageReference, sys *types.SystemContext) (hash string, err error) {
	source, err := ref.NewImageSource(ctx, sys)
	if err != nil {
		return "", err
	}
	defer func() {
		if closeErr := source.Close(); closeErr != nil {
			err = errors.Wrapf(err, " (src: %v)", closeErr)
		}
	}()

	man, _, err := source.GetManifest(ctx, nil)
	if err != nil {
		return "", err
	}

	hash = fmt.Sprintf("%x", sha256.Sum256(man))
	return hash, nil
}
