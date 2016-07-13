/* 
 * Copyright (c) 2015-2016, Gregory M. Kurtzer. All rights reserved.
 * 
 * “Singularity” Copyright (c) 2016, The Regents of the University of California,
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


char *file_id(char *path);
int is_file(char *path);
int is_fifo(char *path);
int is_link(char *path);
int is_dir(char *path);
int is_exec(char *path);
int is_owner(char *path, uid_t uid);
int is_blk(char *path);
int s_mkpath(char *dir, mode_t mode);
int s_rmdir(char *dir);
int copy_file(char * source, char * dest);
char *filecat(char *path);
int fileput(char *path, char *string);
char * container_basedir(char *containerdir, char *dir);
