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

#ifndef __SINGULARITY_UTIL_CRYPT_H_
#define __SINGULARITY_UTIL_CRYPT_H_


#define SIGN_COMMAND "gpg --clearsign"
#define VERIFY_COMMAND "gpg --verify"
#define GPG_SIGNATURE_GOOD "gpg: Good signature"
#define SIFHASH_PREFIX "SIFHASH:\n"


/* a few quantities */
enum{
	SGN_HASHLEN = SHA384_DIGEST_LENGTH,
	SGN_MAXLEN = 2048,

	/* These values are meant to match sif.h Sifhashtype */
	SNG_SHA256 = 1,
	SNG_SHA384 = 2,
	SNG_SHA512 = 3,
	SNG_DEFAULT_HASH = 2
};

typedef enum{
	SGN_ENOERR,	/* Signing errno not set or success */
	SGN_EDUPOUT,	/* Could not duplicate stdout */
	SGN_EPIPE,	/* Could not create pipe */
	SGN_EDUP2OUT,	/* Could not duplicate stdout to pipe */
	SGN_EPSOPEN,	/* Popen failed with SIGN_COMMAND */
	SGN_EPIPESWR,	/* Could not write verifstr to pgp */
	SGN_EFPCLOSE,	/* Pclose failed: unsuccessful GPG operation */
	SGN_EDUP2RSTO,	/* Could not duplicate and restore stdout */
	SGN_ESOFLOW,	/* Buffer too small to hold signature */
	SGN_ERDPIPE,	/* Read error on pgp pipe stream */
	SGN_EDUPERR,	/* Could not duplicate stderr */
	SGN_EDUP2ERR,	/* Could not duplicate stderr to pipe */
	SGN_EPVOPEN,	/* Popen failed with VERIFY_COMMAND */
	SGN_EPIPEVWR,	/* Could not write verifblock to pgp */
	SGN_EDUP2RSTE,	/* Could not duplicate and restore stderr */
	SGN_EVOFLOW,	/* Response buffer too small to hold pgp output */
	SGN_EPCLOSE,	/* Could not close pipe descriptor */
	SGN_ECLOSEOUT,	/* Could not close saved stdout fd */
	SGN_ECLOSEERR,	/* Could not close saved stderr fd */
	SGN_EFNAME,	/* Invalid input file name */
	SGN_EFOPEN,	/* Cannot open input file name */
	SGN_EFSTAT,	/* fstat on input file failed */
	SGN_EFMAP,	/* Cannot mmap input file */
	SGN_EGPGV,	/* Gpg reports an invalid signature */
	SGN_ENOHASH,	/* No hash found in signature message */
	SGN_ESTRDUP	/* Error duplicating signedhash string */
} Sgnerrno;


extern Sgnerrno sgnerrno;

char *sgn_strerror(Sgnerrno sgnerrno);
void sgn_hashtostr(char *hash, char *hashstr);
void sgn_sifhashstr(char *hashstr, char *sifhashstr);
int sgn_getsignedhash(char *signedhash, char *hashstr);

unsigned char *sgn_hashbuffer(char *data, size_t size, char *result);
unsigned char *sgn_hashfile(char *fname, char *result);
int sgn_signhash(char *hashstr, char *signedhash);
int sgn_verifyhash(char *signedhash);

#endif /* __SINGULARITY_UTIL_CRYPT_H_ */
