/* 
 * Copyright (c) 2016-2017, Michael W. Bauer. All rights reserved.
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

#ifndef __BOOTSTRAP_LIB_H_
#define __BOOTSTRAP_LIB_H_

    /* bootdef_parser.c */
    extern int singularity_bootdef_open(char *bootdef_path);
    extern void singularity_bootdef_rewind();
    extern void singularity_bootdef_close();
    extern char *singularity_bootdef_get_value(char *key);
    extern int singularity_bootdef_get_version();
    extern int singularity_bootdef_section_find(char *section_name);
    extern int singularity_bootdef_section_get(char **script, char *section_name);

    /* bootstrap.c */
    

#endif
