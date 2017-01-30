#!/bin/bash

install -d -m 0755 "$SINGULARITY_ROOTFS/bin"
install -d -m 0755 "$SINGULARITY_ROOTFS/dev"
install -d -m 0755 "$SINGULARITY_ROOTFS/home"
install -d -m 0755 "$SINGULARITY_ROOTFS/etc"
install -d -m 0750 "$SINGULARITY_ROOTFS/root"
install -d -m 0755 "$SINGULARITY_ROOTFS/proc"
install -d -m 0755 "$SINGULARITY_ROOTFS/sys"
install -d -m 1777 "$SINGULARITY_ROOTFS/tmp"
install -d -m 1777 "$SINGULARITY_ROOTFS/var/tmp"
touch "$SINGULARITY_ROOTFS/etc/hosts"
touch "$SINGULARITY_ROOTFS/etc/resolv.conf"
