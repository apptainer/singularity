/* 
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 * Copyright (c) 2015-2017, Gregory M. Kurtzer. All rights reserved.
 * Copyright (c) 2017, Vanessa Sochat All rights reserved.
 * 
 */

#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <errno.h> 
#include <string.h>
#include <fcntl.h>  
#include <linux/limits.h>

#include "config.h"
#include "util/util.h"
#include "util/file.h"
#include "util/message.h"
#include "util/privilege.h"
#include "util/registry.h"
#include "util/config_parser.h"


int main(int argc, char ** argv) {

    char *key;
    char *value;

    if ( argc < 2 ) {
        printf("USAGE: %s [key]\n", argv[0]);
        exit(0);
    }

    key = strdup(argv[1]);
    if ( ( value = singularity_registry_get(key) ) != NULL ) {
        printf("%s\n", value);
    }
    return(0);

}
