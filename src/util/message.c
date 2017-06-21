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


#include <ctype.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <string.h>
#include <stdarg.h>
#include <syslog.h>
#include <libgen.h>

#include "config.h"
#include "util/util.h"
#include "util/message.h"

int messagelevel = -99;

extern const char *__progname;

static void message_init(void) {
    char *messagelevel_string = getenv("SINGULARITY_MESSAGELEVEL"); // Flawfinder: ignore (need to get string, validation in atol())

    openlog("Singularity", LOG_CONS | LOG_NDELAY, LOG_LOCAL0);

    if ( messagelevel_string == NULL ) {
        messagelevel = 5;
        singularity_message(DEBUG, "SINGULARITY_MESSAGELEVEL undefined, setting level 5 (debug)\n");
    } else {
        messagelevel = atoi(messagelevel_string); // Flawfinder: ignore
        if ( messagelevel > 9 ) {
            messagelevel = 9;
        }
        singularity_message(VERBOSE, "Set messagelevel to: %d\n", messagelevel);
    }

}


int singularity_message_level(void) {
    if ( messagelevel == -1 ) {
        message_init();
    }

    return(messagelevel);
}

void _singularity_message(int level, const char *function, const char *file_in, int line, char *format, ...) {
    const char *file = file_in;
    int syslog_level = LOG_NOTICE;
    char message[512]; // Flawfinder: ignore (messages are truncated to 512 chars)
    char *prefix = NULL;
    char *color = NULL;
    va_list args;
    va_start (args, format);

    if (vsnprintf(message, 512, format, args) >= 512) { // Flawfinder: ignore (args are not user modifyable)
        memcpy(message+496, "(TRUNCATED...)\n", 15);
        message[511] = '\0';
    }

    va_end (args);

    if ( messagelevel == -99 ) {
        message_init();
    }

    while( ( ! isalpha(file[0]) ) && ( file[0] != '\0') ) {
        file++;
    }

    switch (level) {
        case ABRT:
            prefix = "ABORT";
            color = ANSI_COLOR_RED;
            syslog_level = LOG_ALERT;
            break;
        case ERROR:
            prefix = "ERROR";
            color = ANSI_COLOR_LIGHTRED;
            syslog_level = LOG_ERR;
            break;
        case WARNING:
            prefix = "WARNING";
            color = ANSI_COLOR_YELLOW;
            syslog_level = LOG_WARNING;
            break;
        case LOG:
            prefix = "LOG";
            color = ANSI_COLOR_BLUE;
            break;
        case DEBUG:
            prefix = "DEBUG";
            color = "";
            break;
        case INFO:
            prefix = "INFO";
            color = "";
            break;
        default:
            prefix = "VERBOSE";
            color = "";
            break;
    }

    if ( level <= LOG ) {
        // Note __progname comes from the linker; the UID can be 5 characters and PID can be
        // 10-or-so characters.
        char syslog_string[560]; // Flawfinder: ignore (512 max message length + 48 for header).
        if (snprintf(syslog_string, 540, "%s (U=%d,P=%d)> %s", __progname, geteuid(), getpid(), message) >= 540) {
            // This case shouldn't happen unless we have a very strange __progname; nul-terminating to be sure.
            syslog_string[559] = '\0';
        }

        syslog(syslog_level, "%s", syslog_string);
    }

    if ( level <= messagelevel ) {
        char header_string[100];

        if ( messagelevel >= DEBUG ) {
            char debug_string[25];
            char location_string[60];
            char tmp_header_string[86];
//            snprintf(location_string, 60, "%s:%d:%s()", basename(strdup(file)), line, function); // Flawfinder: ignore
//            snprintf(location_string, 60, "%s:%d ", basename(strdup(file)), line); // Flawfinder: ignore
            if ( function[0] == '_' ) {
                function++;
            }
            snprintf(location_string, 60, "%s()", function); // Flawfinder: ignore
            location_string[59] = '\0';
            snprintf(debug_string, 25, "[U=%d,P=%d]", geteuid(), getpid()); // Flawfinder: ignore
            debug_string[24] = '\0';
            snprintf(tmp_header_string, 86, "%-18s %s", debug_string, location_string); // Flawfinder: ignore
            tmp_header_string[85] = '\0';
            snprintf(header_string, 100, "%s%-7s %-60s ", color, prefix, tmp_header_string); // Flawfinder: ignore
//            header_string[94] = '\0';
        } else {
            snprintf(header_string, 15, "%s%-7s: ", color, prefix); // Flawfinder: ignore
//            header_string[9] = '\0';
        }

        if ( level == INFO && messagelevel == INFO ) {
            printf("%s" ANSI_COLOR_RESET, message); // Flawfinder: ignore (false alarm, format is constant)
        } else if ( level == INFO ) {
            printf("%s%s" ANSI_COLOR_RESET, header_string, message); // Flawfinder: ignore (false alarm, format is constant)
        } else if ( level == LOG && messagelevel <= INFO ) {
            // Don't print anything...
        } else {
            fprintf(stderr, "%s%s" ANSI_COLOR_RESET, header_string, message); // Flawfinder: ignore (false alarm, format is constant)
        }

        fflush(stdout);
        fflush(stderr);

    }

}
