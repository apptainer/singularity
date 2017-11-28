/*
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

#ifndef _XOPEN_SOURCE
#define _XOPEN_SOURCE 700
#endif

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

Siferrno siferrno;

/* count the next descriptor id to generate when adding a new descriptor */
static int desccounter = 1;

typedef struct Siflayout Siflayout;
static struct Siflayout{
	Sifinfo *info;
	Sifdescriptor *descptr;
	char *dataptr;
} siflayout;

enum{
	REGION_GROWSIZE = sizeof(Sifdescriptor)*32 /* descr. area grow size */
};


/*
 * routines associated with debugging and diagnostics
 */

char *
sif_strerror(Siferrno siferrno)
{
	switch(siferrno){
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
	case SIF_ENODESC: return "cannot find data object descriptors while validating image";
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
	else if(!strncmp(name.machine, "arm", 3) && sizeof(void *) == 8)
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

	siflayout.info = info;
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

	/* init the descriptor counter for next id to use */
	desccounter = info->header.ndesc + 1;

	/* point to the first descriptor in SIF file */
	desc = (Sifdescriptor *)(info->mapstart + sizeof(Sifheader));

	/* set output file write pointers (descriptors and data) */
	siflayout.descptr = (Sifdescriptor *)(info->mapstart + info->header.descoff+info->header.desclen);
	siflayout.dataptr = info->mapstart + info->header.dataoff+info->header.datalen;

	/* build up the list of SIF data object descriptors */
	for(i = 0; i < info->header.ndesc; i++){
		Node *n;

		n = listcreate(desc++);
		if(n == NULL){
			siferrno = SIF_ELNOMEM;
			goto bail_unmap;
		}
		listaddtail(&info->deschead, n);
	}

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
	return 0;
}

static int
prepdesc(void *elem)
{
	Cmdesc *cm = (Cmdesc *)elem;

	if(update_headeroffsets(&siflayout.info->header, cm->len))
		return -1;

	switch(cm->datatype){
	case DATA_DEFFILE:
		return prepddesc(elem);
	case DATA_ENVVAR:
		return prepedesc(elem);
	case DATA_LABELS:
		return prepldesc(elem);
	case DATA_PARTITION:
		return preppdesc(elem);
	case DATA_SIGNATURE:
		return prepsdesc(elem);
	default:
		siferrno = SIF_EUDESC;
		return -1;
	}
	return 0;
}

static int
putddesc(void *elem)
{
	Defdesc *d = elem;
	Sifdescriptor *desc = siflayout.descptr;

	/* write data object descriptor */
	desc->cm.datatype = DATA_DEFFILE;
	desc->cm.id = desccounter++;
	desc->cm.groupid = d->cm.groupid;
	desc->cm.link = d->cm.link;
	desc->cm.fileoff = siflayout.dataptr - siflayout.info->mapstart;
	desc->cm.filelen = d->cm.len;

	/* write data object */
	memcpy(siflayout.dataptr, d->mapstart, desc->cm.filelen);

	/* increment file map pointers */
	siflayout.descptr++;
	siflayout.dataptr += desc->cm.filelen;

	return 0;
}

static int
putedesc(void *elem)
{
	Envdesc *e = elem;
	Sifdescriptor *desc = siflayout.descptr;

	/* write data object descriptor */
	desc->cm.datatype = DATA_ENVVAR;
	desc->cm.id = desccounter++;
	desc->cm.groupid = e->cm.groupid;
	desc->cm.link = e->cm.link;
	desc->cm.fileoff = siflayout.dataptr - siflayout.info->mapstart;
	desc->cm.filelen = e->cm.len;

	/* write data object */
	memcpy(siflayout.dataptr, e->vars, desc->cm.filelen);

	/* increment file map pointers */
	siflayout.descptr++;
	siflayout.dataptr += desc->cm.filelen;

	return 0;
}

static int
putldesc(void *elem)
{
	Labeldesc *l = elem;
	Sifdescriptor *desc = siflayout.descptr;

	/* write data object descriptor */
	desc->cm.datatype = DATA_LABELS;
	desc->cm.id = desccounter++;
	desc->cm.groupid = l->cm.groupid;
	desc->cm.link = l->cm.link;
	desc->cm.fileoff = siflayout.dataptr - siflayout.info->mapstart;
	desc->cm.filelen = l->cm.len;

	/* write data object */
	memcpy(siflayout.dataptr, l->mapstart, desc->cm.filelen);

	/* increment file map pointers */
	siflayout.descptr++;
	siflayout.dataptr += desc->cm.filelen;

	return 0;
}

static int
putpdesc(void *elem)
{
	Partdesc *p = elem;
	Sifdescriptor *desc = siflayout.descptr;

	/* write data object descriptor */
	desc->cm.datatype = DATA_PARTITION;
	desc->cm.id = desccounter++;
	desc->cm.groupid = p->cm.groupid;
	desc->cm.link = p->cm.link;
	desc->cm.fileoff = siflayout.dataptr - siflayout.info->mapstart;
	desc->cm.filelen = p->cm.len;
	desc->part.fstype = p->fstype;
	desc->part.parttype = p->parttype;
	strncpy(desc->part.content, p->content, sizeof(desc->part.content)-1);

	/* write data object */
	memcpy(siflayout.dataptr, p->mapstart, desc->cm.filelen);

	/* increment file map pointers */
	siflayout.descptr++;
	siflayout.dataptr += desc->cm.filelen;

	return 0;
}

static int
putsdesc(void *elem)
{
	Sigdesc *s = elem;
	Sifdescriptor *desc = siflayout.descptr;

	/* write data object descriptor */
	desc->cm.datatype = DATA_SIGNATURE;
	desc->cm.id = desccounter++;
	desc->cm.groupid = s->cm.groupid;
	desc->cm.link = s->cm.link;
	desc->cm.fileoff = siflayout.dataptr - siflayout.info->mapstart;
	desc->cm.filelen = s->cm.len;
	desc->sig.hashtype = s->hashtype;
	strncpy(desc->sig.entity, s->entity, sizeof(desc->sig.entity)-1);

	/* write data object */
	memcpy(siflayout.dataptr, s->signature, desc->cm.filelen);

	/* increment file map pointers */
	siflayout.descptr++;
	siflayout.dataptr += desc->cm.filelen;

	return 0;
}

static int
putdesc(void *elem)
{
	Cmdesc *cm = (Cmdesc *)elem;
	switch(cm->datatype){
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
cleanupdesc(void *elem)
{
	Cmdesc *desc = (Cmdesc *)elem;
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
sif_putdataobj(Sifinfo *info, Cmdesc *cm)
{
	printf("sizeof Sifdescriptor: %ld", sizeof(Sifdescriptor));
	printf("sizeof Sifparition: %ld", sizeof(Sifpartition));
	if(prepdesc(cm) < 0)
		return -1;

	if(munmap(info->mapstart, info->filesize) < 0){
		siferrno = SIF_EOUNMAP;
		return -1;
	}
	info->filesize = info->header.dataoff + info->header.datalen;
	if(posix_fallocate(info->fd, 0, info->filesize) != 0){
		siferrno = SIF_EFALLOC;
		return -1;
	}
	info->mapstart = mmap(NULL, info->filesize, PROT_WRITE, MAP_SHARED, info->fd, 0);
	if(info->mapstart == MAP_FAILED){
		siferrno = SIF_EOMAP;
		return -1;
	}

	/* write down the modified header */
	info->header.mtime = time(NULL);
	memcpy(info->mapstart, &info->header, sizeof(Sifheader));

	putdesc(cm);
	cleanupdesc(cm);

	return 0;
}

int
sif_deldataobj(Sifinfo *info, int id)
{
	return 0;
}

int
sif_create(Sifcreateinfo *cinfo)
{
	Sifinfo info;

	siflayout.info = &info;
	memset(&info, 0, sizeof(Sifinfo));

	/* assemble the SIF global header from options (cinfo) */
	memcpy(info.header.launch, cinfo->launchstr, strlen(cinfo->launchstr)+1);
	memcpy(info.header.magic, SIF_MAGIC, SIF_MAGIC_LEN);
	memcpy(info.header.version, cinfo->sifversion, SIF_ARCH_LEN);
	memcpy(info.header.arch, cinfo->arch, SIF_ARCH_LEN);
	uuid_copy(info.header.uuid, cinfo->uuid);
	info.header.ctime = time(NULL);
	info.header.mtime = time(NULL);
	info.header.descoff = sizeof(Sifheader);
	info.header.dataoff = grow_descregion(&info.header);

	/* check the number of data object descriptors and sizes */
	if(listforall(&cinfo->deschead, prepdesc) < 0)
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

	/* set output file write pointers (descriptors and data) */
	siflayout.descptr = (Sifdescriptor *)(info.mapstart + sizeof(Sifheader));
	siflayout.dataptr = info.mapstart + info.header.dataoff;

	/* build SIF: header, data object descriptors, data objects */
	memcpy(info.mapstart, &info.header, sizeof(Sifheader));
	listforall(&cinfo->deschead, putdesc);

	/* cleanup opened and mmap'ed input files (deffile, labels, partition) */
	listforall(&cinfo->deschead, cleanupdesc);

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
