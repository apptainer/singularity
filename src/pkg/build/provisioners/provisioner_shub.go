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
	"io/ioutil"
	"net/http"
	"net/url"
//        "log"
	"os"
        "time"
        "strings"

	"github.com/containers/image/docker"
	"github.com/containers/image/types"
	oci "github.com/containers/image/oci/layout"
	"github.com/singularityware/singularity/src/pkg/image"
	"github.com/singularityware/singularity/src/pkg/sylog"
)


// ShubClient contains the uri and client
type ShubClient struct {
	Client   http.Client
	ImageUri string
	HTTPAddr string
}

// NewRemoteBuilder creates a RemoteBuilder with the specified details.
func NewShubClient(uri string) (shubClient *ShubClient) {

	sylog.Infof("%v\n", uri)
	httpAddr := fmt.Sprintf("www.singularity-hub.org")
	shubClient = &ShubClient{
		Client: http.Client{
			Timeout: 30 * time.Second,
		},
		HTTPAddr: httpAddr,
		ImageUri: uri,
	}

	return
}

func (p *ShubProvisioner) getManifest() (err error) {

	// Create a new Singularity Hub client
	sc := &ShubClient{}

        // TODO: need to parse the tag / digest and send along too
        uri := strings.Split(p.srcRef.StringWithinTransport(), ":")[0]

	// Format the http address, coinciding with the image uri
	httpAddr := fmt.Sprintf("www.singularity-hub.org/api/container%s/", uri)
	sylog.Infof("%v\n", httpAddr)

	// Create the request, add headers context
	url := url.URL{Scheme: "https", Host: sc.HTTPAddr, Path: httpAddr}
        sylog.Infof("%v\n", url)
	req, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return
	}

	// Do the request, if status isn't success, return error
	res, err := sc.Client.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		err = errors.New(res.Status)
		return
	}

	err = json.NewDecoder(res.Body).Decode(&p)
	return

}

// NewShubProvisioner returns a provisioner that can dump a previously
// build Singularity image into a sandbox based on a URI
func NewShubProvisioner(src string) (p *ShubProvisioner, err error) {

	// Question for bauerm - where does this get called?
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
	manifest := p.getManifest()
	sylog.Infof("%v", manifest)

	// retrieve the image
	//err = p.fetch(i)
	//if err != nil {
	//	log.Fatal(err)
	//	return
	//}

	//err = p.unpackTmpfs(i)
	//if err != nil {
	//	log.Fatal(err)
	//	return
	//}

	return nil

	return fmt.Errorf("Shub provisioner not implemented yet")
}
