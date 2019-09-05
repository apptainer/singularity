// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"sort"
	"strings"
	"time"
)

// LibraryModels lists names of valid models in the database
var LibraryModels = []string{"Entity", "Collection", "Container", "Image", "Blob"}

// ModelManager - Generic interface for models which must have a bson ObjectID
type ModelManager interface {
	GetID() string
}

// BaseModel - has an ID, soft deletion marker, and Audit struct
type BaseModel struct {
	ModelManager `json:",omitempty"`
	Deleted      bool      `json:"deleted"`
	CreatedBy    string    `json:"createdBy"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedBy    string    `json:"updatedBy,omitempty"`
	UpdatedAt    time.Time `json:"updatedAt,omitempty"`
	DeletedBy    string    `json:"deletedBy,omitempty"`
	DeletedAt    time.Time `json:"deletedAt,omitempty"`
	Owner        string    `json:"owner,omitempty"`
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

// Check BaseModel implements ModelManager at compile time
var _ ModelManager = (*BaseModel)(nil)

// Entity - Top level entry in the library, contains collections of images
// for a user or group
type Entity struct {
	BaseModel
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Collections []string `json:"collections"`
	Size        int64    `json:"size"`
	Quota       int64    `json:"quota"`
	// DefaultPrivate set true will make any new Collections in ths entity
	// private at the time of creation.
	DefaultPrivate bool `json:"defaultPrivate"`
	// CustomData can hold a user-provided string for integration purposes
	// not used by the library itself.
	CustomData string `json:"customData"`
}

// GetID - Convenience method to get model ID if working with an interface
func (e Entity) GetID() string {
	return e.ID
}

// LibraryURI - library:// URI to the entity
func (e Entity) LibraryURI() string {
	return "library://" + e.Name
}

// Collection - Second level in the library, holds a collection of containers
type Collection struct {
	BaseModel
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Entity      string   `json:"entity"`
	Containers  []string `json:"containers"`
	Size        int64    `json:"size"`
	Private     bool     `json:"private"`
	// CustomData can hold a user-provided string for integration purposes
	// not used by the library itself.
	CustomData string `json:"customData"`
	// Computed fields that will not be stored - JSON response use only
	EntityName string `json:"entityName,omitempty"`
}

// GetID - Convenience method to get model ID if working with an interface
func (c Collection) GetID() string {
	return c.ID
}

// LibraryURI - library:// URI to the collection
func (c Collection) LibraryURI() string {
	return "library://" + c.EntityName + "/" + c.Name
}

// Container - Third level of library. Inside a collection, holds images for
// a particular container
type Container struct {
	BaseModel
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	FullDescription string   `json:"fullDescription"`
	Collection      string   `json:"collection"`
	Images          []string `json:"images"`
	// This base TagMap without architecture support is for old clients only
	// (Singularity <=3.3) to preserve non-architecture-aware behavior
	ImageTags TagMap `json:"imageTags"`
	// We now have a 2 level map for new clients, keeping tags per architecture
	ArchTags      ArchTagMap `json:"archTags"`
	Size          int64      `json:"size"`
	DownloadCount int64      `json:"downloadCount"`
	Stars         int        `json:"stars"`
	Private       bool       `json:"private"`
	ReadOnly      bool       `json:"readOnly"`
	// CustomData can hold a user-provided string for integration purposes
	// not used by the library itself.
	CustomData string `json:"customData"`
	// Computed fields that will not be stored - JSON response use only
	Entity         string `json:"entity,omitempty"`
	EntityName     string `json:"entityName,omitempty"`
	CollectionName string `json:"collectionName,omitempty"`
}

// GetID - Convenience method to get model ID if working with an interface
func (c Container) GetID() string {
	return c.ID
}

// LibraryURI - library:// URI to the container
func (c Container) LibraryURI() string {
	return "library://" + c.EntityName + "/" + c.CollectionName + "/" + c.Name
}

// TagList - return a sorted space delimited list of tags
func (c Container) TagList() string {
	var taglist sort.StringSlice
	for tag := range c.ImageTags {
		taglist = append(taglist, tag)
	}
	taglist.Sort()
	return strings.Join(taglist, " ")
}

// Image - Represents a Singularity image held by the library for a particular
// Container
type Image struct {
	BaseModel
	ID           string   `json:"id"`
	Hash         string   `json:"hash"`
	Description  string   `json:"description"`
	Container    string   `json:"container"`
	Blob         string   `json:"blob,omitempty"`
	Size         int64    `json:"size"`
	Uploaded     bool     `json:"uploaded"`
	Signed       *bool    `json:"signed,omitempty"`
	Architecture *string  `json:"arch,omitempty"`
	Fingerprints []string `json:"fingerprints,omitempty"`
	// CustomData can hold a user-provided string for integration purposes
	// not used by the library itself.
	CustomData string `json:"customData"`
	// Computed fields that will not be stored - JSON response use only
	Entity               string   `json:"entity,omitempty"`
	EntityName           string   `json:"entityName,omitempty"`
	Collection           string   `json:"collection,omitempty"`
	CollectionName       string   `json:"collectionName,omitempty"`
	ContainerName        string   `json:"containerName,omitempty"`
	Tags                 []string `json:"tags,omitempty"`
	ContainerDescription string   `json:"containerDescription,omitempty"`
	ContainerStars       int      `json:"containerStars"`
	ContainerDownloads   int64    `json:"containerDownloads"`
}

// GetID - Convenience method to get model ID if working with an interface
func (img Image) GetID() string {
	return img.ID
}

// Blob - Binary data object (e.g. container image file) stored in a Backend
// Uses object store bucket/key semantics
type Blob struct {
	BaseModel
	ID          string `json:"id"`
	Bucket      string `json:"bucket"`
	Key         string `json:"key"`
	Size        int64  `json:"size"`
	ContentHash string `json:"contentHash"`
	Status      string `json:"status"`
}

// GetID - Convenience method to get model ID if working with an interface
func (b Blob) GetID() string {
	return b.ID
}

// ImageTag - A single mapping from a string to bson ID. Not stored in the DB
// but used by API calls setting tags
type ImageTag struct {
	Tag     string
	ImageID string
}

// TagMap is a mapping of a string tag, to an ObjectID that refers to an Image
// e.g. { "latest": 507f1f77bcf86cd799439011 }
type TagMap map[string]string

// ArchImageTag - A simple mapping from a architecture and tag string to bson
// ID. Not stored in the DB but used by API calls setting tags
type ArchImageTag struct {
	Arch    string
	Tag     string
	ImageID string
}

// ArchTagMap is a mapping of a string architecture to a TagMap, and hence to
// Images.
// e.g. {
//			"amd64":    { "latest": 507f1f77bcf86cd799439011 },
//			"ppc64le":  { "latest": 507f1f77bcf86cd799439012 },
//		}
type ArchTagMap map[string]TagMap
