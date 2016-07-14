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


#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/types.h>
#include <pwd.h>
#include <errno.h> 
#include <string.h>
#include <grp.h>

#include "privilege.h"
#include "config.h"
#include "file.h"
#include "util.h"
#include "message.h"


int get_user_privs(struct s_privinfo *uinfo) {
    uinfo->uid = getuid();
    uinfo->gid = getgid();
    uinfo->gids_count = getgroups(0, NULL);

    message(DEBUG, "Called get_user_privs(struct s_privinfo *uinfo)\n");

    uinfo->gids = (gid_t *) malloc(sizeof(gid_t) * uinfo->gids_count);

    if ( getgroups(uinfo->gids_count, uinfo->gids) < 0 ) {
       message(ERROR, "Could not obtain current supplementary group list: %s\n", strerror(errno));
       ABORT(255);
    }

    uinfo->ready = 1;

    message(DEBUG, "Returning get_user_privs(struct s_privinfo *uinfo) = 0\n");
    return(0);
}


int escalate_privs(void) {

    message(DEBUG, "Called escalate_privs(void)\n");

    if ( seteuid(0) < 0 ) {
        message(ERROR, "Could not escalate effective user privileges: %s\n", strerror(errno));
        ABORT(255);
    }

    if ( setegid(0) < 0 ) {
        message(ERROR, "Could not escalate effective group privileges: %s\n", strerror(errno));
        ABORT(255);
    }

    message(DEBUG, "Returning escalate_privs(void) = 0\n");
    return(0);
}

int drop_privs(struct s_privinfo *uinfo) {

    message(DEBUG, "Called drop_privs(struct s_privinfo *uinfo)\n");

    if ( uinfo->ready != 1 ) {
        message(ERROR, "User info is not ready\n");
        ABORT(255);
    }

    if ( geteuid() == 0 ) {
        message(DEBUG, "Dropping privileges to GID = '%d'\n", uinfo->gid);
        if ( setegid(uinfo->gid) < 0 ) {
            message(ERROR, "Could not drop effective group privileges to gid %d: %s\n", uinfo->gid, strerror(errno));
            ABORT(255);
        }

        message(DEBUG, "Dropping privileges to UID = '%d'\n", uinfo->uid);
        if ( seteuid(uinfo->uid) < 0 ) {
            message(ERROR, "Could not drop effective user privileges to uid %d: %s\n", uinfo->uid, strerror(errno));
            ABORT(255);
        }

    } else {
        message(DEBUG, "Running as root, no privileges to drop\n");
    }

    message(DEBUG, "Confirming we have correct GID\n");
    if ( getgid() != uinfo->gid ) {
        message(ERROR, "Failed to drop effective group privileges to uid %d: %s\n", uinfo->uid, strerror(errno));
        ABORT(255);
    }

    message(DEBUG, "Confirming we have correct UID\n");
    if ( getuid() != uinfo->uid ) {
        message(ERROR, "Failed to drop effective user privileges to uid %d: %s\n", uinfo->uid, strerror(errno));
        ABORT(255);
    }

    message(DEBUG, "Returning drop_privs(struct s_privinfo *uinfo) = 0\n");
    return(0);
}

int drop_privs_perm(struct s_privinfo *uinfo) {

    message(DEBUG, "Called drop_privs_perm(struct s_privinfo *uinfo)\n");

    if ( uinfo->ready != 1 ) {
        message(ERROR, "User info is not ready\n");
        ABORT(255);
    }

    if ( geteuid() == 0 ) {
        message(DEBUG, "Resetting supplementary groups\n");
        if ( setgroups(uinfo->gids_count, uinfo->gids) < 0 ) {
            fprintf(stderr, "ABOFT: Could not reset supplementary group list: %s\n", strerror(errno));
            return(-1);
        }

        message(DEBUG, "Dropping real and effective privileges to GID = '%d'\n", uinfo->gid);
        if ( setregid(uinfo->gid, uinfo->gid) < 0 ) {
            message(ERROR, "Could not dump real and effective group privileges: %s\n", strerror(errno));
            ABORT(255);
        }

        message(DEBUG, "Dropping real and effective privileges to UID = '%d'\n", uinfo->uid);
        if ( setreuid(uinfo->uid, uinfo->uid) < 0 ) {
            message(ERROR, "Could not dump real and effective user privileges: %s\n", strerror(errno));
            ABORT(255);
        }

    } else {
        message(DEBUG, "Running as root, no privileges to drop\n");
    }

    message(DEBUG, "Confirming we have correct GID\n");
    if ( getgid() != uinfo->gid ) {
        message(ERROR, "Failed to drop effective group privileges to uid %d: %s\n", uinfo->uid, strerror(errno));
        ABORT(255);
    }

    message(DEBUG, "Confirming we have correct UID\n");
    if ( getuid() != uinfo->uid ) {
        message(ERROR, "Failed to drop effective user privileges to uid %d: %s\n", uinfo->uid, strerror(errno));
        ABORT(255);
    }

    message(DEBUG, "Returning drop_privs_perm(struct s_privinfo *uinfo) = 0\n");
    return(0);
}

