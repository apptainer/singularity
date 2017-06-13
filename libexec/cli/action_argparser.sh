#!/bin/bash
# 
# Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
#
# Copyright (c) 2015-2017, Gregory M. Kurtzer. All rights reserved.
# 
# Copyright (c) 2016-2017, The Regents of the University of California,
# through Lawrence Berkeley National Laboratory (subject to receipt of any
# required approvals from the U.S. Dept. of Energy).  All rights reserved.
# 
# This software is licensed under a customized 3-clause BSD license.  Please
# consult LICENSE file distributed with the sources of this project regarding
# your rights to use or distribute this software.
# 
# NOTICE.  This Software was developed under funding from the U.S. Department of
# Energy and the U.S. Government consequently retains certain rights. As such,
# the U.S. Government has been granted for itself and others acting on its
# behalf a paid-up, nonexclusive, irrevocable, worldwide license in the Software
# to reproduce, distribute copies to the public, prepare derivative works, and
# perform publicly and display publicly, and to permit other to do so. 
# 
# 


message 2 "Evaluating args: '$*'\n"

while true; do
    case ${1:-} in
        -h|--help|help)
            if [ -e "$SINGULARITY_libexecdir/singularity/cli/$SINGULARITY_COMMAND.help" ]; then
                cat "$SINGULARITY_libexecdir/singularity/cli/$SINGULARITY_COMMAND.help"
            else
                message ERROR "No help exists for this command\n"
                exit 1
            fi
            exit
        ;;
        -l|--labels)
            SINGULARITY_INSPECT_SCRIPT="/.singularity.d/labels.json"
            export SINGULARITY_INSPECT_SCRIPT
            shift
        ;;
        -s|--shell)
            shift
            SINGULARITY_SHELL="${1:-}"
            export SINGULARITY_SHELL
            shift
        ;;
        -u|--user)
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
        --pwd)
            shift
            SINGULARITY_TARGET_PWD="$1"
            export SINGULARITY_TARGET_PWD
            shift
        ;;
        -n|--nv)
            shift
            for i in `ldconfig -p | grep -E "/libnv|/libcuda|/libEGL|/libGL|/libnvcu|/libvdpau|/libOpenCL|/libOpenGL"`; do
                if [ -f "$i" ]; then
                    message 2 "Found NV library: $i\n"
                    if [ -z "${SINGULARITY_CONTAINLIBS:-}" ]; then
                        SINGULARITY_CONTAINLIBS="$i"
                    else
                        SINGULARITY_CONTAINLIBS="$SINGULARITY_CONTAINLIBS,$i"
                    fi
                fi
            done
            if [ -z "${SINGULARITY_CONTAINLIBS:-}" ]; then
                message WARN "Could not find any Nvidia libraries on this host!\n";
            else
                export SINGULARITY_CONTAINLIBS
            fi
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
