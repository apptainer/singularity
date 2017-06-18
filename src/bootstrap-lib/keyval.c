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

#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/stat.h>
#include <unistd.h>
#include <stdlib.h>

#include "util/file.h"
#include "util/util.h"
#include "util/message.h"

#include "util/registry.h"

#define MAX_LINE_LEN 4096


int bootstrap_keyval_parse(char *path) {
    FILE *bootdef_fp;
    char *line;

    if ( is_file(path) != 0 ) {
        singularity_message(ERROR, "Bootstrap definition file not found: %s\n", path);
        ABORT(255);
    }

    if ( ( bootdef_fp = fopen(path, "r") ) == NULL ) {
        singularity_message(ERROR, "Could not open bootstrap definition file %s: %s\n", path, strerror(errno));
        ABORT(255);
    }

    line = (char *)malloc(MAX_LINE_LEN);

    while ( fgets(line, MAX_LINE_LEN, bootdef_fp) ) {
        char *bootdef_key;

        if ( line[0] == '%' ) { // We hit a section, stop parsing for keyword tags
            break;
        } else if ( ( bootdef_key = strtok(line, ":") ) != NULL ) {

            chomp(bootdef_key);

            char *bootdef_value;

            if ( ( bootdef_value = strtok(NULL, "\n") ) != NULL ) {

                chomp_comments(bootdef_value);
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
    }

    free(line);
    fclose(bootdef_fp);

    return(0);
}
