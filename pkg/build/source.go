/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package build

import (
	"github.com/singularityware/singularity/pkg/image"
)

//TODO: Move each data type into provisioner_*.go

// DOCKER
// Docker represents the docker:// URI, pulling from docker hub or
// a private docker registry
type Docker struct {
	container string
	registry  string
	namespace string
	tag       string
	digest    string
}

// Provision provisions a build environment from a docker object
func (p *Docker) Provision() {

}

// DockerFromHeader creates a docker object from a header
func DockerFromHeader(h Definition) *Docker {
	// TODO: Eduardo implement these
	return &Docker{}
}

// SHUB
// shub represents the shub:// URI, pulling from singularity hub
type shub struct {
	container string
	registry  string
	username  string
	tag       string
	digest    string
}

// Provision provisions a build environment from a shub object
func (p *shub) Provision() {

}

// ShubFromHeader creates a shub object from a header
func ShubFromHeader(def Definition) *shub {
	// TODO: Eduardo implement these
	return &shub{}
}

// LOCALIMAGE
// localImage represents bootstrapping from a local image file as a base
type localImage struct {
	image.Image
}

// Provision provisions a build environment from a localImage object
func (p *localImage) Provision() {

}

// LocalImageFromHeader creates a localImage object from a header
func LocalImageFromHeader(def Definition) *localImage {
	// TODO: Eduardo implement these
	return &localImage{}
}

// LOCALARCHIVE
// localArchive represents bootstrapping from a local tar archive
type localArchive struct {
	path string
}

// Provision provisions a build environment from a localArchive object
func (p *localArchive) Provision() {

}

// LocalArchiveFromHeader creates a localArchive object from a header
func LocalArchiveFromHeader(def Definition) *localArchive {
	// TODO: Eduardo implement these
	return &localArchive{}
}

// DEBOOTSTRAP
// debootstrap represents bootstrapping an apt based system (Debian, Ubuntu)
type debootstrap struct {
	url     string
	include string
}

// Provision provisions a build environment from a debootstrap object
func (p *debootstrap) Provision() {

}

// DebootstrapFromHeader creates a debootstrap object from a header
func DebootstrapFromHeader(def Definition) *debootstrap {
	// TODO: Eduardo implement these
	return &debootstrap{}
}

// YUM
// yum represents bootstrapping a yum based system (CentOS, Red Hat)
type yum struct {
	url string
}

// Provision provisions a build environment from a yum object
func (p *yum) Provision() {

}

// YumFromHeader creates a yum object from a header
func YumFromHeader(def Definition) *yum {
	// TODO: Eduardo implement these
	return &yum{}
}

// ARCH
// arch represents bootstrapping an arch linux system
type arch struct {
}

// Provision provisions a build environment from a arch object
func (p *arch) Provision() {

}

// ArchFromHeader creates a arch object from a header
func ArchFromHeader(def Definition) *arch {
	// TODO: Eduardo implement these
	return &arch{}
}

// BUSYBOX
// busybox represents bootstrapping a busybox system
type busybox struct {
	url string
}

// Provision provisions a build environment from a busybox object
func (p *busybox) Provision() {

}

// BusyboxFromHeader creates a busybox object from a header
func BusyboxFromHeader(def Definition) *busybox {
	// TODO: Eduardo implement these
	return &busybox{}
}

// ZYPPER
// zypper represents bootstrapping a zypper based system (SUSE, OpenSUSE)
type zypper struct {
	url     string
	include string
}

// Provision provisions a build environment from a zypper object
func (p *zypper) Provision() {

}

// ZypperFromHeader creates a zypper object from a header
func ZypperFromHeader(def Definition) *zypper {
	// TODO: Eduardo implement these
	return &zypper{}
}
