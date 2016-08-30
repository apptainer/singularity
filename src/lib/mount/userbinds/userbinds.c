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

#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/mount.h>
#include <sys/stat.h>
#include <unistd.h>
#include <stdlib.h>

#include "util/file.h"
#include "util/util.h"
#include "lib/message.h"
#include "lib/privilege.h"
#include "lib/config_parser.h"
#include "lib/rootfs/rootfs.h"
#include "lib/ns/ns.h"



void singularity_mount_userbinds(void) {
    char * tmp_config_string;
    char *container_dir = singularity_rootfs_dir();

    singularity_message(DEBUG, "Checking for 'user bind control' in config\n");
    if ( singularity_config_get_bool("user bind control", 1) <= 0 ) {
        singularity_message(WARNING, "User bind control is disabled by system administrator\n");
        return;
    }

    if ( ( tmp_config_string = getenv("SINGULARITY_BINDPATH") ) != NULL ) {
        singularity_message(DEBUG, "Parsing SINGULARITY_BINDPATH for user-specified bind mounts.\n");
        char *bind = strdup(tmp_config_string);
        if (bind == NULL) {
            singularity_message(ERROR, "Failed to allocate memory for configuration string");
            ABORT(1);
        }
        char *cur = bind;
        char *next = strchr(cur, ':');
        for ( ; 1; next = strchr(cur, ':') ) {
            if (next) {
                *next = '\0'; // What does this do?
            }
            char *source = strtok(cur, ",");
            char *dest = strtok(NULL, ",");
            if ( source == NULL ) {
                break;
            }
            chomp(source);
            if ( dest == NULL ) {
                dest = strdup(source);
            } else {
                if ( dest[0] == ' ' ) {
                    dest++;
                }
                chomp(dest);
            }
            if ( (strlen(cur) == 0) && (next == NULL) ) {
                break;
            }
            singularity_message(VERBOSE2, "Found user-specified 'bind path' = %s, %s\n", source, dest);

            if ( ( is_file(source) != 0 ) && ( is_dir(source) != 0 ) ) {
                singularity_message(WARNING, "Non existant 'bind path' source: '%s'\n", source);
                if (next == NULL) {break;}
                continue;
            }

            singularity_priv_escalate();
            singularity_message(VERBOSE, "Binding '%s' to '%s/%s'\n", source, container_dir, dest);
            if ( mount(source, joinpath(container_dir, dest), NULL, MS_BIND|MS_NOSUID|MS_REC, NULL) < 0 ) {
                singularity_message(ERROR, "There was an error binding the path %s: %s\n", source, strerror(errno));
                ABORT(255);
            }
            singularity_priv_drop();

            cur = next + 1;
            if (next == NULL) {break;}
        }
        free(bind);
        unsetenv("SINGULARITY_BINDPATH");
    } else {
        singularity_message(DEBUG, "No user bind mounts specified.\n");
    }
}

