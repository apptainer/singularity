package oras

import (
	"context"

	"github.com/containerd/containerd/images"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type pullOpts struct {
	allowedMediaTypes []string
	dispatch          func(context.Context, images.Handler, ...ocispec.Descriptor) error
	baseHandlers      []images.Handler
}

// PullOpt allows callers to set options on the oras pull
type PullOpt func(o *pullOpts) error

func pullOptsDefaults() *pullOpts {
	return &pullOpts{
		dispatch: images.Dispatch,
	}
}

// WithAllowedMediaType sets the allowed media types
func WithAllowedMediaType(allowedMediaTypes ...string) PullOpt {
	return func(o *pullOpts) error {
		o.allowedMediaTypes = append(o.allowedMediaTypes, allowedMediaTypes...)
		return nil
	}
}

// WithAllowedMediaTypes sets the allowed media types
func WithAllowedMediaTypes(allowedMediaTypes []string) PullOpt {
	return func(o *pullOpts) error {
		o.allowedMediaTypes = append(o.allowedMediaTypes, allowedMediaTypes...)
		return nil
	}
}

// WithPullByBFS opt to pull in sequence with breath-first search
func WithPullByBFS(o *pullOpts) error {
	o.dispatch = dispatchBFS
	return nil
}

// WithPullBaseHandler provides base handlers, which will be called before
// any pull specific handlers.
func WithPullBaseHandler(handlers ...images.Handler) PullOpt {
	return func(o *pullOpts) error {
		o.baseHandlers = append(o.baseHandlers, handlers...)
		return nil
	}
}
