#!/bin/bash
# 
# Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
# Copyright (c) 2015-2017, Gregory M. Kurtzer. All rights reserved.
# Copyright (c) 2017, Vanessa Sochat. All rights reserved. 

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
export SINGULARITY_MOUNTPOINT SINGULARITY_CHECK SINGULARITY_ROOTFS

##################################################################################
# CHECK SCRIPTS
##################################################################################

#        [SUCCESS] [LEVEL]  [SCRIPT]                                                                         [TAGS]
execute_check    0    HIGH  "bash $SINGULARITY_libexecdir/singularity/helpers/checks/1-hello-world.sh"       security
execute_check    0     LOW  "python $SINGULARITY_libexecdir/singularity/helpers/checks/2-cache-content.py"   clean
execute_check    0    HIGH  "python $SINGULARITY_libexecdir/singularity/helpers/checks/3-cve.py"             security



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

return 0
