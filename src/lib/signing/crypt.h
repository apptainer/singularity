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

#ifndef __SINGULARITY_UTIL_CRYPT_H_
#define __SINGULARITY_UTIL_CRYPT_H_

#include <openssl/sha.h>


#define SIGN_COMMAND "gpg --clearsign"
#define VERIFY_COMMAND "gpg --verify"
#define GPG_SIGNATURE_GOOD "gpg: Good signature"


/* a few quantities */
enum{
	SIGN_HASH_LEN = SHA384_DIGEST_LENGTH,
	SIGN_MAXLEN = 4096
};

typedef enum{
	SIGN_EDUPOUT,	/* Could not duplicate stdout */
	SIGN_EPIPE,	/* Could not create pipe */
	SIGN_EDUP2OUT,	/* Could not duplicate stdout to pipe */
	SIGN_EPSOPEN,	/* Popen failed with SIGN_COMMAND */
	SIGN_EPIPESWR,	/* Could not write verifstr to pgp */
	SIGN_EFPCLOSE,	/* Could not close the pgp pipe stream */
	SIGN_EDUP2RSTO,	/* Could not duplicate and restore stdout */
	SIGN_ESOFLOW,	/* Buffer too small to hold signature */
	SIGN_ERDPIPE,	/* Read error on pgp pipe stream */
	SIGN_EDUPERR,	/* Could not duplicate stderr */
	SIGN_EDUP2ERR,	/* Could not duplicate stderr to pipe */
	SIGN_EPVOPEN,	/* Popen failed with VERIFY_COMMAND */
	SIGN_EPIPEVWR,	/* Could not write verifblock to pgp */
	SIGN_EDUP2RSTE,	/* Could not duplicate and restore stderr */
	SIGN_EVOFLOW,	/* Response buffer too small to hold pgp output */
	SIGN_EPCLOSE,	/* Could not close pipe descriptor */
	SIGN_ECLOSEOUT,	/* Could not close saved stdout fd */
	SIGN_ECLOSEERR	/* Could not close saved stderr fd */
} Signerrno;


extern Signerrno signerrno;

char *sign_strerror(Signerrno signerrno);

unsigned char *compute_buffer_hash(unsigned char *data, size_t size, unsigned char *result);
unsigned char *compute_file_hash(char *fname, unsigned char *result);

int sign_hash(char *hashstr, char *signedhash);
int verify_signedhash(char *signedhash);

#endif /* __SINGULARITY_UTIL_CRYPT_H_ */
