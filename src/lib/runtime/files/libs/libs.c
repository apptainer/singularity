/* 
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

#ifndef _GNU_SOURCE
#define _GNU_SOURCE
#endif

#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/mount.h>
#include <sys/stat.h>
#include <unistd.h>
#include <stdlib.h>
#include <libgen.h>
#include <linux/limits.h>

#include "util/file.h"
#include "util/util.h"
#include "util/message.h"
#include "util/privilege.h"
#include "util/config_parser.h"
#include "util/registry.h"

#include "../file-bind.h"
#include "../../runtime.h"


int _singularity_runtime_files_libs(void) {
    char *container_dir = singularity_runtime_rootfs(NULL);
    char *tmpdir = singularity_registry_get("SESSIONDIR");
    char *includelibs_string;
    char *libdir = joinpath(tmpdir, "/libs");
    char *libdir_contained = joinpath(container_dir, "/.singularity.d/libs");

    if ( ( includelibs_string = singularity_registry_get("CONTAINLIBS") ) != NULL ) {
        char *tok = NULL;
        char *current = strtok_r(strdup(includelibs_string), ",", &tok);
        char *ld_lib_path = envar_path("SINGULARITYENV_LD_LIBRARY_PATH");


        singularity_message(DEBUG, "Parsing SINGULARITY_CONTAINLIBS for user-specified libraries to include.\n");

        free(includelibs_string);

        if ( is_dir(libdir_contained) != 0 ) {
            singularity_message(WARNING, "Library bind directory not present in container, update container\n");
            ABORT(255);
        }

        if ( s_mkpath(libdir, 0755) != 0 ) {
            singularity_message(ERROR, "Failed creating temp lib directory at: %s\n", libdir);
            ABORT(255);
        }

        while (current != NULL ) {
            char *dest = NULL;
            char *source = NULL;

            singularity_message(DEBUG, "Evaluating passed library path: %s\n", current);

            if ( is_link(current) == 0 ) {
                char *link_name;
                ssize_t len;

                link_name = (char *) malloc(PATH_MAX);

                len = readlink(current, link_name, PATH_MAX-1);
                if ( ( len > 0 ) && ( len <= PATH_MAX) ) {
                    link_name[len] = '\0';
                    singularity_message(VERBOSE3, "Found library link source: %s -> %s\n", current, link_name);
                    if ( link_name[0] == '/' ) {
                        source = strdup(link_name);
                    } else {
                        source = joinpath(dirname(strdup(current)), basename(strdup(link_name)));
                    }
                } else {
                    singularity_message(WARNING, "Failed reading library link for %s: %s\n", current, strerror(errno));
                    ABORT(255);
                }
                free(link_name);

            } else if (is_file(current) == 0 ) {
                source = strdup(current);
                singularity_message(VERBOSE3, "Found library source: %s\n", source);
            }

            dest = joinpath(libdir, basename(current));

            singularity_message(DEBUG, "Binding library source here: %s -> %s\n", source, dest);

            if ( fileput(dest, "") != 0 ) {
                singularity_message(ERROR, "Failed creating file at %s: %s\n", dest, strerror(errno));
                ABORT(255);
            }

            singularity_priv_escalate();
            singularity_message(VERBOSE, "Binding file '%s' to '%s'\n", source, dest);
            if ( mount(source, dest, NULL, MS_BIND|MS_NOSUID|MS_REC, NULL) < 0 ) {
                    singularity_priv_drop();
                    singularity_message(ERROR, "There was an error binding %s to %s: %s\n", source, dest, strerror(errno));
                    ABORT(255);
            }
            singularity_priv_drop();

            current = strtok_r(NULL, ",", &tok);
        }

        singularity_priv_escalate();
        singularity_message(VERBOSE, "Binding libdir '%s' to '%s'\n", libdir, libdir_contained);
        if ( mount(libdir, libdir_contained, NULL, MS_BIND|MS_NOSUID|MS_REC, NULL) < 0 ) {
                singularity_priv_drop();
                singularity_message(ERROR, "There was an error binding %s to %s: %s\n", libdir, libdir_contained, strerror(errno));
                ABORT(255);
        }
        singularity_priv_drop();

        if ( ld_lib_path == NULL ) {
            envar_set("SINGULARITYENV_LD_LIBRARY_PATH", "/.singularity.d/libs", 1);
        } else {
            envar_set("SINGULARITYENV_LD_LIBRARY_PATH", strjoin("/.singularity.d/libs:", ld_lib_path), 1);
        }
    }



    return(0);
}
