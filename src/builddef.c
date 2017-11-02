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
#include "util/registry.h"
#include "lib/image/image.h"
#include "lib/runtime/runtime.h"
#include "util/config_parser.h"
#include "util/privilege.h"
#include "util/sessiondir.h"

#include "./bootstrap-lib/include.h"

#ifndef LIBEXECDIR
#error LIBEXECDIR not defined
#endif
#ifndef BINDIR
#error BINDIR not defined
#endif
#ifndef SYSCONFDIR
#error SYSCONFDIR not defined
#endif

#define MAX_LINE_LEN 4096


int main(int argc, char **argv) {
    struct image_object image;
    FILE *bootdef_fp;
    char *line;
    char *builddef;

    singularity_config_init(joinpath(SYSCONFDIR, "/singularity/singularity.conf"));
    singularity_registry_init();
    singularity_priv_init();

    singularity_message(INFO, "Sanitizing environment\n");
    if ( envclean() != 0 ) {
        singularity_message(ERROR, "Failed sanitizing the environment\n");
        ABORT(255);
    }

    singularity_registry_set("WRITABLE", "1");

    if ( singularity_registry_get("WRITABLE") != NULL ) {
        singularity_message(VERBOSE3, "Instantiating writable container image object\n");
        image = singularity_image_init(singularity_registry_get("IMAGE"), O_RDWR);
    } else {
        singularity_message(VERBOSE3, "Instantiating read only container image object\n");
        image = singularity_image_init(singularity_registry_get("IMAGE"), O_RDONLY);
    }

    singularity_runtime_ns(SR_NS_MNT);

    singularity_image_mount(&image, CONTAINER_MOUNTDIR);

    builddef = singularity_registry_get("BUILDDEF");

    if ( is_file(builddef) != 0 ) {
        singularity_message(ERROR, "Bootstrap definition file not found: %s\n", builddef);
        ABORT(255);
    }

    if ( ( bootdef_fp = fopen(builddef, "r") ) == NULL ) {
        singularity_message(ERROR, "Could not open bootstrap definition file %s: %s\n", builddef, strerror(errno));
        ABORT(255);
    }

    line = (char *)malloc(MAX_LINE_LEN);

    while ( fgets(line, MAX_LINE_LEN, bootdef_fp) ) {
        char *bootdef_key;

        chomp_comments(line);

        // skip empty lines (do this after 'chomp')
        if (line[0] == '\0') {
            continue;
        }

        if ( line[0] == '%' ) { // We hit a section, stop parsing for keyword tags
            break;
        } else if ( ( bootdef_key = strtok(line, ":") ) != NULL ) {

            chomp(bootdef_key);

            char *bootdef_value;

            bootdef_value = strtok(NULL, "\n");
            char empty[] = "";
            if (bootdef_value == NULL) {
                bootdef_value = empty;
            } else {
                chomp(bootdef_value);
            }

            singularity_message(VERBOSE2, "Got bootstrap definition key/val '%s' = '%s'\n", bootdef_key, bootdef_value);

            if ( envar_defined(strjoin("SINGULARITY_DEFFILE_", uppercase(bootdef_key))) == 0 ) {
                singularity_message(ERROR, "Duplicate bootstrap definition key found: '%s'\n", bootdef_key);
                ABORT(255);
            }

            if ( strcasecmp(bootdef_key, "import") == 0 ) {
                // Do this again for an imported deffile
                bootstrap_keyval_parse(bootdef_value);
            }

            if ( strcasecmp(bootdef_key, "bootstrap") == 0 ) {
                singularity_registry_set("DRIVER", bootdef_value);
            }

            // Cool little feature, every key defined in def file is transposed
            // to environment
            envar_set(uppercase(bootdef_key), bootdef_value, 1);
            envar_set(strjoin("SINGULARITY_DEFFILE_", uppercase(bootdef_key)), bootdef_value, 1);
        }
    }

    free(line);
    fclose(bootdef_fp);

    envar_set("PATH", "/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin", 1);
    envar_set("SINGULARITY_ROOTFS", CONTAINER_MOUNTDIR, 1);
    envar_set("SINGULARITY_libexecdir", LIBEXECDIR, 1);
    envar_set("SINGULARITY_bindir", BINDIR, 1);
    envar_set("SINGULARITY_IMAGE", singularity_registry_get("IMAGE"), 1);
    envar_set("SINGULARITY_BUILDDEF", singularity_registry_get("BUILDDEF"), 1);
    envar_set("SINGULARITY_CHECKS", singularity_registry_get("CHECKS"), 1);
    envar_set("SINGULARITY_CHECKLEVEL", singularity_registry_get("CHECKLEVEL"), 1);
    envar_set("SINGULARITY_CHECKTAGS", singularity_registry_get("CHECKTAGS"), 1);
    envar_set("SINGULARITY_MESSAGELEVEL", singularity_registry_get("MESSAGELEVEL"), 1);
    envar_set("SINGULARITY_NOTEST", singularity_registry_get("NOTEST"), 1);
    envar_set("SINGULARITY_BUILDSECTION", singularity_registry_get("BUILDSECTION"), 1);
    envar_set("SINGULARITY_BUILDNOBASE", singularity_registry_get("BUILDNOBASE"), 1);
    envar_set("SINGULARITY_DOCKER_PASSWORD", singularity_registry_get("DOCKER_PASSWORD"), 1);
    envar_set("SINGULARITY_DOCKER_USERNAME", singularity_registry_get("DOCKER_USERNAME"), 1);
    envar_set("SINGULARITY_CACHEDIR", singularity_registry_get("CACHEDIR"), 1);
    envar_set("SINGULARITY_NOHTTPS", singularity_registry_get("NOHTTPS"), 1);
    envar_set("SINGULARITY_version", singularity_registry_get("VERSION"), 1);
    envar_set("HOME", singularity_priv_home(), 1);
    envar_set("LANG", "C", 1);

    char *bootstrap = joinpath(LIBEXECDIR, "/singularity/bootstrap-scripts/main-deffile.sh");

    execl(bootstrap, bootstrap, NULL); //Flawfinder: ignore (Yes, yes, we know, and this is required)

    singularity_message(ERROR, "Exec of bootstrap code failed: %s\n", strerror(errno));
    ABORT(255);

    return(0);
}
