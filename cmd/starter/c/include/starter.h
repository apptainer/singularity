/*
  Copyright (c) 2018-2019, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE.md file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

#ifndef _SINGULARITY_STARTER_H
#define _SINGULARITY_STARTER_H

#include <limits.h>
#include <sys/user.h>

#define fatalf(b...)     singularity_message(ERROR, b); \
                         exit(1)
#define debugf(b...)     singularity_message(DEBUG, b)
#define verbosef(b...)   singularity_message(VERBOSE, b)
#define warningf(b...)   singularity_message(WARNING, b)
#define errorf(b...)     singularity_message(ERROR, b)

#define MAX_JSON_SIZE       128*1024
#define MAX_MAP_SIZE        PAGE_SIZE
#define MAX_NS_PATH_SIZE    PATH_MAX
#define MAX_GID             32

struct fdlist {
    int *fds;
    unsigned int num;
};

struct capabilities {
    unsigned long long permitted;
    unsigned long long effective;
    unsigned long long inheritable;
    unsigned long long bounding;
    unsigned long long ambient;
};

struct namespace {
    unsigned int flags;
    char network[MAX_NS_PATH_SIZE];
    char mount[MAX_NS_PATH_SIZE];
    char user[MAX_NS_PATH_SIZE];
    char ipc[MAX_NS_PATH_SIZE];
    char uts[MAX_NS_PATH_SIZE];
    char cgroup[MAX_NS_PATH_SIZE];
    char pid[MAX_NS_PATH_SIZE];
};

struct container {
    pid_t pid;

    unsigned char isSuid;
    unsigned char noNewPrivs;

    char uidMap[MAX_MAP_SIZE];
    char gidMap[MAX_MAP_SIZE];

    uid_t targetUID;
    gid_t targetGID[MAX_GID];
    int numGID;

    unsigned char isInstance;
    unsigned long mountPropagation;
    unsigned char sharedMount;
    unsigned char joinMount;
    unsigned char bringLoopbackInterface;
};

struct json {
    char config[MAX_JSON_SIZE];
    size_t size;
};

struct cConfig {
    struct capabilities capabilities;
    struct namespace namespace;
    struct container container;
    struct json json;
};

#endif /* _SINGULARITY_STARTER_H */
