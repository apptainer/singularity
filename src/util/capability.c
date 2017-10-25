/* 
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 * 
 * This software is licensed under a 3-clause BSD license.  Please
 * consult LICENSE file distributed with the sources of this project regarding
 * your rights to use or distribute this software.
 * 
 */


#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <fcntl.h>
#include <sys/types.h>
#include <sys/prctl.h>
#include <sys/sysmacros.h>
#include <pwd.h>
#include <errno.h>
#include <string.h>
#include <stdio.h>
#include <grp.h>
#include <limits.h>
#include <sched.h>
#include <linux/capability.h>
#include <sys/syscall.h>

#ifdef SINGULARITY_SECUREBITS
#  include <linux/securebits.h>
#else
#  include "util/securebits.h"
#endif /* SINGULARITY_SECUREBITS */

#include "config.h"

#include "util/file.h"
#include "util/util.h"
#include "util/registry.h"
#include "util/privilege.h"
#include "util/message.h"
#include "util/config_parser.h"

#ifndef SYSCONFDIR
#error SYSCONFDIR not defined
#endif

#define NO_CAP      100
#define CAPSET_MAX  40

/* Support only 64 bits sets, since kernel 2.6.25 */
#ifdef _LINUX_CAPABILITY_VERSION_3
#  define LINUX_CAPABILITY_VERSION  _LINUX_CAPABILITY_VERSION_3
#elif defined(_LINUX_CAPABILITY_VERSION_2)
#  define LINUX_CAPABILITY_VERSION  _LINUX_CAPABILITY_VERSION_2
#else
#  error Linux 64 bits capability set not supported
#endif /* _LINUX_CAPABILITY_VERSION_3 */

/*
    if uid != 0 -> no capabilities
    if uid = 0 -> default capabilities
    if uid = 0 and keep_privs -> all capabilities
    if uid = 0 and no_privs -> no capabilities
    if uid = 0 and build stage 2 -> minimal capabilities
*/

static __u32 default_capabilities[] = {
    CAP_SETUID,
    CAP_SETGID,
    CAP_SETPCAP,
    CAP_SETFCAP,
    CAP_SYS_ADMIN,
    CAP_NET_ADMIN,
    CAP_MKNOD,
    CAP_CHOWN,
    CAP_FOWNER,
    CAP_SYS_CHROOT,
    CAP_SYS_PTRACE,
    CAP_DAC_READ_SEARCH,
    CAP_DAC_OVERRIDE,
    CAP_AUDIT_WRITE,
    NO_CAP
};

static __u32 minimal_capabilities[] = {
    CAP_SETUID,
    CAP_SETGID,
    CAP_SETFCAP,
    CAP_CHOWN,
    CAP_FOWNER,
    CAP_SYS_CHROOT,
    CAP_DAC_READ_SEARCH,
    CAP_DAC_OVERRIDE,
    CAP_AUDIT_WRITE,
    NO_CAP
};

static __u32 no_capabilities[] = {
    NO_CAP
};

enum {
    ROOT_DEFCAPS_FULL,
    ROOT_DEFCAPS_FILE,
    ROOT_DEFCAPS_DEFAULT,
    ROOT_DEFCAPS_NO,
    ROOT_DEFCAPS_ERROR
};

static int get_root_default_capabilities(void) {
    char *value = strdup(singularity_config_get_value(ROOT_DEFAULT_CAPABILITIES));

    if ( value == NULL ) {
        return(ROOT_DEFCAPS_ERROR);
    }

    chomp(value);

    if ( strcmp(value, "full") == 0 ) {
        return(ROOT_DEFCAPS_FULL);
    } else if ( strcmp(value, "file") == 0 ) {
        return(ROOT_DEFCAPS_FILE);
    } else if ( strcmp(value, "default") == 0 ) {
        return(ROOT_DEFCAPS_DEFAULT);
    } else if ( strcmp(value, "no") == 0 ) {
        return(ROOT_DEFCAPS_NO);
    }

    return(ROOT_DEFCAPS_ERROR);
}

static __u32 *alloc_capability_set(void) {
    __u32 *caps = (__u32 *)calloc(CAPSET_MAX, sizeof(*caps));

    if ( caps == NULL ) {
        singularity_message(ERROR, "Failed to allocate memory for capability set\n");
        ABORT(255);
    }

    return(caps);
}

static __u32 *get_current_capabilities(void) {
    int i, caps_index = 0;
    __u32 *caps = alloc_capability_set();

    for ( i = CAPSET_MAX - 1; i >= 0; i-- ) {
        if ( prctl(PR_CAPBSET_READ, i) > 0 ) {
            caps[caps_index++] = i;
            if ( caps_index == CAPSET_MAX ) break;
        }
    }

    caps[caps_index] = NO_CAP;
    return(caps);
}

static __u32 *add_capabilities(__u32 *to, __u32 *capabilities) {
    int i, index = 0, add = 1;
    __u32 *ptr;
    __u32 *caps = alloc_capability_set();

    for ( ptr = to; *ptr != NO_CAP; ptr++ ) {
        caps[index++] = *ptr;
    }

    for ( ptr = capabilities; *ptr != NO_CAP; ptr++ ) {
        add = 1;
        for ( i = 0; i < index; i++ ) {
            if ( *ptr == caps[i] ) {
                add = 0;
                break;
            }
        }
        if ( add ) {
            caps[index++] = *ptr;
            if ( index == CAPSET_MAX ) break;
        }
    }

    caps[index] = NO_CAP;
    return(caps);
}

static __u32 *drop_capabilities(__u32 *from, __u32 *capabilities) {
    int index = 0, drop = 0;
    __u32 *ptr, *fptr;
    __u32 *caps = alloc_capability_set();

    for ( fptr = from; *fptr != NO_CAP; fptr++ ) {
        drop = 0;
        for ( ptr = capabilities; *ptr != NO_CAP; ptr++ ) {
            if ( *ptr == *fptr ) {
                drop = 1;
                break;
            }
        }
        if ( ! drop ) {
            caps[index++] = *fptr;
            if ( index == CAPSET_MAX ) break;
        }
    }

    caps[index] = NO_CAP;
    return(caps);
}

static char *cap2str(unsigned long long cap) {
    char *str = (char *)malloc(24);

    if ( str == NULL ) {
        singularity_message(ERROR, "Failed to allocate 24 memory bytes\n");
        ABORT(255);
    }

    memset(str, 0, 24);
    snprintf(str, 23, "%llu", cap);

    return(str);
}

static unsigned long long str2cap(char *value) {
    unsigned long long cap;

    errno = 0;
    cap = strtoull(value, NULL, 10);
    if ( errno != 0 ) {
        singularity_message(WARNING, "Can't convert string %s to unsigned long long\n", value);
        cap = 0;
    }

    return(cap);
}

static __u32 *get_capabilities_from(char *strval) {
    __u32 i, ncaps = 0;
    unsigned long long cap;
    __u32 *caps = alloc_capability_set();

    cap = str2cap(strval);

    for ( i = 0; i < CAPSET_MAX; i++ ) {
        if ( (cap & (1ULL << i)) ) {
            caps[ncaps++] = i;
            if ( ncaps == CAPSET_MAX ) break;
        }
    }

    caps[ncaps] = NO_CAP;
    return(caps);
}

static int capget(cap_user_header_t hdrp, cap_user_data_t datap) {
    return syscall(__NR_capget, hdrp, datap);
}

static int capset(cap_user_header_t hdrp, const cap_user_data_t datap) {
    return syscall(__NR_capset, hdrp, datap);
}

int singularity_capability_keep_privs(void) {
    if ( getuid() == 0 && singularity_registry_get("KEEP_PRIVS") != NULL ) {
        return(1);
    }
    return(0);
}

int singularity_capability_no_privs(void) {
    if ( getuid() == 0 && singularity_registry_get("NO_PRIVS") != NULL ) {
        return(1);
    }
    return(0);
}

static void singularity_capability_set_securebits(void) {
    if ( prctl(PR_SET_SECUREBITS, SECBIT_KEEP_CAPS|
                                  SECBIT_KEEP_CAPS_LOCKED|
                                  SECBIT_NOROOT|
                                  SECBIT_NOROOT_LOCKED|
                                  SECBIT_NO_SETUID_FIXUP|
                                  SECBIT_NO_SETUID_FIXUP_LOCKED) < 0) {
        singularity_message(ERROR, "Failed to set securebits\n");
        ABORT(255);
    }
}

static void singularity_capability_set(__u32 *capabilities) {
    __u32 caps_index;
    __u32 keep_index;
    __u32 keep_cap;
    __u32 last_cap;
    struct __user_cap_header_struct header;
    struct __user_cap_data_struct data[2];

    singularity_message(DEBUG, "Entering in a restricted capability set\n");

    header.version = LINUX_CAPABILITY_VERSION;
    header.pid = getpid();

    if ( capget(&header, data) < 0 ) {
        singularity_message(ERROR, "Failed to get processus capabilities\n");
        ABORT(255);
    }

    // We can't rely on CAP_LAST_CAP if singularity is compiled in a container by
    // example, host is ubuntu with a recent kernel and container is a centos 6
    // container, so CAP_LAST_CAP could be less than CAP_LAST_CAP host and we could
    // forget to drop some capabilities. So we take the MSB of permitted set

    singularity_message(DEBUG, "Determining highest capability of the running process\n");

    /* read bounding set */
    for ( last_cap = CAPSET_MAX; ; last_cap-- ) {
        if ( prctl(PR_CAPBSET_READ, last_cap) > 0 || last_cap == 0 ) {
            break;
        }
    }

    singularity_message(DEBUG, "Dropping capabilities in bounding set\n");
    for ( caps_index = 0; caps_index <= last_cap; caps_index++ ) {
        keep_cap = NO_CAP;
        for ( keep_index = 0; capabilities[keep_index] != NO_CAP; keep_index++ ) {
            if ( caps_index == capabilities[keep_index] ) {
                keep_cap = caps_index;
                break;
            }
        }
        if ( keep_cap == NO_CAP ) {
            if ( prctl(PR_CAPBSET_DROP, caps_index) < 0 ) {
                singularity_message(ERROR, "Failed to drop bounding capabilities set\n");
                ABORT(255);
            }
        }
    }

    // drop all in inheritable set to force childs to inherit capabilities from bounding set
    data[0].inheritable = 0;
    data[1].inheritable = 0;

    if ( capset(&header, data) < 0 ) {
        singularity_message(ERROR, "Failed to set processus capabilities\n");
        ABORT(255);
    }
}

static unsigned long long get_user_capabilities_from_file(void) {
    unsigned long long caps = 0;
    FILE *file = NULL;
    char strcap[24];
    char path[PATH_MAX];

    memset(strcap, 0, 24);
    memset(path, 0, PATH_MAX);

    snprintf(path, PATH_MAX-1, SYSCONFDIR "/singularity/capabilities/user.%d", getuid()); // Flawfinder: ignore

    printf("%s\n", path);

    file = fopen(path, "r");
    if ( file == NULL ) {
        printf("open failed: %s\n",strerror(errno));
        return(caps);
    }

    if ( fgets(strcap, 23, file) == NULL ) {
        printf("read failed\n");
        return(caps);
    }

    caps = str2cap(strcap);
    return(caps);
}

static unsigned long long get_group_capabilities_from_file(void) {
    unsigned long long caps = 0;

    return(caps);
}

static int setup_root_default_capabilities(void) {
    int root_default_caps = get_root_default_capabilities();

    if ( getuid() == 0 ) {
        if ( root_default_caps == ROOT_DEFCAPS_ERROR ) {
            singularity_message(WARNING, "root default capabilities value in configuration is unknown, set to no\n");
            singularity_registry_set("NO_PRIVS", "1");
            singularity_registry_set("KEEP_PRIVS", NULL);

            unsetenv("SINGULARITY_KEEP_PRIVS");
            envar_set("SINGULARITY_NO_PRIVS", "1", 1);
        } else if ( root_default_caps == ROOT_DEFCAPS_FULL ) {
            singularity_registry_set("KEEP_PRIVS", "1");
            envar_set("SINGULARITY_KEEP_PRIVS", "1", 1);
        } else if ( root_default_caps == ROOT_DEFCAPS_FILE ) {
            unsigned long long filecap = get_user_capabilities_from_file();

            if ( singularity_registry_get("ADD_CAPS") == NULL ) {
                singularity_registry_set("ADD_CAPS", cap2str(filecap));
            } else {
                unsigned long long envcap = str2cap(singularity_registry_get("ADD_CAPS"));
                singularity_registry_set("ADD_CAPS", cap2str(envcap | filecap));
            }
            envar_set("SINGULARITY_ADD_CAPS", singularity_registry_get("ADD_CAPS"), 1);

            if ( ! singularity_capability_keep_privs() ) {
                singularity_registry_set("NO_PRIVS", "1");
                envar_set("SINGULARITY_NO_PRIVS", "1", 1);
            }
        }
    }

    envar_set("SINGULARITY_ROOT_DEFAULT_CAPS", int2str(root_default_caps), 1);
    return(root_default_caps);
}

void singularity_capability_init(void) {
    int root_user = (getuid() == 0) ? 1 : 0;

    setup_root_default_capabilities();

    if ( ! singularity_capability_keep_privs() ) {
        if ( singularity_registry_get("ADD_CAPS") && root_user ) {
            __u32 *capabilities = get_capabilities_from(singularity_registry_get("ADD_CAPS"));
            __u32 *final = add_capabilities(default_capabilities, capabilities);

            singularity_capability_set(final);

            free(capabilities);
            free(final);
        } else {
            singularity_capability_set(default_capabilities);
        }
    }
}

/* for mount command */
void singularity_capability_init_default(void) {
    singularity_capability_set(default_capabilities);
    envar_set("SINGULARITY_ROOT_DEFAULT_CAPS", int2str(ROOT_DEFCAPS_DEFAULT), 1);
    unsetenv("SINGULARITY_ADD_CAPS");
    unsetenv("SINGULARITY_DROP_CAPS");
    unsetenv("SINGULARITY_NO_PRIVS");
    unsetenv("SINGULARITY_KEEP_PRIVS");
}

/* for build stage 2 */
void singularity_capability_init_minimal(void) {
    singularity_capability_set(minimal_capabilities);
}

void singularity_capability_drop(void) {
    long int root_default_caps;
    int root_user = (getuid() == 0) ? 1 : 0;

    if ( singularity_registry_get("ROOT_DEFAULT_CAPS") == NULL ) {
        root_default_caps = setup_root_default_capabilities();
    } else {
        if ( str2int(singularity_registry_get("ROOT_DEFAULT_CAPS"), &root_default_caps) == -1 ) {
            singularity_message(ERROR, "Failed to get root default capabilities via environment variable\n");
            ABORT(255);
        }
    }

    if ( root_default_caps == ROOT_DEFCAPS_NO && root_user ) {
        if ( ! singularity_capability_keep_privs() ) {
            singularity_registry_set("NO_PRIVS", "1");
        }
    }

    if ( singularity_capability_no_privs() || ( ! singularity_capability_keep_privs() && ! root_user ) ) {
        singularity_message(DEBUG, "Drop capabilities\n");
        if ( singularity_registry_get("ADD_CAPS") && root_user ) {
            __u32 *capabilities = get_capabilities_from(singularity_registry_get("ADD_CAPS"));
            __u32 *final = add_capabilities(no_capabilities, capabilities);

            singularity_capability_set(final);

            free(capabilities);
            free(final);
        } else {
            singularity_capability_set_securebits();
            singularity_capability_set(no_capabilities);
        }
    }
    if ( singularity_registry_get("DROP_CAPS") && root_user ) {
        __u32 *capabilities = get_capabilities_from(singularity_registry_get("DROP_CAPS"));
        __u32 *current = get_current_capabilities();
        __u32 *final = drop_capabilities(current, capabilities);

        singularity_capability_set(final);

        free(current);
        free(capabilities);
        free(final);
    }
}
