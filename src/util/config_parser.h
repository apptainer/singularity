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


#ifndef __SINGULARITY_CONFIG_H_
#define __SINGULARITY_CONFIG_H_

#include "config_defaults.h"

// Retrieve a single value from the configuration; in the presence of
// multiple values in the configuration file, only the last one is
// returned.
//
// If the configuration file does not have a value for the given key,
// then the compile-time default is returned.  singularity_config_get_value
// is actually a macro and should cause a compile-time error if there is no
// default specified in the code.
//
const char *_singularity_config_get_value_impl(const char *key, const char *default_value);
#define singularity_config_get_value(NAME) \
       _singularity_config_get_value_impl(NAME, NAME ## _DEFAULT)

// Retrieve (possibly) multiple values from the configuration file; the char*
// array is terminated by NULL.
//
const char **_singularity_config_get_value_multi_impl(const char *key, const char *default_value);
#define singularity_config_get_value_multi(NAME) \
       _singularity_config_get_value_multi_impl(NAME, NAME ## _DEFAULT)

// Retrieves a boolean value from the configuration file.  If there are
// multiple values in the configuration file, then only the last one is
// returned.
int _singularity_config_get_bool_impl(const char *key, int default_value);
#define singularity_config_get_bool(NAME) \
       _singularity_config_get_bool_impl(NAME, NAME ## _DEFAULT)

int _singularity_config_get_bool_char_impl(const char *key, const char *value);
#define singularity_config_get_bool_char(NAME) \
       _singularity_config_get_bool_char_impl(NAME, NAME ## _DEFAULT)

// Initialize the configuration table
//
int singularity_config_init(char *config_path);

#endif /* __SINGULARITY_CONFIG_H_ */
