/* 
 * Copyright (c) 2015-2016, Gregory M. Kurtzer. All rights reserved.
 * 
 * “Singularity” Copyright (c) 2016, The Regents of the University of California,
 * through Lawrence Berkeley National Laboratory (subject to receipt of any
 * required approvals from the U.S. Dept. of Energy).  All rights reserved.
 * 
 * If you have questions about your rights to use or distribute this software,
 * please contact Berkeley Lab's Innovation & Partnerships Office at
 * IPO@lbl.gov.
 * 
 * NOTICE.  This Software was developed under funding from the U.S. Department of
 * Energy and the U.S. Government consequently retains certain rights. As such,
 * the U.S. Government has been granted for itself and others acting on its
 * behalf a paid-up, nonexclusive, irrevocable, worldwide license in the Software
 * to reproduce, distribute copies to the public, prepare derivative works, and
 * perform publicly and display publicly, and to permit other to do so. 
 * 
 */


#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <string.h>
#include <stdarg.h>

#include "config.h"
#include "util.h"
#include "message.h"

int messagelevel = -1;

void init(void) {
    char *messagelevel_string = getenv("MESSAGELEVEL");

    if ( messagelevel_string == NULL ) {
        messagelevel = 0;
    } else {
        messagelevel = atol(messagelevel_string);
        message(VERBOSE, "Setting messagelevel to: %d\n", messagelevel);
    }

}

void message(int level, char *format, ...) {
    va_list args;
    va_start (args, format);

    if ( messagelevel == -1 ) {
        init();
    }

    switch (level) {
        case ERROR:
            fprintf(stderr, "ERROR:   ");
            vfprintf(stderr, format, args);
            break;
        case WARNING:
            fprintf(stderr, "WARNING: ");
            vfprintf(stderr, format, args);
            break;
        case INFO:
            vprintf(format, args);
            break;
        default:
            if ( level <= messagelevel && messagelevel >= 5 ) {
                char *debug_string = (char *) malloc(intlen(geteuid()) + intlen(getpid()) + 16);
                snprintf(debug_string, intlen(geteuid()) + intlen(getpid()) + 16, "DEBUG(U=%d,PID=%d):", geteuid(), getpid());
                fprintf(stderr, "%-28s ", debug_string);
                vfprintf(stderr, format, args);
            } else if ( level <= messagelevel ) {
                fprintf(stderr, "VERBOSE: ");
                vfprintf(stderr, format, args);
            }
            break;
    }

    va_end (args);
}

