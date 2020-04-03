/*
 * Copyright (c) 2017-2019, SyLabs, Inc. All rights reserved.
 *
 * Copyright (c) 2016-2017, The Regents of the University of California,
 * through Lawrence Berkeley National Laboratory (subject to receipt of any
 * required approvals from the U.S. Dept. of Energy).  All rights reserved.
 *
 * This software is licensed under a customized 3-clause BSD license.  Please
 * consult LICENSE.md file distributed with the sources of this project regarding
 * your rights to use or distribute this software.
 *
 * NOTICE.  This Software was developed under funding from the U.S. Department of
 * Energy and the U.S. Government consequently retains certain rights. As such,
 * the U.S. Government has been granted for itself and others acting on its
 * behalf a paid-up, nonexclusive, irrevocable, worldwide license in the Software
 * to reproduce, distribute copies to the public, prepare derivative works, and
 * perform publicly and display publicly, and to permit other to do so. 
 */

#define _GNU_SOURCE

#include <ctype.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <string.h>
#include <stdarg.h>
#include <libgen.h>

#include "include/message.h"

int messagelevel = -99;

extern const char *__progname;

int count_digit(int n) {
    int count = 0;
    if ( n == 0 ) {
        return 1;
    }
    count = 1;
    while ( (n /= 10) ) {
        count++;
    }
    return count;
}

void _print(int level, const char *function, const char *file_in, char *format, ...) {
    const char *file = file_in;
    char message[512];
    char *prefix = NULL;
    char *color = NULL;
    char *color_reset = NULL;
    char has_color = 1;
    va_list args;

    if ( messagelevel == -99 ) {
        char *messagelevel_string = getenv(MSGLVL_ENV);

        if ( messagelevel_string == NULL ) {
            messagelevel = 5;
            singularity_message(DEBUG, MSGLVL_ENV " undefined, setting level 5 (debug)\n");
        } else {
            messagelevel = atoi(messagelevel_string);
            if ( messagelevel >= NO_COLOR ) {
                messagelevel -= NO_COLOR;
                has_color = 0;
            } else if ( messagelevel <= -NO_COLOR ) {
                messagelevel += NO_COLOR;
                has_color = 0;
            }
            if ( messagelevel > 9 ) {
                messagelevel = 9;
            }
            singularity_message(VERBOSE, "Set messagelevel to: %d\n", messagelevel);
        }
    }

    if ( level == LOG && messagelevel <= INFO ) {
        return;
    }

    va_start (args, format);

    if (vsnprintf(message, 512, format, args) >= 512) {
        memcpy(message+496, "(TRUNCATED...)", 15);
        message[511] = '\0';
    }

    va_end (args);

    while( ( ! isalpha(file[0]) ) && ( file[0] != '\0') ) {
        file++;
    }

    switch (level) {
        case ABRT:
            prefix = "ABORT";
            if ( has_color == 1 ) {
                color = ANSI_COLOR_RED;
                color_reset = ANSI_COLOR_RESET;
            } else {
                color = "";
                color_reset = "";
            }
            break;
        case ERROR:
            prefix = "ERROR";
            if ( has_color == 1 ) {
                color = ANSI_COLOR_LIGHTRED;
                color_reset = ANSI_COLOR_RESET;
            } else {
                color = "";
                color_reset = "";
            }
            break;
        case WARNING:
            prefix = "WARNING";
            if ( has_color == 1 ) {
                color = ANSI_COLOR_YELLOW;
                color_reset = ANSI_COLOR_RESET;
            } else {
                color = "";
                color_reset = "";
            }
            break;
        case LOG:
            prefix = "LOG";
            if ( has_color == 1 ) {
                color = ANSI_COLOR_BLUE;
                color_reset = ANSI_COLOR_RESET;
            } else {
                color = "";
                color_reset = "";
            }
            break;
        case DEBUG:
            prefix = "DEBUG";
            color = "";
            color_reset = "";
            break;
        case INFO:
            prefix = "INFO";
            color = "";
            color_reset = "";
            break;
        default:
            prefix = "VERBOSE";
            color = "";
            color_reset = "";
            break;
    }

    if ( level <= messagelevel ) {
        char header_string[100];

        if ( messagelevel >= DEBUG ) {
            int count, funclen, length;
            if ( function[0] == '_' ) {
                function++;
            }
            count = 10 - count_digit(geteuid()) - count_digit(getpid());
            if ( count < 0 ) {
                count = 0;
            }
            funclen = 28 - strlen(function);
            if ( funclen < 0 ) {
                funclen = 0;
            }
            length = snprintf(header_string, 100, "%s%-7s [U=%d,P=%d] %*s %s() %*s", color, prefix, geteuid(), getpid(), count, "", function, funclen, "");
            if ( length < 0 ) {
                return;
            } else if ( length > 100 ) {
                header_string[99] = '\0';
            }
            header_string[length-1] = '\0';
        } else {
            snprintf(header_string, 15, "%s%-7s: ", color, prefix);
        }

        if ( level == INFO && messagelevel == INFO ) {
            printf("%s%s", message, color_reset);
        } else if ( level == INFO ) {
            printf("%s%s%s", header_string, message, color_reset);
        } else {
            fprintf(stderr, "%s%s%s", header_string, message, color_reset);
        }

        fflush(stdout);
        fflush(stderr);
    }
    if ( level == ABRT ) {
        exit(255);
    }
}
