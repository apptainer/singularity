/* 
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
#include <time.h>

#include "config.h"
#include "util/util.h"
#include "util/message.h"
#include "util/config_parser.h"

int messagelevel = -1;

extern const char *__progname;

FILE *log_file;

static void message_init(void) {
    char *messagelevel_string = getenv("SINGULARITY_MESSAGELEVEL"); // Flawfinder: ignore (need to get string, validation in atol())

    openlog("Singularity", LOG_CONS | LOG_NDELAY, LOG_LOCAL0);

    if ( messagelevel_string == NULL ) {
        messagelevel = 5;
        singularity_message(DEBUG, "SINGULARITY_MESSAGELEVEL undefined, setting level 5 (debug)\n");
    } else {
        messagelevel = atoi(messagelevel_string); // Flawfinder: ignore
        if ( messagelevel < 0 ) {
            messagelevel = 0;
        } else if ( messagelevel > 9 ) {
            messagelevel = 9;
        }
        singularity_message(VERBOSE, "Set messagelevel to: %d\n", messagelevel);
    }

    if (strcmp(singularity_config_get_value(LOG_SYSTEM), "file") == 0) {
        singularity_priv_init();
        singularity_priv_escalate();
        log_file = fopen("singularity.log", "a");
        fchmod(fileno(log_file), 0644);
        singularity_priv_drop();
    }
}


void _singularity_message(int level, const char *function, const char *file_in, int line, char *format, ...) {
    const char *file = file_in;
    int syslog_level = LOG_NOTICE;
    char message[512]; // Flawfinder: ignore (messages are truncated to 512 chars)
    char *prefix = NULL;
    va_list args;
    va_start (args, format);

    if (vsnprintf(message, 512, format, args) >= 512) {
        memcpy(message+497, "(TRUNCATED...)", 14);
        message[511] = '\0';
    }

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
        // Note __progname comes from the linker; the UID can be 5 characters and PID can be
        // 10-or-so characters.
        char log_message[560]; // Flawfinder: ignore (512 max message length + 48 for header).
        if (snprintf(log_message, 540, "%s", message) >= 540) {
            // This case shouldn't happen unless we have a very strange __progname; nul-terminating to be sure.
            log_message[559] = '\0';
        }

        // get current timestamp for file and stderr logging
        time_t ltime;
        ltime = time(NULL);
        char *timestamp = asctime(localtime(&ltime));
        // remove trailing newline
        strtok(timestamp, "\n");

        if (strcmp(singularity_config_get_value(LOG_SYSTEM), "syslog") == 0) {
            syslog(syslog_level, "%s", log_message);
        } else if (strcmp(singularity_config_get_value(LOG_SYSTEM), "none") == 0) {
            // do nothing...
        } else if (strcmp(singularity_config_get_value(LOG_SYSTEM), "file") == 0) {
            fprintf(log_file, "%s: %s \n", timestamp, log_message);
            fflush(log_file);
        } else {
            level = WARNING;
            prefix = "WARNING";
            snprintf(message, 512, "%s", "Invalid log system option in singularity.conf.\n");
        }

        
    }

    if ( level <= messagelevel ) {
        char header_string[95];

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
            snprintf(header_string, 95, "%-7s %-60s ", prefix, tmp_header_string); // Flawfinder: ignore
            header_string[94] = '\0';
        } else {
            snprintf(header_string, 10, "%-7s: ", prefix); // Flawfinder: ignore
            header_string[9] = '\0';
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

        fflush(stdout);
        fflush(stderr);

    }

}
