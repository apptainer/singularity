/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package config

import (
	"fmt"
)

type RuntimeOciAnnotations interface {
	Get() map[string]string
	Set(annotations map[string]string) error
	Add(key string, value string) error
	Del(key string) error
}

type DefaultRuntimeOciAnnotations struct {
	RuntimeOciSpec *RuntimeOciSpec
}

func (c *DefaultRuntimeOciAnnotations) init() {
	if c.RuntimeOciSpec.Annotations == nil {
		c.RuntimeOciSpec.Annotations = make(map[string]string)
	}
}

func (c *DefaultRuntimeOciAnnotations) Get() map[string]string {
	c.init()
	return c.RuntimeOciSpec.Annotations
}

func (c *DefaultRuntimeOciAnnotations) Set(annotations map[string]string) error {
	c.RuntimeOciSpec.Annotations = annotations
	return nil
}

func (c *DefaultRuntimeOciAnnotations) Add(key string, value string) error {
	c.init()
	if c.RuntimeOciSpec.Annotations[key] != "" {
		return fmt.Errorf("key %s already set", key)
	}
	c.RuntimeOciSpec.Annotations[key] = value
	return nil
}

func (c *DefaultRuntimeOciAnnotations) Del(key string) error {
	c.init()
	if c.RuntimeOciSpec.Annotations[key] == "" {
		return fmt.Errorf("key %s doesn't exists", key)
	}
	delete(c.RuntimeOciSpec.Annotations, key)
	return nil
}
