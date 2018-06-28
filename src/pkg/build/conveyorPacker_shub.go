// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// Copyright (c) 2018, Vanessa Sochat. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	oci "github.com/containers/image/oci/layout"
	"github.com/containers/image/types"
	"github.com/singularityware/singularity/src/pkg/sylog"
)

const shubHTTPAddr string = `www.singularity-hub.org/api/container`
const defaultRegistry string = `singularity-hub.org/api/container/`

type shubAPIResponse struct {
	Image   string `json:"image"`
	Name    string `json:"name"`
	Tag     string `json:"tag"`
	Version string `json:"version"`
}

// ShubClient contains the uri and client, lowercase means not exported
type shubClient struct {
	Client   http.Client
	ImageURI string
	HTTPAddr string
}

// ShubConveyor holds data to be packed into a bundle.
type ShubConveyor struct {
	recipe   Definition
	src      string
	srcRef   types.ImageReference
	srcURI   ShubURI
	tmpfs    string
	tmpfsRef types.ImageReference
	tmpfile  string
}

// ShubConveyorPacker only needs to hold the conveyor to have the needed data to pack
type ShubConveyorPacker struct {
	ShubConveyor
	lp LocalPacker
}

// ShubURI stores the various components of a singularityhub URI
type ShubURI struct {
	registry   string
	user       string
	container  string
	tag        string
	digest     string
	defaultReg bool
}

// Get downloads container from Singularityhub
func (c *ShubConveyor) Get(recipe Definition) (err error) {
	sylog.Debugf("Getting container from Shub")

	c.recipe = recipe

	//use custom parser to make sure we have a valid shub URI
	c.srcURI, err = shubParseReference(recipe.Header["from"])
	if err != nil {
		sylog.Fatalf("Invalid shub URI: %v", err)
		return
	}

	c.tmpfs, err = ioutil.TempDir("", "temp-shub-")
	if err != nil {
		return
	}

	c.tmpfsRef, err = oci.ParseReference(c.tmpfs + ":" + "tmp")
	if err != nil {
		return
	}

	// Get the image manifest
	manifest, err := c.getManifest()

	// retrieve the image
	c.tmpfile, err = c.fetch(manifest.Image)
	if err != nil {
		sylog.Fatalf("Failed to get image from SHub: %v", err)
		return
	}

	return nil
}

// Pack puts relevant objects in a Bundle!
// After image is downloaded, we can use a local packer
func (cp *ShubConveyorPacker) Pack() (b *Bundle, err error) {
	//pack from our temporary downloaded image to our tmpfs
	cp.lp = LocalPacker{cp.tmpfile, cp.tmpfs}

	b, err = cp.lp.Pack()
	if err != nil {
		sylog.Errorf("Local Pack failed", err.Error())
		return nil, err
	}

	b.Recipe = cp.recipe

	return b, nil
}

// Download an image from Singularity Hub, writing as we download instead
// of storing in memory
func (c *ShubConveyor) fetch(url string) (image string, err error) {

	// Create temporary download name
	tmpfile, err := ioutil.TempFile(c.tmpfs, "shub-container")
	sylog.Debugf("\nCreating temporary image file %v\n", tmpfile.Name())
	if err != nil {
		return "", err
	}
	defer tmpfile.Close()

	// Get the image data
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Write the body to file
	bytesWritten, err := io.Copy(tmpfile, resp.Body)
	if err != nil {
		return "", err
	}
	//Simple check to make sure image received is the correct size
	if bytesWritten != resp.ContentLength {
		return "", fmt.Errorf("Image received is not the right size. Supposed to be: %v  Actually: %v", resp.ContentLength, bytesWritten)
	}

	return tmpfile.Name(), err
}

// getManifest will return the image manifest for a container uri
// from Singularity Hub. We return the shubAPIResponse and error
func (c *ShubConveyor) getManifest() (manifest *shubAPIResponse, err error) {

	// Create a new Singularity Hub client
	sc := http.Client{
		Timeout: 30 * time.Second,
	}

	//if we are using a non default registry error out for now
	if !c.srcURI.defaultReg {
		return nil, err
	}

	// Format the http address, coinciding with the image uri
	httpAddr := fmt.Sprintf("www.%s", c.srcURI.String())

	// Create the request, add headers context
	url := url.URL{
		Scheme: "https",
		Host:   "",
		Path:   httpAddr, //path contains host
	}

	req, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, err
	}

	// Do the request, if status isn't success, return error
	res, err := sc.Do(req)
	sylog.Debugf("response: %v\n", res)

	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		err = errors.New(res.Status)
		return nil, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &manifest)
	sylog.Debugf("manifest: %v\n", manifest.Image)
	if err != nil {
		return nil, err
	}

	return manifest, err

}

func shubParseReference(src string) (uri ShubURI, err error) {

	//define regex for each URI component
	registryRegexp := `([-a-zA-Z0-9/]{1,64}\/)?` //target is very open, outside registry
	nameRegexp := `([-a-zA-Z0-9]{1,39}\/)`       //target valid github usernames
	containerRegexp := `([-_.a-zA-Z0-9]{1,64})`  //target valid github repo names
	tagRegexp := `(:[-_.a-zA-Z0-9]{1,64})?`      //target is very open, file extensions or branch names
	digestRegexp := `(\@[a-f0-9]{32})?`          //target md5 sum hash

	shubRegex, err := regexp.Compile(registryRegexp + nameRegexp + containerRegexp + tagRegexp + digestRegexp)
	if err != nil {
		return uri, err
	}

	found := shubRegex.FindString(src)

	//if found string is not equal to the input, input isn't a valid URI
	if strings.Compare(src, found) != 0 {
		return uri, fmt.Errorf("Source string is not a valid URI")
	}

	pieces := strings.SplitAfterN(src, `/`, -1)
	if l := len(pieces); l > 2 {
		//more than two pieces indicates a custom registry
		uri.defaultReg = false
		uri.registry = strings.Join(pieces[:l-2], "")
		uri.user = pieces[l-2]
		src = pieces[l-1]
	} else if l == 2 {
		//two pieces means default registry
		uri.defaultReg = true
		uri.registry = defaultRegistry
		uri.user = pieces[l-2]
		src = pieces[l-1]
	}

	//look for an @ and split if it exists
	if strings.Contains(src, `@`) {
		pieces = strings.Split(src, `@`)
		uri.digest = `@` + pieces[1]
		src = pieces[0]
	}

	//look for a : and split if it exists
	if strings.Contains(src, `:`) {
		pieces = strings.Split(src, `:`)
		uri.tag = `:` + pieces[1]
		src = pieces[0]
	}

	//container name is left over after other parts are split from it
	uri.container = src

	return uri, nil
}

func (s *ShubURI) String() string {
	return s.registry + s.user + s.container + s.tag + s.digest
}
