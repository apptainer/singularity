/* 
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

#include "config.h"
#include "util/file.h"
#include "util/util.h"
#include "util/message.h"
#include "util/privilege.h"
#include "util/config_parser.h"
#include "util/registry.h"
#include "util/mount.h"
#include "util/binary.h"

#include "../file-bind.h"
#include "../../runtime.h"

int _singularity_runtime_files_libs(void) {
    char *container_dir = CONTAINER_FINALDIR;
    char *tmpdir = singularity_registry_get("SESSIONDIR");
    char *includelibs_string;
    char *libdir = joinpath(tmpdir, "/libs");
    char *libdir_contained = joinpath(container_dir, "/.singularity.d/libs");
    const char * const supported_archs[] = { "x86_64", "i686", "x32" };
    int nb_supported_archs = 3;

    if ( ( includelibs_string = singularity_registry_get("CONTAINLIBS") ) != NULL ) {
        char *tok = NULL;
        char *current = strtok_r(strdup(includelibs_string), ",", &tok);

#ifndef SINGULARITY_NO_NEW_PRIVS
        singularity_message(WARNING, "Not mounting libs: host does not support PR_SET_NO_NEW_PRIVS\n");
        return(0);
#endif

        singularity_message(DEBUG, "Parsing SINGULARITY_CONTAINLIBS for user-specified libraries to include.\n");

        free(includelibs_string);

        singularity_message(DEBUG, "Checking if libdir in container exists: %s\n", libdir_contained);
        if ( is_dir(libdir_contained) != 0 ) {
            singularity_message(WARNING, "Library bind directory not present in container, update container\n");
        }

        singularity_message(DEBUG, "Creating session libdir at: %s\n", libdir);
        if ( s_mkpath(libdir, 0755) != 0 ) {
            singularity_message(ERROR, "Failed creating temp lib directory at: %s\n", libdir);
            ABORT(255);
        }

        // Iterate through the requested paths
        while (current != NULL ) {
            char *destdir = NULL;
            char *dest = NULL;
            char *source = NULL;

            singularity_message(DEBUG, "Evaluating requested library path: %s\n", current);

            // Find the library actual path on the host
            if ( is_link(current) == 0 ) {
                char *link_name;
                ssize_t len;
                link_name = (char *) malloc(PATH_MAX);
                len = readlink(current, link_name, PATH_MAX-1); // Flawfinder: ignore
                if ( ( len > 0 ) && ( len <= PATH_MAX) ) {
                    link_name[len] = '\0';
                    singularity_message(VERBOSE3, "Found library link source: %s -> %s\n", current, link_name);
                    if ( link_name[0] == '/' ) {
                        source = strdup(link_name);
                    } else {
                        if ( link_name[0] == '/' ) {
                            source = strdup(link_name);
                        } else {
                            source = joinpath(dirname(strdup(current)), link_name);
                        }
                    }
                } else {
                    singularity_message(WARNING, "Failed reading library link for %s: %s\n", current, strerror(errno));
                    ABORT(255);
                }
                free(link_name);
            } else if (is_file(current) == 0 ) {
                source = strdup(current);
                singularity_message(VERBOSE3, "Found library source: %s\n", source);
            } else {
                singularity_message(WARNING, "Could not find library: %s\n", current);
                current = strtok_r(NULL, ",", &tok);
                continue;
            }

            // Find the full destination path (with optional arch)
            switch(singularity_binary_arch(source)) {
                case BINARY_ARCH_X86_64:
                    destdir = joinpath(libdir, "x86_64");
                    break;
                case BINARY_ARCH_I386:
                    destdir = joinpath(libdir, "i686");
                    break;
                case BINARY_ARCH_X32:
                    destdir = joinpath(libdir, "x32");
                    break;
                default :
                    destdir = strdup(libdir);
            }
            dest = joinpath(destdir, basename(current));

            // This one already exists
            if ( is_file(dest) == 0 ) {
                singularity_message(VERBOSE3, "Staged library exists, skipping: %s\n", current);
                current = strtok_r(NULL, ",", &tok);
                continue;
            }

            // Create the destination arch directory if it does not exists
            singularity_message(DEBUG, "Creating destdir for %s at: %s\n", source, libdir);
            singularity_priv_escalate();
            if ( s_mkpath(destdir, 0755) != 0 ) {
                singularity_priv_drop();
                singularity_message(ERROR, "Failed creating temp lib directory at: %s\n", libdir);
                ABORT(255);
            }
            singularity_priv_drop();


            // Create the bind target file (empty)
            singularity_message(DEBUG, "Binding library source here: %s -> %s\n", source, dest);
            singularity_priv_escalate();
            if ( fileput(dest, "") != 0 ) {
                singularity_priv_drop();
                singularity_message(ERROR, "Failed creating file at %s: %s\n", dest, strerror(errno));
                ABORT(255);
            }
            singularity_priv_drop();

            // Bind the source to the target
            singularity_priv_escalate();
            singularity_message(VERBOSE, "Binding file '%s' to '%s'\n", source, dest);
            if ( singularity_mount(source, dest, NULL, MS_BIND|MS_NOSUID|MS_NODEV|MS_REC, NULL) < 0 ) {
                    singularity_priv_drop();
                    singularity_message(ERROR, "There was an error binding %s to %s: %s\n", source, dest, strerror(errno));
                    ABORT(255);
            }
            singularity_priv_drop();

            free(source);
            free(dest);
            free(destdir);
            current = strtok_r(NULL, ",", &tok);
        }

        char *ld_path;
        // Create the base lib directory inside the container (necessary for old containers)
        if ( is_dir(libdir_contained) != 0 ) {
            singularity_message(DEBUG, "Attempting to create contained libdir\n");
            singularity_priv_escalate();
            if ( s_mkpath(libdir_contained, 0755) != 0 ) {
                singularity_message(ERROR, "Failed creating directory %s :%s\n", libdir_contained, strerror(errno));
                ABORT(255);
            }
            singularity_priv_drop();

            // Set base LD_LIBRARY_PATH
            ld_path = envar_path("LD_LIBRARY_PATH");
            if ( ld_path == NULL ) {
                singularity_message(DEBUG, "Setting LD_LIBRARY_PATH to '/.singularity.d/libs'\n");
                envar_set("LD_LIBRARY_PATH", "/.singularity.d/libs", 1);
            } else {
                singularity_message(DEBUG, "Prepending '/.singularity.d/libs' to LD_LIBRARY_PATH\n");
                envar_set("LD_LIBRARY_PATH", strjoin("/.singularity.d/libs:", ld_path), 1);
            }
        }


        // Add per arch directories to LD_LIBRARY_PATH if those exists
        int idx;
        for (idx = 0; idx < nb_supported_archs; idx++) {
            char *subdir = joinpath(libdir, supported_archs[idx]);
            char *subdir_path;
            singularity_message(DEBUG, "Examining libs subdir arch %d, %s, (%s)\n", idx, supported_archs[idx], subdir);
            if ( is_dir(subdir) == 0 ) {
                singularity_message(VERBOSE, "Prepending subdir '%s' to LD_LIBRARY_PATH\n", subdir_path);
                ld_path = envar_path("LD_LIBRARY_PATH");
                if ( ld_path == NULL ) {
                    subdir_path = joinpath("/.singularity.d/libs/", supported_archs[idx]);
                    singularity_message(DEBUG, "Setting LD_LIBRARY_PATH to '%s'\n", subdir_path);
                    envar_set("LD_LIBRARY_PATH", subdir_path, 1);
                } else {
                    subdir_path = strjoin(joinpath("/.singularity.d/libs/", supported_archs[idx]), ":");
                    singularity_message(DEBUG, "Prepending '%s' to LD_LIBRARY_PATH\n", subdir_path);
                    envar_set("LD_LIBRARY_PATH", strjoin(subdir_path, ld_path), 1);
                }
            }
        }

        // Bind the base directory
        singularity_priv_escalate();
        singularity_message(VERBOSE, "Binding libdir '%s' to '%s'\n", libdir, libdir_contained);
        if ( singularity_mount(libdir, libdir_contained, NULL, MS_BIND|MS_NOSUID|MS_NODEV|MS_REC, NULL) < 0 ) {
                singularity_priv_drop();
                singularity_message(ERROR, "There was an error binding %s to %s: %s\n", libdir, libdir_contained, strerror(errno));
                ABORT(255);
        }
        singularity_priv_drop();

    }

    return(0);
}
