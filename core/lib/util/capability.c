/* 
 * Copyright (c) 2017-2018, SyLabs, Inc. All rights reserved.
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
#  include "securebits.h"
#endif /* SINGULARITY_SECUREBITS */

#include "file.h"
#include "util.h"
#include "registry.h"
#include "privilege.h"
#include "message.h"
#include "config_parser.h"

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
    if uid = 0 -> root default capabilities
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

static unsigned long long get_current_capabilities(void) {
    int i;
    unsigned long long caps = 0;

    for ( i = CAPSET_MAX - 1; i >= 0; i-- ) {
        if ( prctl(PR_CAPBSET_READ, i) > 0 ) {
            caps |= (1ULL << i);
        }
    }

    return(caps);
}

static char *cap2str(unsigned long long cap) {
    char *str = (char *)calloc(24, sizeof(char *));

    if ( str == NULL ) {
        singularity_message(ERROR, "Failed to allocate 24 memory bytes\n");
        ABORT(255);
    }

    snprintf(str, 23, "%llu", cap);

    return(str);
}

static unsigned long long str2cap(char *value) {
    unsigned long long caps = 0;

    if ( value == NULL ) return(caps);

    errno = 0;
    caps = strtoull(value, NULL, 10);
    if ( errno != 0 ) {
        singularity_message(WARNING, "Can't convert string %s to unsigned long long\n", value);
        caps = 0;
    }

    return(caps);
}

static unsigned long long array2cap(__u32 *capabilities) {
    unsigned long long caps = 0;
    __u32 i;

    for ( i = 0; capabilities[i] != NO_CAP; i++ ) {
        caps |= (1ULL << capabilities[i]);
    }

    return(caps);
}

static int capget(cap_user_header_t hdrp, cap_user_data_t datap) {
    return syscall(__NR_capget, hdrp, datap);
}

static int capset(cap_user_header_t hdrp, const cap_user_data_t datap) {
    return syscall(__NR_capset, hdrp, datap);
}

static int singularity_capability_keep_privs(void) {
    if ( singularity_priv_getuid() == 0 && singularity_registry_get("KEEP_PRIVS") != NULL ) {
        return(1);
    }
    return(0);
}

static int singularity_capability_no_privs(void) {
    if ( singularity_priv_getuid() == 0 && singularity_registry_get("NO_PRIVS") != NULL ) {
        return(1);
    }
    return(0);
}

static void singularity_capability_set_securebits(int bits) {
    int current_bits = prctl(PR_GET_SECUREBITS);

    if ( current_bits < 0 ) {
        singularity_message(ERROR, "Failed to read securebits\n");
        ABORT(255);
    }

    if ( ! (current_bits & SECBIT_NO_SETUID_FIXUP_LOCKED) ) {
        if ( singularity_capability_keep_privs() == 1 ) {
            return;
        }

        if ( singularity_priv_getuid() == 0 ) {
            bits &= ~(SECBIT_NOROOT|SECBIT_NOROOT_LOCKED);
        }

        if ( prctl(PR_SET_SECUREBITS, bits) < 0 ) {
            singularity_message(ERROR, "Failed to set securebits\n");
            ABORT(255);
        }
    }
}

void singularity_capability_keep(void) {
    singularity_capability_set_securebits(SECBIT_NO_SETUID_FIXUP);
}

void singularity_capability_set_effective(void) {
    struct __user_cap_header_struct header;
    struct __user_cap_data_struct data[2];

    singularity_message(DEBUG, "Set effective/permitted capabilities for current processus\n");

    header.version = LINUX_CAPABILITY_VERSION;
    header.pid = getpid();

    if ( capget(&header, data) < 0 ) {
        singularity_message(ERROR, "Failed to get processus capabilities\n");
        ABORT(255);
    }

#ifdef USER_CAPABILITIES
    data[1].permitted = data[1].inheritable;
    data[0].permitted = data[0].inheritable;

    data[1].effective = 0;
    data[0].effective = 0;
#else
    data[1].permitted = data[1].effective = data[1].inheritable;
    data[0].permitted = data[0].effective = data[0].inheritable;
#endif

    if ( capset(&header, data) < 0 ) {
        singularity_message(ERROR, "Failed to set processus capabilities\n");
        ABORT(255);
    }
}

static void singularity_capability_set(unsigned long long capabilities) {
    __u32 caps_index;
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
    // forget to drop some capabilities

    singularity_message(DEBUG, "Determining highest capability of the running process\n");

    /* read bounding set */
    for ( last_cap = CAPSET_MAX; ; last_cap-- ) {
        if ( prctl(PR_CAPBSET_READ, last_cap) > 0 || last_cap == 0 ) {
            break;
        }
    }

    singularity_message(DEBUG, "Dropping capabilities in bounding set\n");
    for ( caps_index = 0; caps_index <= last_cap; caps_index++ ) {
        if ( !(capabilities & (1ULL << caps_index)) ) {
            if ( prctl(PR_CAPBSET_DROP, caps_index) < 0 ) {
                singularity_message(ERROR, "Failed to drop bounding capabilities set\n");
                ABORT(255);
            }
        }
    }

    data[1].inheritable = (__u32)(capabilities >> 32);
    data[0].inheritable = (__u32)(capabilities & 0xFFFFFFFF);

    if ( capset(&header, data) < 0 ) {
        singularity_message(ERROR, "Failed to set processus capabilities\n");
        ABORT(255);
    }

#ifdef USER_CAPABILITIES
    // set ambient capabilities if supported
    if ( singularity_config_get_bool(ALLOW_USER_CAPABILITIES) ) {
        int i;
        for (i = 0; i <= CAPSET_MAX; i++ ) {
            if ( (capabilities & (1ULL << i)) ) {
                singularity_message(DEBUG, "Set ambient cap %d\n", i);
                if ( prctl(PR_CAP_AMBIENT, PR_CAP_AMBIENT_RAISE, i, 0, 0) < 0 ) {
                    singularity_message(ERROR, "Failed to set ambient capability\n");
                    ABORT(255);
                }
            }
        }
    }
#endif /* USER_CAPABILITIES */
}

static unsigned long long get_capabilities_from_file(char *ftype, char *id) {
    unsigned long long caps = 0;
    FILE *file = NULL;
    char strcap[24];
    static char path[PATH_MAX];

    singularity_message(DEBUG, "Get capabilities from file for %s %s\n", ftype, id);

    memset(strcap, 0, 24);
    memset(path, 0, PATH_MAX);

    snprintf(path, PATH_MAX-1, SYSCONFDIR "/singularity/capabilities/%s.%s", ftype, id); // Flawfinder: ignore

    file = fopen(path, "r");
    if ( file == NULL ) {
        singularity_message(DEBUG, "Fail to open %s: %s\n", path, strerror(errno));
        return(caps);
    }

    if ( fgets(strcap, 23, file) == NULL ) {
        singularity_message(DEBUG, "Fail to read %s content: %s\n", path, strerror(errno));
        return(caps);
    }

    caps = str2cap(strcap);
    return(caps);
}

static unsigned long long get_user_file_capabilities(void) {
    unsigned long long caps = 0;
    uid_t uid = singularity_priv_getuid();
    struct passwd *pw;

    pw = getpwuid(uid);
    if ( pw == NULL ) {
        singularity_message(ERROR, "Failed to retrieve password file entry for uid %d\n", uid);
        ABORT(255);
    }

    caps = get_capabilities_from_file("user", pw->pw_name);
    return(caps);
}

unsigned long long get_group_file_capabilities(void) {
    unsigned long long caps = 0;
    gid_t *gids = NULL;
    int i, count = getgroups(0, NULL);

    if ( count > NGROUPS_MAX ) {
        singularity_message(ERROR, "Number of user group\n");
        ABORT(255);
    }

    gids = (gid_t *)calloc(count, sizeof(gid_t));
    if ( gids == NULL ) {
        singularity_message(ERROR, "Failed to allocate memory for user groups\n");
        ABORT(255);
    }

    if ( getgroups(count, gids) < 0 ) {
        singularity_message(ERROR, "Failed to retrieve user group\n");
        ABORT(255);
    }

    for ( i = count - 1; i >= 0; i-- ) {
        struct group *gr = getgrgid(gids[i]);

        if ( gr == NULL ) {
            singularity_message(ERROR, "Failed to retrieve group file entry for gid %d\n", gids[i]);
            ABORT(255);
        }
        caps |= get_capabilities_from_file("group", gr->gr_name);
    }

    free(gids);

    return(caps);
}

static unsigned long long setup_user_capabilities(void) {
    unsigned long long caps = 0;

#ifdef USER_CAPABILITIES
    if ( singularity_config_get_bool(ALLOW_USER_CAPABILITIES) ) {
        unsigned long long tcaps = str2cap(singularity_registry_get("ADD_CAPS"));
        caps |= get_user_file_capabilities();
        caps |= get_group_file_capabilities();
        caps = tcaps & caps;
        singularity_registry_set("ADD_CAPS", cap2str(caps));
        envar_set("SINGULARITY_ADD_CAPS", singularity_registry_get("ADD_CAPS"), 1);
    } else {
        envar_set("SINGULARITY_ADD_CAPS", "0", 1);
    }
#else
    if ( singularity_config_get_bool(ALLOW_USER_CAPABILITIES) ) {
        singularity_message(WARNING, "User capabilities are not supported by your kernel\n");
        envar_set("SINGULARITY_ADD_CAPS", "0", 1);
    }
#endif

    return(caps);
}

static int setup_capabilities(void) {
    int root_default_caps = get_root_default_capabilities();

    if ( singularity_priv_getuid() == 0 ) {
        if ( singularity_config_get_bool(ALLOW_ROOT_CAPABILITIES) <= 0 ) {
            singularity_registry_set("ADD_CAPS", NULL);
            unsetenv("SINGULARITY_ADD_CAPS");
            unsetenv("SINGULARITY_DROP_CAPS");
            singularity_registry_set("DROP_CAPS", NULL);
            unsetenv("SINGULARITY_NO_PRIVS");
            singularity_registry_set("NO_PRIVS", NULL);
            unsetenv("SINGULARITY_KEEP_PRIVS");
            singularity_registry_set("KEEP_PRIVS", NULL);
        }

        if ( root_default_caps == ROOT_DEFCAPS_ERROR ) {
            singularity_message(WARNING, "root default capabilities value in configuration is unknown, set to no\n");
            singularity_registry_set("NO_PRIVS", "1");
            singularity_registry_set("KEEP_PRIVS", NULL);

            unsetenv("SINGULARITY_KEEP_PRIVS");
            envar_set("SINGULARITY_NO_PRIVS", "1", 1);
        } else if ( root_default_caps == ROOT_DEFCAPS_FULL || singularity_capability_keep_privs() ) {
            unsigned long long caps = get_current_capabilities();
            if ( singularity_registry_get("NO_PRIVS") == NULL ) {
                singularity_registry_set("KEEP_PRIVS", "1");
                envar_set("SINGULARITY_KEEP_PRIVS", "1", 1);
                singularity_registry_set("ADD_CAPS", cap2str(caps));
                envar_set("SINGULARITY_ADD_CAPS", singularity_registry_get("ADD_CAPS"), 1);
            } else {
                envar_set("SINGULARITY_NO_PRIVS", "1", 1);
                unsetenv("SINGULARITY_KEEP_PRIVS");
                envar_set("SINGULARITY_ADD_CAPS", singularity_registry_get("ADD_CAPS"), 1);
            }
        } else if ( root_default_caps == ROOT_DEFCAPS_FILE ) {
            unsigned long long filecap = get_user_file_capabilities();

            if ( singularity_registry_get("ADD_CAPS") == NULL ) {
                if ( ! singularity_capability_no_privs() ) {
                    singularity_registry_set("ADD_CAPS", cap2str(filecap));
                }
            } else {
                unsigned long long envcap = str2cap(singularity_registry_get("ADD_CAPS"));
                if ( ! singularity_capability_no_privs() ) {
                    singularity_registry_set("ADD_CAPS", cap2str(envcap | filecap));
                }
            }
            envar_set("SINGULARITY_ADD_CAPS", singularity_registry_get("ADD_CAPS"), 1);

            if ( ! singularity_capability_keep_privs() ) {
                singularity_registry_set("NO_PRIVS", "1");
                envar_set("SINGULARITY_NO_PRIVS", "1", 1);
            }
        } else if ( root_default_caps == ROOT_DEFCAPS_NO ) {
            if ( ! singularity_capability_keep_privs() ) {
                singularity_registry_set("NO_PRIVS", "1");
                envar_set("SINGULARITY_NO_PRIVS", "1", 1);
            }
        }
    } else {
        setup_user_capabilities();
    }

    envar_set("SINGULARITY_ROOT_DEFAULT_CAPS", int2str(root_default_caps), 1);
    return(root_default_caps);
}

void singularity_capability_init(void) {
    setup_capabilities();

    if ( ! singularity_capability_keep_privs() ) {
        if ( singularity_registry_get("ADD_CAPS") ) {
            unsigned long long capabilities = str2cap(singularity_registry_get("ADD_CAPS"));
            unsigned long long final = array2cap(default_capabilities) | capabilities;

            singularity_capability_set(final);
        } else {
            singularity_capability_set(array2cap(default_capabilities));
        }
    }
}

/* for mount command */
void singularity_capability_init_default(void) {
    singularity_capability_set(array2cap(default_capabilities));
    envar_set("SINGULARITY_ROOT_DEFAULT_CAPS", int2str(ROOT_DEFCAPS_DEFAULT), 1);
    unsetenv("SINGULARITY_ADD_CAPS");
    unsetenv("SINGULARITY_DROP_CAPS");
    unsetenv("SINGULARITY_NO_PRIVS");
    unsetenv("SINGULARITY_KEEP_PRIVS");
}

/* for build stage 2 */
void singularity_capability_init_minimal(void) {
    singularity_capability_set(array2cap(minimal_capabilities));
    unsetenv("SINGULARITY_ADD_CAPS");
    unsetenv("SINGULARITY_DROP_CAPS");
    unsetenv("SINGULARITY_NO_PRIVS");
    unsetenv("SINGULARITY_KEEP_PRIVS");
}

void singularity_capability_drop(void) {
    long int root_default_caps;
    int root_user = (singularity_priv_getuid() == 0) ? 1 : 0;

    if ( singularity_registry_get("ROOT_DEFAULT_CAPS") == NULL ) {
        root_default_caps = setup_capabilities();
    } else {
        if ( str2int(singularity_registry_get("ROOT_DEFAULT_CAPS"), &root_default_caps) == -1 ) {
            singularity_message(ERROR, "Failed to get root default capabilities via environment variable\n");
            ABORT(255);
        }
    }

    if ( singularity_capability_keep_privs() ) {
        unsigned long long capabilities = str2cap(singularity_registry_get("ADD_CAPS"));
        singularity_capability_set(capabilities);
    }

    if ( singularity_capability_no_privs() || ( ! singularity_capability_keep_privs() && ! root_user ) ) {
        singularity_message(DEBUG, "Set capabilities\n");
        unsigned long long capabilities = str2cap(singularity_registry_get("ADD_CAPS"));
        singularity_capability_set(capabilities);
    }
    if ( singularity_registry_get("DROP_CAPS") ) {
        singularity_message(DEBUG, "Drop capabilities requested by user\n");
        unsigned long long capabilities = str2cap(singularity_registry_get("DROP_CAPS"));
        unsigned long long current = get_current_capabilities();

        current &= ~capabilities;

        singularity_capability_set(current);
    }
    singularity_capability_set_securebits(SECURE_ALL_BITS|SECURE_ALL_LOCKS);
    singularity_capability_set_effective();
}
