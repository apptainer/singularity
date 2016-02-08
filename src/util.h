/*
 *
 * Copyright (c) 2015-2016, Gregory M. Kurtzer
 * All rights reserved.
 *
 *
 * Copyright (c) 2015-2016, The Regents of the University of California,
 * through Lawrence Berkeley National Laboratory (subject to receipt of
 * any required approvals from the U.S. Dept. of Energy).
 * All rights reserved.
 *
 *
 */


int s_is_file(char *path);
int s_is_dir(char *path);
int s_is_exec(char *path);
int s_is_owner(char *path, uid_t uid);
int s_mkpath(char *dir, mode_t mode);
int s_rmdir(char *dir);



