/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package image

// #cgo LDFLAGS: -lsycore -luuid
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

const (
	SIF_DEFAULT_GROUP = C.SIF_DEFAULT_GROUP
	SIF_UNUSED_GROUP  = C.SIF_UNUSED_GROUP

	DATA_DEFFILE   = C.DATA_DEFFILE
	DATA_ENVVAR    = C.DATA_ENVVAR
	DATA_LABELS    = C.DATA_LABELS
	DATA_PARTITION = C.DATA_PARTITION
	DATA_SIGNATURE = C.DATA_SIGNATURE
)

type Sifdescriptor struct {
	desc C.Sifdescriptor
}

type Sifinfo struct {
	sinfo C.Sifinfo
}

func (s *Sifinfo) Mapstart() unsafe.Pointer {
	return unsafe.Pointer(s.sinfo.mapstart)
}

type Sifpartition struct {
	part *C.Sifpartition
}

func (p *Sifpartition) FileOff() uint64 {
	return uint64(p.part.cm.fileoff)
}
func (p *Sifpartition) FileLen() uint64 {
	return uint64(p.part.cm.filelen)
}

type Sifsignature struct {
	sig *C.Sifsignature
}

func (s *Sifsignature) FileOff() uint64 {
	return uint64(s.sig.cm.fileoff)
}

func (s *Sifsignature) FileLen() uint64 {
	return uint64(s.sig.cm.filelen)
}

func (s *Sifsignature) GetEntity() string {
	fingerprint := C.GoBytes(unsafe.Pointer(&s.sig.entity[0]), 20)
	str := fmt.Sprintf("%0X", fingerprint)
	return str
}

type Eleminfo struct {
	einfo C.Eleminfo
}

func (e *Eleminfo) InitSignature(fingerprint [20]byte, signature []byte, part *Sifpartition) {
	C.fill_sigeinfo(C.CBytes(fingerprint[:]), C.CBytes(signature), C.int(len(signature)), &e.einfo, part.part)
}

/*
Wrapper for sif_load()
int sif_load(char *filename, Sifinfo *info, int rdonly)
*/
func SifLoad(filename string, info *Sifinfo, rdonly int) error {
	if ret := C.sif_load(C.CString(filename), &info.sinfo, C.int(rdonly)); ret != 0 {
		return fmt.Errorf("%s", C.GoString(C.sif_strerror(C.siferrno)))
	}
	return nil
}

/*
Wrapper for sif_unload()
int sif_unload(Sifinfo *info)
*/
func SifUnload(info *Sifinfo) error {
	if ret := C.sif_unload(&info.sinfo); ret != 0 {
		return fmt.Errorf("%s", C.GoString(C.sif_strerror(C.siferrno)))
	}
	return nil
}

/*
Wrapper for sif_putdataobj()
int sif_putdataobj(Eleminfo *e, Sifinfo *info)
*/
func SifPutDataObj(e *Eleminfo, info *Sifinfo) error {
	if ret := C.sif_putdataobj(&e.einfo, &info.sinfo); ret != 0 {
		return fmt.Errorf("%s", C.GoString(C.sif_strerror(C.siferrno)))
	}
	return nil
}

/*
 * This portion of the file is for sifaccess.c (search and extract) related wrappers
 */

/*
Wrapper for sif_printheader()
void sif_printheader(Sifinfo *info);
*/
func SifPrintHeader(info *Sifinfo) {
	C.sif_printheader(&info.sinfo)
}

/*
Wrapper for sif_printlist()
void sif_printlist(Sifinfo *info);
*/
func SifPrintList(info *Sifinfo) {
	C.sif_printlist(&info.sinfo)
}

/*
Wrapper for sif_getpartition()
Sifpartition *sif_getpartition(Sifinfo *info, int groupid)
*/
func SifGetPartition(info *Sifinfo, groupid int) (*Sifpartition, error) {
	var ret *C.Sifpartition
	if ret = C.sif_getpartition(&info.sinfo, C.int(groupid)); ret == nil {
		return nil, fmt.Errorf("%s", C.GoString(C.sif_strerror(C.siferrno)))
	}
	return &Sifpartition{part: ret}, nil
}

/*
Wrapper for sif_getlinkeddesc()
Sifdescriptor *sif_getlinkeddesc(Sifinfo *info, int id)
*/
func SifGetLinkedDesc(info *Sifinfo, desc *Sifdescriptor) (*Sifdescriptor, error) {
	var ret *C.Sifdescriptor
	var link = new(Sifdescriptor)
	if ret = C.getlinkeddesc(&info.sinfo, &desc.desc, &link.desc); ret == nil {
		return nil, fmt.Errorf("%s", C.GoString(C.sif_strerror(C.siferrno)))
	}
	return link, nil
}

func SifGetSignature(info *Sifinfo) (*Sifsignature, error) {
	var ret *C.Sifsignature
	if ret = C.getsignature(&info.sinfo); ret == nil {
		return nil, fmt.Errorf("%s", C.GoString(C.sif_strerror(C.siferrno)))
	}
	return &Sifsignature{sig: ret}, nil
}

/*
 * General C <-> Go helpers
 */

func CByteRange(start unsafe.Pointer, offset uint64, len uint64) ([]byte, error) {
	if len > uint64(C.int(^C.uint(0)>>1)) {
		return nil, fmt.Errorf("%s", "error: partition is too large for hashing.")
	}
	addr := unsafe.Pointer(uintptr(start) + uintptr(offset))
	return C.GoBytes(addr, C.int(len)), nil
}
