/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE.md file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

#ifndef _SINGULARITY_STARTER_H
#define _SINGULARITY_STARTER_H

#define fatalf(b...)     singularity_message(ERROR, b); \
                         exit(1)
#define debugf(b...)     singularity_message(DEBUG, b)
#define verbosef(b...)   singularity_message(VERBOSE, b)
#define warningf(b...)   singularity_message(WARNING, b)
#define errorf(b...)     singularity_message(ERROR, b)

#define MAX_NSPATH_SIZE PATH_MAX*7
#define MAX_JSON_SIZE   128*1024
#define MAX_ID_MAPPING  5
#define MAX_GID         32

struct fdlist {
    int *fds;
    unsigned int num;
};

struct uidMapping {
    uid_t hostID;
    uid_t containerID;
    unsigned int size;
};

struct gidMapping {
    gid_t hostID;
    gid_t containerID;
    unsigned int size;
};

struct cConfig {
    unsigned long long capPermitted;
    unsigned long long capEffective;
    unsigned long long capInheritable;
    unsigned long long capBounding;
    unsigned long long capAmbient;
    unsigned long mountPropagation;
    unsigned char sharedMount;
    unsigned int nsFlags;
    pid_t containerPid;
    off_t netNsPathOffset;
    off_t mntNsPathOffset;
    off_t userNsPathOffset;
    off_t ipcNsPathOffset;
    off_t utsNsPathOffset;
    off_t cgroupNsPathOffset;
    off_t pidNsPathOffset;
    unsigned char isSuid;
    unsigned char isInstance;
    unsigned char noNewPrivs;
    struct uidMapping uidMapping[MAX_ID_MAPPING];
    struct gidMapping gidMapping[MAX_ID_MAPPING];
    uid_t targetUID;
    gid_t targetGID[MAX_GID];
    int numGID;
    unsigned int jsonConfSize;
    unsigned int nsPathSize;
};

#endif /* _SINGULARITY_STARTER_H */
