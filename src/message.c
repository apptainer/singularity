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
#include <syslog.h>

#include "config.h"
#include "util.h"
#include "message.h"

int messagelevel = -1;

extern const char *__progname;

void init(void) {
    char *messagelevel_string = getenv("MESSAGELEVEL");

    openlog("Singularity", LOG_CONS | LOG_NDELAY, LOG_LOCAL0);

    if ( messagelevel_string == NULL ) {
        messagelevel = 1;
    } else {
        messagelevel = atol(messagelevel_string);
        message(VERBOSE, "Setting messagelevel to: %d\n", messagelevel);
    }

}


void _message(int level, const char *function, const char *file, int line, char *format, ...) {
    int syslog_level = LOG_NOTICE;
    char message[512];
    char *prefix = "";
    va_list args;
    va_start (args, format);

    vsnprintf(message, 512, format, args);

    va_end (args);

    if ( messagelevel == -1 ) {
        init();
    }

    switch (level) {
        case ABRT:
            prefix = strdup("ABORT");
            syslog_level = LOG_ALERT;
            break;
        case ERROR:
            prefix = strdup("ERROR");
            syslog_level = LOG_ERR;
            break;
        case  WARNING:
            prefix = strdup("WARNING");
            syslog_level = LOG_WARNING;
            break;
        case LOG:
            prefix = strdup("LOG");
            break;
        case DEBUG:
            prefix = strdup("DEBUG");
            break;
        case INFO:
            prefix = strdup("INFO");
            break;
        default:
            prefix = strdup("VERBOSE");
            break;
    }

    if ( level <= LOG ) {
        char syslog_string[540]; // 512 max message length + 28'ish chars for header
        snprintf(syslog_string, 540, "%s (U=%d,P=%d)> %s", __progname, geteuid(), getpid(), message);

        syslog(syslog_level, syslog_string, strlen(syslog_string));
    }

    if ( level <= messagelevel ) {
        char *header_string;

        if ( messagelevel >= DEBUG ) {
            char *debug_string = (char *) malloc(60);
            char *function_string = (char *) malloc(25);
            char *tmp_header_string = (char *) malloc(80);
            header_string = (char *) malloc(80);
            snprintf(function_string, 25, "%s()", function);
            snprintf(debug_string, 40, "[U=%d,P=%d,L=%s:%d]", geteuid(), getpid(), file, line);
            snprintf(tmp_header_string, 80, "%-38s %s", debug_string, function_string);
            snprintf(header_string, 80, "%-7s %-62s: ", prefix, tmp_header_string);
            free(debug_string);
            free(function_string);
            free(tmp_header_string);
        } else {
            header_string = (char *) malloc(11);
            snprintf(header_string, 10, "%-8s ", strjoin(prefix, ":"));
        }

        if ( level == INFO ) {
            printf(strjoin(header_string, message));
        } else if ( level == LOG ) {
            // Don't print anything...
        } else {
            fprintf(stderr, strjoin(header_string, message));
        }


        fflush(stdout);
        fflush(stderr);

    }

}

void singularity_abort(int retval) {
    message(ABRT, "Exiting with RETVAL=%d\n", retval);
    exit(retval);
}
