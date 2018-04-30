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

#include <stdio.h>
#include <string.h>
#include <uuid/uuid.h>

#include "list.h"
#include "sif.h"

#include "sifaccess.h"

char *
sif_archstr(char *arch)
{
	if(strcmp(arch, SIF_ARCH_386) == 0)
		return "386";
	if(strcmp(arch, SIF_ARCH_AMD64) == 0)
		return "AMD64";
	if(strcmp(arch, SIF_ARCH_ARM) == 0)
		return "ARM";
	if(strcmp(arch, SIF_ARCH_AARCH64) == 0)
		return "AARCH64";
	return "Unknown arch";
}

char *
sif_hashstr(Sifhashtype htype)
{
	switch(htype){
	case HASH_SHA256: return "SHA256";
	case HASH_SHA384: return "SHA384";
	case HASH_SHA512: return "SHA512";
	case HASH_BLAKE2S: return "BLAKE2S";
	case HASH_BLAKE2B: return "BLAKE2B";
	}
	return "Unknown hash-type";
}

char *
sif_partstr(Sifparttype ptype)
{
	switch(ptype){
	case PART_SYSTEM: return "System";
	case PART_DATA: return "Data";
	case PART_OVERLAY: return "Overlay";
	}
	return "Unknown part-type";
}

char *
sif_datastr(Sifdatatype dtype)
{
	switch(dtype){
	case DATA_DEFFILE: return "Def.File";
	case DATA_ENVVAR: return "Env.Vars";
	case DATA_LABELS: return "Jason.Labels";
	case DATA_PARTITION: return "FS.Img";
	case DATA_SIGNATURE: return "Signature";
	}
	return "Unknown data-type";
}

char *
sif_fsstr(Siffstype ftype)
{
	switch(ftype){
	case FS_SQUASH: return "Squashfs";
	case FS_EXT3: return "Ext3";
	case FS_IMMOBJECTS: return "Data.Archive";
	case FS_RAW: return "Raw.Data";
	}
	return "Unknown fstype";
}

char *
sif_hreadable(size_t value)
{
	static char conversion[32];
	int divs = 0;

	memset(conversion, 0, 32);

	for(; value; value>>=10) {
		if(value < 1024)
			break;
		divs++;
	}

	switch(divs) {
		case 0: snprintf(conversion, 31, "%zd", value);
			break;
		case 1: snprintf(conversion, 31, "%zdKB", value);
			break;
		case 2: snprintf(conversion, 31, "%zdMB", value);
			break;
		case 3: snprintf(conversion, 31, "%zdGB", value);
			break;
		case 4: snprintf(conversion, 31, "%zdTB", value);
			break;
	}

	return conversion;
}

int
sif_printrow(void *elem, void *data)
{
	static char fposbuf[26];
	Sifdescriptor *desc = elem;

	(void)data;

	printf("%-4d ", desc->cm.id);
	if(desc->cm.groupid == SIF_UNUSED_GROUP)
		printf("|%-7s ", "NONE");
	else
		printf("|%-7d ", desc->cm.groupid & ~SIF_GROUP_MASK);
	if(desc->cm.link == SIF_UNUSED_LINK)
		printf("|%-7s ", "NONE");
	else
		printf("|%-7d ", desc->cm.link);
	sprintf(fposbuf, "|%ld-%ld ", desc->cm.fileoff,
	        desc->cm.fileoff+desc->cm.filelen-1);
	printf("%-26s ", fposbuf);

	switch(desc->cm.datatype){
	case DATA_PARTITION:
		printf("|%s (%s/%s)", sif_datastr(desc->cm.datatype),
		       sif_fsstr(desc->part.fstype), sif_partstr(desc->part.parttype));
		break;
	case DATA_SIGNATURE:
		printf("|%s (%s)", sif_datastr(desc->cm.datatype),
		       sif_hashstr(desc->sig.hashtype));
		break;
	default:
		printf("|%s", sif_datastr(desc->cm.datatype));
		break;
	}
	printf("\n");
	fflush(stdout);

	return 0;
}

void
sif_printlist(Sifinfo *info)
{
	char uuid[37];

	uuid_unparse(info->header.uuid, uuid);

	printf("Container uuid: %s\n", uuid);
	printf("Created on: %s", ctime(&info->header.ctime));
	printf("Modified on: %s", ctime(&info->header.mtime));
	printf("----------------------------------------------------\n\n");

	printf("Descriptor list:\n");

	printf("%-4s %-8s %-8s %-26s %s\n", "ID", "|GROUP", "|LINK", "|SIF POSITION (start-end)", "|TYPE");
	printf("------------------------------------------------------------------------------\n");

	listforall(&info->deschead, sif_printrow, NULL);
}

int
sif_printdesc(void *elem, void *data)
{
	Sifdescriptor *desc = elem;
	int i;

	(void)data;

	printf("desc type: %s\n", sif_datastr(desc->cm.datatype));
	printf("desc id: %d\n", desc->cm.id);
	if(desc->cm.groupid == SIF_UNUSED_GROUP)
		printf("group id: NONE\n");
	else
		printf("group id: %d\n", desc->cm.groupid & ~SIF_GROUP_MASK);
	if(desc->cm.link == SIF_UNUSED_LINK)
		printf("link: NONE\n");
	else
		printf("link: %d\n", desc->cm.link);
	printf("fileoff: %ld\n", desc->cm.fileoff);
	printf("filelen: %zd\n", desc->cm.filelen);

	switch(desc->cm.datatype){
	case DATA_PARTITION:
		printf("fstype: %s\n", sif_fsstr(desc->part.fstype));
		printf("parttype: %s\n", sif_partstr(desc->part.parttype));
		printf("content: %s\n", desc->part.content);
		break;
	case DATA_SIGNATURE:
		printf("hashtype: %s\n", sif_hashstr(desc->sig.hashtype));
		printf("entity: ");
		for(i = 0; i < 20; i++){
			printf("%02hhX", desc->sig.entity[i]);
		}
		printf("\n");
		break;
	default:
		break;
	}
	printf("---------------------------\n");

	return 0;
}

void
sif_printheader(Sifinfo *info)
{
	char uuid[37];

	printf("================ SIF Header ================\n");
	printf("launch: %s\n", info->header.launch);

	printf("magic: %s\n", info->header.magic);
	printf("version: %s\n", info->header.version);
	printf("arch: %s\n", sif_archstr(info->header.arch));
	uuid_unparse(info->header.uuid, uuid);
	printf("uuid: %s\n", uuid);

	printf("creation time: %s", ctime(&info->header.ctime));
	printf("modification time: %s", ctime(&info->header.mtime));

	printf("number of descriptors: %d\n", info->header.ndesc);
	printf("start of descriptors in file: %ld\n", info->header.descoff);
	printf("length of descriptors in file: %s\n", sif_hreadable(info->header.desclen));
	printf("start of data in file: %ld\n", info->header.dataoff);
	printf("length of data in file: %s\n", sif_hreadable(info->header.datalen));
	printf("============================================\n");
}

/* Get the SIF header structure */
Sifheader *
sif_getheader(Sifinfo *info)
{
	return &info->header;
}

static int
sameid(void *cur, void *elem)
{
	Sifdescriptor *c = cur;
	Sifdescriptor *e = elem;

	if(c->cm.id == e->cm.id)
		return 1;
	return 0;
}

Sifdescriptor *
sif_getdescid(Sifinfo *info, int id)
{
	Sifdescriptor lookfor;
	Node *n;

	lookfor.cm.id = id;

	n = listfind(&info->deschead, &lookfor, sameid);
	if(n == NULL){
		siferrno = SIF_ENOID;
		return NULL;
	}
	return n->elem;
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

/* Get descriptors linked to a specific id */
static int
linkmatches(void *cur, void *elem)
{
	Sifdescriptor *c = cur;
	Sifdescriptor *e = elem;

	if(c->cm.link == e->cm.id)
		return 1;
	return 0;
}

Sifdescriptor *
sif_getlinkeddesc(Sifinfo *info, int id)
{
	Sifdescriptor lookfor;
	Node *n;

	lookfor.cm.id = id;

	n = listfind(&info->deschead, &lookfor, linkmatches);
	if(n == NULL){
		siferrno = SIF_ENOLINK;
		return NULL;
	}
	return n->elem;
}
