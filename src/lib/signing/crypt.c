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

#include <sys/mman.h>
#include <sys/stat.h>

#include <fcntl.h>
#include <errno.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>

#include <openssl/sha.h>
#include <uuid/uuid.h>

#include "crypt.h"

Sgnerrno sgnerrno;

char *
sgn_strerror(Sgnerrno sgnerrno)
{
	switch(sgnerrno){
	case SGN_EDUPOUT: return "Could not duplicate stdout";
	case SGN_EPIPE: return "Could not create pipe";
	case SGN_EDUP2OUT: return "Could not duplicate stdout to pipe";
	case SGN_EPSOPEN: return "Popen failed with SIGN_COMMAND";
	case SGN_EPIPESWR: return "Could not write verifstr to pgp";
	case SGN_EFPCLOSE: return "Could not close the pgp pipe stream";
	case SGN_EDUP2RSTO: return "Could not duplicate and restore stdout";
	case SGN_ESOFLOW: return "Buffer too small to hold signature";
	case SGN_ERDPIPE: return "Read error on pgp pipe stream";
	case SGN_EDUPERR: return "Could not duplicate stderr";
	case SGN_EDUP2ERR: return "Could not duplicate stderr to pipe";
	case SGN_EPVOPEN: return "Popen failed with VERIFY_COMMAND";
	case SGN_EPIPEVWR: return "Could not write verifblock to pgp";
	case SGN_EDUP2RSTE: return "Could not duplicate and restore stderr";
	case SGN_EVOFLOW: return "Response buffer too small to hold pgp output";
	case SGN_EPCLOSE: return "Could not close pipe descriptor";
	case SGN_ECLOSEOUT: return "Could not close saved stdout fd";
	case SGN_ECLOSEERR: return "Could not close saved stderr fd";
	case SGN_EFNAME: return "Invalid input file name";
	case SGN_EFOPEN: return "Cannot open input file name";
	default: return "Unknown Signing-lib error";
	}
}

void
sgn_hashtostr(char *hash, char *hashstr)
{
	int i;

	for(i = 0; i < SGN_HASHLEN; i++){
		sprintf(&hashstr[i*2], "%02hhx", hash[i]);
	}
}

void
sgn_sifhashstr(char *hashstr, char *sifhashstr)
{
	strcpy(sifhashstr, SIFHASH_PREFIX);
	strncat(sifhashstr, hashstr, SGN_HASHLEN*2);
}

unsigned char *
sgn_hashbuffer(char *data, size_t size, char *result)
{
	return SHA384((unsigned char *)data, size, (unsigned char *)result);
}

unsigned char *
sgn_hashfile(char *fname, char *result)
{
	int fd;
	unsigned char *mapstart;
	unsigned char *ret;
	struct stat st;

	if(fname == NULL){
		sgnerrno = SGN_EFNAME;
		return NULL;
	}

	fd = open(fname, O_RDONLY);
	if(fd < 0){
		sgnerrno = SGN_EFOPEN;
		return NULL;
	}

	if(fstat(fd, &st) < 0){
		sgnerrno = SGN_EFSTAT;
		close(fd);
		return NULL;
	}

	mapstart = mmap(NULL, st.st_size, PROT_READ, MAP_PRIVATE, fd, 0);
	if(mapstart == MAP_FAILED){
		sgnerrno = SGN_EFMAP;
		close(fd);
		return NULL;
	}

	ret = SHA384(mapstart, st.st_size, (unsigned char *)result);

	munmap(mapstart, st.st_size);
	close(fd);

	return ret;
}

int
sgn_signhash(char *hashstr, char *signedhash)
{
	int ret = -1;
	FILE *pfp;
	int p[2];
	int stdoutfd;

	stdoutfd = dup(1);		/* save the original stdout */
	if(stdoutfd < 0){
		sgnerrno = SGN_EDUPOUT;
		return -1;
	}
	if(pipe(p) < 0){		/* create the pgp run pipe */
		sgnerrno = SGN_EPIPE;
		close(stdoutfd);
	}else if(dup2(p[1], 1) < 0){	/* replace stdout with pipe write end */
		sgnerrno = SGN_EDUP2OUT;
		close(p[0]);
		close(p[1]);
		close(stdoutfd);
	}else if(close(p[1]) < 0){	/* close pipe write end duplicate */
		sgnerrno = SGN_EPCLOSE;
		dup2(stdoutfd, 1);	/* restore stdout and close other fd's */
		close(p[0]);
		close(stdoutfd);
	}else if((pfp = popen(SIGN_COMMAND, "w")) == NULL){
		sgnerrno = SGN_EPSOPEN;
		dup2(stdoutfd, 1);	/* restore stdout and close other fd's */
		close(p[0]);
		close(stdoutfd);
	}else if(fputs(hashstr, pfp) == EOF){
		sgnerrno = SGN_EPIPESWR;
		pclose(pfp);
		dup2(stdoutfd, 1);	/* restore stdout and close other fd's */
		close(p[0]);
		close(stdoutfd);
	}else if(pclose(pfp) < 0){
		sgnerrno = SGN_EFPCLOSE;
		dup2(stdoutfd, 1);	/* restore stdout and close other fd's */
		close(p[0]);
		close(stdoutfd);
	}else if(dup2(stdoutfd, 1) < 0){
		sgnerrno = SGN_EDUP2RSTO;
		close(p[0]);
		close(stdoutfd);
	}else if(close(stdoutfd) < 0){
		sgnerrno = SGN_ECLOSEOUT;
		close(p[0]);
	}else{
		for( ; ; ){
			ret = read(p[0], signedhash, SGN_MAXLEN);
			if(ret == SGN_MAXLEN && read(p[0], signedhash, 1) != 0){
				sgnerrno = SGN_ESOFLOW;
				close(p[0]);
				break;
			}
			if(ret < 0){
				if(errno == EINTR){
					continue;
				}else{
					sgnerrno = SGN_ERDPIPE;
					close(p[0]);
					break;
				}
			}
			ret = 0;
			break;
		}
	}

	return ret;
}

int
sgn_verifyhash(char *signedhash)
{
	int ret = -1;
	FILE *pfp;
	int p[2];
	int stderrfd;
	static char response[2048];

	stderrfd = dup(2);		/* save the original stderr */
	if(stderrfd < 0){
		sgnerrno = SGN_EDUPERR;
		return -1;
	}

	if(pipe(p) < 0){		/* create the pgp run pipe */
		sgnerrno = SGN_EPIPE;
		close(stderrfd);
	}else if(dup2(p[1], 2) < 0){	/* replace stderr with pipe write end */
		sgnerrno = SGN_EDUP2ERR;
		close(p[0]);
		close(p[1]);
		close(stderrfd);
	}else if(close(p[1]) < 0){	/* close pipe write end duplicate */
		sgnerrno = SGN_EPCLOSE;
		dup2(stderrfd, 2);	/* restore stderr and close other fd's */
		close(p[0]);
		close(stderrfd);
	}else if((pfp = popen(VERIFY_COMMAND, "w")) == NULL){
		sgnerrno = SGN_EPVOPEN;
		dup2(stderrfd, 2);	/* restore stderr and close other fd's */
		close(p[0]);
		close(stderrfd);
	}else if(fputs(signedhash, pfp) == EOF){
		sgnerrno = SGN_EPIPEVWR;
		pclose(pfp);
		dup2(stderrfd, 2);	/* restore stderr and close other fd's */
		close(p[0]);
		close(stderrfd);
	}else if(pclose(pfp) < 0){
		sgnerrno = SGN_EFPCLOSE;
		dup2(stderrfd, 2);	/* restore stderr and close other fd's */
		close(p[0]);
		close(stderrfd);
	}else if(dup2(stderrfd, 2) < 0){
		sgnerrno = SGN_EDUP2RSTE;
		close(p[0]);
		close(stderrfd);
	}else if(close(stderrfd) < 0){
		sgnerrno = SGN_ECLOSEERR;
		close(p[0]);
	}else{
		for( ; ; ){
			ret = read(p[0], response, sizeof(response));
			if(ret == sizeof(response) && read(p[0], response, 1) != 0){
				sgnerrno = SGN_EVOFLOW;
				close(p[0]);
				break;
			}
			if(ret < 0){
				if(errno == EINTR){
					continue;
				}else{
					sgnerrno = SGN_ERDPIPE;
					close(p[0]);
					break;
				}
			}
			if(strstr(response, GPG_SIGNATURE_GOOD) != NULL)
				ret = 0;
			break;
		}
	}

	return ret;
}
