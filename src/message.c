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
        messagelevel = 1;
    } else {
        messagelevel = atol(messagelevel_string);
        message(VERBOSE, "Setting messagelevel to: %d\n", messagelevel);
    }

}


void _message(int level, const char *function, const char *file, int line, char *format, ...) {
    char *prefix = "";
    va_list args;
    va_start (args, format);

    if ( messagelevel == -1 ) {
        init();
    }

    switch (level) {
        case ABRT:
            prefix = strdup("ABORT");
            break;
        case DEBUG:
            prefix = strdup("DEBUG");
            break;
        case WARNING:
            prefix = strdup("WARNING");
            break;
        case ERROR:
            prefix = strdup("ERROR");
            break;
        default:
            prefix = strdup("VERBOSE");
            break;
    }

    if ( level <= messagelevel ) {
        if ( level == INFO ) {
            vprintf(format, args);
        } else if ( messagelevel >= 5 ) {
            char *file_string = (char *)  malloc(64);
            char *header_string = (char *) malloc(128);
            char *debug_string = (char *) malloc(128);
            snprintf(file_string, 63, "%s:%d", file, line);
            snprintf(debug_string, 127, "%-7s [U=%d,P=%d %s]: ", prefix, geteuid(), getpid(), file_string);
            snprintf(header_string, 127, "%-48s ", debug_string);
            vfprintf(stderr, strjoin(header_string, format), args);
        } else {
            char *header_string = (char *) malloc(11);
            snprintf(header_string, 10, "%-8s ", strjoin(prefix, ":"));
            vfprintf(stderr, strjoin(header_string, format), args);
        }

    }

    fflush(stdout);
    fflush(stderr);

    va_end (args);

}

//#define MSG(a,b...) message(a, __func__, __FILE__, __LINE__, b)

/*

int main(void) {

    MSG(DEBUG, "Hello World%s\n", "Yes");

    return(0);
}

*/
