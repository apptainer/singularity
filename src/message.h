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

    void _singularity_message(int level, const char *function, const char *file, int line, char *format, ...);

    #define singularity_message(a,b...) _singularity_message(a, __func__, __FILE__, __LINE__, b)

    #define singularity_abort(a,b...) {_singularity_message(ABRT,  __func__, __FILE__, __LINE__, b); _singularity_message(ABRT,  __func__, __FILE__, __LINE__, "Retval = %d\n", a); exit(a);}

#endif /*__SINGULARITY_MESSAGE_H_ */

