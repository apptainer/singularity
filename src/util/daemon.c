/* 
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 * 
 * This software is licensed under a 3-clause BSD license.  Please
 * consult LICENSE file distributed with the sources of this project regarding
 * your rights to use or distribute this software.
 * 
 */


#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <errno.h>
#include <string.h>
#include <fcntl.h>

#include "config.h"
#include "util/file.h"
#include "util/util.h"
#include "util/daemon.h"
#include "util/registry.h"
#include "util/message.c"
#include "lib/image/image.h"
#include "lib/runtime/runtime.h"
#include "util/privilege.h"

void daemon_file_parse(void) {
    singularity_message(DEBUG, "reached file parse\n");
    char *key, *val;
    char *line = (char *)malloc(2048 * sizeof(char *));
    FILE *file = fopen(singularity_registry_get("DAEMON_FILE"), "r");

    while( fgets(line, 2048, file) ) {
        key = strtok(line, "=\n");
        val = strtok(NULL, "=\n");
        singularity_message(DEBUG, "Read key-val pair %s=%s\n", key, val);
        singularity_registry_set(key, val);
    }
}

void daemon_file_write(int fd, char *key, char *val) {
    int retval = 0;
    errno = 0;
    
    singularity_message(DEBUG, "Called daemon_file_write(%d, %s, %s)\n", fd, key, val);
    retval += write(fd, key, strlength(key, 2048));
    retval += write(fd, "=", 1);
    retval += write(fd, val, strlength(val, 2048));
    retval += write(fd, "\n", 1);

    if ( errno != 0 ) {
        singularity_message(ERROR, "Unable to write to daemon file: %s\n", strerror(errno));
        ABORT(255);
    }
}

int daemon_is_owner(char *pid_path) {
    int retval = 0;
    char *proc_status = joinpath(pid_path, "/status");
    char *uid_check = (char *)malloc(2048);
    char *line = (char *)malloc(2048);
    FILE *status = fopen(proc_status, "r");
    pid_t uid = singularity_priv_getuid();

    if ( status == NULL ) {
        singularity_message(ERROR, "Failed to open %s to check instance owner\n", proc_status);
        ABORT(255);
    }

    memset(uid_check, 0, 2048);
    snprintf(uid_check, 2047, "Uid:\t%d\t%d\t%d\t%d\n", uid, uid, uid, uid);

    while ( fgets(line, 2048, status) ) {
        if ( strcmp(line, uid_check) == 0 ) {
            retval = 1;
            break;
        }
    }

    free(uid_check);
    free(line);
    fclose(status);

    return(retval);
}

void daemon_init_join(void) {
    char *ns_path, *ns_fd_str;
    char *pid_path;
    int lock_result, ns_fd;
    int *lock_fd = malloc(sizeof(int));
    char *daemon_file = singularity_registry_get("DAEMON_FILE");
    char *daemon_name = singularity_registry_get("DAEMON_NAME");
    
    /* Check if there is a lock on daemon file */
    singularity_message(DEBUG, "Checking for lock on %s\n", daemon_file);
    lock_result = filelock(daemon_file, lock_fd);

    if ( lock_result == 0 ) {
        /* Successfully obtained lock, no daemon controls this file. */
        singularity_message(ERROR, "Unable to join daemon: %s daemon does not exist\n", daemon_name);
        unlink(daemon_file);
        close(*lock_fd);
        ABORT(255);
        return;
    } else if ( lock_result == EALREADY ) {
        /* EALREADY is set when another process has a lock on the file. */
        singularity_message(DEBUG, "Another process has lock on daemon file\n");

        daemon_file_parse();
                
        pid_path = (char *)malloc(2048 * sizeof(char *));
        sprintf(pid_path, "/proc/%s", singularity_registry_get("DAEMON_PID")); //Flawfinder: ignore

        if ( daemon_is_owner(pid_path) == 0 ) {
            singularity_message(ERROR, "Unable to join instance: you are not the owner\n");
            ABORT(255);
        }

        ns_path = joinpath(pid_path, "/ns");

        /* Open FD to /proc/[PID]/ns directory to call openat() for ns files */
        singularity_priv_escalate();
        if ( ( ns_fd = open(ns_path, O_RDONLY | O_CLOEXEC) ) == -1 ) {
            singularity_message(ERROR, "Unable to open ns directory of PID in daemon file: %s\n", strerror(errno));
            ABORT(255);
        }
        singularity_priv_drop();

        ns_fd_str = int2str(ns_fd);

        /* Set DAEMON_NS_FD to /proc/[PID]/ns FD in registry */
        singularity_registry_set("DAEMON_NS_FD", ns_fd_str);
    } else {
        singularity_message(ERROR, "Unable to join daemon: %s daemon does not exist\n", daemon_name);
        ABORT(255);
    }
}

void daemon_init_start(void) {
    char *daemon_file = singularity_registry_get("DAEMON_FILE");
    char *daemon_name = singularity_registry_get("DAEMON_NAME");
    char *daemon_file_dir = strdup(daemon_file);
    char *daemon_pid = (char *)malloc(256 * sizeof(char));
    char *daemon_image;
    int daemon_fd;
    int lock;
    
    /* Check if /var/tmp/.singularity-daemon-[UID]/ directory exists, if not create it */
    if ( is_dir(dirname(daemon_file_dir)) == -1 ) {
        s_mkpath(daemon_file_dir, 0755);
    }
    
    /* Attempt to open lock on daemon file */
    lock = filelock(daemon_file, &daemon_fd);

    if( lock == 0 ) {
        singularity_message(DEBUG, "Successfully obtained excluse lock on %s\n", daemon_file);

        /* Calling readlink on /proc/self returns the PID of the thread in the host PID NS */
        memset(daemon_pid, 0, 256);
        if ( readlink("/proc/self", daemon_pid, 256) == -1 ) { //Flawfinder: ignore
            singularity_message(ERROR, "Unable to open /proc/self: %s\n", strerror(errno));
            ABORT(255);
        } else {
            singularity_message(DEBUG, "PID in host namespace: %s\n", daemon_pid);
        }

        if ( !(daemon_image = realpath(singularity_registry_get("IMAGE"), NULL)) ) { //Flawfinder: ignore
            singularity_message(DEBUG, "ERROR: %s\n", strerror(errno));
        }
        
        /* Successfully obtained lock, write to daemon fd */
        lseek(daemon_fd, 0, SEEK_SET);
        if ( ftruncate(daemon_fd, 0) == -1 ) {
            singularity_message(ERROR, "Unable to truncate %d: %s\n", daemon_fd, strerror(errno));
        }

        daemon_file_write(daemon_fd, "DAEMON_PID", daemon_pid);
        daemon_file_write(daemon_fd, "DAEMON_IMAGE", daemon_image);
        daemon_file_write(daemon_fd, "DAEMON_ROOTFS", singularity_registry_get("ROOTFS"));

        singularity_registry_set("DAEMON_FD", int2str(daemon_fd));
    } else if( lock == EALREADY ) {
        /* Another daemon controls this file already */
        singularity_message(ERROR, "Daemon %s already exists: %s\n", daemon_name, strerror(errno));
        ABORT(255);
    } else {
        singularity_message(ERROR, "Cannot lock %s: %s\n", daemon_file, strerror(errno));
        ABORT(255);
    }
}

void singularity_daemon_init(void) {
    if ( singularity_registry_get("DAEMON_START") ) {

#if defined (SINGULARITY_NO_SETNS) && !defined (SINGULARITY_SETNS_SYSCALL)
        singularity_message(ERROR, "Instance feature is disabled, your kernel is too old\n");
        ABORT(255);
#endif

        daemon_init_start();
        return;
    } else if ( singularity_registry_get("DAEMON_JOIN") ) {

#if defined (SINGULARITY_NO_SETNS) && !defined (SINGULARITY_SETNS_SYSCALL)
        singularity_message(ERROR, "Instance feature is disabled, your kernel is too old\n");
        ABORT(255);
#endif

        daemon_init_join();
        return;
    } else {
        singularity_message(DEBUG, "Not joining a daemon, daemon join not set\n");
        return;
    }
}
