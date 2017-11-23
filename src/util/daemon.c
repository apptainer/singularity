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
#include <stdarg.h>
#include <unistd.h>
#include <errno.h>
#include <string.h>
#include <fcntl.h>
#include <limits.h>
#include <libgen.h>

#include "config.h"
#include "util/file.h"
#include "util/util.h"
#include "util/daemon.h"
#include "util/registry.h"
#include "util/message.h"
#include "lib/image/image.h"
#include "lib/runtime/runtime.h"
#include "util/privilege.h"

void *xmalloc(size_t size) {
    void *mem = malloc(size);
    if ( mem == NULL ) {
        singularity_message(ERROR, "Failed to allocate %lu memory bytes\n", size);
        ABORT(255);
    }
    memset(mem, 0, size);
    return (void *)mem;
}

int xsnprintf(char **buf, size_t size, char *fmt, ...) {
    int ret;
    va_list ap;
    if ( *buf == NULL ) {
        *buf = (char *)xmalloc(size);
    }

    va_start(ap, fmt);
    ret = vsnprintf(*buf, size - 1, fmt, ap); // Flawfinder: ignore
    va_end(ap);

    return ret;
}

void daemon_file_parse(void) {
    char *key, *val;
    char *line = (char *)xmalloc(2048);
    FILE *file = fopen(singularity_registry_get("DAEMON_FILE"), "r");
    char *daemon_name = singularity_registry_get("DAEMON_NAME");

    singularity_message(DEBUG, "reached file parse for daemon %s\n", daemon_name);

    if ( file == NULL ) {
        singularity_message(ERROR, "%s daemon does not exist\n", daemon_name);
        ABORT(255);
    }

    while( fgets(line, 2048, file) ) {
        key = strtok(line, "=\n");
        val = strtok(NULL, "=\n");
        singularity_message(DEBUG, "Read key-val pair %s=%s\n", key, val);
        singularity_registry_set(key, val);
    }
    free(line);
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

int daemon_is_running(char *pid_path) {
    int retval = 0;
    char *daemon_name = singularity_registry_get("DAEMON_NAME");
    char *daemon_cmdline = NULL;
    char *daemon_procname = NULL;
    FILE *file_cmdline;

    xsnprintf(&daemon_procname, 2048, "singularity-instance: %s [%s]", singularity_priv_getuser(), daemon_name);

    file_cmdline = fopen(joinpath(pid_path, "/cmdline"), "r");
    if ( file_cmdline == NULL ) {
        singularity_message(ERROR, "Can't open process command line, is instance %s running ?\n", daemon_name);
        ABORT(255);
    }

    daemon_cmdline = (char *)xmalloc(2048);
    if ( fgets(daemon_cmdline, 2048, file_cmdline) == NULL ) {
        singularity_message(ERROR, "Can't read command line, is instance %s running ?\n", daemon_name);
        ABORT(255);
    }

    if ( strcmp(daemon_procname, daemon_cmdline) == 0 ) {
        retval = 1;
    }

    free(daemon_cmdline);
    free(daemon_procname);
    fclose(file_cmdline);

    return(retval);
}

int daemon_is_owner(char *pid_path) {
    int retval = 0;
    char *proc_status = joinpath(pid_path, "/status");
    char *uid_check = (char *)xmalloc(2048);
    char *line = (char *)xmalloc(2048);
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
    int ns_fd;
    char *pid_path = NULL;
    char *daemon_file = singularity_registry_get("DAEMON_FILE");
    char *daemon_name = singularity_registry_get("DAEMON_NAME");

    if ( daemon_name == NULL ) {
        singularity_message(ERROR, "No instance name specified\n");
        ABORT(255);
    }

    if ( daemon_file == NULL ) {
        singularity_message(ERROR, "No instance file found for instance %s\n", daemon_name);
        ABORT(255);
    }

    singularity_message(DEBUG, "Check if instance %s is running\n", daemon_name);

    daemon_file_parse();

    if ( singularity_registry_get("DAEMON_PID") == NULL ) {
        singularity_message(ERROR, "%s seems corrupted or bad formatted", daemon_file);
        ABORT(255);
    }

    xsnprintf(&pid_path, 2048, "/proc/%s", singularity_registry_get("DAEMON_PID"));

    if ( daemon_is_owner(pid_path) == 0 ) {
        singularity_message(ERROR, "Unable to join instance: you are not the owner\n");
        ABORT(255);
    }

    if ( daemon_is_running(pid_path) ) {
        ns_path = joinpath(pid_path, "/ns");

        /* Open FD to /proc/[PID]/ns directory to call openat() for ns files */
        singularity_priv_escalate();
        if ( ( ns_fd = open(ns_path, O_RDONLY) ) == -1 ) {
            singularity_message(ERROR, "Unable to open ns directory of PID in daemon file: %s\n", strerror(errno));
            ABORT(255);
        }
        singularity_priv_drop();

        if ( fcntl(ns_fd, F_SETFD, FD_CLOEXEC) != 0 ) {
            singularity_message(ERROR, "Unable to set CLOEXEC on file descriptor\n");
            ABORT(255);
        }

        ns_fd_str = int2str(ns_fd);

        /* Set DAEMON_NS_FD to /proc/[PID]/ns FD in registry */
        singularity_registry_set("DAEMON_NS_FD", ns_fd_str);

        free(pid_path);
    } else {
        singularity_message(ERROR, "No instance named %s found\n", daemon_name);
        ABORT(255);
    }
}

void daemon_init_start(void) {
    char *daemon_file = singularity_registry_get("DAEMON_FILE");
    char *daemon_name = singularity_registry_get("DAEMON_NAME");
    char *daemon_file_dir = strdup(daemon_file);
    char *daemon_pid = (char *)xmalloc(256);
    char *daemon_image;
    int daemon_fd;
    
    /* Check if /var/tmp/.singularity-daemon-[UID]/ directory exists, if not create it */
    if ( is_dir(dirname(daemon_file_dir)) == -1 ) {
        s_mkpath(daemon_file_dir, 0755);
    }

    if ( access(daemon_file, F_OK) == 0 ) { // Flawfinder: ignore
        char *pid_path = NULL;

        /* check if it's a singularity daemon */
        daemon_file_parse();

        if ( singularity_registry_get("DAEMON_PID") == NULL ) {
            singularity_message(ERROR, "%s seems corrupted or bad formatted", daemon_file);
            ABORT(255);
        }

        xsnprintf(&pid_path, 2048, "/proc/%s", singularity_registry_get("DAEMON_PID"));

        if ( daemon_is_running(pid_path) ) {
            singularity_message(ERROR, "An instance named %s is already running\n", daemon_name);
            ABORT(255);
        }
        unlink(daemon_file);
        free(pid_path);
    }

    /* file don't exists assume no daemon is running */
    if ( ( daemon_fd = open(daemon_file, O_RDWR | O_CREAT | O_SYNC, 0644) ) < 0 ) {
        singularity_message(ERROR, "Unable to create daemon file %s\n", daemon_file);
        ABORT(255);
    }

    /* Calling readlink on /proc/self returns the PID of the thread in the host PID NS */
    if ( readlink("/proc/self", daemon_pid, 255) == -1 ) { //Flawfinder: ignore
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

    if ( singularity_registry_get("ADD_CAPS") ) {
        daemon_file_write(daemon_fd, "ADD_CAPS", singularity_registry_get("ADD_CAPS"));
    }
    if ( singularity_registry_get("DROP_CAPS") ) {
        daemon_file_write(daemon_fd, "DROP_CAPS", singularity_registry_get("DROP_CAPS"));
    }
    if ( singularity_registry_get("NO_PRIVS") ) {
        daemon_file_write(daemon_fd, "NO_PRIVS", singularity_registry_get("NO_PRIVS"));
    }
    if ( singularity_registry_get("KEEP_PRIVS") ) {
        daemon_file_write(daemon_fd, "KEEP_PRIVS", singularity_registry_get("KEEP_PRIVS"));
    }

    close(daemon_fd);

    free(daemon_pid);
    free(daemon_image);
    free(daemon_file_dir);
}

int singularity_daemon_has_namespace(char *namespace) {
    int retval = 0;
    char *self_ns_path = NULL;
    char *target_ns_path = (char *)xmalloc(PATH_MAX);
    struct stat self_ns;
    struct stat target_ns;
    char *target_pid = singularity_registry_get("DAEMON_PID");

    if ( target_pid == NULL ) {
        singularity_message(ERROR, "DAEMON_PID is not set\n");
        ABORT(255);
    }

    if ( target_ns_path == NULL ) {
        singularity_message(ERROR, "Can't allocate %d memory bytes buffer\n", PATH_MAX);
        ABORT(255);
    }

    if ( namespace == NULL ) {
        singularity_message(ERROR, "No namespace specified\n");
        ABORT(255);
    }

    if ( xsnprintf(&target_ns_path, PATH_MAX, "/proc/%s/ns/%s", target_pid, namespace) >= PATH_MAX ) {
        singularity_message(ERROR, "Path too long\n");
        ABORT(255);
    }

    self_ns_path = joinpath("/proc/self/ns/", namespace);

    if ( stat(self_ns_path, &self_ns) < 0 ) {
        singularity_message(ERROR, "Stat failed on link %s\n", self_ns_path);
        ABORT(255);
    }

    if ( stat(target_ns_path, &target_ns) < 0 ) {
        singularity_message(ERROR, "Stat failed on link %s\n", target_ns_path);
        ABORT(255);
    }

    if ( self_ns.st_ino != target_ns.st_ino ) {
        retval = 1;
    }

    free(target_ns_path);
    free(self_ns_path);

    return(retval);
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
