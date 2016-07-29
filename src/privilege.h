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

typedef struct {
    int ready;
    uid_t uid;
    gid_t gid;
    gid_t *gids;
    size_t gids_count;
    int userns_ready;
    int disable_setgroups;
    uid_t orig_uid;
    uid_t orig_gid;
    int target_mode;  // Set to 1 if we are running in "target mode" (admin specifies UID/GID)
} s_privinfo;

int priv_userns_enabled();
int priv_target_mode();
uid_t priv_getuid();
gid_t priv_getgid();
const gid_t *priv_getgids();
int priv_getgidcount();

// These all return void because on failure they ABORT()
void update_uid_map(pid_t child, uid_t outside, int);
void update_gid_map(pid_t child, gid_t outside, int);
void priv_drop_perm(void);
void priv_drop(void);
void priv_escalate(void);
void priv_init(void);
// Initialize the user namespace from outside the container.
void priv_init_userns_outside();
// Finish initialization of user namespace; must be called inside
// the container but *before* PID namespaces are setup.
void priv_init_userns_inside();
