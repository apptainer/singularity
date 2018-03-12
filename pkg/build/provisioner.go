/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package build

import (
	"fmt"
	"net/url"
)

// NewProvisionerFromURI is used for providing a provisioner for any command that accepts
// a remote image source (e.g. singularity run docker://..., singularity build image.sif docker://...)
func NewProvisionerFromURI(uri string) (p Provisioner, err error) {
	u, err := url.ParseRequestURI(uri)

	if u.Scheme == "" {
		return nil, err
	}

	switch u.Scheme {
	case "docker":
		return NewDockerProvisioner(), nil
	case "shub":
		return NewSHubProvisioner(), nil
	default:
		return nil, fmt.Errorf("Provisioner \"%s\" not supported", u.Scheme)
	}
}

func IsValidURI(uri string) bool {
	_, err := url.ParseRequestURI(uri)

	if err != nil {
		return false
	} else {
		return true
	}
}

// Provisioner is the interface used to represent how we convert any image
// source into a chroot tree on disk. All necessary input (URL, etc...) should be
// set up when we're creating the specific data structure.
type Provisioner interface {
	Provision(path string)
}

// ===== Docker =====
func NewDockerProvisioner() (d *DockerProvisioner) {
	return &DockerProvisioner{}
}

type DockerProvisioner struct {
}

func (d *DockerProvisioner) Provision(path string) {

}

// ===== SHub =====
func NewSHubProvisioner() (s *SHubProvisioner) {
	return &SHubProvisioner{}
}

type SHubProvisioner struct {
}

func (s *SHubProvisioner) Provision(path string) {

}
