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


#ifndef __SINGULARITY_REGISTRY_H_
#define __SINGULARITY_REGISTRY_H_

#define REGISTRY_SIZE 128
#define MAX_KEY_LEN 128

// Initalize the registry. It will automatically be initalized on the
// first call to _get() or _set(), but if you want to ensure it gets run
// at startup time, do it manually.
//
// This will also retrieve any environment variables that are prefixed with
// "SINGULARITY_" and load that data into the registry automatically.
extern void singularity_registry_init(void);

// Set a value in the registry. If it already exists, this will overwrite the
// previous entry as only one value for each key can be stored.
extern int singularity_registry_set(char *key, char *value);

// Get any value that is currently being stored in the registry. If the key
// is not currently set, it will return with NULL.
extern char *singularity_registry_get(char *key);

#endif /* __SINGULARITY_REGISTRY_H_ */
