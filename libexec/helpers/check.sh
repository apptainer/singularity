#!/bin/bash
#
# Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
# Copyright (c) 2017, Vanessa Sochat. All rights reserved.
#
# See the COPYRIGHT.md file at the top-level directory of this distribution and at
# https://github.com/singularityware/singularity/blob/master/COPYRIGHT.md.
#
# This file is part of the Singularity Linux container project. It is subject to the license
# terms in the LICENSE.md file found in the top-level directory of this distribution and
# at https://github.com/singularityware/singularity/blob/master/LICENSE.md. No part
# of Singularity, including this file, may be copied, modified, propagated, or distributed
# except according to the terms contained in the LICENSE.md file.



## Basic sanity
if [ -z "$SINGULARITY_libexecdir" ]; then
    echo "Could not identify the Singularity libexecdir."
    exit 1
fi


## Load functions
if [ -f "$SINGULARITY_libexecdir/singularity/functions" ]; then
    . "$SINGULARITY_libexecdir/singularity/functions"
else
    echo "Error loading functions: $SINGULARITY_libexecdir/singularity/functions"
    exit 1
fi

if [ ! -d "${SINGULARITY_MOUNTPOINT}" ]; then
    message ERROR "The mount point does not exist: ${SINGULARITY_MOUNTPOINT}\n"
    ABORT 255
fi


if [ -z "${SINGULARITY_MOUNTPOINT}" ]; then
    message ERROR "The mount point does not exist: ${SINGULARITY_MOUNTPOINT}\n"
    ABORT 255
fi


SINGULARITY_ROOTFS=${SINGULARITY_MOUNTPOINT}
export SINGULARITY_MOUNTPOINT SINGULARITY_CHECKLEVEL \
       SINGULARITY_ROOTFS SINGULARITY_CHECKTAGS

message DEFAULT "Checking tags $SINGULARITY_CHECKTAGS"

##################################################################################
# USER TAGS
##################################################################################


##################################################################################
# CHECK SCRIPTS
##################################################################################

#        [SUCCESS] [LEVEL]  [SCRIPT] [TAGS]
                                                                       
exec_check 0 LOW  "python $SINGULARITY_libexecdir/singularity/helpers/checks/1-bash-hiddens.py" security clean bootstrap
exec_check 0 LOW  "bash $SINGULARITY_libexecdir/singularity/helpers/checks/1-hello-world.sh" testing
exec_check 0 LOW  "python $SINGULARITY_libexecdir/singularity/helpers/checks/1-cache-content.py" default clean bootstrap
exec_check 0 HIGH "python $SINGULARITY_libexecdir/singularity/helpers/checks/3-cve.py" security
exec_check 0 LOW  "python $SINGULARITY_libexecdir/singularity/helpers/checks/1-docker.py" docker



##################################################################################
# Checks we want to add
##################################################################################

# 1) history cleanup (no trace of credentials like "echo changemenow| sudo passwd --stdin root")
# 2) remove any file specific to the machine and not required on the container 
#    (same files as for cloning instances) e.g /etc/ssh/host, 
#    cleanup /etc/hosts, /etc/# sysconfig/network-scripts/ifcfg-, 
#    but not ifcfg-lo, /etc/udev/rules.d/persistent ) /var/log/{secure,messages*,...}
# 3) yum clean all and the apt-get equivalent
# 4) SElinux or ACLs transfer? tar --selinux --xattrs
# 5) sparse file handling
# 6) remove any non required users via userdel?
# 7) maybe reuse|have a peak at cloud-init ?
# 8) maybe http://libguestfs.org/virt-sysprep.1.html#operations would be a better start
