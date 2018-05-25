// +build singularity_sif

// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sif

// #cgo CFLAGS: -I../../runtime/c/lib
// #cgo LDFLAGS: -L../../../builddir/lib -lruntime -luuid
/*
#include <sys/types.h>
#include <stdio.h>
#include <string.h>

#include <uuid/uuid.h>

#include <sif/list.h>
#include <sif/sif.h>
#include <sif/sifaccess.h>

void
fill_sigeinfo(void *fingerprint, void *signature, int siglen, Eleminfo *e, Sifpartition *desc)
{
	e->cm.datatype = DATA_SIGNATURE;
	e->cm.groupid = SIF_UNUSED_GROUP;
	e->cm.link = desc->cm.id;
	e->cm.len = siglen;
	e->sigdesc.signature = signature;
	e->sigdesc.hashtype = HASH_SHA384;
	memset(e->sigdesc.entity, 0, sizeof(e->sigdesc.entity));
	memcpy(e->sigdesc.entity, fingerprint, 20);
}

Sifdescriptor *
getlinkeddesc(Sifinfo *info, Sifdescriptor *desc, Sifdescriptor *link)
{
	Sifdescriptor *d = sif_getlinkeddesc(info, desc->cm.id);
	memcpy(link, d, sizeof(Sifdescriptor));
}

Sifsignature *
getsignature(Sifinfo *info)
{
	Sifpartition *part;
	Sifdescriptor *link;

	part = sif_getpartition(info, SIF_DEFAULT_GROUP);
	if(part == NULL){
		fprintf(stderr, "Cannot find partition from SIF file: %s\n",
		        sif_strerror(siferrno));
		return NULL;
	}
	link = sif_getlinkeddesc(info, part->cm.id);
	if(link == NULL){
		fprintf(stderr, "Cannot find signature for id %d: %s\n",
		        part->cm.id, sif_strerror(siferrno));
		return NULL;
	}

	return (Sifsignature *)link;
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

/*
 * This portion of the file is for sif.c (internal SIF) related wrappers
 */

// SIF-related constants
const (
	DefaultGroup = C.SIF_DEFAULT_GROUP
	UnusedGroup  = C.SIF_UNUSED_GROUP

	DataDefFile   = C.DATA_DEFFILE
	DataEnvVar    = C.DATA_ENVVAR
	DataLabels    = C.DATA_LABELS
	DataPartition = C.DATA_PARTITION
	DataSignature = C.DATA_SIGNATURE
)

// Descriptor represents a SIF descriptor.
type Descriptor struct {
	desc C.Sifdescriptor
}

// Info represents information about a SIF.
type Info struct {
	sinfo C.Sifinfo
}

// Mapstart returns the start of the memory map of the opened SIF file.
func (i *Info) Mapstart() unsafe.Pointer {
	return unsafe.Pointer(i.sinfo.mapstart)
}

// Partition contains information about a SIF partition.
type Partition struct {
	part *C.Sifpartition
}

// FileOff returns the offset of the start of the image file.
func (p *Partition) FileOff() uint64 {
	return uint64(p.part.cm.fileoff)
}

// FileLen returns the length of the data in the file.
func (p *Partition) FileLen() uint64 {
	return uint64(p.part.cm.filelen)
}

// Signature describes a SIF signature block.
type Signature struct {
	sig *C.Sifsignature
}

// FileOff returns the offset of the SIF signature block.
func (s *Signature) FileOff() uint64 {
	return uint64(s.sig.cm.fileoff)
}

// FileLen returns the length of the SIF signature block.
func (s *Signature) FileLen() uint64 {
	return uint64(s.sig.cm.filelen)
}

// GetEntity returns the fingerprint of the SIF signature block.
func (s *Signature) GetEntity() string {
	fingerprint := C.GoBytes(unsafe.Pointer(&s.sig.entity[0]), 20)
	str := fmt.Sprintf("%0X", fingerprint)
	return str
}

// Eleminfo describes information about a SIF element.
type Eleminfo struct {
	einfo C.Eleminfo
}

// InitSignature initializes a SIF element with signature details.
func (e *Eleminfo) InitSignature(fingerprint [20]byte, signature []byte, part *Partition) {
	C.fill_sigeinfo(C.CBytes(fingerprint[:]), C.CBytes(signature), C.int(len(signature)), &e.einfo, part.part)
}

// Load loads a SIF image.
//
// Wrapper for sif_load()
// int sif_load(char *filename, Sifinfo *info, int rdonly)
func Load(filename string, info *Info, rdonly int) error {
	if ret := C.sif_load(C.CString(filename), &info.sinfo, C.int(rdonly)); ret != 0 {
		return fmt.Errorf("%s", C.GoString(C.sif_strerror(C.siferrno)))
	}
	return nil
}

// Unload unloads a SIF image.
//
// Wrapper for sif_unload()
// int sif_unload(Sifinfo *info)
func Unload(info *Info) error {
	if ret := C.sif_unload(&info.sinfo); ret != 0 {
		return fmt.Errorf("%s", C.GoString(C.sif_strerror(C.siferrno)))
	}
	return nil
}

// PutDataObj adds an element to the SIF image.
//
// Wrapper for sif_putdataobj()
// int sif_putdataobj(Eleminfo *e, Sifinfo *info)
func PutDataObj(e *Eleminfo, info *Info) error {
	if ret := C.sif_putdataobj(&e.einfo, &info.sinfo); ret != 0 {
		return fmt.Errorf("%s", C.GoString(C.sif_strerror(C.siferrno)))
	}
	return nil
}

/*
 * This portion of the file is for sifaccess.c (search and extract) related wrappers
 */

// PrintHeader prints the SIF header details
//
// Wrapper for sif_printheader()
// void sif_printheader(Sifinfo *info);
func PrintHeader(info *Info) {
	C.sif_printheader(&info.sinfo)
}

// PrintList prints the list of SIF descriptors
//
// Wrapper for sif_printlist()
// void sif_printlist(Sifinfo *info);
func PrintList(info *Info) {
	C.sif_printlist(&info.sinfo)
}

// GetPartition returns the Partition with the supplied groupid.
//
// Wrapper for sif_getpartition()
// Sifpartition *sif_getpartition(Sifinfo *info, int groupid)
func GetPartition(info *Info, groupid int) (*Partition, error) {
	var ret *C.Sifpartition
	if ret = C.sif_getpartition(&info.sinfo, C.int(groupid)); ret == nil {
		return nil, fmt.Errorf("%s", C.GoString(C.sif_strerror(C.siferrno)))
	}
	return &Partition{part: ret}, nil
}

// GetLinkedDesc returns the Descriptor lined to the supplied Descriptor.
//
// Wrapper for sif_getlinkeddesc()
// Sifdescriptor *sif_getlinkeddesc(Sifinfo *info, int id)
func GetLinkedDesc(info *Info, desc *Descriptor) (*Descriptor, error) {
	var ret *C.Sifdescriptor
	var link = new(Descriptor)
	if ret = C.getlinkeddesc(&info.sinfo, &desc.desc, &link.desc); ret == nil {
		return nil, fmt.Errorf("%s", C.GoString(C.sif_strerror(C.siferrno)))
	}
	return link, nil
}

// GetSignature gets the SIF signature info.
func GetSignature(info *Info) (*Signature, error) {
	var ret *C.Sifsignature
	if ret = C.getsignature(&info.sinfo); ret == nil {
		return nil, fmt.Errorf("%s", C.GoString(C.sif_strerror(C.siferrno)))
	}
	return &Signature{sig: ret}, nil
}

/*
 * General C <-> Go helpers
 */

// CByteRange is a helper to get a byte slice given a pointer, offset, and length.
func CByteRange(start unsafe.Pointer, offset uint64, len uint64) ([]byte, error) {
	if len > uint64(C.int(^C.uint(0)>>1)) {
		return nil, fmt.Errorf("%s", "error: partition is too large for hashing.")
	}
	addr := unsafe.Pointer(uintptr(start) + uintptr(offset))
	return C.GoBytes(addr, C.int(len)), nil
}
