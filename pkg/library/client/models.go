/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.
*/

package client

import (
"fmt"
"time"

"gopkg.in/mgo.v2/bson"
)

// ModelManager - Generic interface for models which must have a bson ObjectID
type ModelManager interface {
	GetID() bson.ObjectId
	setID(id bson.ObjectId) bson.ObjectId
	IsDeleted() bool
	GetCreated() (auditUser string, auditTime time.Time)
	GetUpdated() (auditUser string, auditTime time.Time)
	GetDeleted() (auditUser string, auditTime time.Time)
	setCreated(auditUser string, auditTime ...time.Time)
	setUpdated(auditUser string, auditTime ...time.Time)
	setDeleted(auditUser string, auditTime ...time.Time)
}

// BaseModel - has an ID, soft deletion marker, and Audit struct
type BaseModel struct {
	ModelManager `bson:",omitempty" json:",omitempty"`
	ID           bson.ObjectId `bson:"_id" json:"id"`
	Deleted      bool          `bson:"IsDeleted" json:"IsDeleted"`
	CreatedBy    string        `bson:"CreatedBy" json:"CreatedBy"`
	CreatedAt    time.Time     `bson:"CreatedAt" json:"CreatedAt"`
	UpdatedBy    string        `bson:"UpdatedBy,omitempty" json:"UpdatedBy,omitempty"`
	UpdatedAt    time.Time     `bson:"UpdatedAt,omitempty" json:"UpdatedAt,omitempty"`
	DeletedBy    string        `bson:"DeletedBy,omitempty" json:"DeletedBy,omitempty"`
	DeletedAt    time.Time     `bson:"DeletedAt,omitempty" json:"DeletedAt,omitempty"`
}

// GetID - Convenience method to get model ID if working with an interface
func (m BaseModel) GetID() bson.ObjectId {
	return m.ID
}

// SetID - Convenience method to set model ID if working with an interface
func (m *BaseModel) setID(id bson.ObjectId) bson.ObjectId {
	m.ID = id
	return m.ID
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

func (m *BaseModel) setCreated(auditUser string, auditTime ...time.Time) {
	if len(auditTime) > 0 {
		m.CreatedAt = auditTime[0]
	} else {
		m.CreatedAt = time.Now()
	}
	m.CreatedBy = auditUser
}

func (m *BaseModel) setUpdated(auditUser string, auditTime ...time.Time) {
	if len(auditTime) > 0 {
		m.UpdatedAt = auditTime[0]
	} else {
		m.UpdatedAt = time.Now()
	}
	m.UpdatedBy = auditUser
}

func (m *BaseModel) setDeleted(auditUser string, auditTime ...time.Time) {
	if len(auditTime) > 0 {
		m.DeletedAt = auditTime[0]
	} else {
		m.DeletedAt = time.Now()
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
	Name        string          `bson:"name" json:"name"`
	Description string          `bson:"description" json:"description"`
	Collections []bson.ObjectId `bson:"collections,omitempty" json:"collections,omitempty"`
}

// AddCollection - Associate a collection with an Entity
func (e *Entity) AddCollection(collectionID bson.ObjectId) ([]bson.ObjectId, error) {
	if idInSlice(collectionID, e.Collections) {
		return e.Collections, fmt.Errorf("Collection %s is already a member of entity %s", collectionID.Hex(), e.ID.Hex())
	}
	e.Collections = append(e.Collections, collectionID)
	return e.Collections, nil
}

// RemoveCollection - Dissociate a collection from an Entity
func (e *Entity) RemoveCollection(collectionID bson.ObjectId) ([]bson.ObjectId, error) {
	if !idInSlice(collectionID, e.Collections) {
		return e.Collections, fmt.Errorf("Collection %s is not a member of entity %s", collectionID.Hex(), e.ID.Hex())
	}
	e.Collections = sliceWithoutID(e.Collections, collectionID)
	return e.Collections, nil
}

// Collection - Second level in the library, holds a collection of containers
type Collection struct {
	BaseModel
	Name        string          `bson:"name" json:"name"`
	Description string          `bson:"description" json:"description"`
	Entity      bson.ObjectId   `bson:"entity" json:"entity"`
	Containers  []bson.ObjectId `bson:"containers,omitempty" json:"containers,omitempty"`
}

// AddContainer - Associate a container with a collection
func (c *Collection) AddContainer(containerID bson.ObjectId) ([]bson.ObjectId, error) {
	if idInSlice(containerID, c.Containers) {
		return c.Containers, fmt.Errorf("Collection %s is already a member of entity %s", containerID.Hex(), c.ID.Hex())
	}
	c.Containers = append(c.Containers, containerID)
	return c.Containers, nil
}

// RemoveContainer - Dissociate a container from an Collection
func (c *Collection) RemoveCollection(containerID bson.ObjectId) ([]bson.ObjectId, error) {
	if idInSlice(containerID, c.Containers) {
		return c.Containers, fmt.Errorf("Collection %s is already a member of entity %s", containerID.Hex(), c.ID.Hex())
	}
	c.Containers = sliceWithoutID(c.Containers, containerID)
	return c.Containers, nil
}

// Container - Third level of library. Inside a collection, holds images for
// a particular container
type Container struct {
	BaseModel
	Name        string          `bson:"name" json:"name"`
	Description string          `bson:"description" json:"description"`
	Collection  bson.ObjectId   `bson:"collection" json:"collection"`
	Images      []bson.ObjectId `son:"images,omitempty" json:"images,omitempty"`
}

// AddImage - Associate an image with a container
func (c *Container) AddImage(imageID bson.ObjectId) ([]bson.ObjectId, error) {
	if idInSlice(imageID, c.Images) {
		return c.Images, fmt.Errorf("Image %s is already a member of container %s", imageID.Hex(), c.ID.Hex())
	}
	c.Images = append(c.Images, imageID)
	return c.Images, nil
}

// RemoveImage - Dissociate an image from a container
func (c *Container) RemoveImage(imageID bson.ObjectId) ([]bson.ObjectId, error) {
	if idInSlice(imageID, c.Images) {
		return c.Images, fmt.Errorf("Image %s is already a member of container %s", imageID.Hex(), c.ID.Hex())
	}
	c.Images = sliceWithoutID(c.Images, imageID)
	return c.Images, nil
}

// Image - Represents a Singularity image held by the library for a particular
// Container
type Image struct {
	BaseModel
	Name        string        `bson:"name" json:"name"`
	Description string        `bson:"description" json:"description"`
	Container   bson.ObjectId `bson:"container" json:"container"`
	Blob        bson.ObjectId `bson:"blob" json:"blob"`
	Tags        []string      `bson:"tags" json:"tags"`
}

// Blob - Binary data object (e.g. container image file) stored in a backend
// Uses object store bucket/key semantics
type Blob struct {
	BaseModel
	Bucket      string `bson:"bucket" json:"bucket"`
	Key         string `bson:"key" json:"key"`
	Size        int64  `bson:"size" json:"size"`
	ContentHash string `bson:"contentHash" json:"contentHash"`
	Status      string `bson:"status" json:"status"`
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func idInSlice(a bson.ObjectId, list []bson.ObjectId) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func sliceWithoutID(list []bson.ObjectId, a bson.ObjectId) []bson.ObjectId {

	var newList []bson.ObjectId

	for _, b := range list {
		if b != a {
			newList = append(newList, b)
		}
	}
	return newList
}