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

#ifndef __FILE_H_
#define __FILE_H_

char *file_id(char *path);
char *file_devino(char *path);
#include <sys/stat.h>
int chk_perms(char *path, mode_t mode);
int chk_mode(char *path, mode_t mode, mode_t mask);
int is_file(char *path);
int is_fifo(char *path);
int is_link(char *path);
int is_dir(char *path);
int is_exec(char *path);
int is_write(char *path);
int is_suid(char *path);
int is_owner(char *path, uid_t uid);
int is_blk(char *path);
int is_chr(char *path);
int s_mkpath(char *dir, mode_t mode);
int s_rmdir(char *dir);
int copy_file(char * source, char * dest);
char *filecat(char *path);
int fileput(char *path, char *string);
int filelock(const char *const filepath, int *const fdptr);
char *basedir(char *dir);

#endif
