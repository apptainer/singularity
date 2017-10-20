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


/*
 * Singularity Image Format SIF definition
 */


/*
 * Layout of a SIF file (example)
 *
 * .================================================================.
 * | GLOBAL HEADER: Sifheader                                       |
 * | - launch: "#!/usr/bin/env..."                                  |
 * | - magic: "SIF_MAGIC"                                           |
 * | - version: "1"                                                 |
 * | - arch: "4"                                                    |
 * | - uuid: b2659d4e-bd50-4ea5-bd17-eec5e54f918e                   |
 * | - ctime: 1504657553                                            |
 * | - ndesc: 4                                                     |
 * | - descoff: 88                                                  | --.
 * | - dataoff: 280                                                 |   |
 * | - datalen: 42111314                                            |   |
 * |----------------------------------------------------------------|   |
 * | DESCR[0]: Siflabels  					    | <-'
 * | - Sifcommon                                                    |
 * |   - datatype: DATA_LABELS                                      |
 * |   - groupid: inter-object relation                             |
 * |   - fileoff: #bytes from start                                 | --.
 * |   - filelen: #bytes used                                       |   |
 * |----------------------------------------------------------------|   |
 * | DESCR[1]: Sifdeffile                                           |   |
 * | - Sifcommon                                                    |   |
 * |   - datatype: DATA_LABELS                                      |   |
 * |   - groupid: inter-object relation                             |   |
 * |   - fileoff: #bytes from start                                 | ----.
 * |   - filelen: #bytes used                                       |   | |
 * |----------------------------------------------------------------|   | |
 * | DESCR[2]: Sifenvvar                                            |   | |
 * | - Sifcommon                                                    |   | |
 * |   - datatype: DATA_LABELS                                      |   | |
 * |   - groupid: inter-object relation                             |   | |
 * |   - fileoff: #bytes from start                                 | ------.
 * |   - filelen: #bytes used                                       |   | | |
 * |----------------------------------------------------------------|   | | |
 * | DESCR[3]: Sifsignature                                         |   | | |
 * | - Sifcommon                                                    |   | | |
 * |   - datatype: DATA_LABELS                                      |   | | |
 * |   - groupid: inter-object relation                             |   | | |
 * |   - fileoff: #bytes from start                                 | --------.
 * |   - filelen: #bytes used                                       |   | | | |
 * | - hashtype: HASH_SHA384                                        |   | | | |
 * | - entity: "Joe Bloe <jbloe..."                                 |   | | | |
 * |----------------------------------------------------------------|   | | | |
 * | DESCR[4]: Sifpartition                                         |   | | | |
 * | - Sifcommon                                                    |   | | | |
 * |   - datatype: DATA_LABELS                                      |   | | | |
 * |   - groupid: inter-object relation                             |   | | | |
 * |   - fileoff: #bytes from start                                 | ----------.
 * |   - filelen: #bytes used                                       |   | | | | |
 * | - fstype: FS_SQUASH                                            |   | | | | |
 * | - parttype: PART_SYSTEM                                        |   | | | | |
 * | - content: "RHEL 7.4 / kernel 3.10.0-693 / Webmail server"     |   | | | | |
 * |----------------------------------------------------------------| <-' | | | |
 * | JSON labels data                                               |     | | | |
 * | .                                                              |     | | | |
 * | .                                                              |     | | | |
 * | .                                                              |     | | | |
 * |----------------------------------------------------------------| <---' | | |
 * | Definition file data                                           |       | | |
 * | .                                                              |       | | |
 * | .                                                              |       | | |
 * | .                                                              |       | | |
 * |----------------------------------------------------------------| <-----' | |
 * | Environment variables data                                     |         | |
 * | .                                                              |         | |
 * | .                                                              |         | |
 * | .                                                              |         | |
 * |----------------------------------------------------------------| <-------' |
 * | Signed verification data                                       |           |
 * | .                                                              |           |
 * | .                                                              |           |
 * | .                                                              |           |
 * |----------------------------------------------------------------| <---------'
 * | File system partition image                                    |
 * | .                                                              |
 * | .                                                              |
 * | .                                                              |
 * `================================================================'
 */

#ifndef __SINGULARITY_SIF_H_
#define __SINGULARITY_SIF_H_

#define SIF_LAUNCH	"#!/usr/bin/env run-singularity\n"
#define SIF_MAGIC	"SIF_MAGIC"
#define SIF_VERSION	"0"
#define SIF_ARCH_386	"2"
#define SIF_ARCH_AMD64	"4"
#define SIF_ARCH_ARM	"8"
#define SIF_ARCH_AARCH64 "16"

/* various SIF related quantities */
enum{
	SIF_LAUNCH_LEN = 32,		/* sizeof("#!/usr/bin/env... "); */
	SIF_MAGIC_LEN = 10,		/* sizeof("SIF_MAGIC") */
	SIF_VERSION_LEN = 3,		/* sizeof("99"); */
	SIF_ARCH_LEN = 3,		/* sizeof("99"); */
	SIF_ENTITY_LEN = 64,		/* "Joe Bloe <jbloe@gmail.com>..." */
	SIF_CONTENT_LEN = 256,		/* "RHEL 7.4 / kernel 3.10.0-693 / ..." */

	SIF_DEFAULT_GROUP = 0		/* first groupid number created */
};

/* types of data objects stored in the image */
typedef enum{
	DATA_DEFFILE = 0x4001,		/* definition file data object */
	DATA_ENVVAR,			/* environment variables data object */
	DATA_LABELS,			/* JSON labels data object */
	DATA_PARTITION,			/* file system data object */
	DATA_SIGNATURE			/* signing/verification data object */
} Sifdatatype;

/* types of file systems found in partition data objects */
typedef enum{
	FS_SQUASH,			/* Squashfs file system, RDONLY */
	FS_EXT3,			/* EXT3 file system, RDWR (deprecated) */
	FS_IMMOBJECTS,			/* immutable object archive */
	FS_RAW				/* raw data */
} Siffstype;

/* type of container partition and usage purpose */
typedef enum{
	PART_SYSTEM,			/* partition hosts an operating system */
	PART_DATA,			/* partition hosts data only */
	PART_OVERLAY			/* partition hosts an overlay */
}Sifconttype;

/* types of hashing function used to fingerprint data objects */
typedef enum{
	HASH_SHA256,
	HASH_SHA384,
	HASH_SHA512
} Sifhashtype;

/* SIF data object descriptor info common to all object type */
typedef struct Sifcommon Sifcommon;
struct Sifcommon{
	Sifdatatype datatype;		/* informs of descriptor type */
	int groupid;			/* object this data object is related to */
	off_t fileoff;			/* offset from start of image file */
	size_t filelen; 		/* length of data in file */
};

/* definition of an definition-file data object descriptor */
typedef struct Sifdeffile Sifdeffile;
struct Sifdeffile{
	Sifcommon cm;
};

/* definition of an JSON-labels data object descriptor */
typedef struct Siflabels Siflabels;
struct Siflabels{
	Sifcommon cm;
};

/* definition of an envvar data object descriptor */
typedef struct Sifenvvar Sifenvvar;
struct Sifenvvar{
	Sifcommon cm;
};

/* definition of an partition data object descriptor */
typedef struct Sifpartition Sifpartition;
struct Sifpartition{
	Sifcommon cm;
	Siffstype fstype;
	Sifconttype parttype;
	char content[SIF_CONTENT_LEN];
};

/* definition of an signature data object descriptor */
typedef struct Sifsignature Sifsignature;
struct Sifsignature{
	Sifcommon cm;
	Sifhashtype hashtype;
	char entity[SIF_ENTITY_LEN];
};

/* Singularity image format (SIF) global header */
typedef struct Sifheader Sifheader;
struct Sifheader{
	char launch[SIF_LAUNCH_LEN];	/* #! shell execution line */

	/* identify SIF version/support (ASCII) */
	char magic[SIF_MAGIC_LEN];	/* look for "SIF_MAGIC" */
	char version[SIF_VERSION_LEN];	/* SIF version */
	char arch[SIF_ARCH_LEN];	/* arch the image is built for */
	uuid_t uuid;			/* image unique identifier */

	/* start of common header */
	time_t ctime;			/* image creation time */

	/* info about data object descriptors */
	int ndesc;			/* total # of data object descr. */
	off_t descoff;			/* bytes into file where descs start */
	off_t dataoff;			/* bytes into file where data starts */
	size_t datalen;			/* combined size of all data objects */
};

typedef struct Sifinfo Sifinfo;
struct Sifinfo{
	Sifheader header;		/* the loaded SIF global header */
	int fd;				/* file descriptor of opened SIF file */
	size_t filesize;		/* file size of the opened SIF file */
	char *mapstart;			/* memory map of opened SIF file */
	Node deschead;			/* list of loaded descriptors from SIF file */
};


/*
 * This section describes SIF creation data structures used when building
 * a new SIF file. Transient data not found in the final SIF file.
 */

/* information needed to create an definition-file data object descriptor */
typedef struct Ddesc Ddesc;
struct Ddesc{
	Sifdatatype datatype;
	char *fname;
	int fd;
	unsigned char *mapstart;
	size_t len;
};

/* information needed to create an envvar data object descriptor */
typedef struct Edesc Edesc;
struct Edesc{
	Sifdatatype datatype;
	char *vars;
	size_t len;
};

/* information needed to create an JSON-labels data object descriptor */
typedef struct Ldesc Ldesc;
struct Ldesc{
	Sifdatatype datatype;
	char *fname;
	int fd;
	unsigned char *mapstart;
	size_t len;
};

/* information needed to create an partition data object descriptor */
typedef struct Pdesc Pdesc;
struct Pdesc{
	Sifdatatype datatype;
	char *fname;
	int fd;
	unsigned char *mapstart;
	size_t len;
	Siffstype fstype;
	Sifconttype parttype;
	char content[SIF_CONTENT_LEN];
};

/* information needed to create an signature data object descriptor */
typedef struct Sdesc Sdesc;
struct Sdesc{
	Sifdatatype datatype;
	char *signature;
	size_t len;
	Sifhashtype hashtype;
	char entity[SIF_ENTITY_LEN];
};

/* all creation info needed wrapped into a struct */
typedef struct Sifcreateinfo Sifcreateinfo;
struct Sifcreateinfo{
	char *pathname;		/* the end result output filename */
	char *launchstr;	/* the shell run command */
	char *sifversion;	/* the SIF specification version used */
	char *arch;		/* the architecture targetted */
	uuid_t uuid;		/* image unique identifier */
	Node deschead;		/* list head of info for all descriptors to create */
};


/*
 * description for diagnostics and utility routines
 */

typedef enum{
	SIF_ENOERR,	/* SIF errno not set or success */
	SIF_EMAGIC,	/* invalid SIF magic */
	SIF_EFNAME,	/* invalid input file name */
	SIF_EFOPEN,	/* cannot open input file name */
	SIF_EFSTAT,	/* fstat on input file failed */
	SIF_EFMAP,	/* cannot mmap input file */
	SIF_ELNOMEM,	/* cannot allocate memory for list node */
	SIF_EFUNMAP,	/* cannot munmap input file */
	SIF_EUNAME,	/* uname error while validating image */
	SIF_EUARCH,	/* unknown host architecture while validating image */
	SIF_ESIFVER,	/* unsupported SIF version while validating image */
	SIF_ERARCH,	/* architecture mismatch while validating image */
	SIF_ENODESC,	/* cannot find data object descriptors while validating image */
	SIF_ENODEF,	/* cannot find definition file descriptor */
	SIF_ENOENV,	/* cannot find envvar descriptor */
	SIF_ENOLAB,	/* cannot find jason label descriptor */
	SIF_ENOPAR,	/* cannot find partition descriptor */
	SIF_ENOSIG,	/* cannot find signature descriptor */
	SIF_EFDDEF,	/* cannot open definition file */
	SIF_EMAPDEF,	/* cannot mmap definition file */
	SIF_EFDLAB,	/* cannot open jason-labels file */
	SIF_EMAPLAB,	/* cannot mmap jason-labels file */
	SIF_EFDPAR,	/* cannot open partition file */
	SIF_EMAPPAR,	/* cannot mmap partition file */
	SIF_EUDESC,	/* unknown data descriptor type */
	SIF_EEMPTY,	/* nothing to generate into SIF file (empty) */
	SIF_ECREAT,	/* cannot create output SIF file, check permissions */
	SIF_EFALLOC,	/* fallocate on SIF output file failed */
	SIF_EOMAP,	/* cannot mmap SIF output file */
	SIF_EOUNMAP,	/* cannot unmmap SIF output file */
	SIF_EOCLOSE	/* closing SIF file failed, file corrupted, don't use */
} Siferrno;


/*
 * SIF API and exported routines
 */

extern Siferrno siferrno;

char *sif_strerror(Siferrno siferrno);
void printsifhdr(Sifinfo *info);
int sif_load(char *filename, Sifinfo *info);
int sif_unload(Sifinfo *info);
int sif_create(Sifcreateinfo *cinfo);
Sifheader *sif_getheader(Sifinfo *info);
Sifdeffile *sif_getdeffile(Sifinfo *info, int groupid);
Siflabels *sif_getlabels(Sifinfo *info, int groupid);
Sifenvvar *sif_getenvvar(Sifinfo *info, int groupid);
Sifpartition *sif_getpartition(Sifinfo *info, int groupid);
Sifsignature *sif_getsignature(Sifinfo *info, int groupid);

#endif /* __SINGULARITY_SIF_H_ */
