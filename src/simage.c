/* 
 * Copyright (c) 2016, Michael W. Bauer. All rights reserved.
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
#include <sys/mount.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <sys/param.h>
#include <errno.h>
#include <signal.h>
#include <sched.h>
#include <string.h>
#include <fcntl.h>
#include <grp.h>
#include <libgen.h>
#include <linux/limits.h>

#include "config.h"
#include "lib/singularity.h"
#include "util/file.h"
#include "util/util.h"
#include "util/config_parser.h"


#ifndef SYSCONFDIR
#define SYSCONFDIR "/etc"
#endif


int main(int argc_in, char ** argv_in) {
    // Note: SonarQube complains when we change the value of parameters, even
    // in obviously-OK cases like this one...
    char **argv = argv_in;
    int argc = argc_in;
    long int size = 1024;
    
    if ( argv[1] == NULL ) {
        fprintf(stderr, "USAGE: %s [bootstrap/mount/bind/create/expand] [args]\n", argv[0]);
        return(1);
    }

    /* Open the config file for parsing */
    singularity_config_init(joinpath(SYSCONFDIR, "/singularity/singularity.conf"));

    /* 
     * Even though we don't have SUID for this binary, singularity_priv_init and 
     * singularity_priv_escalate can be used to ensure that the calling user is root.
     */
    singularity_priv_init();

    singularity_config_init(joinpath(SYSCONFDIR, "/singularity/singularity.conf"));

    /* Loop until we've gone through argv and returned */
    while ( 1 ) {
        singularity_message(DEBUG, "Running %s %s workflow\n", argv[0], argv[1]);

        singularity_priv_escalate();
        if ( argv[1] == NULL ) {
            singularity_message(DEBUG, "Finished running simage command and returning\n");
            return(0);
        }

        /* Run image mount workflow */
        else if ( strcmp(argv[1], "mount") == 0 ) {
            singularity_ns_mnt_unshare();
            if ( singularity_image_mount(argc - 1, &argv[1]) != 0 ) {
                singularity_priv_drop_perm();
                return(1);
            }
        }

        /* Run image bind workflow */
        else if ( strcmp(argv[1], "bind") == 0 ) {
            if ( singularity_image_bind(argc - 1, &argv[1]) != 0 ) {
                singularity_priv_drop_perm();
                return(1);
            }
        }
        
        /* Run image create workflow */
        else if ( strcmp(argv[1], "create") == 0 ) {
            if ( argv[2] == NULL ) {
                fprintf(stderr, "USAGE: %s create [singularity container image] [size in MiB]\n", argv[0]);
            }
            if ( argv[3] != NULL ) {
                size = ( strtol(argv[3], (char **)NULL, 10) );
            }
            return(singularity_image_create(argv[2], size));
        }
        
        /* Run image expand workflow */
        else if ( strcmp(argv[1], "expand") == 0 ) {
            if ( argv[2] == NULL ) {
                fprintf(stderr, "USAGE: %s expand [singularity container image] [size in MiB]\n", argv[0]);
            }
            if ( argv[3] != NULL ) {
                size = ( strtol(argv[3], (char **)NULL, 10) );
            }
            return(singularity_image_expand(argv[2], size));
        }

        /* Run image bootstrap workflow */
        else if ( strcmp(argv[1], "bootstrap") == 0 ) {
            if ( (argv[2] == NULL) || (argv[3] == NULL) ) {
                fprintf(stderr, "USAGE: %s bootstrap [singularity container image] [bootstrap definition file]\n", argv[0]);
                return(1);
            }
            return(singularity_bootstrap(argv[2], argv[3]));
        } 

        /* If there is a trailing arg containing a script, attempt to execute it */
        else {
            singularity_priv_drop_perm();
            return(singularity_fork_exec(&argv[1]));
        }
        
        argv++;
        argc--;
        singularity_priv_drop();
    }
}
