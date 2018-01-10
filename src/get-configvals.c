/* 
* Copyright (c) 2017, SingularityWare, LLC., Inc. All rights reserved.
* 
* This software is licensed under a 3-clause BSD license.  Please
* consult LICENSE file distributed with the sources of this project regarding
* your rights to use or distribute this software.
* 
*/


#include <stdio.h>
#include <string.h>

#include "util/util.h"
#include "util/config_parser.h"

#ifndef SYSCONFDIR
#error SYSCONFDIR not defined
#endif

int main(int argc, char **argv) {

    if ( argc < 2 ) {
        printf("USAGE: %s [key]\n", argv[0]);
        exit(0);
    }

    singularity_config_init(joinpath(SYSCONFDIR, "/singularity/singularity.conf"));

    /* 
    If the key does not exist in the singularity.conf file, this is just 
    going to return "NULL".  The function was originally designed to return the
    default value if the value does not exist in the conf file, but I don't
    know how to do that using only strings, and the key needs to be based on 
    user input, not hardcoded. 
    */
    printf("%s\n", _singularity_config_get_value_impl(argv[1], "NULL"));

    return(0);
}
