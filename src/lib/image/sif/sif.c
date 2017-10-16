/*
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
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

/* current placement group id; increments with every new definition-file */
static int currentgroup;

typedef struct Siflayout Siflayout;
static struct Siflayout{
	Sifheader header;
	unsigned char *mapstart;
	unsigned char *descptr;
	unsigned char *dataptr;
	int defindex;
	int envindex;
	int parindex;
	int sigindex;
} siflayout;



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
	case SIF_ENODEF: return "cannot find partition descriptor";
	case SIF_ENOENV: return "cannot find envvar descriptor";
	case SIF_ENOLAB: return "cannot find jason label descriptor";
	case SIF_ENOPAR: return "cannot find partition descriptor";
	case SIF_ENOSIG: return "cannot find signature descriptor";
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
	default: return "Unknown SIF error";
	}
}

static int
printdesc(void *elem)
{
	Sifcommon *cm = (Sifcommon *)elem;
	Sifpartition *p = (Sifpartition *)elem;
	Sifsignature *s = (Sifsignature *)elem;

	printf("desc type: %x\n", cm->datatype);
	printf("group id: %d\n", cm->groupid);
	printf("fileoff: %ld\n", cm->fileoff);
	printf("filelen: %ld\n", cm->filelen);

	switch(cm->datatype){
	case DATA_PARTITION:
		printf("fstype: %d\n", p->fstype);
		printf("parttype: %d\n", p->parttype);
		printf("content: %s\n", p->content);
		break;
	case DATA_SIGNATURE:
		printf("hashtype: %d\n", s->hashtype);
		printf("entity: %s\n", s->entity);
		break;
	default:
		break;
	}
	printf("---------------------------\n");

	return 0;
}

void
printsifhdr(Sifinfo *info)
{
	char uuid[37];

	printf("================ SIF Header ================\n");
	printf("launch: |%s|\n", info->header.launch);

	printf("magic: |%s|\n", info->header.magic);
	printf("version: |%s|\n", info->header.version);
	printf("arch: |%s|\n", info->header.arch);
	uuid_unparse(info->header.uuid, uuid);
	printf("uuid: |%s|\n", uuid);

	printf("creation time: %s", ctime(&info->header.ctime));

	printf("number of descriptors: %d\n", info->header.ndesc);
	printf("start of descriptors in file: %ld\n", info->header.descoff);
	printf("start of data in file: %ld\n", info->header.dataoff);
	printf("length of data in file: %ld\n", info->header.datalen);
	printf("============================================\n");

	listforall(&info->deschead, printdesc);
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
sif_load(char *filename, Sifinfo *info)
{
	int ret = -1;
	int i;
	struct stat st;
	char *p;

	memset(info, 0, sizeof(Sifinfo));

	if(filename == NULL){
		siferrno = SIF_EFNAME;
		return -1;
	}

	info->fd = open(filename, O_RDONLY);
	if(info->fd < 0){
		siferrno = SIF_EFOPEN;
		return -1;
	}

	if(fstat(info->fd, &st) < 0){
		siferrno = SIF_EFSTAT;
		goto bail_close;
	}
	info->filesize = st.st_size;

	info->mapstart = mmap(NULL, info->filesize, PROT_READ, MAP_PRIVATE, info->fd, 0);
	if(info->mapstart == MAP_FAILED){
		siferrno = SIF_EFMAP;
		goto bail_close;
	}

	memcpy(&info->header, info->mapstart, sizeof(Sifheader));
	if(sif_validate(info) < 0)
		goto bail_unmap;

	/* build up the list of SIF data object descriptors */
	for(i = 0, p = info->mapstart+sizeof(Sifheader); i < info->header.ndesc; i++){
		Node *n;
		Sifcommon *cm = (Sifcommon *)p;

		n = listcreate(cm);
		if(n == NULL){
			siferrno = SIF_ELNOMEM;
			goto bail_unmap;
		}
		listaddtail(&info->deschead, n);

		switch(cm->datatype){
		case DATA_DEFFILE:
			p += sizeof(Sifdeffile);
			break;
		case DATA_ENVVAR:
			p += sizeof(Sifenvvar);
			break;
		case DATA_LABELS:
			p += sizeof(Siflabels);
			break;
		case DATA_PARTITION:
			p += sizeof(Sifpartition);
			break;
		case DATA_SIGNATURE:
			p += sizeof(Sifsignature);
			break;
		}
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
	munmap(info->mapstart, info->filesize);
	close(info->fd);

	return 0;
}

/* Get the SIF header structure */
Sifheader *
sif_getheader(Sifinfo *info)
{
	return &info->header;
}

static int
isdeffile(void *cur, void *elem)
{
	Sifdeffile *c = (Sifdeffile *)cur;
        Sifdeffile *e = (Sifdeffile *)elem;

	if(c->cm.datatype == DATA_DEFFILE && c->cm.groupid == e->cm.groupid)
		return 1;
	return 0;
}

/* Get a definition-file descriptor based on groupid */
Sifdeffile *
sif_getdeffile(Sifinfo *info, int groupid)
{
	Sifdeffile lookfor;
	Node *n;

	lookfor.cm.groupid = groupid;

	n = listfind(&info->deschead, &lookfor, isdeffile);
	if(n == NULL){
		siferrno = SIF_ENODEF;
		return NULL;
	}
	return n->elem;
}

/* Get an JSON-labels descriptor based on groupid */
static int
islabels(void *cur, void *elem)
{
	Siflabels *c = (Siflabels *)cur;
        Siflabels *e = (Siflabels *)elem;

	if(c->cm.datatype == DATA_LABELS && c->cm.groupid == e->cm.groupid)
		return 1;
	return 0;
}

Siflabels *
sif_getlabels(Sifinfo *info, int groupid)
{
	Siflabels lookfor;
	Node *n;

	lookfor.cm.groupid = groupid;

	n = listfind(&info->deschead, &lookfor, islabels);
	if(n == NULL){
		siferrno = SIF_ENOLAB;
		return NULL;
	}
	return n->elem;
}

/* Get an environment var descriptor based on groupid */
static int
isenvvar(void *cur, void *elem)
{
	Sifenvvar *c = (Sifenvvar *)cur;
        Sifenvvar *e = (Sifenvvar *)elem;

	if(c->cm.datatype == DATA_ENVVAR && c->cm.groupid == e->cm.groupid)
		return 1;
	return 0;
}

Sifenvvar *
sif_getenvvar(Sifinfo *info, int groupid)
{
	Sifenvvar lookfor;
	Node *n;

	lookfor.cm.groupid = groupid;

	n = listfind(&info->deschead, &lookfor, isenvvar);
	if(n == NULL){
		siferrno = SIF_ENOENV;
		return NULL;
	}
	return n->elem;
}

/* Get an partition descriptor based on groupid */
static int
ispartition(void *cur, void *elem)
{
	Sifpartition *c = (Sifpartition *)cur;
        Sifpartition *e = (Sifpartition *)elem;

	if(c->cm.datatype == DATA_PARTITION && c->cm.groupid == e->cm.groupid)
		return 1;
	return 0;
}

Sifpartition *
sif_getpartition(Sifinfo *info, int groupid)
{
	Sifpartition lookfor;
	Node *n;

	lookfor.cm.groupid = groupid;

	n = listfind(&info->deschead, &lookfor, ispartition);
	if(n == NULL){
		siferrno = SIF_ENOPAR;
		return NULL;
	}
	return n->elem;
}

/* Get an signature/verification descriptor based on groupid */
static int
issignature(void *cur, void *elem)
{
	Sifsignature *c = (Sifsignature *)cur;
        Sifsignature *e = (Sifsignature *)elem;

	if(c->cm.datatype == DATA_SIGNATURE && c->cm.groupid == e->cm.groupid)
		return 1;
	return 0;
}

Sifsignature *
sif_getsignature(Sifinfo *info, int groupid)
{
	Sifsignature lookfor;
	Node *n;

	lookfor.cm.groupid = groupid;

	n = listfind(&info->deschead, &lookfor, issignature);
	if(n == NULL){
		siferrno = SIF_ENOSIG;
		return NULL;
	}
	return n->elem;
}

/*
 * routines associated with the creation of a new SIF image file
 */

static int
prepddesc(void *elem)
{
	Ddesc *d = elem;

	siflayout.header.ndesc++;
	siflayout.header.dataoff += sizeof(Sifdeffile);
	siflayout.header.datalen += d->len;

	/* prep input file (definition file) */
	d->fd = open(d->fname, O_RDONLY);
	if(d->fd < 0){
		siferrno = SIF_EFDDEF;
		return -1;
	}
	/* map input definition file into memory for SIF creation coming up after */
	d->mapstart = mmap(NULL, d->len, PROT_READ, MAP_PRIVATE, d->fd, 0);
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
	Edesc *e = elem;

	siflayout.header.ndesc++;
	siflayout.header.dataoff += sizeof(Sifenvvar);
	siflayout.header.datalen += e->len;

	return 0;
}

static int
prepldesc(void *elem)
{
	Ldesc *l = elem;

	siflayout.header.ndesc++;
	siflayout.header.dataoff += sizeof(Siflabels);
	siflayout.header.datalen += l->len;

	/* prep input file (JSON-label file) */
	l->fd = open(l->fname, O_RDONLY);
	if(l->fd < 0){
		siferrno = SIF_EFDLAB;
		return -1;
	}
	/* map input JSON-label file into memory for SIF creation coming up after */
	l->mapstart = mmap(NULL, l->len, PROT_READ, MAP_PRIVATE, l->fd, 0);
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
	Pdesc *p = elem;

	siflayout.header.ndesc++;
	siflayout.header.dataoff += sizeof(Sifpartition);
	siflayout.header.datalen += p->len;

	/* prep input file (partition file) */
	p->fd = open(p->fname, O_RDONLY);
	if(p->fd < 0){
		siferrno = SIF_EFDPAR;
		return -1;
	}
	/* map input partition file into memory for SIF creation coming up after */
	p->mapstart = mmap(NULL, p->len, PROT_READ, MAP_PRIVATE, p->fd, 0);
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
	Sdesc *s = elem;

	siflayout.header.ndesc++;
	siflayout.header.dataoff += sizeof(Sifsignature);
	siflayout.header.datalen += s->len;

	return 0;
}

static int
prepdesc(void *elem)
{
	Sifdatatype *dt = (Sifdatatype *)elem;
	switch(*dt){
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
	Ddesc *d = elem;
	Sifdeffile *desc = (Sifdeffile *)siflayout.descptr;

	/* write data object descriptor */
	desc->cm.datatype = DATA_DEFFILE;
	desc->cm.groupid = currentgroup;
	desc->cm.fileoff = siflayout.dataptr - siflayout.mapstart;
	desc->cm.filelen = d->len;

	/* write data object */
	memcpy(siflayout.dataptr, d->mapstart, desc->cm.filelen);

	/* increment file map pointers */
	siflayout.descptr += sizeof(Sifdeffile);
	siflayout.dataptr += desc->cm.filelen;

	/* increment definition-file object descriptor placement index */
	siflayout.defindex++;

	return 0;
}

static int
putedesc(void *elem)
{
	Edesc *e = elem;
	Sifenvvar *desc = (Sifenvvar *)siflayout.descptr;

	/* write data object descriptor */
	desc->cm.datatype = DATA_ENVVAR;
	desc->cm.groupid = currentgroup;
	desc->cm.fileoff = siflayout.dataptr - siflayout.mapstart;
	desc->cm.filelen = e->len;

	/* write data object */
	memcpy(siflayout.dataptr, e->vars, desc->cm.filelen);

	/* increment file map pointers */
	siflayout.descptr += sizeof(Sifenvvar);
	siflayout.dataptr += desc->cm.filelen;

	/* increment definition-file object descriptor placement index */
	siflayout.envindex++;

	return 0;
}

static int
putldesc(void *elem)
{
	Ldesc *l = elem;
	Siflabels *desc = (Siflabels *)siflayout.descptr;

	/* write data object descriptor */
	desc->cm.datatype = DATA_LABELS;
	desc->cm.groupid = currentgroup;
	desc->cm.fileoff = siflayout.dataptr - siflayout.mapstart;
	desc->cm.filelen = l->len;

	/* write data object */
	memcpy(siflayout.dataptr, l->mapstart, desc->cm.filelen);

	/* increment file map pointers */
	siflayout.descptr += sizeof(Siflabels);
	siflayout.dataptr += desc->cm.filelen;

	/* increment JSON-label object descriptor placement index */
	siflayout.envindex++;

	return 0;
}

static int
putpdesc(void *elem)
{
	Pdesc *p = elem;
	Sifpartition *desc = (Sifpartition *)siflayout.descptr;

	/* write data object descriptor */
	desc->cm.datatype = DATA_PARTITION;
	desc->cm.groupid = currentgroup;
	desc->cm.fileoff = siflayout.dataptr - siflayout.mapstart;
	desc->cm.filelen = p->len;
	desc->fstype = p->fstype;
	strncpy(desc->content, p->content, sizeof(desc->content)-1);

	/* write data object */
	memcpy(siflayout.dataptr, p->mapstart, desc->cm.filelen);

	/* increment file map pointers */
	siflayout.descptr += sizeof(Sifpartition);
	siflayout.dataptr += desc->cm.filelen;

	/* increment definition-file object descriptor placement index */
	siflayout.parindex++;

	return 0;
}

static int
putsdesc(void *elem)
{
	Sdesc *s = elem;
	Sifsignature *desc = (Sifsignature *)siflayout.descptr;

	/* write data object descriptor */
	desc->cm.datatype = DATA_SIGNATURE;
	desc->cm.groupid = currentgroup;
	desc->cm.fileoff = siflayout.dataptr - siflayout.mapstart;
	desc->cm.filelen = s->len;
	desc->hashtype = s->hashtype;
	strncpy(desc->entity, s->entity, sizeof(desc->entity)-1);

	/* write data object */
	memcpy(siflayout.dataptr, s->signature, desc->cm.filelen);

	/* increment file map pointers */
	siflayout.descptr += sizeof(Sifsignature);
	siflayout.dataptr += desc->cm.filelen;

	/* increment definition-file object descriptor placement index */
	siflayout.sigindex++;

	return 0;
}

static int
putdesc(void *elem)
{
	Sifdatatype *dt = (Sifdatatype *)elem;
	switch(*dt){
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
	Ddesc *d = elem;

	munmap(d->mapstart, d->len);
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
	Pdesc *p = elem;

	munmap(p->mapstart, p->len);
	close(p->fd);

	return 0;
}

static int
cleanupldesc(void *elem)
{
	Ldesc *l = elem;

	munmap(l->mapstart, l->len);
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
	Sifdatatype *dt = (Sifdatatype *)elem;
	switch(*dt){
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
sif_create(Sifcreateinfo *cinfo)
{
	int fd;

	/* assemble the SIF global header from options (cinfo) */
	memcpy(siflayout.header.launch, cinfo->launchstr, strlen(cinfo->launchstr)+1);
	memcpy(siflayout.header.magic, SIF_MAGIC, SIF_MAGIC_LEN);
	memcpy(siflayout.header.version, cinfo->sifversion, SIF_ARCH_LEN);
	memcpy(siflayout.header.arch, cinfo->arch, SIF_ARCH_LEN);
	uuid_copy(siflayout.header.uuid, cinfo->uuid);
	siflayout.header.ctime = time(NULL);
	siflayout.header.descoff = sizeof(Sifheader);
	siflayout.header.dataoff = sizeof(Sifheader); /* augmented by prep?desc (below) */

	/* check the number of data object descriptors and sizes */
	if(listforall(&cinfo->deschead, prepdesc) < 0)
		return -1;

	if(siflayout.header.ndesc == 0){
		siferrno = SIF_EEMPTY;
		return -1;
	}

	/* create and grow output file */
	fd = open(cinfo->pathname, O_CREAT|O_TRUNC|O_RDWR,
	          S_IRWXU|S_IRGRP|S_IXGRP|S_IROTH|S_IXOTH);
	if(fd < 0){
		siferrno = SIF_ECREAT;
		return -1;
	}
	if(posix_fallocate(fd, 0, siflayout.header.dataoff + siflayout.header.datalen) != 0){
		siferrno = SIF_EFALLOC;
		close(fd);
		return -1;
	}

	/* represent output file in memory */
	siflayout.mapstart = mmap(NULL, siflayout.header.dataoff + siflayout.header.datalen,
	                          PROT_WRITE, MAP_SHARED, fd, 0);
	if(siflayout.mapstart == MAP_FAILED){
		siferrno = SIF_EOMAP;
		close(fd);
		return -1;
	}

	/* set output file write pointers (descriptors and data) */
	siflayout.descptr = siflayout.mapstart;
	siflayout.dataptr = siflayout.mapstart + siflayout.header.dataoff;

	/* build SIF: header, section descriptors, section data */
	memcpy(siflayout.descptr, &siflayout.header, sizeof(Sifheader));
	siflayout.descptr += sizeof(Sifheader);
	listforall(&cinfo->deschead, putdesc);

	/* cleanup opened and mmap'ed input files (deffile, labels, partition) */
	listforall(&cinfo->deschead, cleanupdesc);

	/* unmap and close resulting output file */
	if(munmap(siflayout.mapstart, siflayout.header.dataoff + siflayout.header.datalen) < 0){
		siferrno = SIF_EOUNMAP;
		close(fd);
		return -1;
	}
	if(close(fd) < 0){
		siferrno = SIF_EOCLOSE;
		return -1;
	}

	return 0;
}

