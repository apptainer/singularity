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

#include <errno.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <uuid/uuid.h>

#include "crypt.h"

Signerrno signerrno;

char *
sign_strerror(Signerrno signerrno)
{
	switch(signerrno){
	case SIGN_EDUPOUT: return "Could not duplicate stdout";
	case SIGN_EPIPE: return "Could not create pipe";
	case SIGN_EDUP2OUT: return "Could not duplicate stdout to pipe";
	case SIGN_EPSOPEN: return "Popen failed with SIGN_COMMAND";
	case SIGN_EPIPESWR: return "Could not write verifstr to pgp";
	case SIGN_EFPCLOSE: return "Could not close the pgp pipe stream";
	case SIGN_EDUP2RSTO: return "Could not duplicate and restore stdout";
	case SIGN_ESOFLOW: return "Buffer too small to hold signature";
	case SIGN_ERDPIPE: return "Read error on pgp pipe stream";
	case SIGN_EDUPERR: return "Could not duplicate stderr";
	case SIGN_EDUP2ERR: return "Could not duplicate stderr to pipe";
	case SIGN_EPVOPEN: return "Popen failed with VERIFY_COMMAND";
	case SIGN_EPIPEVWR: return "Could not write verifblock to pgp";
	case SIGN_EDUP2RSTE: return "Could not duplicate and restore stderr";
	case SIGN_EVOFLOW: return "Response buffer too small to hold pgp output";
	case SIGN_EPCLOSE: return "Could not close pipe descriptor";
	case SIGN_ECLOSEOUT: return "Could not close saved stdout fd";
	case SIGN_ECLOSEERR: return "Could not close saved stderr fd";
	default: return "Unknown Signing error";
	}
}

unsigned char *
compute_buffer_hash(unsigned char *data, size_t size, unsigned char *result)
{
	return SHA384(data, size, result);
}

unsigned char *
compute_file_hash(char *fname, unsigned char *result)
{
	// return SHA384(data, size, result);
	return NULL;
}

int
sign_hash(char *hashstr, char *signedhash)
{
	int ret = -1;
	FILE *pfp;
	int p[2];
	int stdoutfd;

	stdoutfd = dup(1);		/* save the original stdout */
	if(stdoutfd < 0){
		signerrno = SIGN_EDUPOUT;
		return -1;
	}

	if(pipe(p) < 0){		/* create the pgp run pipe */
		signerrno = SIGN_EPIPE;
		close(stdoutfd);
	}else if(dup2(p[1], 1) < 0){	/* replace stdout with pipe write end */
		signerrno = SIGN_EDUP2OUT;
		close(p[0]);
		close(p[1]);
		close(stdoutfd);
	}else if(close(p[1]) < 0){	/* close pipe write end duplicate */
		signerrno = SIGN_EPCLOSE;
		dup2(stdoutfd, 1);	/* restore stdout and close other fd's */
		close(p[0]);
		close(stdoutfd);
	}else if((pfp = popen(SIGN_COMMAND, "w")) == NULL){
		signerrno = SIGN_EPSOPEN;
		dup2(stdoutfd, 1);	/* restore stdout and close other fd's */
		close(p[0]);
		close(stdoutfd);
	}else if(fputs(hashstr, pfp) == EOF){
		signerrno = SIGN_EPIPESWR;
		pclose(pfp);
		dup2(stdoutfd, 1);	/* restore stdout and close other fd's */
		close(p[0]);
		close(stdoutfd);
	}else if(pclose(pfp) < 0){
		signerrno = SIGN_EFPCLOSE;
		dup2(stdoutfd, 1);	/* restore stdout and close other fd's */
		close(p[0]);
		close(stdoutfd);
	}else if(dup2(stdoutfd, 1) < 0){
		signerrno = SIGN_EDUP2RSTO;
		close(p[0]);
		close(stdoutfd);
	}else if(close(stdoutfd) < 0){
		signerrno = SIGN_ECLOSEOUT;
		close(p[0]);
	}else{
		for( ; ; ){
			ret = read(p[0], signedhash, SIGN_MAXLEN);
			if(ret == SIGN_MAXLEN && read(p[0], signedhash, 1) != 0){
				signerrno = SIGN_ESOFLOW;
				close(p[0]);
				break;
			}
			if(ret < 0){
				if(errno == EINTR){
					continue;
				}else{
					signerrno = SIGN_ERDPIPE;
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
verify_signedhash(char *signedhash)
{
	int ret = -1;
	FILE *pfp;
	int p[2];
	int stderrfd;
	static char response[2048];

	stderrfd = dup(2);		/* save the original stderr */
	if(stderrfd < 0){
		signerrno = SIGN_EDUPERR;
		return -1;
	}

	if(pipe(p) < 0){		/* create the pgp run pipe */
		signerrno = SIGN_EPIPE;
		close(stderrfd);
	}else if(dup2(p[1], 2) < 0){	/* replace stderr with pipe write end */
		signerrno = SIGN_EDUP2ERR;
		close(p[0]);
		close(p[1]);
		close(stderrfd);
	}else if(close(p[1]) < 0){	/* close pipe write end duplicate */
		signerrno = SIGN_EPCLOSE;
		dup2(stderrfd, 2);	/* restore stderr and close other fd's */
		close(p[0]);
		close(stderrfd);
	}else if((pfp = popen(VERIFY_COMMAND, "w")) == NULL){
		signerrno = SIGN_EPVOPEN;
		dup2(stderrfd, 2);	/* restore stderr and close other fd's */
		close(p[0]);
		close(stderrfd);
	}else if(fputs(signedhash, pfp) == EOF){
		signerrno = SIGN_EPIPEVWR;
		pclose(pfp);
		dup2(stderrfd, 2);	/* restore stderr and close other fd's */
		close(p[0]);
		close(stderrfd);
	}else if(pclose(pfp) < 0){
		signerrno = SIGN_EFPCLOSE;
		dup2(stderrfd, 2);	/* restore stderr and close other fd's */
		close(p[0]);
		close(stderrfd);
	}else if(dup2(stderrfd, 2) < 0){
		signerrno = SIGN_EDUP2RSTE;
		close(p[0]);
		close(stderrfd);
	}else if(close(stderrfd) < 0){
		signerrno = SIGN_ECLOSEERR;
		close(p[0]);
	}else{
		for( ; ; ){
			ret = read(p[0], response, sizeof(response));
			if(ret == sizeof(response) && read(p[0], response, 1) != 0){
				signerrno = SIGN_EVOFLOW;
				close(p[0]);
				break;
			}
			if(ret < 0){
				if(errno == EINTR){
					continue;
				}else{
					signerrno = SIGN_ERDPIPE;
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
