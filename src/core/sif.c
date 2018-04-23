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

#ifndef _XOPEN_SOURCE
#define _XOPEN_SOURCE 500
#endif

#include <sys/stat.h>
#include <sys/types.h>
#include <sys/utsname.h>

#include <libgen.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <strings.h>
#include <time.h>
#include <unistd.h>

#include <uuid/uuid.h>

#include "lib/sif/list.h"
#include "lib/sif/sif.h"
#include "lib/sif/sifaccess.h"

#include "util/util.h"

char *progname;

static void
usage(void)
{
	fprintf(stderr, "usage: %s COMMAND OPTION FILE\n", progname);
	fprintf(stderr, "\n\n");
	fprintf(stderr, "create --  Create a new sif file with input data objects\n");
	fprintf(stderr, "del    id  Delete a specified set of descriptor+object\n");
	fprintf(stderr, "dump   id  Display data object content\n");
	fprintf(stderr, "header --  Display SIF header\n");
	fprintf(stderr, "info   id  Print data object descriptor info\n");
	fprintf(stderr, "list   --  List SIF data descriptors from an input SIF file\n");
	fprintf(stderr, "\n\n");
	fprintf(stderr, "create options:\n");
	fprintf(stderr, "\t-D deffile : include definitions file `deffile'\n");
	fprintf(stderr, "\t-E : include environment variables\n");
	fprintf(stderr, "\t-P partfile : include file system partition `partfile'\n");
	fprintf(stderr, "\t\t-c CONTENT : freeform partition content string\n");
	fprintf(stderr, "\t\t-f FSTYPE : filesystem type: EXT3, SQUASHFS\n");
	fprintf(stderr, "\t\t-p PARTTYPE : filesystem partition type: SYSTEM, DATA, OVERLAY\n");
	fprintf(stderr, "\t\t-u uuid : pass a uuid to use instead of generating a new one\n");
	fprintf(stderr, "\n");
	fprintf(stderr, "example: sif create -P /tmp/fs.squash -f \"SQUASHFS\" -p \"SYSTEM\" -c \"Linux\" /tmp/container.sif\n\n");
}

Node *
ddescadd(Node *head, char *fname)
{
	Eleminfo *e;
	Node *n;
	struct stat st;

	e = malloc(sizeof(Eleminfo));
	if(e == NULL){
		fprintf(stderr, "Error allocating memory for Eleminfo\n");
		return NULL;
	}
	e->cm.datatype = DATA_DEFFILE;
	e->cm.groupid = SIF_DEFAULT_GROUP;
	e->cm.link = SIF_UNUSED_LINK;
	e->defdesc.fname = strdup(fname);
	if(e->defdesc.fname == NULL){
		fprintf(stderr, "Error allocating memory for e->defdesc.fname\n");
		return NULL;
	}
	if(stat(e->defdesc.fname, &st) < 0){
		perror("Error calling stat");
		free(e);
		return NULL;
	}
	e->cm.len = st.st_size;
	n = listcreate(e);
	if(n == NULL){
		fprintf(stderr, "Error allocating Eleminfo node\n");
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
	Eleminfo *e;
	Node *n;

	e = malloc(sizeof(Eleminfo));
	if(e == NULL){
		fprintf(stderr, "Error allocating memory for Eleminfo\n");
		return NULL;
	}
	e->cm.datatype = DATA_ENVVAR;
	e->cm.groupid = SIF_DEFAULT_GROUP;
	e->cm.link = SIF_UNUSED_LINK;
	e->cm.len = sizeof(testenvs);
	e->envdesc.vars = testenvs;

	n = listcreate(e);
	if(n == NULL){
		fprintf(stderr, "Error allocating Eleminfo node\n");
		free(e);
		return NULL;
	}
	listaddtail(head, n);

	return n;
}

Node *
ldescadd(Node *head, char *fname)
{
	Eleminfo *e;
	Node *n;
	struct stat st;

	e = malloc(sizeof(Eleminfo));
	if(e == NULL){
		fprintf(stderr, "Error allocating memory for Eleminfo\n");
		return NULL;
	}
	e->cm.datatype = DATA_LABELS;
	e->cm.groupid = SIF_DEFAULT_GROUP;
	e->cm.link = SIF_UNUSED_LINK;
	e->labeldesc.fname = strdup(fname);
	if(e->labeldesc.fname == NULL){
		fprintf(stderr, "Error allocating memory for e->labeldesc.fname\n");
		return NULL;
	}
	if(stat(e->labeldesc.fname, &st) < 0){
		perror("Error calling stat");
		free(e);
		return NULL;
	}
	e->cm.len = st.st_size;
	n = listcreate(e);
	if(n == NULL){
		fprintf(stderr, "Error allocating Eleminfo node\n");
		free(e);
		return NULL;
	}
	listaddtail(head, n);

	return n;
}

Node *
sdescadd(Node *head, char *signedhash, Sifhashtype hashtype)
{
	Eleminfo *e;
	Node *n;
	char entity[SIF_ENTITY_LEN] = { };

	e = malloc(sizeof(Eleminfo));
	if(e == NULL){
		fprintf(stderr, "Error allocating memory for Eleminfo\n");
		return NULL;
	}
	e->cm.datatype = DATA_SIGNATURE;
	e->cm.groupid = SIF_DEFAULT_GROUP;
	e->cm.link = SIF_UNUSED_LINK;
	e->cm.len = strlen(signedhash)+1;
	e->sigdesc.signature = strdup(signedhash);
	e->sigdesc.hashtype = hashtype;
	strcpy(e->sigdesc.entity, entity); /* Flawfinder: ignore */

	n = listcreate(e);
	if(n == NULL){
		fprintf(stderr, "Error allocating Eleminfo node\n");
		free(e);
		return NULL;
	}
	listaddtail(head, n);

	return n;
}

Node *
pdescadd(Node *head, char *fname, int argc, char *argv[])
{
	Eleminfo *e;
	Node *n;
	int opt;
	char content[SIF_CONTENT_LEN] = { };
	Siffstype fstype = -1;
	Sifparttype parttype = -1;
	struct stat st;

	while((opt = getopt(argc, argv, "c:f:p:")) != -1){ /* Flawfinder: ignore */
		switch(opt){
		case 'c':
			strncpy(content, optarg, sizeof(content)-1);
			break;
		case 'f':
			if(strncasecmp(optarg, "SQUASHFS", strlen("SQUASHFS")) == 0)
				fstype = FS_SQUASH;
			else if(strncasecmp(optarg, "EXT3", strlen("EXT3")) == 0)
				fstype = FS_EXT3;
			else
				fstype = 1000; /* unknown */
			break;
		case 'p':
			if(strncasecmp(optarg, "SYSTEM", strlen("SYSTEM")) == 0)
				parttype = PART_SYSTEM;
			else if(strncasecmp(optarg, "DATA", strlen("DATA")) == 0)
				parttype = PART_DATA;
			else if(strncasecmp(optarg, "OVERLAY", strlen("OVERLAY")) == 0)
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

	e = malloc(sizeof(Eleminfo));
	if(e == NULL){
		fprintf(stderr, "Error allocating memory for Eleminfo\n");
		return NULL;
	}
	e->cm.datatype = DATA_PARTITION;
	e->cm.groupid = SIF_DEFAULT_GROUP;
	e->cm.link = SIF_UNUSED_LINK;
	e->partdesc.fname = strdup(fname);
	if(e->partdesc.fname == NULL){
		fprintf(stderr, "Error allocating memory for e->partdesc.fname\n");
		return NULL;
	}
	if(stat(e->partdesc.fname, &st) < 0){
		perror("Error calling stat");
		free(e);
		return NULL;
	}
	e->cm.len = st.st_size;
	e->partdesc.fstype = fstype;
	e->partdesc.parttype = parttype;
	strcpy(e->partdesc.content, content); /* Flawfinder: ignore */

	n = listcreate(e);
	if(n == NULL){
		fprintf(stderr, "Error allocating Eleminfo node\n");
		free(e);
		return NULL;
	}
	listaddtail(head, n);

	return n;
}

int
cmd_create(int argc, char *argv[])
{
	int ret;
	int opt;
	int dopts = 0;
	int eopts = 0;
	int lopts = 0;
	int popts = 0;
	int extuuid = 0;
	struct utsname name;
	Sifcreateinfo createinfo = { };
	Node *n;

	/* get rid of command */
	argc--;
	argv++;

	while((opt = getopt(argc, argv, "u:D:EL:P:")) != -1){ /* Flawfinder: ignore */
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
		case 'u':
			if(uuid_parse(optarg, createinfo.uuid) < 0){
				fprintf(stderr, "Make sure the uuid passed is correctly formated:\n");
				fprintf(stderr, "Expecting format: %s\n", "`%08x-%04x-%04x-%04x-%012x'");
				return -1;
			}
			extuuid = 1;
			break;
		default:
			usage();
			return -1;
		}
	}
	if(popts == 0){
		fprintf(stderr, "Error: At least one partition (-P) is required\n");
		return -1;
	}
	if(optind >= argc){
		fprintf(stderr, "Error: Expected argument after options\n");
		usage();
		return -1;
	}
	argc -= optind;
	argv += optind;

	createinfo.pathname = argv[0];
	createinfo.launchstr = SIF_LAUNCH;
	createinfo.sifversion = SIF_VERSION;
	createinfo.arch = SIF_ARCH_AMD64;
	if(!extuuid)
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

int
cmd_list(int argc, char *argv[])
{
	Sifinfo sif;

	if(argc < 3){
		usage();
		return -1;
	}

	if(sif_load(argv[2], &sif, 1) < 0){
		fprintf(stderr, "Cannot load SIF image: %s\n", sif_strerror(siferrno));
		return(-1);
	}
	sif_printlist(&sif);

	sif_unload(&sif);

	return 0;
}

int
cmd_info(int argc, char *argv[])
{
	int id;
	Sifinfo sif;
	Sifdescriptor *desc;

	if(argc < 4){
		usage();
		return -1;
	}

	id = atoi(argv[2]);

	if(sif_load(argv[3], &sif, 1) < 0){
		fprintf(stderr, "Cannot load SIF image: %s\n", sif_strerror(siferrno));
		return(-1);
	}

	desc = sif_getdescid(&sif, id);
	if(desc == NULL){
		fprintf(stderr, "Cannot find descriptor %d from SIF file: %s\n", id,
		        sif_strerror(siferrno));
		sif_unload(&sif);
		return -1;
	}

	printf("Descriptor info:\n");
	printf("---------------------------\n");
	sif_printdesc(desc, NULL);

	sif_unload(&sif);

	return 0;
}

int
cmd_dump(int argc, char *argv[])
{
	int id;
	Sifinfo sif;
	Sifdescriptor *desc;
	char *c;

	if(argc < 4){
		usage();
		return -1;
	}

	id = atoi(argv[2]);

	if(sif_load(argv[3], &sif, 1) < 0){
		fprintf(stderr, "Cannot load SIF image: %s\n", sif_strerror(siferrno));
		return(-1);
	}

	desc = sif_getdescid(&sif, id);
	if(desc == NULL){
		fprintf(stderr, "Cannot find descriptor %d from SIF file: %s\n", id,
		        sif_strerror(siferrno));
		sif_unload(&sif);
		return -1;
	}

	for(c = sif.mapstart+desc->cm.fileoff;
	    c < sif.mapstart+desc->cm.fileoff+desc->cm.filelen;
	    c++)
		printf("%c", *c);

	sif_unload(&sif);

	return 0;
}

int
cmd_del(int argc, char *argv[])
{
	int ret;
	int id;
	Sifinfo sif;

	if(argc < 4){
		usage();
		return -1;
	}

	id = atoi(argv[2]);

	if(sif_load(argv[3], &sif, 0) < 0){
		fprintf(stderr, "Cannot load SIF image: %s\n", sif_strerror(siferrno));
		return(-1);
	}

	ret = sif_deldataobj(&sif, id, DEL_ZERO);
	if(ret < 0){
		fprintf(stderr, "Cannot delete object with id %d from SIF file: %s\n", id,
		        sif_strerror(siferrno));
		sif_unload(&sif);
		return -1;
	}

	sif_unload(&sif);

	return 0;
}

int
cmd_header(int argc, char *argv[])
{
	Sifinfo sif;

	if(argc < 3) {
		usage();
		return -1;
	}

	if(sif_load(argv[2], &sif, 1) < 0) {
		fprintf(stderr, "Cannot load SIF image: %s\n", sif_strerror(siferrno));
		return -1;
	}

	sif_printheader(&sif);

	sif_unload(&sif);

	return 0;
}

int
main(int argc, char *argv[])
{
	progname = basename(argv[0]);

	if(argc < 2){
		usage();
		return -1;
	}
	if(strncmp(argv[1], "create", 6) == 0)
		return cmd_create(argc, argv);
	if(strncmp(argv[1], "list", 4) == 0)
		return cmd_list(argc, argv);
	if(strncmp(argv[1], "info", 4) == 0)
		return cmd_info(argc, argv);
	if(strncmp(argv[1], "dump", 4) == 0)
		return cmd_dump(argc, argv);
	if(strncmp(argv[1], "del", 3) == 0)
		return cmd_del(argc, argv);
	if(strncmp(argv[1], "header", 6) == 0)
		return cmd_header(argc, argv);

	usage();
	return -1;
}
