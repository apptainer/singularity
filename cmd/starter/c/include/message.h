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
#define NO_COLOR 90

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

#define MSGLVL_ENV              "SINGULARITY_MESSAGELEVEL"

void _print(int level, const char *function, const char *file, char *format, ...) __attribute__ ((__format__(printf, 4, 5)));

#define singularity_message(a,b...) _print(a, __func__, __FILE__, b)

#endif /*_SINGULARITY_MESSAGE_H */
