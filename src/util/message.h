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


#ifndef __SINGULARITY_MESSAGE_H_
#define __SINGULARITY_MESSAGE_H_

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

    int singularity_message_level(void);
    void _singularity_message(int level, const char *function, const char *file, int line, char *format, ...) __attribute__ ((__format__(printf, 5, 6))); // Flawfinder: ignore

    #define singularity_message(a,b...) _singularity_message(a, __func__, __FILE__, __LINE__, b)

    #define singularity_abort(a,b...) do {_singularity_message(ABRT,  __func__, __FILE__, __LINE__, b); _singularity_message(ABRT,  __func__, __FILE__, __LINE__, "Retval = %d\n", a); exit(a);} while(0)

#endif /*__SINGULARITY_MESSAGE_H_ */

