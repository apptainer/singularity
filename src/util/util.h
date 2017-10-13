/* 
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 *
 * Copyright (c) 2015-2017, Gregory M. Kurtzer. All rights reserved.
 * 
 * Copyright (c) 2016-2017, The Regents of the University of California,
 * through Lawrence Berkeley National Laboratory (subject to receipt of any
 * required approvals from the U.S. Dept. of Energy).  All rights reserved.
 * 
 * This software is licensed under a customized 3-clause BSD license.  Please
 * consult LICENSE file distributed with the sources of this project regarding
 * your rights to use or distribute this software.
 * 
 * NOTICE.  This Software was developed under funding from the U.S. Department of
 * Energy and the U.S. Government consequently retains certain rights. As such,
 * the U.S. Government has been granted for itself and others acting on its
 * behalf a paid-up, nonexclusive, irrevocable, worldwide license in the Software
 * to reproduce, distribute copies to the public, prepare derivative works, and
 * perform publicly and display publicly, and to permit other to do so. 
 * 
 */


#include <unistd.h>
#include <stdlib.h>
#include <stdio.h>

#include "util/message.h"

#ifndef __UTIL_H_
#define __UTIL_H_

#define TRUE 1
#define FALSE 0

struct tempfile {
    FILE *fp;
    int fd;
    char filename[64];
};

char *envar_get(char *name, char *allowed, int len);
char *envar_path(char *name);
int envar_defined(char *name);
int envar_set(char *key, char *value, int overwrite);
int intlen(int input);
char *int2str(int num);
char *joinpath(const char * path1, const char * path2);
char *strjoin(char *str1, char *str2);
char *uppercase(char *string);
void chomp_noline(char *str);
void chomp_comments(char *str);
void chomp(char *str);
int strlength(const char *string, int max_len);
int envclean(void);
char *random_string(int length);
void free_tempfile(struct tempfile *tf);
struct tempfile *make_tempfile(void);
struct tempfile *make_logfile(char *label);

// Given a const char * string containing a base-10 integer,
// try to convert to an C integer.
// This is a bit less error prone (and stricter!) than strtoll:
// - Returns -1 on error and sets errno appropriately.
// - On failure, output_num is not touched.
// - On success, sets output_num to the parsed value (if output_num
//   is not null).
// - If the whole string isn't consumed, then -1 is returned and
//   errno is set to EINVAL
int str2int(const char *input_str, long int *output_num);

struct passwd;

#define ABORT(a) do {singularity_message(ABRT, "Retval = %d\n", a); exit(a);} while (0)


#endif
