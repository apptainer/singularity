package oras

import (
	"context"

	orascontent "github.com/deislabs/oras/pkg/content"

	"github.com/containerd/containerd/content"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// ensure interface
var (
	_ content.Provider = &hybridStore{}
	_ content.Ingester = &hybridStore{}
)

type hybridStore struct {
	cache    *orascontent.Memorystore
	provider content.Provider
	ingester content.Ingester
}

func newHybridStoreFromProvider(provider content.Provider) *hybridStore {
	return &hybridStore{
		cache:    orascontent.NewMemoryStore(),
		provider: provider,
	}
}

func newHybridStoreFromIngester(ingester content.Ingester) *hybridStore {
	return &hybridStore{
		cache:    orascontent.NewMemoryStore(),
		ingester: ingester,
	}
}

func (s *hybridStore) Set(desc ocispec.Descriptor, content []byte) {
	s.cache.Set(desc, content)
}

// ReaderAt provides contents
func (s *hybridStore) ReaderAt(ctx context.Context, desc ocispec.Descriptor) (content.ReaderAt, error) {
	readerAt, err := s.cache.ReaderAt(ctx, desc)
	if err == nil {
		return readerAt, nil
	}
	if s.provider != nil {
		return s.provider.ReaderAt(ctx, desc)
	}
	return nil, err
}

// Writer begins or resumes the active writer identified by desc
func (s *hybridStore) Writer(ctx context.Context, opts ...content.WriterOpt) (content.Writer, error) {
	var wOpts content.WriterOpts
	for _, opt := range opts {
		if err := opt(&wOpts); err != nil {
			return nil, err
		}
	}

	if isAllowedMediaType(wOpts.Desc.MediaType, ocispec.MediaTypeImageManifest, ocispec.MediaTypeImageIndex) || s.ingester == nil {
		return s.cache.Writer(ctx, opts...)
	}
	return s.ingester.Writer(ctx, opts...)
}
