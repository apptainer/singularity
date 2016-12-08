/* 
 * Copyright (c) 2015-2016, Gregory M. Kurtzer. All rights reserved.
 * 
 * “Singularity” Copyright (c) 2016, The Regents of the University of California,
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


#include <ctype.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <string.h>
#include <stdarg.h>
#include <syslog.h>

#include "config.h"
#include "util/util.h"
#include "lib/message.h"

int messagelevel = -1;

extern const char *__progname;

static void message_init(void) {
    char *messagelevel_string = getenv("MESSAGELEVEL"); // Flawfinder: ignore (need to get string, validation in atol())

    openlog("Singularity", LOG_CONS | LOG_NDELAY, LOG_LOCAL0);

    if ( messagelevel_string == NULL ) {
        messagelevel = 1;
    } else {
        messagelevel = atoi(messagelevel_string); // Flawfinder: ignore
        if ( messagelevel < 0 ) {
            messagelevel = 0;
        } else if ( messagelevel > 9 ) {
            messagelevel = 9;
        }
        singularity_message(VERBOSE, "Set messagelevel to: %d\n", messagelevel);
    }

}


void _singularity_message(int level, const char *function, const char *file_in, int line, char *format, ...) {
    const char *file = file_in;
    int syslog_level = LOG_NOTICE;
    char message[512]; // Flawfinder: ignore (messages are truncated to 512 chars)
    char *prefix = NULL;
    va_list args;
    va_start (args, format);

    vsnprintf(message, 512, format, args); // Flawfinder: ignore (format is internally defined)

    va_end (args);

    if ( messagelevel == -1 ) {
        message_init();
    }

    while( ( ! isalpha(file[0]) ) && ( file[0] != '\0') ) {
        file++;
    }

    switch (level) {
        case ABRT:
            prefix = "ABORT";
            syslog_level = LOG_ALERT;
            break;
        case ERROR:
            prefix = "ERROR";
            syslog_level = LOG_ERR;
            break;
        case  WARNING:
            prefix = "WARNING";
            syslog_level = LOG_WARNING;
            break;
        case LOG:
            prefix = "LOG";
            break;
        case DEBUG:
            prefix = "DEBUG";
            break;
        case INFO:
            prefix = "INFO";
            break;
        default:
            prefix = "VERBOSE";
            break;
    }

    if ( level <= LOG ) {
        char syslog_string[540]; // Flawfinder: ignore (512 max message length + 28'ish chars for header)
        snprintf(syslog_string, 540, "%s (U=%d,P=%d)> %s", __progname, geteuid(), getpid(), message); // Flawfinder: ignore

        syslog(syslog_level, syslog_string, strlength(syslog_string, 1024)); // Flawfinder: ignore (format is internally defined)
    }

    if ( level <= messagelevel ) {
        char *header_string = NULL;

        if ( messagelevel >= DEBUG ) {
            char *debug_string = (char *) malloc(25);
            char *location_string = (char *) malloc(60);
            char *tmp_header_string = (char *) malloc(80);
            header_string = (char *) malloc(80);
            snprintf(location_string, 60, "%s:%d:%s()", file, line, function); // Flawfinder: ignore
            snprintf(debug_string, 25, "[U=%d,P=%d]", geteuid(), getpid()); // Flawfinder: ignore
            snprintf(tmp_header_string, 80, "%-18s %s", debug_string, location_string); // Flawfinder: ignore
            snprintf(header_string, 80, "%-7s %-62s: ", prefix, tmp_header_string); // Flawfinder: ignore
            free(debug_string);
            free(location_string);
            free(tmp_header_string);
        } else {
            header_string = (char *) malloc(11);
            snprintf(header_string, 10, "%-7s: ", prefix); // Flawfinder: ignore
        }

        if ( level == INFO && messagelevel == INFO ) {
            printf("%s", message);
        } else if ( level == INFO ) {
            printf("%s%s", header_string, message);
        } else if ( level == LOG && messagelevel <= INFO ) {
            // Don't print anything...
        } else {
            fprintf(stderr, "%s%s", header_string, message);
        }

        free(header_string);
        fflush(stdout);
        fflush(stderr);

    }

}
