// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	jsonresp "github.com/sylabs/json-resp"
)

// EntityResponse - Response from the API for an Entity request
type EntityResponse struct {
	Data  Entity          `json:"data"`
	Error *jsonresp.Error `json:"error,omitempty"`
}

// CollectionResponse - Response from the API for an Collection request
type CollectionResponse struct {
	Data  Collection      `json:"data"`
	Error *jsonresp.Error `json:"error,omitempty"`
}

// ContainerResponse - Response from the API for an Container request
type ContainerResponse struct {
	Data  Container       `json:"data"`
	Error *jsonresp.Error `json:"error,omitempty"`
}

// ImageResponse - Response from the API for an Image request
type ImageResponse struct {
	Data  Image           `json:"data"`
	Error *jsonresp.Error `json:"error,omitempty"`
}

// TagsResponse - Response from the API for a tags request
type TagsResponse struct {
	Data  TagMap          `json:"data"`
	Error *jsonresp.Error `json:"error,omitempty"`
}

// SearchResults - Results structure for searches
type SearchResults struct {
	Entities    []Entity     `json:"entity"`
	Collections []Collection `json:"collection"`
	Containers  []Container  `json:"container"`
	Images      []Image      `json:"image"`
}

// SearchResponse - Response from the API for a search request
type SearchResponse struct {
	Data  SearchResults   `json:"data"`
	Error *jsonresp.Error `json:"error,omitempty"`
}

// UploadImage - Contains requisite data for direct S3 image upload support
type UploadImage struct {
	UploadURL string `json:"uploadURL"`
}

// UploadImageResponse - Response from the API for an image upload request
type UploadImageResponse struct {
	Data  UploadImage     `json:"data"`
	Error *jsonresp.Error `json:"error,omitempty"`
}
