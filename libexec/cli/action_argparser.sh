#!/bin/bash
#
# Copyright (c) 2017-2018, Sylabs, Inc. All rights reserved.
#
# See the COPYRIGHT.md file at the top-level directory of this distribution and at
# https://github.com/singularityware/singularity/blob/master/COPYRIGHT.md.
#
# This file is part of the Singularity Linux container project. It is subject to the license
# terms in the LICENSE.md file found in the top-level directory of this distribution and
# at https://github.com/singularityware/singularity/blob/master/LICENSE.md. No part
# of Singularity, including this file, may be copied, modified, propagated, or distributed
# except according to the terms contained in the LICENSE.md file.
#
# This file also contains content that is covered under the LBNL/DOE/UC modified
# 3-clause BSD license and is subject to the license terms in the LICENSE-LBNL.md
# file found in the top-level directory of this distribution and at
# https://github.com/singularityware/singularity/blob/master/LICENSE-LBNL.md.

## Load capabilities
if [ -f "$SINGULARITY_libexecdir/singularity/capabilities" ]; then
    . "$SINGULARITY_libexecdir/singularity/capabilities"
    singularity_get_env_capabilities
else
    echo "Error loading capabilities: $SINGULARITY_libexecdir/singularity/capabilities"
    exit 1
fi

message 2 "Evaluating args: '$*'\n"

while true; do
    case ${1:-} in
        -h|--help|help)
            exec "$SINGULARITY_libexecdir/singularity/cli/help.exec" "$SINGULARITY_COMMAND"
        ;;
        -o|--overlay)
            shift
            SINGULARITY_OVERLAYIMAGE="${1:-}"
            export SINGULARITY_OVERLAYIMAGE
            shift

            if [ ! -e "${SINGULARITY_OVERLAYIMAGE:-}" ]; then
                message ERROR "Overlay image must be a file or directory!\n"
                ABORT 255
            fi
        ;;
        -s|--shell)
            shift
            SINGULARITY_SHELL="${1:-}"
            export SINGULARITY_SHELL
            shift
        ;;
        -u|--user|--userns)
            SINGULARITY_NOSUID=1
            export SINGULARITY_NOSUID
            shift
        ;;
        -w|--writable)
            shift
            SINGULARITY_WRITABLE=1
            export SINGULARITY_WRITABLE
        ;;
        -H|--home)
            shift
            SINGULARITY_HOME="$1"
            export SINGULARITY_HOME
            shift
        ;;
        -W|--wdir|--workdir|--workingdir)
            shift
            SINGULARITY_WORKDIR="$1"
            export SINGULARITY_WORKDIR
            shift
        ;;
        -S|--scratchdir|--scratch-dir|--scratch)
            shift
            SINGULARITY_SCRATCHDIR="$1,${SINGULARITY_SCRATCHDIR:-}"
            export SINGULARITY_SCRATCHDIR
            shift
        ;;
        app|--app|-a)
            shift
            SINGULARITY_APPNAME="${1:-}"
            export SINGULARITY_APPNAME
            shift
        ;;
        -B|--bind)
            shift
            SINGULARITY_BINDPATH="${SINGULARITY_BINDPATH:-},${1:-}"
            export SINGULARITY_BINDPATH
            shift
        ;;
        -c|--contain)
            shift
            SINGULARITY_CONTAIN=1
            export SINGULARITY_CONTAIN
        ;;
        -C|--containall|--CONTAIN)
            shift
            SINGULARITY_CONTAIN=1
            SINGULARITY_UNSHARE_PID=1
            SINGULARITY_UNSHARE_IPC=1
            SINGULARITY_CLEANENV=1
            export SINGULARITY_CONTAIN SINGULARITY_UNSHARE_PID SINGULARITY_UNSHARE_IPC SINGULARITY_CLEANENV
        ;;
        -e|--cleanenv)
            shift
            SINGULARITY_CLEANENV=1
            export SINGULARITY_CLEANENV
        ;;
        -p|--pid)
            shift
            SINGULARITY_UNSHARE_PID=1
            export SINGULARITY_UNSHARE_PID
        ;;
        -i|--ipc)
            shift
            SINGULARITY_UNSHARE_IPC=1
            export SINGULARITY_UNSHARE_IPC
        ;;
        -n|--net)
            shift
            SINGULARITY_UNSHARE_NET=1
            export SINGULARITY_UNSHARE_NET
        ;;
        --uts)
            shift
            SINGULARITY_UNSHARE_UTS=1
            export SINGULARITY_UNSHARE_UTS
        ;;
        --hostname)
            shift
            SINGULARITY_UNSHARE_UTS=1
            SINGULARITY_HOSTNAME="$1"
            export SINGULARITY_UNSHARE_UTS SINGULARITY_HOSTNAME
            shift
        ;;
        --pwd)
            shift
            SINGULARITY_TARGET_PWD="$1"
            export SINGULARITY_TARGET_PWD
            shift
        ;;
        --nv)
            shift
            SINGULARITY_NV=1
        ;;
        -f|--fakeroot)
            shift
            SINGULARITY_NOSUID=1
            SINGULARITY_USERNS_UID=0
            SINGULARITY_USERNS_GID=0
            export SINGULARITY_USERNS_UID SINGULARITY_USERNS_GID SINGULARITY_NOSUID
        ;;
        --keep-privs)
            if [ "$(id -ru)" = "0" ]; then
                if [ "$(singularity_config_value 'allow root capabilities')" != "yes" ]; then
                    message ERROR "keep-privs is disabled when allow root capabilities directive is set to no\n"
                    exit 1
                fi
                SINGULARITY_KEEP_PRIVS=1
                export SINGULARITY_KEEP_PRIVS
                message 4 "Requesting keep privileges\n"
            else
                message WARNING "Keeping privileges is for root only\n"
            fi
            shift
        ;;
        --no-privs)
            if [ "$(id -ru)" = "0" ]; then
                SINGULARITY_NO_PRIVS=1
                export SINGULARITY_NO_PRIVS
                message 4 "Requesting no privileges\n"
            fi
            shift
        ;;
        --add-caps)
            shift
            singularity_add_capabilities "$1"
            shift
        ;;
        --drop-caps)
            shift
            singularity_drop_capabilities "$1"
            shift
        ;;
        --allow-setuid)
            shift
            if [ "$(id -ru)" != "0" ]; then
                message ERROR "allow-setuid is for root only\n"
                exit 1
            fi
            if [ "$(singularity_config_value 'allow root capabilities')" != "yes" ]; then
                message ERROR "allow-setuid is disabled when allow root capabilities directive is set to no\n"
                exit 1
            fi
            SINGULARITY_ALLOW_SETUID=1
            export SINGULARITY_ALLOW_SETUID
        ;;
        -*)
            message ERROR "Unknown option: ${1:-}\n"
            exit 1
        ;;
        *)
            break;
        ;;
    esac
done

if [ -z "${SINGULARITY_NV_OFF:-}" ]; then # this is a "kill switch" provided for user
    # if singularity.conf specifies --nv
    if [ `$SINGULARITY_libexecdir/singularity/bin/get-configvals "always use nv"` == "yes" ]; then 
        message 2 "'always use nv = yes' found in singularity.conf\n"
        message 2 "binding nvidia files into container\n"
        bind_nvidia_files
    # or if the user asked for --nv    
    elif [ -n "${SINGULARITY_NV:-}" ]; then
        bind_nvidia_files
    fi
fi 


