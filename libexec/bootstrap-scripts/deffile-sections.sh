#!/bin/bash
#
# Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
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
if [ -f "$SINGULARITY_libexecdir/singularity/bootstrap-scripts/functions" ]; then
    . "$SINGULARITY_libexecdir/singularity/bootstrap-scripts/functions"
else
    echo "Error loading functions: $SINGULARITY_libexecdir/singularity/bootstrap-scripts/functions"
    exit 1
fi

if [ -z "${SINGULARITY_ROOTFS:-}" ]; then
    message ERROR "Singularity root file system not defined\n"
    exit 1
fi

if [ -z "${SINGULARITY_BUILDDEF:-}" ]; then
    message ERROR "Singularity bootstrap definition file not defined!\n"
    exit 1
fi

if [ ! -f "${SINGULARITY_BUILDDEF:-}" ]; then
    message ERROR "Singularity bootstrap definition file not found!\n"
    exit 1
fi


umask 0002

# First priority goes to runscript defined in build file
runscript_command=$(singularity_section_get "runscript" "$SINGULARITY_BUILDDEF")

# If the command is not empty, write to file.
if [ ! -z "$runscript_command" ]; then
    echo "User defined %runscript found! Taking priority."
    echo "$runscript_command" > "$SINGULARITY_ROOTFS/singularity"    
fi

test -d "$SINGULARITY_ROOTFS/proc" || install -d -m 755 "$SINGULARITY_ROOTFS/proc"
test -d "$SINGULARITY_ROOTFS/sys" || install -d -m 755 "$SINGULARITY_ROOTFS/sys"
test -d "$SINGULARITY_ROOTFS/tmp" || install -d -m 755 "$SINGULARITY_ROOTFS/tmp"
test -d "$SINGULARITY_ROOTFS/dev" || install -d -m 755 "$SINGULARITY_ROOTFS/dev"

mount --no-mtab -t proc proc "$SINGULARITY_ROOTFS/proc"
mount --no-mtab -t sysfs sysfs "$SINGULARITY_ROOTFS/sys"
mount --no-mtab --rbind "/tmp" "$SINGULARITY_ROOTFS/tmp"
mount --no-mtab --rbind "/dev" "$SINGULARITY_ROOTFS/dev"

cp /etc/hosts           "$SINGULARITY_ROOTFS/etc/hosts"
cp /etc/resolv.conf     "$SINGULARITY_ROOTFS/etc/resolv.conf"

### EXPORT ENVARS
DEBIAN_FRONTEND=noninteractive
SINGULARITY_ENVIRONMENT="/.singularity.d/env/91-environment.sh"
export DEBIAN_FRONTEND SINGULARITY_ENVIRONMENT

# Script helper paths
ADD_LABEL=$SINGULARITY_libexecdir/singularity/python/helpers/json/add.py

##########################################################################################
#
# MAIN SECTIONS
#
##########################################################################################


### SETUP
if [ -z "${SINGULARITY_BUILDSECTION:-}" -o "${SINGULARITY_BUILDSECTION:-}" == "setup" ]; then
    if singularity_section_exists "setup" "$SINGULARITY_BUILDDEF"; then
        ARGS=`singularity_section_args "setup" "$SINGULARITY_BUILDDEF"`
        singularity_section_get "setup" "$SINGULARITY_BUILDDEF" | /bin/sh -e -x $ARGS || ABORT 255
    fi

    if [ ! -x "$SINGULARITY_ROOTFS/bin/sh" -a ! -L "$SINGULARITY_ROOTFS/bin/sh" ]; then
        message ERROR "Could not locate /bin/sh inside the container\n"
        exit 255
    fi
else
    message 2 "Skipping setup section\n"
fi


### FILES
if [ -z "${SINGULARITY_BUILDSECTION:-}" -o "${SINGULARITY_BUILDSECTION:-}" == "files" ]; then
    if singularity_section_exists "files" "$SINGULARITY_BUILDDEF"; then
        message 1 "Adding files to container\n"

        singularity_section_get "files" "$SINGULARITY_BUILDDEF" | sed -e 's/#.*//' | while read origin dest; do
            if [ -n "${origin:-}" ]; then
                if [ -z "${dest:-}" ]; then
                    dest="$origin"
                fi
                message 1 "Copying '$origin' to '$dest'\n"
                if ! /bin/cp -fLr $origin "$SINGULARITY_ROOTFS/$dest"; then
                    message ERROR "Failed copying file(s) into container\n"
                    exit 255
                fi
            fi
        done
    fi
else
    message 2 "Skipping files section\n"
fi


### ENVIRONMENT
if [ -z "${SINGULARITY_BUILDSECTION:-}" -o "${SINGULARITY_BUILDSECTION:-}" == "environment" ]; then
    if singularity_section_exists "environment" "$SINGULARITY_BUILDDEF"; then
        message 1 "Adding environment to container\n"

        singularity_section_get "environment" "$SINGULARITY_BUILDDEF" >> "$SINGULARITY_ROOTFS/.singularity.d/env/90-environment.sh"
    fi
else
    message 2 "Skipping environment section\n"
fi


### RUN POST
if [ -z "${SINGULARITY_BUILDSECTION:-}" -o "${SINGULARITY_BUILDSECTION:-}" == "post" ]; then
    if singularity_section_exists "post" "$SINGULARITY_BUILDDEF"; then
        message 1 "Running post scriptlet\n"
        
        ARGS=`singularity_section_args "post" "$SINGULARITY_BUILDDEF"`
        singularity_section_get "post" "$SINGULARITY_BUILDDEF" | chroot "$SINGULARITY_ROOTFS" /bin/sh -e -x $ARGS || ABORT 255
    fi
else
    message 2 "Skipping post section\n"
fi


### LABELS
if [ -z "${SINGULARITY_BUILDSECTION:-}" -o "${SINGULARITY_BUILDSECTION:-}" == "labels" ]; then
    if singularity_section_exists "labels" "$SINGULARITY_BUILDDEF"; then
        message 1 "Adding deffile section labels to container\n"

        singularity_section_get "labels" "$SINGULARITY_BUILDDEF" | while read KEY VAL; do
            if [ -n "$KEY" -a -n "$VAL" ]; then
                if [ "${SINGULARITY_DEFFILE_BOOTSTRAP:-}" = "shub" -o "${SINGULARITY_DEFFILE_BOOTSTRAP:-}" = "localimage" ]; then
                    $ADD_LABEL --key "$KEY" --value "$VAL" --file "$SINGULARITY_ROOTFS/.singularity.d/labels.json" -f
                else
                    $ADD_LABEL --key "$KEY" --value "$VAL" --file "$SINGULARITY_ROOTFS/.singularity.d/labels.json"
                
                fi
            fi
        done
    fi
else
    message 2 "Skipping labels section\n"
fi

### RUNSCRIPT
if [ -z "${SINGULARITY_BUILDSECTION:-}" -o "${SINGULARITY_BUILDSECTION:-}" == "startscript" ]; then
    if singularity_section_exists "startscript" "$SINGULARITY_BUILDDEF"; then
        message 1 "Adding startscript\n"

        echo -n "#!/bin/sh " > "$SINGULARITY_ROOTFS/.singularity.d/startscript"
        singularity_section_args "startscript" "$SINGULARITY_BUILDDEF" >> "$SINGULARITY_ROOTFS/.singularity.d/startscript"
        echo "" >> "$SINGULARITY_ROOTFS/.singularity.d/startscript"
        singularity_section_get "startscript" "$SINGULARITY_BUILDDEF" >> "$SINGULARITY_ROOTFS/.singularity.d/startscript"

    fi
else
    message 2 "Skipping startscript section\n"
fi


### RUNSCRIPT
if [ -z "${SINGULARITY_BUILDSECTION:-}" -o "${SINGULARITY_BUILDSECTION:-}" == "runscript" ]; then
    if singularity_section_exists "runscript" "$SINGULARITY_BUILDDEF"; then
        message 1 "Adding runscript\n"

        echo -n "#!/bin/sh " > "$SINGULARITY_ROOTFS/.singularity.d/runscript"
        singularity_section_args "runscript" "$SINGULARITY_BUILDDEF" >> "$SINGULARITY_ROOTFS/.singularity.d/runscript"
        echo "" >> "$SINGULARITY_ROOTFS/.singularity.d/runscript"
        singularity_section_get "runscript" "$SINGULARITY_BUILDDEF" >> "$SINGULARITY_ROOTFS/.singularity.d/runscript"

    fi
else
    message 2 "Skipping runscript section\n"

fi


### HELP FOR RUNSCRIPT
if [ -z "${SINGULARITY_BUILDSECTION:-}" -o "${SINGULARITY_BUILDSECTION:-}" == "help" ]; then
    if singularity_section_exists "help" "$SINGULARITY_BUILDDEF"; then
        message 1 "Adding runscript help\n"
        singularity_section_args "help" "$SINGULARITY_BUILDDEF" > "$SINGULARITY_ROOTFS/.singularity.d/runscript.help"
        echo "" >> "$SINGULARITY_ROOTFS/.singularity.d/runscript.help"
        singularity_section_get "help" "$SINGULARITY_BUILDDEF" >> "$SINGULARITY_ROOTFS/.singularity.d/runscript.help"
        # Add label for the file
        HELPLABEL="org.label-schema.usage.singularity.runscript.help"
        HELPFILE="/.singularity.d/runscript.help"
        $SINGULARITY_libexecdir/singularity/python/helpers/json/add.py --key "$HELPLABEL" --value "$HELPFILE" --file "$SINGULARITY_ROOTFS/.singularity.d/labels.json"
    fi
else
    message 2 "Skipping runscript help section\n"
fi


### RUN TEST
if [ -z "${SINGULARITY_BUILDSECTION:-}" -o "${SINGULARITY_BUILDSECTION:-}" == "test" ]; then
    if [ -z "${SINGULARITY_NOTEST:-}" ]; then
        if singularity_section_exists "test" "$SINGULARITY_BUILDDEF"; then
            message 1 "Running test scriptlet\n"

            ARGS=`singularity_section_args "test" "$SINGULARITY_BUILDDEF"`
            echo "#!/bin/sh" > "$SINGULARITY_ROOTFS/.singularity.d/test"
            echo "" >> "$SINGULARITY_ROOTFS/.singularity.d/test"
            singularity_section_get "test" "$SINGULARITY_BUILDDEF" >> "$SINGULARITY_ROOTFS/.singularity.d/test"

            chmod 0755 "$SINGULARITY_ROOTFS/.singularity.d/test"

            chroot "$SINGULARITY_ROOTFS" /bin/sh -e -x $ARGS "/.singularity.d/test" "$@" || ABORT 255
        fi
    fi
else
    message 2 "Skipping test section\n"
fi


##########################################################################################
#
# APP SECTIONS
#
##########################################################################################

### APPFILES
if [ -z "${SINGULARITY_BUILDSECTION:-}" -o "${SINGULARITY_BUILDSECTION:-}" == "appfiles" ]; then
    if singularity_section_exists "appfiles" "$SINGULARITY_BUILDDEF"; then
        APPNAMES=(`singularity_section_args "appfiles" "$SINGULARITY_BUILDDEF"`)

        for APPNAME in "${APPNAMES[@]}"; do
            message 1 "Adding files to ${APPNAME}\n"
            singularity_app_init "${APPNAME}" "${SINGULARITY_ROOTFS}"
            get_section "appfiles ${APPNAME}" "$SINGULARITY_BUILDDEF" | sed -e 's/#.*//' | while read origin dest; do
                if [ -n "${origin:-}" ]; then
                    if [ -z "${dest:-}" ]; then
                        # files must be relative to app
                        dest="scif/apps/${APPNAME}"
                    else
                        dest="scif/apps/${APPNAME}/$dest"
                    fi
                    message 1 "+ $origin to $dest\n"
                    if ! /bin/cp -fLr $origin "$SINGULARITY_ROOTFS/$dest"; then
                        message ERROR "Failed copying file(s) for app ${APPNAME} into container\n"
                        exit 255
                    fi
                fi
            done
        done
    fi
fi


### APPHELP
if [ -z "${SINGULARITY_BUILDSECTION:-}" -o "${SINGULARITY_BUILDSECTION:-}" == "apphelp" ]; then
    if singularity_section_exists "apphelp" "$SINGULARITY_BUILDDEF"; then
        APPNAMES=(`singularity_section_args "apphelp" "$SINGULARITY_BUILDDEF"`)

        for APPNAME in "${APPNAMES[@]}"; do
            message 1 "${APPNAME} has help section\n"
            singularity_app_init "${APPNAME}" "${SINGULARITY_ROOTFS}"
            APPHELP=$(get_section "apphelp ${APPNAME}" "$SINGULARITY_BUILDDEF")

            if [ ! -z "$APPHELP" ]; then
                echo "$APPHELP" > "$SINGULARITY_ROOTFS/scif/apps/${APPNAME}/scif/runscript.help"    
            fi
        done
    fi
fi


### APPRUNSCRIPT
if [ -z "${SINGULARITY_BUILDSECTION:-}" -o "${SINGULARITY_BUILDSECTION:-}" == "apprun" ]; then
    if singularity_section_exists "apprun" "$SINGULARITY_BUILDDEF"; then
        APPNAMES=(`singularity_section_args "apprun" "$SINGULARITY_BUILDDEF"`)
        
        for APPNAME in "${APPNAMES[@]}"; do
            message 1 "${APPNAME} has runscript definition\n"
            singularity_app_init "${APPNAME}" "${SINGULARITY_ROOTFS}"
            APPRUN=$(get_section "apprun ${APPNAME}" "$SINGULARITY_BUILDDEF")

            if [ ! -z "$APPRUN" ]; then
                echo "$APPRUN" > "$SINGULARITY_ROOTFS/scif/apps/${APPNAME}/scif/runscript"
                chmod 0755 "$SINGULARITY_ROOTFS/scif/apps/${APPNAME}/scif/runscript"  
            fi

            # Make sure we have metadata
            APPBASE="$SINGULARITY_ROOTFS/scif/apps/${APPNAME}"
            APPFOLDER_SIZE=$(singularity_calculate_size "${APPBASE}")
            $ADD_LABEL --key "SINGULARITY_APP_SIZE" --value "${APPFOLDER_SIZE}MB" --file "$APPBASE/scif/labels.json"
            $ADD_LABEL --key "SINGULARITY_APP_NAME" --value "${APPNAME}" --file "${APPBASE}/scif/labels.json"

        done
    fi
fi


### APPENVIRONMENT
if [ -z "${SINGULARITY_BUILDSECTION:-}" -o "${SINGULARITY_BUILDSECTION:-}" == "appenv" ]; then
    if singularity_section_exists "appenv" "$SINGULARITY_BUILDDEF"; then
        APPNAMES=(`singularity_section_args "appenv" "$SINGULARITY_BUILDDEF"`)

        for APPNAME in "${APPNAMES[@]}"; do
            message 1 "Adding custom environment to ${APPNAME}\n"
            singularity_app_init "${APPNAME}" "${SINGULARITY_ROOTFS}"
            get_section "appenv ${APPNAME}" "$SINGULARITY_BUILDDEF" >> "$SINGULARITY_ROOTFS/scif/apps/${APPNAME}/scif/env/90-environment.sh"
        done
    fi
fi


### APPLABELS
if [ -z "${SINGULARITY_BUILDSECTION:-}" -o "${SINGULARITY_BUILDSECTION:-}" == "appfiles" ]; then
    if singularity_section_exists "applabels" "$SINGULARITY_BUILDDEF"; then
        APPNAMES=(`singularity_section_args "applabels" "$SINGULARITY_BUILDDEF"`)

        for APPNAME in "${APPNAMES[@]}"; do
            message 1 "Adding labels to ${APPNAME}\n"
            singularity_app_init "${APPNAME}" "${SINGULARITY_ROOTFS}"
            get_section "applabels ${APPNAME}" "$SINGULARITY_BUILDDEF" | while read KEY VAL; do
                if [ -n "$KEY" -a -n "$VAL" ]; then
                    $ADD_LABEL --key "$KEY" --value "$VAL" --file "$SINGULARITY_ROOTFS/scif/apps/${APPNAME}/scif/labels.json"
                fi
            done
        done
     fi
fi

### APPINSTALL
if [ -z "${SINGULARITY_BUILDSECTION:-}" -o "${SINGULARITY_BUILDSECTION:-}" == "appinstall" ]; then
    if singularity_section_exists "appinstall" "$SINGULARITY_BUILDDEF"; then
        APPNAMES=(`singularity_section_args "appinstall" "$SINGULARITY_BUILDDEF"`)
        
        for APPNAME in "${APPNAMES[@]}"; do
            message 1 "Installing ${APPNAME}\n"
            APPBASE="$SINGULARITY_ROOTFS/scif/apps/${APPNAME}"
            SINGULARITY_APPROOT="/scif/apps/${APPNAME}"
            export SINGULARITY_APPROOT
            singularity_app_init "${APPNAME}" "${SINGULARITY_ROOTFS}"
            singularity_app_save "${APPNAME}" "$SINGULARITY_BUILDDEF" "${APPBASE}/scif/Singularity"
            singularity_app_install_get "${APPNAME}" "$SINGULARITY_BUILDDEF" | chroot "$SINGULARITY_ROOTFS" /bin/sh -xe || ABORT 255

            APPFOLDER_SIZE=$(singularity_calculate_size "${APPBASE}")
            $ADD_LABEL --key "SINGULARITY_APP_SIZE" --value "${APPFOLDER_SIZE}MB" --file "$APPBASE/scif/labels.json" --quiet -f
            $ADD_LABEL --key "SINGULARITY_APP_NAME" --value "${APPNAME}" --file "${APPBASE}/scif/labels.json" --quiet -f

        done
    fi
else
    message 2 "No applications detected for install\n"
fi

## APPGLOBAL

APPGLOBAL="${SINGULARITY_ROOTFS}/.singularity.d/env/94-appsbase.sh"

for app in ${SINGULARITY_ROOTFS}/scif/apps/*; do
    if [ -d "$app" ]; then

        app="${app##*/}"
        app=(`echo $app | sed -e "s/-/_/g"`)
        appbase="${SINGULARITY_ROOTFS}/scif/apps/$app"
        appmeta="${appbase}/scif"

        # Export data, root, metadata, labels, environment
        echo "APPDATA_$app=/scif/data/$app" >> "${APPGLOBAL}"
        echo "APPMETA_$app=/scif/apps/$app/scif" >> "${APPGLOBAL}"
        echo "APPROOT_$app=/scif/apps/$app" >> "${APPGLOBAL}"
        echo "APPBIN_$app=/scif/apps/$app/bin" >> "${APPGLOBAL}"
        echo "APPLIB_$app=/scif/apps/$app/lib" >> "${APPGLOBAL}"
        echo "export APPDATA_$app APPROOT_$app APPMETA_$app APPBIN_$app APPLIB_$app"  >> "${APPGLOBAL}"

        # Environment
        if [ -e "${appmeta}/env/90-environment.sh" ]; then
            echo  "APPENV_${app}=/scif/apps/$app/scif/env/90-environment.sh" >> "${APPGLOBAL}"
            echo  "export APPENV_${app}" >> "${APPGLOBAL}"
        fi

        # Labels
        if [ -e "${appmeta}/labels.json" ]; then
            echo  "APPLABELS_${app}=/scif/apps/$app/scif/labels.json" >> "${APPGLOBAL}"
            echo  "export APPLABELS_${app}" >> "${APPGLOBAL}"
        fi

        # Runscript
        if [ -e "${appmeta}/runscript" ]; then
            echo  "APPRUN_${app}=/scif/apps/$app/scif/runscript" >> "${APPGLOBAL}"
            echo  "export APPRUN_${app}" >> "${APPGLOBAL}"
        fi
    fi
done


##########################################################################################
#
# Finalizing
#
##########################################################################################


> "$SINGULARITY_ROOTFS/etc/hosts"
> "$SINGULARITY_ROOTFS/etc/resolv.conf"


# If we have a runscript, whether docker, user defined, change permissions
if [ -s "$SINGULARITY_ROOTFS/.singularity.d/runscript" ]; then
    chmod 0755 "$SINGULARITY_ROOTFS/.singularity.d/runscript"
fi

# Copy the definition file into the container.  If one already exists, archive.
if [ -f "$SINGULARITY_ROOTFS/.singularity.d/Singularity" ]; then
    message 1 "Found an existing definition file\n"
    message 1 "Adding a bootstrap_history directory\n"
    mkdir -p "$SINGULARITY_ROOTFS/.singularity.d/bootstrap_history"
    count=0
    while true; do 
        if [ ! -f "$SINGULARITY_ROOTFS/.singularity.d/bootstrap_history/Singularity${count}" ]; then
            mv "$SINGULARITY_ROOTFS/.singularity.d/Singularity" "$SINGULARITY_ROOTFS/.singularity.d/bootstrap_history/Singularity${count}"
            break
        fi
        count=`expr $count + 1`
    done
fi
install -m 644 "$SINGULARITY_BUILDDEF" "$SINGULARITY_ROOTFS/.singularity.d/Singularity"
