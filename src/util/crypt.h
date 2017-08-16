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

#define HASH_LEN SHA384_DIGEST_LENGTH

unsigned char *compute_hash(const unsigned char *data, size_t size, unsigned char *result);
int sign_verifblock(char *verifstr, char *verifblock);

#endif /* __SINGULARITY_UTIL_CRYPT_H_ */
