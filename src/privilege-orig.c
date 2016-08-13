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


#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <fcntl.h>
#include <sys/types.h>
#include <pwd.h>
#include <errno.h> 
#include <string.h>
#include <stdio.h>
#include <grp.h>
#ifdef SINGULARITY_USERNS
#include <sched.h>
#endif  // SINGULARITY_USERNS

#include "privilege.h"
#include "config.h"
#include "file.h"
#include "util.h"
#include "message.h"
#include "singularity.h"


void update_uid_map(pid_t child, uid_t outside, int is_child) {
    char * map_file;
    char * map;
    ssize_t map_len;
    int fd;

    message(DEBUG, "Updating UID map.\n");
    if (asprintf(&map_file, "/proc/%i/uid_map", child) < 0) {
        message(ERROR, "Can't allocate uid map filename\n");
        ABORT(255);
    }
    if (is_child) {
        map_len = asprintf(&map, "%i 0 1\n", outside);
    } else {
        map_len = asprintf(&map, "0 %i 1\n", outside);
    }
    if (map_len < 0) {
        free(map_file);
        message(ERROR, "Can't allocate uid map\n");
        ABORT(255);
    }

    message(DEBUG, "Updating UID map with policy: %s", map);
    fd = open(map_file, O_RDWR);
    free(map_file);
    if (fd == -1) {
        message(ERROR, "Failure when opening UID mapfile: %s\n", strerror(errno));
        free(map);
        ABORT(255);
    }
    if (write(fd, map, map_len) != map_len) {
        message(ERROR, "Failure when writing policy to mapfile: %s", strerror(errno));
        free(map);
        ABORT(255);
    }
    free(map);
    close(fd);
}


void update_gid_map(pid_t child, gid_t outside, int is_child) {
    char * setgroups_file, * map_file;
    char * map;
    ssize_t map_len;
    int fd;

    if (asprintf(&map_file, "/proc/%i/gid_map", child) < 0) {
        message(ERROR, "Can't allocate uid map filename\n");
        ABORT(255);
    }
    if (is_child) {
        map_len = asprintf(&map, "%i 0 1\n", outside);
    } else {
        map_len = asprintf(&map, "0 %i 1\n", outside);
    }
    if (map_len < 0) {
        free(map_file);
        message(ERROR, "Can't allocate gid map\n");
        ABORT(255);
    }
    if (asprintf(&setgroups_file, "/proc/%i/setgroups", child) < 0) {
        message(ERROR, "Can't allocate setgroups filename\n");
        ABORT(255);
    }
    message(DEBUG, "Disabling setgroups file.\n");
    fd = open(setgroups_file, O_RDWR);
    if (fd == -1) {
        free(setgroups_file);
        if (!is_child || (errno != EACCES)) {
            message(ERROR, "Failure when opening %s: %s\n", setgroups_file, strerror(errno));
            free(map_file);
            free(map);
            ABORT(255);
        }
    } else {
        free(setgroups_file);
        if (write(fd, "deny", 4) != 4) {
            message(ERROR, "Failure when writing setgroups deny: %s", strerror(errno));
            free(map_file);
            free(map);
            close(fd);
        }
        message(DEBUG, "Setgroups file successfully disabled.\n");
        close(fd);
    }

    message(DEBUG, "Updating GID map %s with policy: %s", map_file, map);
    fd = open(map_file, O_RDWR);
    if (fd == -1) {
        message(ERROR, "Failure when opening GID mapfile (%s): %s\n", map_file, strerror(errno));
        free(map_file);
        free(map);
        exit(-1);
    }
    if (write(fd, map, map_len) != map_len) {
        message(ERROR, "Failure when writing GID map (%s): %s", map_file, map);
        free(map);
        free(map_file);
        close(fd);
        exit(-1);
    }
    free(map);
    free(map_file);
    close(fd);
}


static s_privinfo uinfo;


void priv_init(void) {
    memset(&uinfo, '\0', sizeof(uinfo));

    // If we are *not* the setuid binary and started as root, then
    //
    long int target_uid = -1;
    long int target_gid = -1;
#ifdef SINGULARITY_NOSUID
    char *target_uid_str = NULL;
    char *target_gid_str = NULL;
    if ( getuid() == 0 ) {
        target_uid_str = getenv("SINGULARITY_TARGET_UID");
        target_gid_str = getenv("SINGULARITY_TARGET_GID");
        if ( target_uid_str && !target_gid_str ) {
            message(ERROR, "A target UID is set (%s) but a target GID is not set (SINGULARITY_TARGET_GID).  Both must be specified.\n", target_uid_str);
            ABORT(255);
        }
        if (target_uid_str) {
            if ( -1 == str2int(target_uid_str, &target_uid) ) {
                message(ERROR, "Unable to convert target UID (%s) to integer: %s\n", target_uid_str, strerror(errno));
                ABORT(255);
            }
            if (target_uid < 500) {
                message(ERROR, "Target UID (%ld) must be 500 or greater to avoid system users.\n", target_uid);
                ABORT(255);
            }
            if (target_uid > 65534) { // Avoid anything greater than the traditional overflow UID.
                message(ERROR, "Target UID (%ld) cannot be greater than 65534.\n", target_uid);
                ABORT(255);
            }
        }
        if ( !target_uid_str && target_gid_str ) {
            message(ERROR, "A target GID is set (%s) but a target UID is not set (SINGULARITY_TARGET_UID).  Both must be specified.\n", target_gid_str);
            ABORT(255);
        }
        if (target_gid_str) {
            if ( -1 == str2int(target_gid_str, &target_gid) ) {
                message(ERROR, "Unable to convert target GID (%s) to integer: %s\n", target_gid_str, strerror(errno));
                ABORT(255);
            }
            if (target_gid < 500) {
                message(ERROR, "Target GID (%ld) must be 500 or greater to avoid system groups.\n", target_gid);
                ABORT(255);
            }
            if (target_gid > 65534) { // Avoid anything greater than the traditional overflow GID.
                message(ERROR, "Target GID (%ld) cannot be greater than 65534.\n", target_gid);
                ABORT(255);
            }
        }
    }
#endif
    if ( (target_uid >= 500) && (target_gid >= 500) ) {
        uinfo.target_mode = 1;
        uinfo.uid = target_uid;
        uinfo.gid = target_gid;
        uinfo.gids_count = 0;
        uinfo.gids = NULL;
    } else {
        uinfo.uid = getuid();
        uinfo.gid = getgid();
        uinfo.gids_count = getgroups(0, NULL);

        message(DEBUG, "Called priv_init(void)\n");

        uinfo.gids = (gid_t *) malloc(sizeof(gid_t) * uinfo.gids_count);

        if ( getgroups(uinfo.gids_count, uinfo.gids) < 0 ) {
            message(ERROR, "Could not obtain current supplementary group list: %s\n", strerror(errno));
            ABORT(255);
        }
    }
    uinfo.ready = 1;

    priv_drop();

    message(DEBUG, "Returning priv_init(void)\n");
}


int priv_userns_enabled() {
    return uinfo.userns_ready;
}


int priv_target_mode() {
    if ( !uinfo.ready ) {
        message(ERROR, "Invoked before privilege info initialized!\n");
        ABORT(255);
    }
    return uinfo.target_mode;
}


uid_t priv_getuid() {
    if ( !uinfo.ready ) {
        message(ERROR, "Invoked before privilege info initialized!\n");
        ABORT(255);
    }
    return uinfo.uid;
}


gid_t priv_getgid() {
    if ( !uinfo.ready ) {
        message(ERROR, "Invoked before privilege info initialized!\n");
        ABORT(255);
    }
    return uinfo.gid;
}


const gid_t *priv_getgids() {
    if ( !uinfo.ready ) {
        message(ERROR, "Invoked before privilege info initialized!\n");
        ABORT(255);
    }
    return uinfo.gids;
}


int priv_getgidcount() {
    if ( !uinfo.ready ) {
        message(ERROR, "Invoked before privilege info initialized!\n");
        ABORT(255);
    }
    return uinfo.gids_count;
}


void priv_init_userns_outside() {
#ifdef SINGULARITY_USERNS
    if (!uinfo.ready) {
        message(ERROR, "Internal error: User NS initialization before general privilege initiation.\n");
        ABORT(255);
    }

    uinfo.orig_uid = uinfo.uid;
    uinfo.orig_gid = uinfo.gid;
    uinfo.orig_pid = getpid();

    int ret = unshare(CLONE_NEWUSER);
    if (ret == -1) {
        message(ERROR, "Failed to unshare namespace: %s.\n", strerror(errno));
        ABORT(255);
    }
    update_gid_map(uinfo.orig_pid, uinfo.orig_gid, 0);
    update_uid_map(uinfo.orig_pid, uinfo.orig_uid, 0);
    uinfo.uid = 0;
    uinfo.gid = 0;
    uinfo.userns_ready = 1;
#else  // SINGULARITY_USERNS
    message(ERROR, "Internal error: User NS function invoked without compiled-in support.\n");
    ABORT(255);
#endif  // SINGULARITY_USERNS
}

void priv_init_userns_inside_init() {
#ifdef SINGULARITY_USERNS
    if (!uinfo.userns_ready) {
        message(ERROR, "Internal error: User NS privilege data structure not initialized.\n");
        ABORT(255);
    }
    uinfo.uid = uinfo.orig_uid;
    uinfo.gid = uinfo.orig_gid;
#else  // SINGULARITY_USERNS
    message(ERROR, "Internal error: User NS function invoked without compiled-in support.\n");
    ABORT(255);
#endif  // SINGULARITY_USERNS
}


void priv_init_userns_inside_final() {
#ifdef SINGULARITY_USERNS
    if (!uinfo.userns_ready) {
        message(ERROR, "Internal error: User NS privilege data structure not initialized.\n");
        ABORT(255);
    }
    int ret = unshare(CLONE_NEWUSER);
    if (ret == -1) {
        message(ERROR, "Failed to unshare namespace: %s.\n", strerror(errno));
        ABORT(255);
    }
    update_gid_map(1, uinfo.orig_gid, 1);
    update_uid_map(1, uinfo.orig_uid, 1);
#else  // SINGULARITY_USERNS
    message(ERROR, "Internal error: User NS function invoked without compiled-in support.\n");
    ABORT(255);
#endif  // SINGULARITY_USERNS
}


void priv_escalate(void) {

    if ( getuid() != 0 ) {
        message(DEBUG, "Temporarily escalating privileges (U=%d)\n", getuid());

        if ( ( seteuid(0) < 0 ) || ( setegid(0) < 0 ) ) {
            message(ERROR, "The feature you are requesting requires privilege you do not have\n");
            ABORT(255);
        }

    } else {
        message(DEBUG, "Running as root, not changing privileges\n");
    }
}

void priv_drop(void) {

    if ( uinfo.ready != 1 ) {
        message(ERROR, "User info is not available\n");
        ABORT(255);
    }

    if ( getuid() != 0 ) {
        message(DEBUG, "Dropping privileges to UID=%d, GID=%d\n", uinfo.uid, uinfo.gid);

        if ( setegid(uinfo.gid) < 0 ) {
            message(ERROR, "Could not drop effective group privileges to gid %d: %s\n", uinfo.gid, strerror(errno));
            ABORT(255);
        }

        if ( seteuid(uinfo.uid) < 0 ) {
            message(ERROR, "Could not drop effective user privileges to uid %d: %s\n", uinfo.uid, strerror(errno));
            ABORT(255);
        }

        message(DEBUG, "Confirming we have correct UID/GID\n");
        if ( getgid() != uinfo.gid ) {
#ifdef SINGULARITY_NOSUID
            if ( uinfo.target_mode && getgid() != 0 ) {
                message(ERROR, "Non-zero real GID for target mode: %d\n", getgid());
                    ABORT(255);
                } else if ( !uinfo.target_mode )
#endif  // SINGULARITY_NOSUID
                {
                    message(ERROR, "Failed to drop effective group privileges to gid %d (currently %d)\n", uinfo.gid, getgid());
                    ABORT(255);
                }
            }

            if ( getuid() != uinfo.uid ) {
#ifdef SINGULARITY_NOSUID
            if ( uinfo.target_mode && getuid() != 0 ) {
                message(ERROR, "Non-zero real UID for target mode: %d\n", getuid());
                ABORT(255);
            } else if ( !uinfo.target_mode )
#endif  // SINGULARITY_NOSUID
            {
                message(ERROR, "Failed to drop effective user privileges to uid %d (currently %d)\n", uinfo.uid, getuid());
                ABORT(255);
            }
        }
    } else {
        message(DEBUG, "Running as root, not changing privileges\n");
    }
}

void priv_drop_perm(void) {
    message(DEBUG, "Called priv_drop_perm(void)\n");

    if ( uinfo.ready != 1 ) {
        message(ERROR, "User info is not available\n");
        ABORT(255);
    }

    return;

    if ( geteuid() == 0 ) {
        if ( !uinfo.userns_ready ) {
            message(DEBUG, "Resetting supplementary groups\n");
            if ( setgroups(uinfo->gids_count, uinfo->gids) < 0 ) {
                message(ERROR, "Could not reset supplementary group list: %s\n", strerror(errno));
                ABORT(255);
            }
        } else {
            message(DEBUG, "Not resetting supplementary groups as we are running in a user namespace.\n");
        }

        message(DEBUG, "Dropping real and effective privileges to GID = '%d'\n", uinfo.gid);
        if ( setregid(uinfo.gid, uinfo.gid) < 0 ) {
            message(ERROR, "Could not dump real and effective group privileges: %s\n", strerror(errno));
            ABORT(255);
        }

        message(DEBUG, "Dropping real and effective privileges to UID = '%d'\n", uinfo.uid);
        if ( setreuid(uinfo.uid, uinfo.uid) < 0 ) {
            message(ERROR, "Could not dump real and effective user privileges: %s\n", strerror(errno));
            ABORT(255);
        }

    } else {
        message(DEBUG, "Running as root, no privileges to drop\n");
    }

    message(DEBUG, "Confirming we have correct GID\n");
    if ( getgid() != uinfo.gid ) {
        message(ERROR, "Failed to drop effective group privileges to uid %d: %s\n", uinfo.uid, strerror(errno));
        ABORT(255);
    }

    message(DEBUG, "Confirming we have correct UID\n");
    if ( getuid() != uinfo.uid ) {
        message(ERROR, "Failed to drop effective user privileges to uid %d: %s\n", uinfo.uid, strerror(errno));
        ABORT(255);
    }

    message(DEBUG, "Returning priv_drop_perm(void)\n");
}

