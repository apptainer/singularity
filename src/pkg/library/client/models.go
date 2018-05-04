/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package client

import (
	"fmt"
	"time"

	"github.com/globalsign/mgo/bson"
)

// LibraryModels lists names of valid models in the database
var LibraryModels = []string{"Entity", "Collection", "Container", "Image", "Blob"}

// ModelManager - Generic interface for models which must have a bson ObjectID
type ModelManager interface {
	GetID() bson.ObjectId
	SetID(id bson.ObjectId) bson.ObjectId
	IsDeleted() bool
	GetCreated() (auditUser string, auditTime time.Time)
	GetUpdated() (auditUser string, auditTime time.Time)
	GetDeleted() (auditUser string, auditTime time.Time)
	SetCreated(auditUser string, auditTime ...time.Time)
	SetUpdated(auditUser string, auditTime ...time.Time)
	SetDeleted(auditUser string, auditTime ...time.Time)
}

// BaseModel - has an ID, soft deletion marker, and Audit struct
type BaseModel struct {
	ModelManager `bson:",omitempty" json:",omitempty"`
	Deleted      bool      `bson:"IsDeleted" json:"IsDeleted"`
	CreatedBy    string    `bson:"CreatedBy" json:"CreatedBy"`
	CreatedAt    time.Time `bson:"CreatedAt" json:"CreatedAt"`
	UpdatedBy    string    `bson:"UpdatedBy,omitempty" json:"UpdatedBy,omitempty"`
	UpdatedAt    time.Time `bson:"UpdatedAt,omitempty" json:"UpdatedAt,omitempty"`
	DeletedBy    string    `bson:"DeletedBy,omitempty" json:"DeletedBy,omitempty"`
	DeletedAt    time.Time `bson:"DeletedAt,omitempty" json:"DeletedAt,omitempty"`
}

// IsDeleted - Convenience method to check soft deletion state if working with
// an interface
func (m BaseModel) IsDeleted() bool {
	return m.Deleted
}

// GetCreated - Convenience method to get creation stamps if working with an
// interface
func (m BaseModel) GetCreated() (auditUser string, auditTime time.Time) {
	return m.CreatedBy, m.CreatedAt
}

// GetUpdated - Convenience method to get update stamps if working with an
// interface
func (m BaseModel) GetUpdated() (auditUser string, auditTime time.Time) {
	return m.UpdatedBy, m.UpdatedAt
}

// GetDeleted - Convenience method to get deletino stamps if working with an
// interface
func (m BaseModel) GetDeleted() (auditUser string, auditTime time.Time) {
	return m.DeletedBy, m.DeletedAt
}

// SetCreated - Convenience method to set creation stamps if working with an
// interface
func (m *BaseModel) SetCreated(auditUser string, auditTime ...time.Time) {
	if len(auditTime) > 0 {
		m.CreatedAt = auditTime[0]
	} else {
		m.CreatedAt = BsonUTCNow()
	}
	m.CreatedBy = auditUser
}

// SetUpdated - Convenience method to set update stamps if working with an
// interface
func (m *BaseModel) SetUpdated(auditUser string, auditTime ...time.Time) {
	if len(auditTime) > 0 {
		m.UpdatedAt = auditTime[0]
	} else {
		m.UpdatedAt = BsonUTCNow()
	}
	m.UpdatedBy = auditUser
}

// SetDeleted - Convenience method to set deletino stamps if working with an
// interface
func (m *BaseModel) SetDeleted(auditUser string, auditTime ...time.Time) {
	if len(auditTime) > 0 {
		m.DeletedAt = auditTime[0]
	} else {
		m.DeletedAt = BsonUTCNow()
	}
	m.DeletedBy = auditUser
	m.Deleted = true
}

// Check BaseModel implements ModelManager at compile time
var _ ModelManager = (*BaseModel)(nil)

// Entity - Top level entry in the library, contains collections of images
// for a user or group
type Entity struct {
	BaseModel
	ID          bson.ObjectId   `bson:"_id" json:"id"`
	Name        string          `bson:"name" json:"name"`
	Description string          `bson:"description" json:"description"`
	Collections []bson.ObjectId `bson:"collections" json:"collections"`
}

// GetID - Convenience method to get model ID if working with an interface
func (e Entity) GetID() bson.ObjectId {
	return e.ID
}

// SetID - Convenience method to set model ID if working with an interface
func (e *Entity) SetID(id bson.ObjectId) bson.ObjectId {
	e.ID = id
	return e.ID
}

// AddCollection - Associate a collection with an Entity
func (e *Entity) AddCollection(collectionID bson.ObjectId) ([]bson.ObjectId, error) {
	if IDInSlice(collectionID, e.Collections) {
		return e.Collections, fmt.Errorf("Collection %s is already a member of entity %s", collectionID.Hex(), e.ID.Hex())
	}
	e.Collections = append(e.Collections, collectionID)
	return e.Collections, nil
}

// RemoveCollection - Dissociate a collection from an Entity
func (e *Entity) RemoveCollection(collectionID bson.ObjectId) ([]bson.ObjectId, error) {
	if !IDInSlice(collectionID, e.Collections) {
		return e.Collections, fmt.Errorf("Collection %s is not a member of entity %s", collectionID.Hex(), e.ID.Hex())
	}
	e.Collections = SliceWithoutID(e.Collections, collectionID)
	return e.Collections, nil
}

// Collection - Second level in the library, holds a collection of containers
type Collection struct {
	BaseModel
	ID          bson.ObjectId   `bson:"_id" json:"id"`
	Name        string          `bson:"name" json:"name"`
	Description string          `bson:"description" json:"description"`
	Entity      bson.ObjectId   `bson:"entity" json:"entity"`
	Containers  []bson.ObjectId `bson:"containers" json:"containers"`
}

// GetID - Convenience method to get model ID if working with an interface
func (c Collection) GetID() bson.ObjectId {
	return c.ID
}

// SetID - Convenience method to set model ID if working with an interface
func (c *Collection) SetID(id bson.ObjectId) bson.ObjectId {
	c.ID = id
	return c.ID
}

// AddContainer - Associate a container with a collection
func (c *Collection) AddContainer(containerID bson.ObjectId) ([]bson.ObjectId, error) {
	if IDInSlice(containerID, c.Containers) {
		return c.Containers, fmt.Errorf("Container %s is already a member of entity %s", containerID.Hex(), c.ID.Hex())
	}
	c.Containers = append(c.Containers, containerID)
	return c.Containers, nil
}

// RemoveContainer - Dissociate a container from an Collection
func (c *Collection) RemoveContainer(containerID bson.ObjectId) ([]bson.ObjectId, error) {
	if !IDInSlice(containerID, c.Containers) {
		return c.Containers, fmt.Errorf("Collection %s is not a member of entity %s", containerID.Hex(), c.ID.Hex())
	}
	c.Containers = SliceWithoutID(c.Containers, containerID)
	return c.Containers, nil
}

// Container - Third level of library. Inside a collection, holds images for
// a particular container
type Container struct {
	BaseModel
	ID          bson.ObjectId            `bson:"_id" json:"id"`
	Name        string                   `bson:"name" json:"name"`
	Description string                   `bson:"description" json:"description"`
	Collection  bson.ObjectId            `bson:"collection" json:"collection"`
	Images      []bson.ObjectId          `bson:"images" json:"images"`
	ImageTags   map[string]bson.ObjectId `bson:"imageTags" json:"imageTags"`
}

// GetID - Convenience method to get model ID if working with an interface
func (c Container) GetID() bson.ObjectId {
	return c.ID
}

// SetID - Convenience method to set model ID if working with an interface
func (c *Container) SetID(id bson.ObjectId) bson.ObjectId {
	c.ID = id
	return c.ID
}

// AddImage - Associate an image with a container
func (c *Container) AddImage(imageID bson.ObjectId) ([]bson.ObjectId, error) {
	if IDInSlice(imageID, c.Images) {
		return c.Images, fmt.Errorf("Image %s is already a member of container %s", imageID.Hex(), c.ID.Hex())
	}
	c.Images = append(c.Images, imageID)
	return c.Images, nil
}

// RemoveImage - Dissociate an image from a container
func (c *Container) RemoveImage(imageID bson.ObjectId) ([]bson.ObjectId, error) {
	if !IDInSlice(imageID, c.Images) {
		return c.Images, fmt.Errorf("Image %s is not a member of container %s", imageID.Hex(), c.ID.Hex())
	}
	c.Images = SliceWithoutID(c.Images, imageID)
	return c.Images, nil
}

// GetImageTag gets the image ID associated with a provided tag
func (c *Container) GetImageTag(tag string) (imgID bson.ObjectId, err error) {
	if !IsTag(tag) {
		return "", fmt.Errorf("Not a valid tag: %s", tag)
	}
	if c.ImageTags == nil {
		return "", fmt.Errorf("No tags exist on container %s", c.ID.Hex())
	}
	imgID, ok := c.ImageTags[tag]
	if !ok {
		return "", fmt.Errorf("Tag %s does not exist on container %s", tag, c.ID.Hex())
	}
	return imgID, nil
}

// SetImageTag adds or sets a tag to point to a particular image
func (c *Container) SetImageTag(tag string, imgID bson.ObjectId) error {
	if !IsTag(tag) {
		return fmt.Errorf("Not a valid tag: %s", tag)
	}
	if !IDInSlice(imgID, c.Images) {
		return fmt.Errorf("Image %s is not associated with container %s", imgID.Hex(), c.ID.Hex())
	}
	if c.ImageTags == nil {
		c.ImageTags = map[string]bson.ObjectId{}
	}
	c.ImageTags[tag] = imgID
	return nil
}

// DeleteImageTag removes an image tag
func (c *Container) DeleteImageTag(tag string) error {
	if !IsTag(tag) {
		return fmt.Errorf("Not a valid tag: %s", tag)
	}
	_, ok := c.ImageTags[tag]
	if c.ImageTags == nil {
		return fmt.Errorf("No tags exist on container %s", c.ID.Hex())
	}
	if !ok {
		return fmt.Errorf("Tag %s does not exist on container %s", tag, c.ID.Hex())
	}
	delete(c.ImageTags, tag)
	return nil
}

// Image - Represents a Singularity image held by the library for a particular
// Container
type Image struct {
	BaseModel
	ID          bson.ObjectId `bson:"_id" json:"id"`
	Hash        string        `bson:"hash" json:"hash"`
	Description string        `bson:"description" json:"description"`
	Container   bson.ObjectId `bson:"container" json:"container"`
	Blob        bson.ObjectId `bson:"blob,omitempty" json:"blob,omitempty"`
	Size        int64         `bson:"size" json:"size"`
}

// GetID - Convenience method to get model ID if working with an interface
func (img Image) GetID() bson.ObjectId {
	return img.ID
}

// SetID - Convenience method to set model ID if working with an interface
func (img *Image) SetID(id bson.ObjectId) bson.ObjectId {
	img.ID = id
	return img.ID
}

// Blob - Binary data object (e.g. container image file) stored in a Backend
// Uses object store bucket/key semantics
type Blob struct {
	BaseModel
	ID          bson.ObjectId `bson:"_id" json:"id"`
	Bucket      string        `bson:"bucket" json:"bucket"`
	Key         string        `bson:"key" json:"key"`
	Size        int64         `bson:"size" json:"size"`
	ContentHash string        `bson:"contentHash" json:"contentHash"`
	Status      string        `bson:"status" json:"status"`
}

// GetID - Convenience method to get model ID if working with an interface
func (b Blob) GetID() bson.ObjectId {
	return b.ID
}

// SetID - Convenience method to set model ID if working with an interface
func (b *Blob) SetID(id bson.ObjectId) bson.ObjectId {
	b.ID = id
	return b.ID
}

// ImageTag - A simple mapping from a string to bson ID. Not stored in the DB
// but used by API calls setting tags
type ImageTag struct {
	Tag     string
	ImageID bson.ObjectId
}
