// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	jsonresp "github.com/sylabs/json-resp"
)

// getEntity returns the specified entity; returns ErrNotFound if entity is not
// found, otherwise error
func (c *Client) getEntity(ctx context.Context, entityRef string) (*Entity, error) {
	url := "/v1/entities/" + entityRef
	entJSON, err := c.apiGet(ctx, url)
	if err != nil {
		return nil, err
	}
	var res EntityResponse
	if err := json.Unmarshal(entJSON, &res); err != nil {
		return nil, fmt.Errorf("error decoding entity: %v", err)
	}
	return &res.Data, nil
}

// getCollection returns the specified collection; returns ErrNotFound if
// collection is not found, otherwise error.
func (c *Client) getCollection(ctx context.Context, collectionRef string) (*Collection, error) {
	url := "/v1/collections/" + collectionRef
	colJSON, err := c.apiGet(ctx, url)
	if err != nil {
		return nil, err
	}
	var res CollectionResponse
	if err := json.Unmarshal(colJSON, &res); err != nil {
		return nil, fmt.Errorf("error decoding collection: %v", err)
	}
	return &res.Data, nil
}

// getContainer returns container by ref id; returns ErrNotFound if container
// is not found, otherwise error.
func (c *Client) getContainer(ctx context.Context, containerRef string) (*Container, error) {
	url := "/v1/containers/" + containerRef
	conJSON, err := c.apiGet(ctx, url)
	if err != nil {
		return nil, err
	}
	var res ContainerResponse
	if err := json.Unmarshal(conJSON, &res); err != nil {
		return nil, fmt.Errorf("error decoding container: %v", err)
	}
	return &res.Data, nil
}

// createEntity creates an entity (must be authorized)
func (c *Client) createEntity(ctx context.Context, name string) (*Entity, error) {
	e := Entity{
		Name:        name,
		Description: "No description",
	}
	entJSON, err := c.apiCreate(ctx, "/v1/entities", e)
	if err != nil {
		return nil, err
	}
	var res EntityResponse
	if err := json.Unmarshal(entJSON, &res); err != nil {
		return nil, fmt.Errorf("error decoding entity: %v", err)
	}
	return &res.Data, nil
}

// createCollection creates a new collection
func (c *Client) createCollection(ctx context.Context, name string, entityID string) (*Collection, error) {
	newCollection := Collection{
		Name:        name,
		Description: "No description",
		Entity:      entityID,
	}
	colJSON, err := c.apiCreate(ctx, "/v1/collections", newCollection)
	if err != nil {
		return nil, err
	}
	var res CollectionResponse
	if err := json.Unmarshal(colJSON, &res); err != nil {
		return nil, fmt.Errorf("error decoding collection: %v", err)
	}
	return &res.Data, nil
}

// createContainer creates a container in the specified collection
func (c *Client) createContainer(ctx context.Context, name string, collectionID string) (*Container, error) {
	newContainer := Container{
		Name:        name,
		Description: "No description",
		Collection:  collectionID,
	}
	conJSON, err := c.apiCreate(ctx, "/v1/containers", newContainer)
	if err != nil {
		return nil, err
	}
	var res ContainerResponse
	if err := json.Unmarshal(conJSON, &res); err != nil {
		return nil, fmt.Errorf("error decoding container: %v", err)
	}
	return &res.Data, nil
}

// createImage creates a new image
func (c *Client) createImage(ctx context.Context, hash string, containerID string, description string) (*Image, error) {
	i := Image{
		Hash:        hash,
		Description: description,
		Container:   containerID,
	}
	imgJSON, err := c.apiCreate(ctx, "/v1/images", i)
	if err != nil {
		return nil, err
	}
	var res ImageResponse
	if err := json.Unmarshal(imgJSON, &res); err != nil {
		return nil, fmt.Errorf("error decoding image: %v", err)
	}
	return &res.Data, nil
}

// setTags applies tags to the specified container
func (c *Client) setTags(ctx context.Context, containerID, imageID string, tags []string) error {
	// Get existing tags, so we know which will be replaced
	existingTags, err := c.getTags(ctx, containerID)
	if err != nil {
		return err
	}

	for _, tag := range tags {
		c.Logger.Logf("Setting tag %s", tag)

		if _, ok := existingTags[tag]; ok {
			c.Logger.Logf("%s replaces an existing tag", tag)
		}

		imgTag := ImageTag{
			tag,
			imageID,
		}
		err := c.setTag(ctx, containerID, imgTag)
		if err != nil {
			return err
		}
	}
	return nil
}

// getTags returns a tag map for the specified containerID
func (c *Client) getTags(ctx context.Context, containerID string) (TagMap, error) {
	url := fmt.Sprintf("/v1/tags/%s", containerID)
	c.Logger.Logf("getTags calling %s", url)
	req, err := c.newRequest(http.MethodGet, url, "", nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request to server:\n\t%v", err)
	}
	res, err := c.HTTPClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("error making request to server:\n\t%v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		err := jsonresp.ReadError(res.Body)
		if err != nil {
			return nil, fmt.Errorf("creation did not succeed: %v", err)
		}
		return nil, fmt.Errorf("unexpected http status code: %d", res.StatusCode)
	}
	var tagRes TagsResponse
	err = json.NewDecoder(res.Body).Decode(&tagRes)
	if err != nil {
		return nil, fmt.Errorf("error decoding tags: %v", err)
	}
	return tagRes.Data, nil
}

// setTag sets tag on specified containerID
func (c *Client) setTag(ctx context.Context, containerID string, t ImageTag) error {
	url := "/v1/tags/" + containerID
	c.Logger.Logf("setTag calling %s", url)
	s, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("error encoding object to JSON:\n\t%v", err)
	}
	req, err := c.newRequest("POST", url, "", bytes.NewBuffer(s))
	if err != nil {
		return fmt.Errorf("error creating POST request:\n\t%v", err)
	}
	res, err := c.HTTPClient.Do(req.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("error making request to server:\n\t%v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		err := jsonresp.ReadError(res.Body)
		if err != nil {
			return fmt.Errorf("creation did not succeed: %v", err)
		}
		return fmt.Errorf("creation did not succeed: http status code: %d", res.StatusCode)
	}
	return nil
}

// GetImage returns the Image object if exists; returns ErrNotFound if image is
// not found, otherwise error.
func (c *Client) GetImage(ctx context.Context, imageRef string) (*Image, error) {
	url := "/v1/images/" + imageRef
	imgJSON, err := c.apiGet(ctx, url)
	if err != nil {
		return nil, err
	}
	var res ImageResponse
	if err := json.Unmarshal(imgJSON, &res); err != nil {
		return nil, fmt.Errorf("error decoding image: %v", err)
	}
	return &res.Data, nil
}
