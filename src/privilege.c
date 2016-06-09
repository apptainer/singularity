/* 
 * Copyright (c) 2015-2016, Gregory M. Kurtzer. All rights reserved.
 * 
 * “Singularity” Copyright (c) 2016, The Regents of the University of California,
 * through Lawrence Berkeley National Laboratory (subject to receipt of any
 * required approvals from the U.S. Dept. of Energy).  All rights reserved.
 * 
 * If you have questions about your rights to use or distribute this software,
 * please contact Berkeley Lab's Innovation & Partnerships Office at
 * IPO@lbl.gov.
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

#include "config.h"
#include "file.h"
#include "util.h"
#include "privilege.h"


int get_user_privs(struct s_privinfo *uinfo) {
    uinfo->uid = getuid();
    uinfo->gid = getgid();
    uinfo->gids_count = getgroups(0, NULL);

    uinfo->gids = (gid_t *) malloc(sizeof(gid_t) * uinfo->gids_count);

    if ( getgroups(uinfo->gids_count, uinfo->gids) < 0 ) {
       fprintf(stderr, "ERROR: Could not obtain current supplementary group list: %s\n", strerror(errno));
       return(-1);
    }

    uinfo->ready = 1;

    return(0);
}


int escalate_privs(void) {

    if ( seteuid(0) < 0 ) {
        fprintf(stderr, "ERROR: Could not escalate effective user privileges %s\n", strerror(errno));
        return(-1);
    }

    if ( setegid(0) < 0 ) {
        fprintf(stderr, "ERROR: Could not escalate effective group privileges: %s\n", strerror(errno));
        return(-1);
    }

    return(0);
}

int drop_privs(struct s_privinfo *uinfo) {

    if ( uinfo->ready != 1 ) {
        fprintf(stderr, "ERROR: User info is not ready\n");
        return(-1);
    }

    if ( getuid() == 0 ) {
        if ( seteuid(uinfo->uid) < 0 ) {
            fprintf(stderr, "ERROR: Could not drop effective user privileges to uid %d: %s\n", uinfo->uid, strerror(errno));
            return(-1);
        }

        if ( setegid(uinfo->gid) < 0 ) {
            fprintf(stderr, "ERROR: Could not drop effective group privileges to gid %d: %s\n", uinfo->gid, strerror(errno));
            return(-1);
        }
    } else {
        fprintf(stderr, "ERROR: Can not drop privileges from non privileged access level\n");
    }

    return(0);
}

int drop_privs_perm(struct s_privinfo *uinfo) {

    if ( uinfo->ready != 1 ) {
        fprintf(stderr, "ERROR: User info is not ready\n");
        return(-1);
    }

    if ( getuid() == 0 ) {
        if ( setgroups(uinfo->gids_count, uinfo->gids) < 0 ) {
            fprintf(stderr, "ABOFT: Could not reset supplementary group list: %s\n", strerror(errno));
            return(-1);
        }
        if ( setregid(uinfo->gid, uinfo->gid) < 0 ) {
            fprintf(stderr, "ERROR: Could not dump real and effective group privileges: %s\n", strerror(errno));
            return(-1);
        }
        if ( setreuid(uinfo->uid, uinfo->uid) < 0 ) {
            fprintf(stderr, "ERROR: Could not dump real and effective user privileges: %s\n", strerror(errno));
            return(-1);
        }
    } else {
        fprintf(stderr, "ERROR: Can not drop privileges from non privileged access level\n");
    }

    return(0);
}

