/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package provisioners

import (
	"encoding/json"
        "errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
        "log"
	"os"
        "time"
        "strings"

	"github.com/containers/image/docker"
	"github.com/containers/image/types"
	oci "github.com/containers/image/oci/layout"
	"github.com/singularityware/singularity/src/pkg/image"
	"github.com/singularityware/singularity/src/pkg/sylog"
)


type shubAPIResponse struct {
    Image string `json:"image"`
    Name string `json:"name"`
    Tag string `json:"tag"`
    Version string `json:"version"`
}


// ShubClient contains the uri and client, lowercase means not exported
type shubClient struct {
	Client   http.Client
	ImageUri string
	HTTPAddr string
}

// NewRemoteBuilder creates a RemoteBuilder with the specified details.
func NewshubClient(uri string) (sc *shubClient) {

	sylog.Infof("%v\n", uri)
	sc = &shubClient{
		Client: http.Client{
			Timeout: 30 * time.Second,
		},
		HTTPAddr: "www.singularity-hub.org",
		ImageUri: uri,
	}

	return
}

// getManifest will return the image manifest for a container uri
// from Singularity Hub. We return the shubAPIResponse and error
func (p *ShubProvisioner) getManifest() (manifest *shubAPIResponse, err error) {

	// Create a new Singularity Hub client
	sc := &shubClient{}

        // TODO: need to parse the tag / digest and send along too
        uri := strings.Split(p.srcRef.StringWithinTransport(), ":")[0]

	// Format the http address, coinciding with the image uri
	httpAddr := fmt.Sprintf("www.singularity-hub.org/api/container%s/", uri)
	sylog.Infof("%v\n", httpAddr)

	// Create the request, add headers context
	url := url.URL{
                Scheme: "https", 
                Host: sc.HTTPAddr, 
                Path: httpAddr,
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
        if(err != nil){
                return nil, err
        }

        return manifest, err

}

// Download an image from Singularity Hub, writing as we download instead
// of storing in memory
func (p *ShubProvisioner) fetch(url string) (err error) {

    // Create temporary download name
    tmpfile, err := ioutil.TempFile(p.tmpfs, "shub-image")
    sylog.Infof("Created temporary file %v\n", tmpfile.Name())
    if err != nil {
        return err
    }
    defer tmpfile.Close()

    // Get the image data
    resp, err := http.Get(url)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    // Write the body to file
    _, err = io.Copy(tmpfile, resp.Body)
    if err != nil {
        return err
    }

    return nil
}

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

	p.tmpfs, err = ioutil.TempDir("", "temp-oci-")
	if err != nil {
		return &ShubProvisioner{}, err
	}

	p.tmpfsRef, err = oci.ParseReference(p.tmpfs + ":" + "tmp")
	if err != nil {
		return &ShubProvisioner{}, err
	}

	return p, nil
}

// ShubProvisioner provisions a sandbox environment from a shub URI.
type ShubProvisioner struct {
	src      string
	srcRef   types.ImageReference
	tmpfs    string
	tmpfsRef types.ImageReference
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
	err = p.fetch(manifest.Image)
	if err != nil {
		log.Fatal(err)
		return
	}

	//err = p.unpackTmpfs(i)
	//if err != nil {
	//	log.Fatal(err)
	//	return
	//}

	return nil

	return fmt.Errorf("Shub provisioner not implemented yet")
}
