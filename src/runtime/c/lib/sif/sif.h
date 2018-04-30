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


/*
 * Singularity Image Format SIF definition
 */


/*
 * Layout of a SIF file (example)
 *
 * .================================================.
 * | GLOBAL HEADER: Sifheader                       |
 * | - launch: "#!/usr/bin/env..."                  |
 * | - magic: "SIF_MAGIC"                           |
 * | - version: "1"                                 |
 * | - arch: "4"                                    |
 * | - uuid: b2659d4e-bd50-4ea5-bd17-eec5e54f918e   |
 * | - ctime: 1504657553                            |
 * | - mtime: 1504657653                            |
 * | - ndesc: 3                                     |
 * | - descoff: 120                                 | --.
 * | - desclen: 432                                 |   |
 * | - dataoff: 4096                                |   |
 * | - datalen: 619362                              |   |
 * |------------------------------------------------| <-'
 * | DESCR[0]: Sifdeffile                           |
 * | - Sifcommon                                    |
 * |   - datatype: DATA_DEFFILE                     |
 * |   - id: 1                                      |
 * |   - groupid: 1                                 |
 * |   - link: NONE                                 |
 * |   - fileoff: 4096                              | --.
 * |   - filelen: 222                               |   |
 * |------------------------------------------------|   |
 * | DESCR[1]: Sifpartition                         |   |
 * | - Sifcommon                                    |   |
 * |   - datatype: DATA_PARTITION                   |   |
 * |   - id: 2                                      |   |
 * |   - groupid: 1                                 |   |
 * |   - link: NONE                                 |   |
 * |   - fileoff: 4318                              | ----.
 * |   - filelen: 618496                            |   | |
 * | - fstype: Squashfs                             |   | |
 * | - parttype: System                             |   | |
 * | - content: Linux                               |   | |
 * |------------------------------------------------|   | |
 * | DESCR[2]: Sifsignature                         |   | |
 * | - Sifcommon                                    |   | |
 * |   - datatype: DATA_SIGNATURE                   |   | |
 * |   - id: 3                                      |   | |
 * |   - groupid: NONE                              |   | |
 * |   - link: 2                                    |   | |
 * |   - fileoff: 622814                            | ------.
 * |   - filelen: 644                               |   | | |
 * | - hashtype: SHA384                             |   | | |
 * | - entity: @                                    |   | | |
 * |------------------------------------------------| <-' | |
 * | Definition file data                           |     | |
 * | .                                              |     | |
 * | .                                              |     | |
 * | .                                              |     | |
 * |------------------------------------------------| <---' |
 * | File system partition image                    |       |
 * | .                                              |       |
 * | .                                              |       |
 * | .                                              |       |
 * |------------------------------------------------| <-----'
 * | Signed verification data                       |
 * | .                                              |
 * | .                                              |
 * | .                                              |
 * `================================================'
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
	SIF_CONTENT_LEN = 64,		/* "RHEL 7.4 / kernel 3.10.0-693 / ..." */

	SIF_GROUP_MASK = 0xf0000000,	/* groups start at that offset */
	SIF_UNUSED_GROUP = SIF_GROUP_MASK,/* descriptor without a group */
	SIF_DEFAULT_GROUP = SIF_GROUP_MASK|1,/* first groupid number created */

	SIF_UNUSED_LINK = 0		/* descriptor without link to other */
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
	FS_SQUASH = 1,			/* Squashfs file system, RDONLY */
	FS_EXT3,			/* EXT3 file system, RDWR (deprecated) */
	FS_IMMOBJECTS,			/* immutable data object archive */
	FS_RAW				/* raw data */
} Siffstype;

/* type of container partition and usage purpose */
typedef enum{
	PART_SYSTEM = 1,		/* partition hosts an operating system */
	PART_DATA,			/* partition hosts data only */
	PART_OVERLAY			/* partition hosts an overlay */
} Sifparttype;

/* types of hashing function used to fingerprint data objects */
typedef enum{
	HASH_SHA256 = 1,
	HASH_SHA384,
	HASH_SHA512,
	HASH_BLAKE2S,
	HASH_BLAKE2B
} Sifhashtype;

enum{
	DEL_ZERO = 1,
	DEL_COMPACT
};

/* SIF data object descriptor info common to all object type */
typedef struct Sifcommon Sifcommon;
struct Sifcommon{
	Sifdatatype datatype;		/* informs of descriptor type */
	int id;				/* a unique id for this data object */
	int groupid;			/* object group this data object is related to */
	int link;			/* special link or relation to an id or group */
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
	Sifparttype parttype;
	char content[SIF_CONTENT_LEN];
};

/* definition of an signature data object descriptor */
typedef struct Sifsignature Sifsignature;
struct Sifsignature{
	Sifcommon cm;
	Sifhashtype hashtype;
	char entity[SIF_ENTITY_LEN];
};

typedef union Sifdescriptor Sifdescriptor;
union Sifdescriptor{
	Sifcommon cm;
	Sifdeffile def;
	Siflabels label;
	Sifenvvar env;
	Sifpartition part;
	Sifsignature sig;
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
	time_t mtime;			/* last modification time */

	/* info about data object descriptors */
	int ndesc;			/* total # of data object descr. */
	off_t descoff;			/* bytes into file where descs start */
	size_t desclen;			/* bytes used by all current descriptors */
	off_t dataoff;			/* bytes into file where data starts */
	size_t datalen;			/* bytes used by all data objects */
};

typedef struct Sifinfo Sifinfo;
struct Sifinfo{
	Sifheader header;		/* the loaded SIF global header */
	int nextid;			/* The next id to use for new descriptors */
	int fd;				/* file descriptor of opened SIF file */
	size_t filesize;		/* file size of the opened SIF file */
	char *mapstart;			/* memory map of opened SIF file */
	Node deschead;			/* list of loaded descriptors from SIF file */
};


/*
 * This section describes SIF creation data structures used when building
 * a new SIF file. Transient data not found in the final SIF file.
 */

/* common information needed to create a data object descriptor */
typedef struct Cmdesc Cmdesc;
struct Cmdesc{
	Sifdatatype datatype;
	int groupid;
	int link;
	size_t len;
};

/* information needed to create an definition-file data object descriptor */
typedef struct Defdesc Defdesc;
struct Defdesc{
	Cmdesc cm;
	char *fname;
	int fd;
	unsigned char *mapstart;
};

/* information needed to create an envvar data object descriptor */
typedef struct Envdesc Envdesc;
struct Envdesc{
	Cmdesc cm;
	char *vars;
};

/* information needed to create an JSON-labels data object descriptor */
typedef struct Labeldesc Labeldesc;
struct Labeldesc{
	Cmdesc cm;
	char *fname;
	int fd;
	unsigned char *mapstart;
};

/* information needed to create an partition data object descriptor */
typedef struct Partdesc Partdesc;
struct Partdesc{
	Cmdesc cm;
	char *fname;
	int fd;
	unsigned char *mapstart;
	Siffstype fstype;
	Sifparttype parttype;
	char content[SIF_CONTENT_LEN];
};

/* information needed to create an signature data object descriptor */
typedef struct Sigdesc Sigdesc;
struct Sigdesc{
	Cmdesc cm;
	char *signature;
	Sifhashtype hashtype;
	char entity[SIF_ENTITY_LEN];
};

/* Most SIF manipulations require Sifinfo and *desc */
typedef struct Eleminfo Eleminfo;
struct Eleminfo{
	Sifinfo *info;
	Sifdescriptor *desc;
	union{
		Cmdesc cm;
		Defdesc defdesc;
		Envdesc envdesc;
		Labeldesc labeldesc;
		Partdesc partdesc;
		Sigdesc sigdesc;
	};
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
	SIF_ENODESC,	/* cannot find data object descriptor(s) */
	SIF_ENODEF,	/* cannot find definition file descriptor */
	SIF_ENOENV,	/* cannot find envvar descriptor */
	SIF_ENOLAB,	/* cannot find jason label descriptor */
	SIF_ENOPAR,	/* cannot find partition descriptor */
	SIF_ENOSIG,	/* cannot find signature descriptor */
	SIF_ENOLINK,	/* cannot find descriptor linked to specified id */
	SIF_ENOID,	/* cannot find descriptor with specified id */
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
	SIF_EOCLOSE,	/* closing SIF file failed, file corrupted, don't use */
	SIF_EDNOMEM,	/* no more space to add new descriptors */
	SIF_ENOSUPP	/* operation not implemented/supported */
} Siferrno;


/*
 * SIF API and exported routines
 */

extern Siferrno siferrno;

char *sif_strerror(Siferrno errnum);

int sif_load(char *filename, Sifinfo *info, int rdonly);
int sif_unload(Sifinfo *info);

int sif_create(Sifcreateinfo *cinfo);
int sif_putdataobj(Eleminfo *e, Sifinfo *info);
int sif_deldataobj(Sifinfo *info, int id, int flags);

#endif /* __SINGULARITY_SIF_H_ */
