// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package mount

import (
	"fmt"
)

// hookFn describes function prototype for function
// to be called before/after mounting a tag list
type hookFn func(*System) error

// mountFn describes function prototype for function responsible
// of mount operation
type mountFn func(*Point, *System) error

// System defines a mount system allowing to register before/after
// hook functions for specific tag during mount phase
type System struct {
	Points         *Points
	Mount          mountFn
	currentTag     AuthorizedTag
	beforeTagHooks map[AuthorizedTag][]hookFn
	afterTagHooks  map[AuthorizedTag][]hookFn
}

func (b *System) init() {
	if b.beforeTagHooks == nil {
		b.beforeTagHooks = make(map[AuthorizedTag][]hookFn)
	}
	if b.afterTagHooks == nil {
		b.afterTagHooks = make(map[AuthorizedTag][]hookFn)
	}
}

// RunBeforeTag registers a hook function executed before mounting points
// of tag list
func (b *System) RunBeforeTag(tag AuthorizedTag, fn hookFn) error {
	if _, ok := authorizedTags[tag]; !ok {
		return fmt.Errorf("tag %s is not an authorized tag", tag)
	}
	b.init()
	b.beforeTagHooks[tag] = append(b.beforeTagHooks[tag], fn)
	return nil
}

// RunAfterTag registers a hook function executed after mounting points
// of tag list
func (b *System) RunAfterTag(tag AuthorizedTag, fn hookFn) error {
	if _, ok := authorizedTags[tag]; !ok {
		return fmt.Errorf("tag %s is not an authorized tag", tag)
	}
	b.init()
	b.afterTagHooks[tag] = append(b.afterTagHooks[tag], fn)
	return nil
}

// CurrentTag returns the tag being processed by MountAll.
func (b *System) CurrentTag() AuthorizedTag {
	return b.currentTag
}

// MountAll iterates over mount point list and mounts every point
// by calling hook before/after hook functions
func (b *System) MountAll() error {
	b.init()
	for _, tag := range GetTagList() {
		b.currentTag = tag
		for _, fn := range b.beforeTagHooks[tag] {
			if err := fn(b); err != nil {
				return fmt.Errorf("hook function for tag %s returns error: %s", tag, err)
			}
		}
		for _, point := range b.Points.GetByTag(tag) {
			if b.Mount != nil {
				if err := b.Mount(&point, b); err != nil {
					return fmt.Errorf("mount %s->%s error: %s", point.Source, point.Destination, err)
				}
			}
		}
		for _, fn := range b.afterTagHooks[tag] {
			if err := fn(b); err != nil {
				return fmt.Errorf("hook function for tag %s returns error: %s", tag, err)
			}
		}
	}
	return nil
}
