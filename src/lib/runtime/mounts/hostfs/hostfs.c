/* 
 * Copyright (c) 2017-2018, SyLabs, Inc. All rights reserved.
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

#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/mount.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <unistd.h>
#include <stdlib.h>
#include <pwd.h>
#include <linux/limits.h>

#include "config.h"
#include "util/file.h"
#include "util/util.h"
#include "util/message.h"
#include "util/config_parser.h"
#include "util/registry.h"
#include "util/mountlist.h"

#include "../../runtime.h"

#define MAX_LINE_LEN 4096


int _singularity_runtime_mount_hostfs(struct mountlist *mountlist) {
    FILE *mounts;
    char *line = NULL;

    if ( singularity_config_get_bool(MOUNT_HOSTFS) <= 0 ) {
        singularity_message(DEBUG, "Not mounting host file systems per configuration\n");
        return(0);
    }

    singularity_message(DEBUG, "Checking to see if /proc/mounts exists\n");
    if ( is_file("/proc/mounts") < 0 ) {
        singularity_message(WARNING, "Can not probe for currently mounted host file systems\n");
        return(1);
    }

    singularity_message(DEBUG, "Opening /proc/mounts\n");
    if ( ( mounts = fopen("/proc/mounts", "r") ) == NULL ) { // Flawfinder: ignore
        singularity_message(ERROR, "Could not open /proc/mounts for reading: %s\n", strerror(errno));
        return(1);
    }

    line = (char *)malloc(MAX_LINE_LEN);

    singularity_message(DEBUG, "Getting line by line\n");
    while ( fgets(line, MAX_LINE_LEN, mounts) ) {
        char *source;
        char *mountpoint;
        char *filesystem;

        if ( line == NULL ) {
            singularity_message(DEBUG, "Skipping empty line in /proc/mounts\n");
            continue;
        }

        chomp(line);

        if ( line[0] == '#' || strlength(line, 2) <= 1 ) { // Flawfinder: ignore
            singularity_message(VERBOSE3, "Skipping blank or comment line in /proc/mounts\n");
            continue;
        }
        if ( ( source = strtok(line, " ") ) == NULL ) {
            singularity_message(VERBOSE3, "Could not obtain mount source from /proc/mounts: %s\n", line);
            continue;
        }
        if ( ( mountpoint = strtok(NULL, " ") ) == NULL ) {
            singularity_message(VERBOSE3, "Could not obtain mount point from /proc/mounts: %s\n", line);
            continue;
        }
        if ( ( filesystem = strtok(NULL, " ") ) == NULL ) {
            singularity_message(VERBOSE3, "Could not obtain file system from /proc/mounts: %s\n", line);
            continue;
        }

        if ( strcmp(mountpoint, "/") == 0 ) {
            singularity_message(DEBUG, "Skipping root (/): %s,%s,%s\n", source, mountpoint, filesystem);
            continue;
        }
        if ( strncmp(mountpoint, "/sys", 4) == 0 ) {
            singularity_message(DEBUG, "Skipping /sys based file system: %s,%s,%s\n", source, mountpoint, filesystem);
            continue;
        }
        if ( strncmp(mountpoint, "/boot", 5) == 0 ) {
            singularity_message(DEBUG, "Skipping /boot based file system: %s,%s,%s\n", source, mountpoint, filesystem);
            continue;
        }
        if ( strncmp(mountpoint, "/proc", 5) == 0 ) {
            singularity_message(DEBUG, "Skipping /proc based file system: %s,%s,%s\n", source, mountpoint, filesystem);
            continue;
        }
        if ( strncmp(mountpoint, "/dev", 4) == 0 ) {
            singularity_message(DEBUG, "Skipping /dev based file system: %s,%s,%s\n", source, mountpoint, filesystem);
            continue;
        }
        if ( strncmp(mountpoint, "/run", 4) == 0 ) {
            singularity_message(DEBUG, "Skipping /run based file system: %s,%s,%s\n", source, mountpoint, filesystem);
            continue;
        }
        if ( strncmp(mountpoint, "/var", 4) == 0 ) {
            singularity_message(DEBUG, "Skipping /var based file system: %s,%s,%s\n", source, mountpoint, filesystem);
            continue;
        }
        if ( strncmp(mountpoint, CONTAINER_FINALDIR, strlength(CONTAINER_FINALDIR, PATH_MAX)) == 0 ) {
            singularity_message(DEBUG, "Skipping final_dir (%s) based file system: %s,%s,%s\n", CONTAINER_FINALDIR, source, mountpoint, filesystem);
            continue;
        }
        if ( strcmp(mountpoint, CONTAINER_MOUNTDIR) == 0 ) {
            singularity_message(DEBUG, "Skipping container_dir (%s) based file system: %s,%s,%s\n", CONTAINER_MOUNTDIR, source, mountpoint, filesystem);
            continue;
        }
        if ( strcmp(filesystem, "tmpfs") == 0 ) {
            singularity_message(DEBUG, "Skipping tmpfs file system: %s,%s,%s\n", source, mountpoint, filesystem);
            continue;
        }
        if ( strcmp(filesystem, "cgroup") == 0 ) {
            singularity_message(DEBUG, "Skipping cgroup file system: %s,%s,%s\n", source, mountpoint, filesystem);
            continue;
        }
        if ( strcmp(filesystem, "ramfs") == 0 ) {
            singularity_message(DEBUG, "Skipping ramfs file system: %s,%s,%s\n", source, mountpoint, filesystem);
            continue;
        }

        singularity_message(VERBOSE, "Queuing bind mount of '%s'(%s) to '%s'\n", mountpoint, filesystem, mountpoint);
        mountlist_add(mountlist, NULL, strdup(mountpoint), NULL, MS_BIND|MS_NOSUID|MS_NODEV|MS_REC, 0);

    }

    free(line);
    fclose(mounts);
    return(0);
}
