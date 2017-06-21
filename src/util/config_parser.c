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
#define _GNU_SOURCE

#include <ctype.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <errno.h> 
#include <string.h>
#include <fcntl.h>  
#include <limits.h>
#include <search.h>
#include <glob.h>

#include "config.h"
#include "util/util.h"
#include "util/message.h"
#include "util/file.h"
#include "config_parser.h"

#define MAX_LINE_LEN (PATH_MAX + 128)
#define MAX_CONFIG_ENTRIES 64
#define NULLONE ((char*)1)

static int config_initialized = 0;
static struct hsearch_data config_table;

// Return a new, empty hash entry appropriate for adding to the config hash.
//
// By default, each hash bucket can have 7 values.  We set currently-empty
// entries
static ENTRY *new_hash_entry(char *key, char *value) {
    char **hash_value = (char**) malloc(sizeof(char*) * MAX_CONFIG_ENTRIES+1);
    int idx;
    hash_value[0] = value;
    for (idx=1; idx < MAX_CONFIG_ENTRIES; idx++) {hash_value[idx] = (char*)1;}
    hash_value[MAX_CONFIG_ENTRIES] = NULL;

    ENTRY *hash_entry = (ENTRY*)malloc(sizeof(ENTRY));
    memset(hash_entry, '\0', sizeof(ENTRY));
    hash_entry->key = key;
    hash_entry->data = hash_value;
    return hash_entry;
}

static void add_entry(char *key, char *value) {
    ENTRY search_item;
    search_item.key = key;
    search_item.data = NULL;
    ENTRY * old_entry = NULL;

    if (hsearch_r(search_item, FIND, &old_entry, &config_table)) {
        char **hash_value = old_entry->data;
        int idx = 0;
        while ( (hash_value[idx] != NULL) && (hash_value[idx] != NULLONE) ) {idx++;}
        if ( idx >= MAX_CONFIG_ENTRIES ) {
            singularity_message(ERROR, "Maximum of %d allowed configuration entries for: %s\n", MAX_CONFIG_ENTRIES, key);
            ABORT(255);
        }
        if (hash_value[idx] == NULLONE) {
            hash_value[idx] = value;
            return;
        }
        if (hash_value[idx] == NULL) {
            int max_size = 2*(idx+1);
            hash_value = realloc(hash_value, sizeof(char*)*max_size);
            hash_value[idx] = value;
            for (; idx<max_size-1; idx++) {
                hash_value[idx] = NULLONE;
            }
            hash_value[max_size-1] = NULL;
            return;
        }
        return;
    }
    ENTRY *new_entry = new_hash_entry(key, value);
    if (!hsearch_r(*new_entry, ENTER, &new_entry, &config_table)) {
        singularity_message(ERROR, "Internal error - unable to initialize configuration entry %s=%s.\n", key, value);
        ABORT(255);
    }
}

// Logs any errors that occur while the config glob is run.
static int log_glob_error(const char *epath, int eerrno) {
    singularity_message(ERROR, "Failed to evaluate config include glob due to error at %s: %s (errno=%d).\n", epath, strerror(eerrno), eerrno);
    ABORT(255);
    return 1;
}

/* 
 * Parses the singularity configuration into memory.
 *
 * @param char *config_path pointer to string containing path to configuration file
 * @returns 0 if successful, -1 if failure
 */
int singularity_config_parse(char *config_path) {
    singularity_message(VERBOSE, "Initialize configuration file: %s\n", config_path);
    if ( is_file(config_path) != 0 ) {
        singularity_message(ERROR, "Specified configuration file %s does not appear to be a normal file.\n", config_path);
    }
    FILE *config_fp = fopen(config_path, "r");
    if ( config_fp == NULL ) { // Flawfinder: ignore (we have to open the file...)
        singularity_message(ERROR, "Could not open configuration file %s: %s\n", config_path, strerror(errno));
        return -1;
    }

    char *line = (char *)malloc(MAX_LINE_LEN);

    singularity_message(DEBUG, "Starting parse of configuration file %s\n", config_path);
    while ( fgets(line, MAX_LINE_LEN, config_fp) ) {
        char *line_buf = line;
        // Skip over whitespace
        while (*line_buf && isspace(*line_buf)) {line_buf++;}

        // Skip comment lines.
        if (!*line_buf || (*line_buf == '#')) {
            continue;
        }

        // Include files
        if (strncmp("%include", line_buf, 8) == 0) {
            char *fname_glob = line_buf += 8;
            if (isspace(*fname_glob)) {
                chomp(fname_glob);
                singularity_message(DEBUG, "Parsing '%%include %s' directive.\n", fname_glob);
                glob_t glob_results;
                int err = glob(fname_glob, 0, log_glob_error, &glob_results);
                if (err == GLOB_NOSPACE) {
                    singularity_message(ERROR, "Failed to evaluate '%%include %s' due to running out of memory.\n", fname_glob);
                    ABORT(255);
                } else if (err == GLOB_ABORTED) {
                    singularity_message(ERROR, "Failed to evaluate '%%include %s' due read error.\n", fname_glob);
                    ABORT(255);
                } else if (err == GLOB_NOMATCH) {
                    singularity_message(ERROR, "No file matches '%%include %s'\n", fname_glob);
                    ABORT(255);
                } else if (err) {
                    singularity_message(ERROR, "Unknown error when evaluating '%%include %s'\n", fname_glob);
                    ABORT(255);
                }
                int idx;
                for (idx=0; idx<glob_results.gl_pathc; idx++) {
                    singularity_config_parse(glob_results.gl_pathv[idx]);
                }
                globfree(&glob_results);
                continue;
            }
        }

        // Parse assignments.
        char *config_key = strtok(line, "=");
        if ( config_key != NULL ) {
            config_key = strdup(config_key);
            chomp(config_key);

            char *config_value_tmp = strtok(NULL, "=");
            if ( config_value_tmp != NULL ) {
                char *config_value = strdup(config_value_tmp);
                chomp(config_value);
                singularity_message(VERBOSE2, "Got config key %s = '%s'\n", config_key, config_value);

                add_entry(config_key, config_value);
            }
        }
    }
    free(line);

    singularity_message(DEBUG, "Finished parsing configuration file '%s'\n", config_path);

    fclose(config_fp);
    return 0;
}

/*
 * Initialize the configuration, starting at a particular file.
 *
 * Returns 0 if the configuration was already initialized or the operation
 * was successful
 *
 * Returns non-zero on error.
 */
int singularity_config_init(char *config_path) {
    if (config_initialized) {
        return 0;
    }
    config_initialized = 1;

    hcreate_r(60, &config_table);
    int retval = singularity_config_parse(config_path);
    if (retval) {  // Error case.
        hdestroy_r(&config_table);
        config_initialized = 0;
    }
    return retval;
}

/* 
 * Retrieve a single configuration entry.
 *
 * @param char *key pointer to string containing key to search for in config_fp
 * @returns NULL if key not found, otherways returns 
 */
const char *_singularity_config_get_value_impl(const char *key, const char *default_value)
{
    if (!config_initialized) {
        singularity_message(ERROR, "Called singularity_config_get_value on uninitialized config subsystem\n");
        ABORT(255);
    }

    ENTRY search_item;
    search_item.key = (char*)key;
    search_item.data = NULL;
    ENTRY * old_entry = NULL;
    if (!hsearch_r(search_item, FIND, &old_entry, &config_table)) {  // hsearch_r returns 0 on error.
        singularity_message(DEBUG, "No configuration entry found for '%s'; returning default value '%s'\n", key, default_value);
        return default_value;
    }

    char **values = old_entry->data;
    int idx = 0;
    const char *retval = default_value;
    while ((values[idx] != NULL) && (values[idx] != NULLONE)) {
        retval = values[idx];
        idx++;
    }

    singularity_message(DEBUG, "Returning configuration value %s='%s'\n", key, retval);
    return retval;
}


static const char *_default_entry[2];

const char **_singularity_config_get_value_multi_impl(const char *key, const char *default_value)
{
    if (!config_initialized) {
        singularity_message(ERROR, "Called singularity_config_get_value on uninitialized config subsystem\n");
        ABORT(255);
    }
    _default_entry[1] = '\0';
    _default_entry[0] = default_value;

    ENTRY search_item;
    search_item.key = (char*)key;
    search_item.data = NULL;
    ENTRY * old_entry = NULL;
    if (!hsearch_r(search_item, FIND, &old_entry, &config_table)) {
        singularity_message(DEBUG, "No configuration entry found for '%s'; returning default value '%s'\n", key, default_value);
        return _default_entry;
    }
    char **values = old_entry->data;
    if ( (values[0] == NULL || values[0] == NULLONE) ) {
        singularity_message(DEBUG, "No configuration entry found for '%s'; returning default value '%s'\n", key, default_value);
        return _default_entry;
    }

    int idx = 1;
    while (values[idx] != NULL) {
        if (values[idx] == NULLONE) {
            values[idx] = NULL;
        }
        idx++;
    }
    return (const char **)values;
}

/*
 * Gets the associated boolean value of key from config_fp. Passes
 * key into singularity_get_config_value() and then checks if that
 * value is yes, no, or NULL. If not yes or no and not NULL, errors out.
 * 
 * @param char *key pointer to key to search for
 * @param int def integer representing the default value of key
 * @returns 1 for yes, 0 for no, def if NULL
 */
int _singularity_config_get_bool_impl(const char *key, int def) {
    return _singularity_config_get_bool_char_impl(key, def ? "yes" : "no");
}

int _singularity_config_get_bool_char_impl(const char *key, const char *def) {
    const char *config_value;

    singularity_message(DEBUG, "Called singularity_config_get_bool(%s, %s)\n", key, def);

    if ( ( config_value = _singularity_config_get_value_impl(key, def) ) != NULL ) {
        if ( strcmp(config_value, "yes") == 0 ||
                strcmp(config_value, "y") == 0 ||
                strcmp(config_value, "1") == 0 ) {
            singularity_message(DEBUG, "Return singularity_config_get_bool(%s, %s) = 1\n", key, def);
            return(1);
        } else if ( strcmp(config_value, "no") == 0 ||
                strcmp(config_value, "n") == 0 ||
                strcmp(config_value, "0") == 0 ) {
            singularity_message(DEBUG, "Return singularity_config_get_bool(%s, %s) = 0\n", key, def);
            return(0);
        } else {
            singularity_message(ERROR, "Unsupported value for configuration boolean key '%s' = '%s'\n", key, config_value);
            singularity_message(ERROR, "Returning default value: %s\n", def);
            ABORT(255);
        }
    } else {
        singularity_message(ERROR, "Undefined configuration for '%s'; should have resulted in a compile error.\n", key);
        ABORT(255);
    }

    return(-1);
}
