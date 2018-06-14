// Copyright (c) 2018, Sylabs Inc. All rights reserved.
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
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/containers/image/docker"
	oci "github.com/containers/image/oci/layout"
	"github.com/containers/image/types"
	"github.com/singularityware/singularity/src/pkg/image"
	"github.com/singularityware/singularity/src/pkg/sylog"
)

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

// NewshubClient creates a RemoteBuilder with the specified details.
func newshubClient(uri string) (sc *shubClient) {

	sylog.Infof("%v\n", uri)
	sc = &shubClient{
		Client: http.Client{
			Timeout: 30 * time.Second,
		},
		HTTPAddr: "www.singularity-hub.org",
		ImageURI: uri,
	}

	return
}

// ShubConveyor holds data to be packed into a bundle.
type ShubConveyor struct {
	recipe   Definition
	src      string
	srcRef   types.ImageReference
	tmpfs    string
	tmpfsRef types.ImageReference
}

// ShubConveyorPacker only needs to hold the conveyor to have the needed data to pack
type ShubConveyorPacker struct {
	ShubConveyor
}

// Get downloads container information from Singularityhub
func (c *ShubConveyor) Get(recipe Definition) (err error) {

	c.recipe = recipe

	//prepending slashes to src for ParseReference expected string format
	src := "//" + recipe.Header["from"]

	// Shub URI largely follows same namespace convention
	c.srcRef, err = docker.ParseReference(src)
	if err != nil {
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

	// The full Google Storage download media link
	sylog.Infof("%v\n", manifest.Image)

	// retrieve the image
	//tmpfile, err := c.fetch(manifest.Image)
	_, err = c.fetch(manifest.Image)
	if err != nil {
		log.Fatal(err)
		return
	}

	return nil
}

// Pack puts relevant objects in a Bundle!
func (cp *ShubConveyorPacker) Pack() (b *Bundle, err error) {
	//defer os.RemoveAll(p.tmpfs)

	//err = cp.unpackTmpfs(tmpfile, i)
	if err != nil {
		log.Fatal(err)
		return
	}

	return nil, nil
}

// Download an image from Singularity Hub, writing as we download instead
// of storing in memory
func (c *ShubConveyor) fetch(url string) (image string, err error) {

	// Create temporary download name
	tmpfile, err := ioutil.TempFile(c.tmpfs, "shub-container")
	sylog.Infof("\nCreating temporary image file %v\n", tmpfile.Name())
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
	_, err = io.Copy(tmpfile, resp.Body)
	if err != nil {
		return "", err
	}

	return tmpfile.Name(), err
}

// getManifest will return the image manifest for a container uri
// from Singularity Hub. We return the shubAPIResponse and error
func (c *ShubConveyor) getManifest() (manifest *shubAPIResponse, err error) {

	// Create a new Singularity Hub client
	sc := &shubClient{}

	// TODO: need to parse the tag / digest and send along too
	uri := strings.Split(c.srcRef.StringWithinTransport(), ":")[0]

	// Format the http address, coinciding with the image uri
	httpAddr := fmt.Sprintf("www.singularity-hub.org/api/container%s/", uri)
	sylog.Infof("%v\n", httpAddr)

	// Create the request, add headers context
	url := url.URL{
		Scheme: "https",
		Host:   sc.HTTPAddr,
		Path:   httpAddr,
	}

	sylog.Infof("%v\n", url)
	req, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, err
	}

	// Do the request, if status isn't success, return error
	res, err := sc.Client.Do(req)
	sylog.Infof("response: %v\n", res)

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
	sylog.Infof("manifest: %v\n", manifest.Image)
	if err != nil {
		return nil, err
	}

	return manifest, err

}

func (cp *ShubConveyorPacker) unpackTmpfs(tmpfile string, i *image.Sandbox) (err error) {
	//refs := []string{"name=tmp"}
	sylog.Infof("Here we need to unpack %v to %v\n", tmpfile, i)
	//err = imagetools.UnpackLayout(p.tmpfs, i.Rootfs(), "amd64", refs)
	return err
}

/*
// NewShubProvisioner returns a provisioner that can dump a previously
// build Singularity image into a sandbox based on a URI
func NewShubProvisioner(src string) (p *ShubProvisioner, err error) {

	// Called in src/pkg/build/provisioner.go for shub uri
	p = &ShubProvisioner{}

	// Shub URI largely follows same namespace convention
	p.srcRef, err = docker.ParseReference(src)
	if err != nil {
		return &ShubProvisioner{}, err
	}

	p.tmpfs, err = ioutil.TempDir("", "temp-shub-")
	if err != nil {
		return &ShubProvisioner{}, err
	}

	p.tmpfsRef, err = oci.ParseReference(p.tmpfs + ":" + "tmp")
	if err != nil {
		return &ShubProvisioner{}, err
	}

	return p, nil
}

// Provision provisions a sandbox from the Shub source URI into the location
// specified by i.
func (p *ShubProvisioner) Provision(i *image.Sandbox) (err error) {

	defer os.RemoveAll(p.tmpfs)

	// Get the image manifest
	manifest, err := p.getManifest()

	// The full Google Storage download media link
	sylog.Infof("%v\n", manifest.Image)

	// retrieve the image
	tmpfile, err := p.fetch(manifest.Image)
	if err != nil {
		log.Fatal(err)
		return
	}

	err = p.unpackTmpfs(tmpfile, i)
	if err != nil {
		log.Fatal(err)
		return
	}

	return nil
}

*/
