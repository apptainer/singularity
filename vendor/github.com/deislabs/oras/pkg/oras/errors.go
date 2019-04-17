package oras

import "errors"

// Common errors
var (
	ErrResolverUndefined = errors.New("resolver_undefined")
	ErrEmptyDescriptors  = errors.New("empty_descriptors")
)

// Path validation related errors
var (
	ErrDirtyPath               = errors.New("dirty_path")
	ErrPathNotSlashSeparated   = errors.New("path_not_slash_separated")
	ErrAbsolutePathDisallowed  = errors.New("absolute_path_disallowed")
	ErrPathTraversalDisallowed = errors.New("path_traversal_disallowed")
)
