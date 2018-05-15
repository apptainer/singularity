/*
 * Copyright (c) 2017-2018, SyLabs, Inc. All rights reserved.
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 * Copyright (c) 2017, Yannick Cote <yhcote@gmail.com> All rights reserved.
 *
 * See the COPYRIGHT.md file at the top-level directory of this distribution and at
 * https://github.com/singularityware/singularity/blob/master/COPYRIGHT.md.
 *
 * This file is part of the Singularity Linux container project. It is subject to the license
 * terms in the LICENSE.md file found in the top-level directory of this distribution and
 * at https://github.com/singularityware/singularity/blob/master/LICENSE.md. No part
 * of Singularity, including this file, may be copied, modified, propagated, or distributed
 * except according to the terms contained in the LICENSE.md file.
 */

#define _GNU_SOURCE

#include <sys/mman.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <sys/utsname.h>

#include <fcntl.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <unistd.h>
#include <uuid/uuid.h>

#include "list.h"
#include "sif.h"
#include "sifaccess.h"

Siferrno siferrno;

enum{
	REGION_GROWSIZE = sizeof(Sifdescriptor)*32 /* descr. area grow size */
};


/*
 * routines associated with debugging and diagnostics
 */

char *
sif_strerror(Siferrno errnum)
{
	switch(errnum){
	case SIF_ENOERR: return "SIF errno not set or success";
	case SIF_EMAGIC: return "invalid SIF magic";
	case SIF_EFNAME: return "invalid input file name";
	case SIF_EFOPEN: return "cannot open input file name";
	case SIF_EFSTAT: return "fstat on input file failed";
	case SIF_EFMAP: return "cannot mmap input file";
	case SIF_ELNOMEM: return "cannot allocate memory for list node";
	case SIF_EFUNMAP: return "cannot munmap input file";
	case SIF_EUNAME: return "uname error while validating image";
	case SIF_EUARCH: return "unknown host architecture while validating image";
	case SIF_ESIFVER: return "unsupported SIF version while validating image";
	case SIF_ERARCH: return "architecture mismatch while validating image";
	case SIF_ENODESC: return "cannot find data object descriptor(s)";
	case SIF_ENODEF: return "cannot find definition file descriptor";
	case SIF_ENOENV: return "cannot find envvar descriptor";
	case SIF_ENOLAB: return "cannot find jason label descriptor";
	case SIF_ENOPAR: return "cannot find partition descriptor";
	case SIF_ENOSIG: return "cannot find signature descriptor";
	case SIF_ENOLINK: return "cannot find descriptor linked to specified id";
	case SIF_ENOID: return "cannot find descriptor with specified id";
	case SIF_EFDDEF: return "cannot open definition file";
	case SIF_EMAPDEF: return "cannot mmap definition file";
	case SIF_EFDLAB: return "cannot open jason-labels file";
	case SIF_EMAPLAB: return "cannot mmap jason-labels file";
	case SIF_EFDPAR: return "cannot open partition file";
	case SIF_EMAPPAR: return "cannot mmap partition file";
	case SIF_EUDESC: return "unknown data descriptor type";
	case SIF_EEMPTY: return "nothing to generate into SIF file (empty)";
	case SIF_ECREAT: return "cannot create output SIF file, check permissions";
	case SIF_EFALLOC: return "fallocate on SIF output file failed";
	case SIF_EOMAP: return "cannot mmap SIF output file";
	case SIF_EOUNMAP: return "cannot unmmap SIF output file";
	case SIF_EOCLOSE: return "closing SIF file failed, file corrupted, don't use";
	case SIF_EDNOMEM: return "no more space to add new descriptors";
	default: return "Unknown SIF error";
	}
}

/*
 * routines associated with the loading of an SIF image file
 */


static int
sif_validate(Sifinfo *info)
{
	struct utsname name;
	char *currarch;

	if(uname(&name) < 0){
		siferrno = SIF_EUNAME;
		return -1;
	}

	if(!strncmp(name.machine, "x86_64", 6)){
		if(sizeof(void *) == 8)
			currarch = SIF_ARCH_AMD64;
		else
			currarch = SIF_ARCH_386;
	}else if(name.machine[0] == 'i' && name.machine[2] == '8' &&
	        name.machine[3] == '6')
		currarch = SIF_ARCH_386;
	else if(!strncmp(name.machine, "arm", 3) && sizeof(void *) == 4)
		currarch = SIF_ARCH_ARM;
	else if(!strncmp(name.machine, "aarch64", 7))
		currarch = SIF_ARCH_AARCH64;
	else{
		siferrno = SIF_EUARCH;
		return -1;
	}

	if(strncmp(info->header.magic, SIF_MAGIC, strlen(SIF_MAGIC))){
		siferrno = SIF_EMAGIC;
		return -1;
	}
	if(strncmp(info->header.version, SIF_VERSION, strlen(SIF_VERSION))){
		siferrno = SIF_ESIFVER;
		return -1;
	}
	if(strncmp(currarch, info->header.arch, strlen(currarch))){
		siferrno = SIF_ERARCH;
		return -1;
	}
	if(info->header.ndesc <= 0){
		siferrno = SIF_ENODESC;
		return -1;
	}

	return 0;
}

/* load and returns the SIF header and populate the list of data object descriptor */
int
sif_load(char *filename, Sifinfo *info, int rdonly)
{
	int ret = -1;
	int i;
	int oflags, mprot, mflags;
	struct stat st;
	Sifdescriptor *desc;

	memset(info, 0, sizeof(Sifinfo));

	if(filename == NULL){
		siferrno = SIF_EFNAME;
		return -1;
	}

	if(rdonly){
		oflags = O_RDONLY;
		mprot = PROT_READ;
		mflags = MAP_PRIVATE;
	}else{
		oflags = O_RDWR;
		mprot = PROT_WRITE;
		mflags = MAP_SHARED;
	}

	info->fd = open(filename, oflags);
	if(info->fd < 0){
		siferrno = SIF_EFOPEN;
		return -1;
	}

	if(fstat(info->fd, &st) < 0){
		siferrno = SIF_EFSTAT;
		goto bail_close;
	}
	info->filesize = st.st_size;

	info->mapstart = mmap(NULL, info->filesize, mprot, mflags, info->fd, 0);
	if(info->mapstart == MAP_FAILED){
		siferrno = SIF_EFMAP;
		goto bail_close;
	}

	memcpy(&info->header, info->mapstart, sizeof(Sifheader));
	if(sif_validate(info) < 0)
		goto bail_unmap;

	/* point to the first descriptor in SIF file */
	desc = (Sifdescriptor *)(info->mapstart + sizeof(Sifheader));
	info->nextid = desc->cm.id;

	/* build up the list of SIF data object descriptors */
	for(i = 0; i < info->header.ndesc; i++){
		Node *n;

		if(desc->cm.id > info->nextid)
			info->nextid = desc->cm.id;

		n = listcreate(desc++);
		if(n == NULL){
			siferrno = SIF_ELNOMEM;
			goto bail_unmap;
		}
		listaddtail(&info->deschead, n);
	}
	info->nextid++;

	return 0;

bail_unmap:
	munmap(info->mapstart, info->filesize);
bail_close:
	close(info->fd);

	return ret;
}

int
sif_unload(Sifinfo *info)
{
	if(munmap(info->mapstart, info->filesize) < 0){
		siferrno = SIF_EOUNMAP;
		return -1;
	}
	if(close(info->fd) < 0){
		siferrno = SIF_EOCLOSE;
		return -1;
	}

	return 0;
}

/*
 * routines associated with the creation of a new SIF image file
 */

static off_t
grow_descregion(Sifheader *header)
{
	header->dataoff += REGION_GROWSIZE;
	return header->dataoff;
}

static int
update_headeroffsets(Sifheader *header, size_t datasize)
{
	header->ndesc++;
	if((header->descoff + header->desclen + sizeof(Sifdescriptor)) >= header->dataoff){
		siferrno = SIF_EDNOMEM;
		return -1;
	}

	header->desclen += sizeof(Sifdescriptor);
	header->datalen += datasize;

	return 0;
}

static int
prepddesc(void *elem)
{
	Defdesc *d = elem;

	/* prep input file (definition file) */
	d->fd = open(d->fname, O_RDONLY);
	if(d->fd < 0){
		siferrno = SIF_EFDDEF;
		return -1;
	}
	/* map input definition file into memory for SIF creation coming up after */
	d->mapstart = mmap(NULL, d->cm.len, PROT_READ, MAP_PRIVATE, d->fd, 0);
	if(d->mapstart == MAP_FAILED){
		siferrno = SIF_EMAPDEF;
		close(d->fd);
		return -1;
	}

	return 0;
}

static int
prepedesc(void *elem)
{
	(void)elem;
	return 0;
}

static int
prepldesc(void *elem)
{
	Labeldesc *l = elem;

	/* prep input file (JSON-label file) */
	l->fd = open(l->fname, O_RDONLY);
	if(l->fd < 0){
		siferrno = SIF_EFDLAB;
		return -1;
	}
	/* map input JSON-label file into memory for SIF creation coming up after */
	l->mapstart = mmap(NULL, l->cm.len, PROT_READ, MAP_PRIVATE, l->fd, 0);
	if(l->mapstart == MAP_FAILED){
		siferrno = SIF_EMAPLAB;
		close(l->fd);
		return -1;
	}
	return 0;
}

static int
preppdesc(void *elem)
{
	Partdesc *p = elem;

	/* prep input file (partition file) */
	p->fd = open(p->fname, O_RDONLY);
	if(p->fd < 0){
		siferrno = SIF_EFDPAR;
		return -1;
	}
	/* map input partition file into memory for SIF creation coming up after */
	p->mapstart = mmap(NULL, p->cm.len, PROT_READ, MAP_PRIVATE, p->fd, 0);
	if(p->mapstart == MAP_FAILED){
		siferrno = SIF_EMAPPAR;
		close(p->fd);
		return -1;
	}
	return 0;
}

static int
prepsdesc(void *elem)
{
	(void)elem;
	return 0;
}

static int
prepdesc(void *elem, void *data)
{
	Eleminfo *e = elem;

	/* for each eleminfo node to prepare, set the info ptr */
	e->info = data;

	if(update_headeroffsets(&e->info->header, e->cm.len))
		return -1;

	switch(e->cm.datatype){
	case DATA_DEFFILE:
		return prepddesc(&e->defdesc);
	case DATA_ENVVAR:
		return prepedesc(&e->envdesc);
	case DATA_LABELS:
		return prepldesc(&e->labeldesc);
	case DATA_PARTITION:
		return preppdesc(&e->partdesc);
	case DATA_SIGNATURE:
		return prepsdesc(&e->sigdesc);
	default:
		siferrno = SIF_EUDESC;
		return -1;
	}
	return 0;
}

static int
putddesc(void *elem)
{
	Eleminfo *e = elem;

	e->desc = (Sifdescriptor *)(e->info->mapstart + e->info->header.descoff + e->info->header.desclen);

	/* write data object descriptor */
	e->desc->cm.datatype = DATA_DEFFILE;
	e->desc->cm.id = e->info->nextid++;
	e->info->header.ndesc++;
	e->desc->cm.groupid = e->cm.groupid;
	e->desc->cm.link = e->cm.link;
	e->desc->cm.fileoff = e->info->header.dataoff + e->info->header.datalen;
	e->desc->cm.filelen = e->cm.len;

	/* write data object */
	memcpy(e->info->mapstart + e->desc->cm.fileoff, e->defdesc.mapstart, e->desc->cm.filelen);

	/* increment header desclen and datalen */
	e->info->header.desclen += sizeof(Sifdescriptor);
	e->info->header.datalen += e->desc->cm.filelen;

	return 0;
}

static int
putedesc(void *elem)
{
	Eleminfo *e = elem;

	e->desc = (Sifdescriptor *)(e->info->mapstart + e->info->header.descoff + e->info->header.desclen);

	/* write data object descriptor */
	e->desc->cm.datatype = DATA_ENVVAR;
	e->desc->cm.id = e->info->nextid++;
	e->info->header.ndesc++;
	e->desc->cm.groupid = e->cm.groupid;
	e->desc->cm.link = e->cm.link;
	e->desc->cm.fileoff = e->info->header.dataoff + e->info->header.datalen;
	e->desc->cm.filelen = e->cm.len;

	/* write data object */
	memcpy(e->info->mapstart + e->desc->cm.fileoff, e->envdesc.vars, e->desc->cm.filelen);

	/* increment header desclen and datalen */
	e->info->header.desclen += sizeof(Sifdescriptor);
	e->info->header.datalen += e->desc->cm.filelen;

	return 0;
}

static int
putldesc(void *elem)
{
	Eleminfo *e = elem;

	e->desc = (Sifdescriptor *)(e->info->mapstart + e->info->header.descoff + e->info->header.desclen);

	/* write data object descriptor */
	e->desc->cm.datatype = DATA_LABELS;
	e->desc->cm.id = e->info->nextid++;
	e->info->header.ndesc++;
	e->desc->cm.groupid = e->cm.groupid;
	e->desc->cm.link = e->cm.link;
	e->desc->cm.fileoff = e->info->header.dataoff + e->info->header.datalen;
	e->desc->cm.filelen = e->cm.len;

	/* write data object */
	memcpy(e->info->mapstart + e->desc->cm.fileoff, e->labeldesc.mapstart, e->desc->cm.filelen);

	/* increment header desclen and datalen */
	e->info->header.desclen += sizeof(Sifdescriptor);
	e->info->header.datalen += e->desc->cm.filelen;

	return 0;
}

static int
putpdesc(void *elem)
{
	Eleminfo *e = elem;

	e->desc = (Sifdescriptor *)(e->info->mapstart + e->info->header.descoff + e->info->header.desclen);

	/* write data object descriptor */
	e->desc->cm.datatype = DATA_PARTITION;
	e->desc->cm.id = e->info->nextid++;
	e->info->header.ndesc++;
	e->desc->cm.groupid = e->cm.groupid;
	e->desc->cm.link = e->cm.link;
	e->desc->cm.fileoff = e->info->header.dataoff + e->info->header.datalen;
	e->desc->cm.filelen = e->cm.len;
	e->desc->part.fstype = e->partdesc.fstype;
	e->desc->part.parttype = e->partdesc.parttype;
	strncpy(e->desc->part.content, e->partdesc.content, SIF_CONTENT_LEN);

	/* write data object */
	memcpy(e->info->mapstart + e->desc->cm.fileoff, e->partdesc.mapstart, e->desc->cm.filelen);

	/* increment header desclen and datalen */
	e->info->header.desclen += sizeof(Sifdescriptor);
	e->info->header.datalen += e->desc->cm.filelen;

	return 0;
}

static int
putsdesc(void *elem)
{
	Eleminfo *e = elem;

	e->desc = (Sifdescriptor *)(e->info->mapstart + e->info->header.descoff + e->info->header.desclen);

	/* write data object descriptor */
	e->desc->cm.datatype = DATA_SIGNATURE;
	e->desc->cm.id = e->info->nextid++;
	e->info->header.ndesc++;
	e->desc->cm.groupid = e->cm.groupid;
	e->desc->cm.link = e->cm.link;
	e->desc->cm.fileoff = e->info->header.dataoff + e->info->header.datalen;
	e->desc->cm.filelen = e->cm.len;
	e->desc->sig.hashtype = e->sigdesc.hashtype;
	strncpy(e->desc->sig.entity, e->sigdesc.entity, SIF_ENTITY_LEN);

	/* write data object */
	memcpy(e->info->mapstart + e->desc->cm.fileoff, e->sigdesc.signature, e->desc->cm.filelen);

	/* increment header desclen and datalen */
	e->info->header.desclen += sizeof(Sifdescriptor);
	e->info->header.datalen += e->desc->cm.filelen;

	return 0;
}

static int
putdesc(void *elem, void *data)
{
	Eleminfo *e = elem;

	(void)data;

	switch(e->cm.datatype){
	case DATA_DEFFILE:
		return putddesc(elem);
	case DATA_ENVVAR:
		return putedesc(elem);
	case DATA_LABELS:
		return putldesc(elem);
	case DATA_PARTITION:
		return putpdesc(elem);
	case DATA_SIGNATURE:
		return putsdesc(elem);
	default:
		siferrno = SIF_EUDESC;
		return -1;
	}
	return 0;
}

static int
cleanupddesc(void *elem)
{
	Defdesc *d = elem;

	munmap(d->mapstart, d->cm.len);
	close(d->fd);

	return 0;
}

static int
cleanupedesc(void *elem)
{
	(void)elem;

	return 0;
}

static int
cleanuppdesc(void *elem)
{
	Partdesc *p = elem;

	munmap(p->mapstart, p->cm.len);
	close(p->fd);

	return 0;
}

static int
cleanupldesc(void *elem)
{
	Labeldesc *l = elem;

	munmap(l->mapstart, l->cm.len);
	close(l->fd);

	return 0;
}

static int
cleanupsdesc(void *elem)
{
	(void)elem;

	return 0;
}

static int
cleanupdesc(void *elem, void *data)
{
	Cmdesc *desc = (Cmdesc *)elem;

	(void)data;

	switch(desc->datatype){
	case DATA_DEFFILE:
		return cleanupddesc(elem);
	case DATA_ENVVAR:
		return cleanupedesc(elem);
	case DATA_LABELS:
		return cleanupldesc(elem);
	case DATA_PARTITION:
		return cleanuppdesc(elem);
	case DATA_SIGNATURE:
		return cleanupsdesc(elem);
	default:
		siferrno = SIF_EUDESC;
		return -1;
	}
	return 0;
}

int
sif_putdataobj(Eleminfo *e, Sifinfo *info)
{
	size_t oldsize;
	void *oldmap;
	int oldndesc = info->header.ndesc;
	size_t olddesclen = info->header.desclen;
	size_t olddatalen = info->header.datalen;

	if(prepdesc(e, info) < 0)
		return -1;

	oldmap = info->mapstart;
	oldsize = info->filesize;

	info->filesize = info->header.dataoff + info->header.datalen;
	if(posix_fallocate(info->fd, 0, info->filesize) != 0){
		siferrno = SIF_EFALLOC;
		return -1;
	}

	info->mapstart = mremap(oldmap, oldsize, info->filesize, MREMAP_MAYMOVE);
	if(info->mapstart == MAP_FAILED){
		siferrno = SIF_EOMAP;
		return -1;
	}

	/* reset values modified by update_headeroffsets() */
	info->header.ndesc = oldndesc;
	info->header.desclen = olddesclen;
	info->header.datalen = olddatalen;

	putdesc(e, NULL);
	cleanupdesc(&e->cm, NULL);

	/* write down the modified header */
	info->header.mtime = time(NULL);
	memcpy(info->mapstart, &info->header, sizeof(Sifheader));

	return 0;
}

int
sif_deldataobj(Sifinfo *info, int id, int flags)
{
	Sifdescriptor *desc;

	desc = sif_getdescid(info, id);
	if(desc == NULL){
		siferrno = SIF_ENODESC;
		return -1;
	}

	/* zero out or remove data portion */
	switch(flags){
	case DEL_ZERO:
		memset(info->mapstart + desc->cm.fileoff, 0, desc->cm.filelen);
		break;
	case DEL_COMPACT:
		siferrno = SIF_ENOSUPP;
		return -1;
	}

	/* remove and shuffle descriptors */
	if(info->header.ndesc > 1){
		memmove(desc, (char *)desc + sizeof(Sifdescriptor), sizeof(Sifdescriptor));
	}else{
		memset(desc, 0, sizeof(Sifdescriptor));
	}

	/* write down the modified header */
	info->header.ndesc--;
	info->header.mtime = time(NULL);
	info->header.desclen -= sizeof(Sifdescriptor);

	memcpy(info->mapstart, &info->header, sizeof(Sifheader));

	return 0;
}

int
sif_create(Sifcreateinfo *cinfo)
{
	Sifinfo info;

	memset(&info, 0, sizeof(Sifinfo));

	/* assemble the SIF global header from options (cinfo) */
	memcpy(info.header.launch, cinfo->launchstr, strlen(cinfo->launchstr)+1);
	memcpy(info.header.magic, SIF_MAGIC, SIF_MAGIC_LEN);
	memcpy(info.header.version, cinfo->sifversion, SIF_ARCH_LEN);
	memcpy(info.header.arch, cinfo->arch, SIF_ARCH_LEN);
	uuid_copy(info.header.uuid, cinfo->uuid);
	info.nextid = 1;
	info.header.ctime = time(NULL);
	info.header.mtime = time(NULL);
	info.header.descoff = sizeof(Sifheader);
	info.header.dataoff = grow_descregion(&info.header);

	/* check the number of data object descriptors and sizes */
	if(listforall(&cinfo->deschead, prepdesc, &info) < 0)
		return -1;

	if(info.header.ndesc == 0){
		siferrno = SIF_EEMPTY;
		return -1;
	}

	/* create and grow output file */
	info.fd = open(cinfo->pathname, O_CREAT|O_TRUNC|O_RDWR,
	               S_IRWXU|S_IRGRP|S_IXGRP|S_IROTH|S_IXOTH);
	if(info.fd < 0){
		siferrno = SIF_ECREAT;
		return -1;
	}
	if(posix_fallocate(info.fd, 0, info.header.dataoff + info.header.datalen) != 0){
		siferrno = SIF_EFALLOC;
		close(info.fd);
		return -1;
	}

	/* represent output file in memory */
	info.mapstart = mmap(NULL, info.header.dataoff + info.header.datalen,
	                          PROT_WRITE, MAP_SHARED, info.fd, 0);
	if(info.mapstart == MAP_FAILED){
		siferrno = SIF_EOMAP;
		close(info.fd);
		return -1;
	}

	/* reset values modified by update_headeroffsets() */
	info.header.ndesc = 0;
	info.header.desclen = 0;
	info.header.datalen = 0;

	/* build SIF: header, data object descriptors, data objects */
	listforall(&cinfo->deschead, putdesc, NULL);
	memcpy(info.mapstart, &info.header, sizeof(Sifheader));

	/* cleanup opened and mmap'ed input files (deffile, labels, partition) */
	listforall(&cinfo->deschead, cleanupdesc, NULL);

	/* unmap and close resulting output file */
	if(munmap(info.mapstart, info.header.dataoff + info.header.datalen) < 0){
		siferrno = SIF_EOUNMAP;
		close(info.fd);
		return -1;
	}
	if(close(info.fd) < 0){
		siferrno = SIF_EOCLOSE;
		return -1;
	}

	return 0;
}
