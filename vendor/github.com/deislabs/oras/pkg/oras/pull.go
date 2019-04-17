package oras

import (
	"context"
	"sync"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/remotes"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	orascontent "github.com/deislabs/oras/pkg/content"
)

// Pull pull files from the remote
func Pull(ctx context.Context, resolver remotes.Resolver, ref string, ingester content.Ingester, allowedMediaTypes ...string) ([]ocispec.Descriptor, error) {
	if resolver == nil {
		return nil, ErrResolverUndefined
	}

	_, desc, err := resolver.Resolve(ctx, ref)
	if err != nil {
		return nil, err
	}

	fetcher, err := resolver.Fetcher(ctx, ref)
	if err != nil {
		return nil, err
	}

	return fetchContent(ctx, fetcher, desc, ingester, allowedMediaTypes...)
}

func fetchContent(ctx context.Context, fetcher remotes.Fetcher, desc ocispec.Descriptor, ingester content.Ingester, allowedMediaTypes ...string) ([]ocispec.Descriptor, error) {
	var descriptors []ocispec.Descriptor
	lock := &sync.Mutex{}
	picker := images.HandlerFunc(func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		if isAllowedMediaType(desc.MediaType, allowedMediaTypes...) {
			if name, ok := orascontent.ResolveName(desc); ok && len(name) > 0 {
				lock.Lock()
				defer lock.Unlock()
				descriptors = append(descriptors, desc)
			}
			return nil, nil
		}
		return nil, nil
	})
	store := newHybridStoreFromIngester(ingester)
	handlers := images.Handlers(
		filterHandler(allowedMediaTypes...),
		remotes.FetchHandler(store, fetcher),
		picker,
		images.ChildrenHandler(store),
	)
	if err := images.Dispatch(ctx, handlers, desc); err != nil {
		return nil, err
	}

	return descriptors, nil
}

func filterHandler(allowedMediaTypes ...string) images.HandlerFunc {
	return func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		switch {
		case isAllowedMediaType(desc.MediaType, ocispec.MediaTypeImageManifest, ocispec.MediaTypeImageIndex):
			return nil, nil
		case isAllowedMediaType(desc.MediaType, allowedMediaTypes...):
			if name, ok := orascontent.ResolveName(desc); ok && len(name) > 0 {
				return nil, nil
			}
			log.G(ctx).Warnf("blob_no_name: %v", desc.Digest)
		default:
			log.G(ctx).Warnf("unknown_type: %v", desc.MediaType)
		}
		return nil, images.ErrStopHandler
	}
}

func isAllowedMediaType(mediaType string, allowedMediaTypes ...string) bool {
	if len(allowedMediaTypes) == 0 {
		return true
	}
	for _, allowedMediaType := range allowedMediaTypes {
		if mediaType == allowedMediaType {
			return true
		}
	}
	return false
}
