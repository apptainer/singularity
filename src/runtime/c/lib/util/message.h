/*
 * Copyright (c) 2018, Sylabs, Inc. All rights reserved.
 *
 * This software is licensed under a 3-clause BSD license. Please
 * consult LICENSE.md file distributed with the sources of this
 * project regarding your rights to use or distribute this software.
 */

#ifndef _SINGULARITY_MESSAGE_H
#define _SINGULARITY_MESSAGE_H

#define ABRT -4
#define ERROR -3
#define WARNING -2
#define LOG -1
#define INFO 1
#define VERBOSE 2
#define VERBOSE1 2
#define VERBOSE2 3
#define VERBOSE3 4
#define DEBUG 5

#define ANSI_COLOR_RED          "\x1b[31m"
#define ANSI_COLOR_GREEN        "\x1b[32m"
#define ANSI_COLOR_YELLOW       "\x1b[33m"
#define ANSI_COLOR_BLUE         "\x1b[34m"
#define ANSI_COLOR_MAGENTA      "\x1b[35m"
#define ANSI_COLOR_CYAN         "\x1b[36m"
#define ANSI_COLOR_GRAY         "\x1b[37m"
#define ANSI_COLOR_LIGHTGRAY    "\x1b[90m"
#define ANSI_COLOR_LIGHTRED     "\x1b[91m"
#define ANSI_COLOR_LIGHTGREEN   "\x1b[92m"
#define ANSI_COLOR_LIGHTYELLOW  "\x1b[93m"
#define ANSI_COLOR_LIGHTBLUE    "\x1b[94m"
#define ANSI_COLOR_LIGHTMAGENTA "\x1b[95m"
#define ANSI_COLOR_LIGHTCYAN    "\x1b[96m"
#define ANSI_COLOR_RESET        "\x1b[0m"

void _print(int level, const char *function, const char *file, char *format, ...) __attribute__ ((__format__(printf, 4, 5)));

#define singularity_message(a,b...) _print(a, __func__, __FILE__, b)

#endif /*_SINGULARITY_MESSAGE_H */
