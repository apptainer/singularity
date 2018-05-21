// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"github.com/singularityware/singularity/src/pkg/image"
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

// SHub represents the shub:// URI, pulling from singularity hub
type SHub struct {
	container string
	registry  string
	username  string
	tag       string
	digest    string
}

// Provision provisions a build environment from a SHub object
func (p *SHub) Provision() {

}

// ShubFromHeader creates a SHub object from a header
func ShubFromHeader(def Definition) *SHub {
	// TODO: Eduardo implement these
	return &SHub{}
}

// LOCALIMAGE

// LocalImage represents bootstrapping from a local image file as a base
type LocalImage struct {
	image.Image
}

// Provision provisions a build environment from a localImage object
func (p *LocalImage) Provision() {

}

// LocalImageFromHeader creates a localImage object from a header
func LocalImageFromHeader(def Definition) *LocalImage {
	// TODO: Eduardo implement these
	return &LocalImage{}
}

// LOCALARCHIVE

// LocalArchive represents bootstrapping from a local tar archive
type LocalArchive struct {
	path string
}

// Provision provisions a build environment from a localArchive object
func (p *LocalArchive) Provision() {

}

// LocalArchiveFromHeader creates a localArchive object from a header
func LocalArchiveFromHeader(def Definition) *LocalArchive {
	// TODO: Eduardo implement these
	return &LocalArchive{}
}

// DEBOOTSTRAP

// Debootstrap represents bootstrapping an apt based system (Debian, Ubuntu)
type Debootstrap struct {
	url     string
	include string
}

// Provision provisions a build environment from a Debootstrap object
func (p *Debootstrap) Provision() {

}

// DebootstrapFromHeader creates a Debootstrap object from a header
func DebootstrapFromHeader(def Definition) *Debootstrap {
	// TODO: Eduardo implement these
	return &Debootstrap{}
}

// YUM

// YUM represents bootstrapping a YUM based system (CentOS, Red Hat)
type YUM struct {
	url string
}

// Provision provisions a build environment from a YUM object
func (p *YUM) Provision() {

}

// YUMFromHeader creates a YUM object from a header
func YUMFromHeader(def Definition) *YUM {
	// TODO: Eduardo implement these
	return &YUM{}
}

// ARCH

// Arch represents bootstrapping an Arch linux system
type Arch struct {
}

// Provision provisions a build environment from a Arch object
func (p *Arch) Provision() {

}

// ArchFromHeader creates a Arch object from a header
func ArchFromHeader(def Definition) *Arch {
	// TODO: Eduardo implement these
	return &Arch{}
}

// BUSYBOX

// BusyBox represents bootstrapping a BusyBox system
type BusyBox struct {
	url string
}

// Provision provisions a build environment from a busyBusyBoxbox object
func (p *BusyBox) Provision() {

}

// BusyboxFromHeader creates a BusyBox object from a header
func BusyboxFromHeader(def Definition) *BusyBox {
	// TODO: Eduardo implement these
	return &BusyBox{}
}

// ZYPPER

// Zypper represents bootstrapping a Zypper based system (SUSE, OpenSUSE)
type Zypper struct {
	url     string
	include string
}

// Provision provisions a build environment from a zypper object
func (p *Zypper) Provision() {

}

// ZypperFromHeader creates a zypper object from a header
func ZypperFromHeader(def Definition) *Zypper {
	// TODO: Eduardo implement these
	return &Zypper{}
}
