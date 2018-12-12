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
	"io"
	"strings"

	"github.com/containers/image/copy"
	"github.com/containers/image/oci/layout"
	"github.com/containers/image/signature"
	"github.com/containers/image/transports"
	"github.com/containers/image/types"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

// ImageReference wraps containers/image ImageReference type
type ImageReference struct {
	source types.ImageReference
	types.ImageReference
}

// ConvertReference converts a source reference into a cache.ImageReference to cache its blobs
func ConvertReference(src types.ImageReference, sys *types.SystemContext) (types.ImageReference, error) {
	// Our cache dir is an OCI directory. We are using this as a 'blob pool'
	// storing all incoming containers under unique tags, which are a hash of
	// their source URI.
	cacheTag, err := calculateRefHash(src, sys)
	if err != nil {
		return nil, err
	}

	c, err := layout.ParseReference(cache.OciBlob() + ":" + cacheTag)
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

	// First we are fetching into the cache
	err = copy.Image(context.Background(), policyCtx, t.ImageReference, t.source, &copy.Options{
		ReportWriter: w,
		SourceCtx: &types.SystemContext{
			OCIInsecureSkipTLSVerify:    true,
			DockerInsecureSkipTLSVerify: true,
		},
	})
	if err != nil {
		return nil, err
	}
	return t.ImageReference.NewImageSource(ctx, sys)
}

// ParseImageName parses a uri (e.g. docker://ubuntu) into it's transport:reference
// combination and then returns the proper reference
func ParseImageName(uri string, sys *types.SystemContext) (types.ImageReference, error) {
	ref, err := parseURI(uri)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse image name %v: %v", uri, err)
	}

	return ConvertReference(ref, sys)
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

// TempImageExists returns whether or not the uri exists splatted out in the cache.OciTemp() directory
func TempImageExists(uri string) (bool, string, error) {
	sum, err := ImageSHA(uri, nil)
	if err != nil {
		return false, "", err
	}

	split := strings.Split(uri, ":")
	if len(split) < 2 {
		return false, "", fmt.Errorf("poorly formatted URI %v", uri)
	}

	exists, err := cache.OciTempExists(sum, split[1])
	return exists, cache.OciTempImage(sum, split[1]), err
}

// ImageSHA calculates the SHA of a uri's manifest
func ImageSHA(uri string, sys *types.SystemContext) (string, error) {
	ref, err := parseURI(uri)
	if err != nil {
		return "", fmt.Errorf("Unable to parse image name %v: %v", uri, err)
	}

	return calculateRefHash(ref, sys)
}

func calculateRefHash(ref types.ImageReference, sys *types.SystemContext) (string, error) {
	source, err := ref.NewImageSource(context.TODO(), sys)
	if err != nil {
		return "", err
	}

	man, _, err := source.GetManifest(context.TODO(), nil)
	if err != nil {
		return "", err
	}

	hash := fmt.Sprintf("%x", sha256.Sum256(man))
	return hash, nil
}
