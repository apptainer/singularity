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
#define _XOPEN_SOURCE 500
#endif

#include <sys/stat.h>
#include <sys/types.h>
#include <sys/utsname.h>

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <unistd.h>
#include <uuid/uuid.h>

#include "lib/sif/list.h"
#include "lib/sif/sif.h"

#include "util/util.h"

static void
usage(char *argv[])
{
	fprintf(stderr, "usage: %s OPTION FILE\n", argv[0]);
	fprintf(stderr, "\nGenerates a SIF output FILE with descriptors from -D,-E,-P,-S options.\n");
	fprintf(stderr, "\ndescriptor options: -D deffile [-E] -P partfile [-S signfile]\n");
	fprintf(stderr, "descriptor attributes:\n");
	fprintf(stderr, " for -P: -c CONTENT, -f FSTYPE\n");
	fprintf(stderr, " for -S: -h HASHTYPE, -e ENTITY\n");
	fprintf(stderr, "\n");
}

Node *
ddescadd(Node *head, char *fname)
{
	Ddesc *e;
	Node *n;
	struct stat st;

	e = malloc(sizeof(Ddesc));
	if(e == NULL){
		fprintf(stderr, "Error allocating memory for Ddesc\n");
		return NULL;
	}
	e->datatype = DATA_DEFFILE;
	e->fname = strdup(fname);
	if(e->fname == NULL){
		fprintf(stderr, "Error allocating memory for e->fname\n");
		return NULL;
	}
	if(stat(e->fname, &st) < 0){
		perror("Error calling stat");
		free(e);
		return NULL;
	}
	e->len = st.st_size;
	n = listcreate(e);
	if(n == NULL){
		fprintf(stderr, "Error allocating Ddesc node\n");
		free(e);
		return NULL;
	}
	listaddtail(head, n);

	return n;
}

static char testenvs[] = "VAR0=VALUE0\nVAR1=VALUE1\nVAR2=VALUE2";
Node *
edescadd(Node *head)
{
	Edesc *e;
	Node *n;

	e = malloc(sizeof(Edesc));
	if(e == NULL){
		fprintf(stderr, "Error allocating memory for Edesc\n");
		return NULL;
	}
	e->datatype = DATA_ENVVAR;
	e->vars = testenvs;
	e->len = sizeof(testenvs);

	n = listcreate(e);
	if(n == NULL){
		fprintf(stderr, "Error allocating Edesc node\n");
		free(e);
		return NULL;
	}
	listaddtail(head, n);

	return n;
}

Node *
ldescadd(Node *head, char *fname)
{
	Ldesc *e;
	Node *n;
	struct stat st;

	e = malloc(sizeof(Ldesc));
	if(e == NULL){
		fprintf(stderr, "Error allocating memory for Ldesc\n");
		return NULL;
	}
	e->datatype = DATA_LABELS;
	e->fname = strdup(fname);
	if(e->fname == NULL){
		fprintf(stderr, "Error allocating memory for e->fname\n");
		return NULL;
	}
	if(stat(e->fname, &st) < 0){
		perror("Error calling stat");
		free(e);
		return NULL;
	}
	e->len = st.st_size;
	n = listcreate(e);
	if(n == NULL){
		fprintf(stderr, "Error allocating Ldesc node\n");
		free(e);
		return NULL;
	}
	listaddtail(head, n);

	return n;
}

Node *
pdescadd(Node *head, char *fname, int argc, char *argv[])
{
	Pdesc *e;
	Node *n;
	int opt;
	char content[SIF_CONTENT_LEN] = { };
	Siffstype fstype = -1;
	Sifparttype parttype = -1;
	struct stat st;
	char *fstypestr;
	char *parttypestr;

	while((opt = getopt(argc, argv, "c:f:p:")) != -1){ /* Flawfinder: ignore */
		switch(opt){
		case 'c':
			strncpy(content, optarg, sizeof(content)-1);
			break;
		case 'f':
			fstypestr = uppercase(optarg);
			if(strncmp(fstypestr, "SQUASHFS", strlen("SQUASHFS")) == 0)
				fstype = FS_SQUASH;
			else if(strncmp(fstypestr, "EXT3", strlen("EXT3")) == 0)
				fstype = FS_EXT3;
			else
				fstype = 1000; /* unknown */
			free(fstypestr);
			break;
		case 'p':
			parttypestr = uppercase(optarg);
			if(strncmp(parttypestr, "SYSTEM", strlen("SYSTEM")) == 0)
				parttype = PART_SYSTEM;
			else if(strncmp(parttypestr, "DATA", strlen("DATA")) == 0)
				parttype = PART_DATA;
			else if(strncmp(parttypestr, "OVERLAY", strlen("OVERLAY")) == 0)
				parttype = PART_OVERLAY;
			else
				parttype = 1000; /* unknown */
			break;
		default:
			fprintf(stderr, "Error expecting -c CONTENT, -f FSTYPE and -p PARTTYPE\n");
			return NULL;
		}
		/* done parsing attributes for option 'P' */
		if(fstype != -1 && strlen(content) != 0 && parttype != -1)
			break;
	}
	if(strlen(content) == 0){
		fprintf(stderr, "Error invalid content string, use -c CONTENT\n");
		return NULL;
	}
	if(fstype == -1){
		fprintf(stderr, "Error extracting FSTYPE\n");
		return NULL;
	}
	if(parttype == -1){
		fprintf(stderr, "Error extracting PARTTYPE\n");
		return NULL;
	}

	e = malloc(sizeof(Pdesc));
	if(e == NULL){
		fprintf(stderr, "Error allocating memory for Pdesc\n");
		return NULL;
	}
	e->datatype = DATA_PARTITION;
	e->fname = strdup(fname);
	if(e->fname == NULL){
		fprintf(stderr, "Error allocating memory for e->fname\n");
		return NULL;
	}
	if(stat(e->fname, &st) < 0){
		perror("Error calling stat");
		free(e);
		return NULL;
	}
	e->len = st.st_size;
	e->fstype = fstype;
	e->parttype = parttype;
	strcpy(e->content, content); /* Flawfinder: ignore */

	n = listcreate(e);
	if(n == NULL){
		fprintf(stderr, "Error allocating Pdesc node\n");
		free(e);
		return NULL;
	}
	listaddtail(head, n);

	return n;
}

static char testsign[] = "-----BEGIN PGP SIGNED MESSAGE-----";
Node *
sdescadd(Node *head, int argc, char *argv[])
{
	Sdesc *e;
	Node *n;
	int opt;
	char entity[SIF_ENTITY_LEN] = { };
	int hashtype = -1;

	while((opt = getopt(argc, argv, "e:h:")) != -1){ /* Flawfinder: ignore */
		switch(opt){
		case 'e':
			strncpy(entity, optarg, sizeof(entity)-1);
			break;
		case 'h':
			hashtype = atoi(optarg);
			break;
		default:
			fprintf(stderr, "Error expecting -e ENTITY and -h HASHTYPE\n");
			return NULL;
		}
		/* done parsing attributes for option 'S' */
		if(hashtype != -1 && strlen(entity) != 0)
			break;
	}
	if(strlen(entity) == 0){
		fprintf(stderr, "Error invalid signing entity string, use -e ENTITY\n");
		return NULL;
	}
	if(hashtype == -1){
		fprintf(stderr, "Error extracting HASHTYPE, use -h HASHTYPE\n");
		return NULL;
	}

	e = malloc(sizeof(Sdesc));
	if(e == NULL){
		fprintf(stderr, "Error allocating memory for Sdesc\n");
		return NULL;
	}
	e->datatype = DATA_SIGNATURE;
	e->signature = testsign;
	e->len = sizeof(testsign);
	e->hashtype = hashtype;
	strcpy(e->entity, entity); /* Flawfinder: ignore */

	n = listcreate(e);
	if(n == NULL){
		fprintf(stderr, "Error allocating Sdesc node\n");
		free(e);
		return NULL;
	}
	listaddtail(head, n);

	return n;
}

int
main(int argc, char *argv[])
{
	int ret;
	int opt;
	int dopts = 0;
	int eopts = 0;
	int lopts = 0;
	int popts = 0;
	int sopts = 0;
	struct utsname name;
	Sifcreateinfo createinfo = { };
	Node *n;

	while((opt = getopt(argc, argv, "D:EL:P:S")) != -1){ /* Flawfinder: ignore */
		switch(opt){
		case 'D':
			n = ddescadd(&createinfo.deschead, optarg);
			if(n == NULL){
				fprintf(stderr, "Could not add a deffile descriptor\n");
				return -1;
			}
			dopts++;
			break;
		case 'E':
			n = edescadd(&createinfo.deschead);
			if(n == NULL){
				fprintf(stderr, "Could not add an envvar descriptor\n");
				return -1;
			}
			eopts++;
			break;
		case 'L':
			n = ldescadd(&createinfo.deschead, optarg);
			if(n == NULL){
				fprintf(stderr, "Could not add a JSON-labels descriptor\n");
				return -1;
			}
			lopts++;
			break;
		case 'P':
			n = pdescadd(&createinfo.deschead, optarg, argc, argv);
			if(n == NULL){
				fprintf(stderr, "Could not add a partition descriptor\n");
				return -1;
			}
			popts++;
			break;
		case 'S':
			n = sdescadd(&createinfo.deschead, argc, argv);
			if(n == NULL){
				fprintf(stderr, "Could not add a signature descriptor\n");
				return -1;
			}
			sopts++;
			break;
		default:
			usage(argv);
			return -1;
		}
	}
	if(popts == 0){
		fprintf(stderr, "Error: At least one partition (-P) is required\n");
		return -1;
	}
	if(optind >= argc){
		fprintf(stderr, "Error: Expected argument after options\n");
		usage(argv);
		return -1;
	}
	argc -= optind;
	argv += optind;

	createinfo.pathname = argv[0];
	createinfo.launchstr = SIF_LAUNCH;
	createinfo.sifversion = SIF_VERSION;
	createinfo.arch = SIF_ARCH_AMD64;
	uuid_generate(createinfo.uuid);
	if(uname(&name) < 0){
		fprintf(stderr, "Error: Calling uname failed\n");
		return -1;
	}
	if(!strncmp(name.machine, "x86_64", 6)){
		if(sizeof(void *) == 8)
			createinfo.arch = SIF_ARCH_AMD64;
		else
			createinfo.arch = SIF_ARCH_386;
	}else if(name.machine[0] == 'i' && name.machine[2] == '8' &&
	        name.machine[3] == '6')
		createinfo.arch = SIF_ARCH_386;
	else if(!strncmp(name.machine, "arm", 3) && sizeof(void *) == 4)
		createinfo.arch = SIF_ARCH_ARM;
	else if(!strncmp(name.machine, "arm", 3) && sizeof(void *) == 8)
		createinfo.arch = SIF_ARCH_AARCH64;
	else{
		fprintf(stderr, "Error: Cannot determine running arch\n");
		return -1;
	}


	ret = sif_create(&createinfo);
	if(ret < 0){
		fprintf(stderr, "Error creating SIF file %s: %s\n",
		        createinfo.pathname, sif_strerror(siferrno));
		return -1;
	}

	return 0;
}
