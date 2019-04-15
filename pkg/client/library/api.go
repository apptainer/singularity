// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/globalsign/mgo/bson"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
)

// HTTP timeout in seconds
const httpTimeout = 10

func getEntity(baseURL string, authToken string, entityRef string) (entity Entity, found bool, err error) {
	url := (baseURL + "/v1/entities/" + entityRef)
	entJSON, found, err := apiGet(url, authToken)
	if err != nil {
		return entity, false, err
	}
	if !found {
		return entity, false, nil
	}
	var res EntityResponse
	if err := json.Unmarshal(entJSON, &res); err != nil {
		return entity, false, fmt.Errorf("error decoding entity: %v", err)
	}
	return res.Data, found, nil
}

func getCollection(baseURL string, authToken string, collectionRef string) (collection Collection, found bool, err error) {
	url := baseURL + "/v1/collections/" + collectionRef
	colJSON, found, err := apiGet(url, authToken)
	if err != nil {
		return collection, false, err
	}
	if !found {
		return collection, false, nil
	}
	var res CollectionResponse
	if err := json.Unmarshal(colJSON, &res); err != nil {
		return collection, false, fmt.Errorf("error decoding collection: %v", err)
	}
	return res.Data, found, nil
}

func getContainer(baseURL string, authToken string, containerRef string) (container Container, found bool, err error) {
	url := baseURL + "/v1/containers/" + containerRef
	conJSON, found, err := apiGet(url, authToken)
	if err != nil {
		return container, false, err
	}
	if !found {
		return container, false, nil
	}
	var res ContainerResponse
	if err := json.Unmarshal(conJSON, &res); err != nil {
		return container, false, fmt.Errorf("error decoding container: %v", err)
	}
	return res.Data, found, nil
}

func getImage(baseURL string, authToken string, imageRef string) (image Image, found bool, err error) {
	url := baseURL + "/v1/images/" + imageRef
	imgJSON, found, err := apiGet(url, authToken)
	if err != nil {
		return image, false, err
	}
	if !found {
		return image, false, nil
	}
	var res ImageResponse
	if err := json.Unmarshal(imgJSON, &res); err != nil {
		return image, false, fmt.Errorf("error decoding image: %v", err)
	}
	return res.Data, found, nil
}

func createEntity(baseURL string, authToken string, name string) (entity Entity, err error) {
	e := Entity{
		Name:        name,
		Description: "No description",
	}
	entJSON, err := apiCreate(e, baseURL+"/v1/entities", authToken)
	if err != nil {
		return entity, err
	}
	var res EntityResponse
	if err := json.Unmarshal(entJSON, &res); err != nil {
		return entity, fmt.Errorf("error decoding entity: %v", err)
	}
	return res.Data, nil
}

func createCollection(baseURL string, authToken string, name string, entityID string) (collection Collection, err error) {
	c := Collection{
		Name:        name,
		Description: "No description",
		Entity:      bson.ObjectIdHex(entityID),
	}
	colJSON, err := apiCreate(c, baseURL+"/v1/collections", authToken)
	if err != nil {
		return collection, err
	}
	var res CollectionResponse
	if err := json.Unmarshal(colJSON, &res); err != nil {
		return collection, fmt.Errorf("error decoding collection: %v", err)
	}
	return res.Data, nil
}

func createContainer(baseURL string, authToken string, name string, collectionID string) (container Container, err error) {
	c := Container{
		Name:        name,
		Description: "No description",
		Collection:  bson.ObjectIdHex(collectionID),
	}
	conJSON, err := apiCreate(c, baseURL+"/v1/containers", authToken)
	if err != nil {
		return container, err
	}
	var res ContainerResponse
	if err := json.Unmarshal(conJSON, &res); err != nil {
		return container, fmt.Errorf("error decoding container: %v", err)
	}
	return res.Data, nil
}

func createImage(baseURL string, authToken string, hash string, containerID string, description string) (image Image, err error) {
	i := Image{
		Hash:        hash,
		Description: description,
		Container:   bson.ObjectIdHex(containerID),
	}
	imgJSON, err := apiCreate(i, baseURL+"/v1/images", authToken)
	if err != nil {
		return image, err
	}
	var res ImageResponse
	if err := json.Unmarshal(imgJSON, &res); err != nil {
		return image, fmt.Errorf("error decoding image: %v", err)
	}
	return res.Data, nil
}

func setTags(baseURL string, authToken string, containerID string, imageID string, tags []string) error {
	// Get existing tags, so we know which will be replaced
	existingTags, err := apiGetTags(baseURL+"/v1/tags/"+containerID, authToken)
	if err != nil {
		return err
	}

	for _, tag := range tags {
		sylog.Infof("Setting tag %s\n", tag)

		if _, ok := existingTags[tag]; ok {
			sylog.Warningf("%s replaces an existing tag\n", tag)
		}

		imgTag := ImageTag{
			tag,
			bson.ObjectIdHex(imageID),
		}
		err := apiSetTag(baseURL+"/v1/tags/"+containerID, authToken, imgTag)
		if err != nil {
			return err
		}
	}
	return nil
}

func search(baseURL string, authToken string, value string) (results SearchResults, err error) {
	u, err := url.Parse(baseURL + "/v1/search")
	if err != nil {
		return
	}
	q := u.Query()
	q.Set("value", value)
	u.RawQuery = q.Encode()

	resJSON, _, err := apiGet(u.String(), authToken)
	if err != nil {
		return results, err
	}

	var res SearchResponse
	if err := json.Unmarshal(resJSON, &res); err != nil {
		return results, fmt.Errorf("error decoding reesults: %v", err)
	}

	return res.Data, nil
}

func apiCreate(o interface{}, url string, authToken string) (objJSON []byte, err error) {
	sylog.Debugf("apiCreate calling %s\n", url)
	s, err := json.Marshal(o)
	if err != nil {
		return []byte{}, fmt.Errorf("error encoding object to JSON:\n\t%v", err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(s))
	req.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
	req.Header.Set("User-Agent", useragent.Value())

	client := &http.Client{
		Timeout: (httpTimeout * time.Second),
	}
	res, err := client.Do(req)
	if err != nil {
		return []byte{}, fmt.Errorf("error making request to server:\n\t%v", err)
	}
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		jRes, err := ParseErrorBody(res.Body)
		if err != nil {
			jRes = ParseErrorResponse(res)
		}
		return []byte{}, fmt.Errorf("creation did not succeed: %d %s\n\t%v",
			jRes.Error.Code, jRes.Error.Status, jRes.Error.Message)
	}
	objJSON, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return []byte{}, fmt.Errorf("error reading response from server:\n\t%v", err)
	}
	return objJSON, nil
}

func apiGet(url string, authToken string) (objJSON []byte, found bool, err error) {
	sylog.Debugf("apiGet calling %s\n", url)
	client := &http.Client{
		Timeout: (httpTimeout * time.Second),
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return []byte{}, false, fmt.Errorf("error creating request to server:\n\t%v", err)
	}
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
	req.Header.Set("User-Agent", useragent.Value())
	res, err := client.Do(req)
	if err != nil {
		return []byte{}, false, fmt.Errorf("error making request to server:\n\t%v", err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return []byte{}, false, nil
	}
	if res.StatusCode == http.StatusOK {
		objJSON, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return []byte{}, false, fmt.Errorf("error reading response from server:\n\t%v", err)
		}
		return objJSON, true, nil
	}
	// Not OK, not 404.... error
	jRes, err := ParseErrorBody(res.Body)
	if err != nil {
		jRes = ParseErrorResponse(res)
	}
	return []byte{}, false, fmt.Errorf("get did not succeed: %d %s\n\t%v",
		jRes.Error.Code, jRes.Error.Status, jRes.Error.Message)
}

func apiGetTags(url string, authToken string) (tags TagMap, err error) {
	sylog.Debugf("apiGetTags calling %s\n", url)
	client := &http.Client{
		Timeout: (httpTimeout * time.Second),
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request to server:\n\t%v", err)
	}
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
	req.Header.Set("User-Agent", useragent.Value())
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request to server:\n\t%v", err)
	}
	if res.StatusCode != http.StatusOK {
		jRes, err := ParseErrorBody(res.Body)
		if err != nil {
			jRes = ParseErrorResponse(res)
		}
		return nil, fmt.Errorf("creation did not succeed: %d %s\n\t%v",
			jRes.Error.Code, jRes.Error.Status, jRes.Error.Message)
	}
	var tagRes TagsResponse
	err = json.NewDecoder(res.Body).Decode(&tagRes)
	if err != nil {
		return tags, fmt.Errorf("error decoding tags: %v", err)
	}
	return tagRes.Data, nil

}

func apiSetTag(url string, authToken string, t ImageTag) (err error) {
	sylog.Debugf("apiSetTag calling %s\n", url)
	s, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("error encoding object to JSON:\n\t%v", err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(s))
	req.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
	req.Header.Set("User-Agent", useragent.Value())
	client := &http.Client{
		Timeout: (httpTimeout * time.Second),
	}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error making request to server:\n\t%v", err)
	}
	if res.StatusCode != http.StatusOK {
		jRes, err := ParseErrorBody(res.Body)
		if err != nil {
			jRes = ParseErrorResponse(res)
		}
		return fmt.Errorf("creation did not succeed: %d %s\n\t%v",
			jRes.Error.Code, jRes.Error.Status, jRes.Error.Message)
	}
	return nil
}

// GetImage returns the Image object if exists, otherwise returns error
func GetImage(baseURL string, authToken string, imageRef string) (image Image, err error) {
	entityName, collectionName, containerName, tags := parseLibraryRef(imageRef)

	i, f, err := getImage(baseURL, authToken, entityName+"/"+collectionName+"/"+containerName+":"+tags[0])
	if err != nil {
		return Image{}, err
	} else if !f {
		return Image{}, fmt.Errorf("image '%s:%s' was not found in the library", containerName, tags[0])
	}

	return i, nil
}
