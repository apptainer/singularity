/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package build

import (
	"fmt"
	"strings"

	"github.com/singularityware/singularity/pkg/image"
)

var validProvisioners = map[string]bool{
	"docker": true,
	"shub":   true,
}

// NewProvisionerFromURI is used for providing a provisioner for any command that accepts
// a remote image source (e.g. singularity run docker://..., singularity build image.sif docker://...)
func NewProvisionerFromURI(uri string) (p Provisioner, err error) {
	u := strings.SplitN(uri, ":", 2)

	switch u[0] {
	case "docker":
		return NewDockerProvisioner(u[1])
	case "shub":
		return NewSHubProvisioner()
	default:
		return nil, fmt.Errorf("Provisioner \"%s\" not supported", u[0])
	}
}

func IsValidURI(uri string) (valid bool, err error) {
	u := strings.SplitN(uri, ":", 2)

	if len(u) != 2 {
		return false, nil
	}

	if _, ok := validProvisioners[u[0]]; ok {
		return true, nil
	}

	return false, fmt.Errorf("Invaled image URI: %s", uri)
}

// Provisioner is the interface used to represent how we convert any image
// source into a chroot tree on disk. All necessary input (URL, etc...) should be
// set up when we're creating the specific data structure.
type Provisioner interface {
	Provision(i *image.Sandbox)
}

// ===== SHub =====
func NewSHubProvisioner() (s *SHubProvisioner, err error) {
	return &SHubProvisioner{}, nil
}

type SHubProvisioner struct {
}

func (s *SHubProvisioner) Provision(i *image.Sandbox) {

}
