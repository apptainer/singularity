// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package config

import (
	"fmt"
)

// RuntimeOciAnnotations describes the methods required for an OCI annotations implementation.
type RuntimeOciAnnotations interface {
	Get() map[string]string
	Set(annotations map[string]string) error
	Add(key string, value string) error
	Del(key string) error
}

// DefaultRuntimeOciAnnotations describes the default runtime OCI annotations.
type DefaultRuntimeOciAnnotations struct {
	RuntimeOciSpec *RuntimeOciSpec
}

func (c *DefaultRuntimeOciAnnotations) init() {
	if c.RuntimeOciSpec.Annotations == nil {
		c.RuntimeOciSpec.Annotations = make(map[string]string)
	}
}

// Get retrieves the runtime annotations.
func (c *DefaultRuntimeOciAnnotations) Get() map[string]string {
	c.init()
	return c.RuntimeOciSpec.Annotations
}

// Set sets the runtime annotations.
func (c *DefaultRuntimeOciAnnotations) Set(annotations map[string]string) error {
	c.RuntimeOciSpec.Annotations = annotations
	return nil
}

// Add adds the supplied key/value to the runtime annotations.
func (c *DefaultRuntimeOciAnnotations) Add(key string, value string) error {
	c.init()
	if c.RuntimeOciSpec.Annotations[key] != "" {
		return fmt.Errorf("key %s already set", key)
	}
	c.RuntimeOciSpec.Annotations[key] = value
	return nil
}

// Del deletes the supplied key from the runtime annotations.
func (c *DefaultRuntimeOciAnnotations) Del(key string) error {
	c.init()
	if c.RuntimeOciSpec.Annotations[key] == "" {
		return fmt.Errorf("key %s doesn't exists", key)
	}
	delete(c.RuntimeOciSpec.Annotations, key)
	return nil
}
