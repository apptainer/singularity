#!/bin/bash
# 
# Copyright (c) 2015-2016, Gregory M. Kurtzer. All rights reserved.
# 
# “Singularity” Copyright (c) 2016, The Regents of the University of California,
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

## Basic sanity
if [ -z "$SINGULARITY_libexecdir" ]; then
    echo "Could not identify the Singularity libexecdir."
    exit 1
fi

# Ensure the user has provided a docker image name with "From"
if [ -z "$SINGULARITY_DOCKER_IMAGE" ]; then
    echo "Please specify the Docker image name with From: in the definition file."
    exit 1
fi

## Load functions
if [ -f "$SINGULARITY_libexecdir/singularity/functions" ]; then
    . "$SINGULARITY_libexecdir/singularity/functions"
else
    echo "Error loading functions: $SINGULARITY_libexecdir/singularity/functions"
    exit 1
fi

if [ -z "${SINGULARITY_ROOTFS:-}" ]; then
    message ERROR "Singularity root file system not defined\n"
    exit 1
fi

if [ -z "${SINGULARITY_BUILDDEF:-}" ]; then
    message ERROR "Singularity build definition file not defined\n"
    exit 1
fi


########## BEGIN BOOTSTRAP SCRIPT ##########


: ' ADMIN IMAGE STUFFS -------------------------------------------
 Here we create the admin account, and define hosts
'

mkdir -p -m 0755 "$SINGULARITY_ROOTFS/bin"
mkdir -p -m 0755 "$SINGULARITY_ROOTFS/etc"

echo "root:!:0:0:root:/root:/bin/sh" > "$SINGULARITY_ROOTFS/etc/passwd"
echo " root:x:0:" > "$SINGULARITY_ROOTFS/etc/group"
echo "127.0.0.1   localhost localhost.localdomain localhost4 localhost4.localdomain4" > "$SINGULARITY_ROOTFS/etc/hosts"




: ' INPUT PARSING -------------------------------------------
 Parse image name, repo name, and namespace
'

# First split the docker image name by /
IFS='/' read -ra DOCKER_ADDR <<< "$SINGULARITY_DOCKER_IMAGE"

# If there are two parts, we have namespace with repo (and maybe tab)
if [ ${#DOCKER_ADDR[@]} -eq 2 ]; then
    namespace=${DOCKER_ADDR[0]}
    SINGULARITY_DOCKER_IMAGE=${DOCKER_ADDR[1]}

# Otherwise, we must be using library namespace
else
    namespace="library"
fi

# Now split the docker image name by :
IFS=':' read -ra DOCKER_ADDR <<< "$SINGULARITY_DOCKER_IMAGE"

if [ ${#DOCKER_ADDR[@]} -eq 2 ]; then
    repo_name=${DOCKER_ADDR[0]}
    repo_tag=${DOCKER_ADDR[1]}

# Otherwise, assume latest of an image
else
    repo_name=${DOCKER_ADDR[0]}
    repo_tag="latest"
fi

: ' AUTHORIZATION -------------------------------------------
 To get the image layers, we need a valid token to read the repo
'

# This is a version 1.0 registry auth token, version (2.0), which isn't currently working/finished, is below
token=$(curl -si https://registry.hub.docker.com/v1/repositories/$namespace/$repo_name/images -H 'X-Docker-Token: true' | grep X-Docker-Token)
token=$(echo ${token/X-Docker-Token:/})
token=$(echo Authorization\: Token $token)

#token=$(curl -si https://auth.docker.io/token?service=registry.docker.io&scope=repository:$repo_name/$repo_tag:read)
#token=`echo ${token/{\"token\":\"/}` # leaves a space at beginning
#token=`echo ${token/\"\}/}`
#token=`echo ${token/ \}/}`
#token=`echo 'Authorization: Token signature=' $token`


: ' IMAGE METADATA -------------------------------------------
 Use Docker Registry API (version 1.0) to get manifest
'

# Was the image manifest found?
manifest=$(curl -k https://registry.hub.docker.com/v1/repositories/$namespace/$repo_name/tags/$repo_tag)
if [ "$manifest" = "Tag not found" ]; then
    message ERROR "Image manifest for $namespace/$repo_name:$repo_tag not found using Docker Registry.\n"
    exit 1
fi

# Find images
repo_images=$(curl -si https://registry.hub.docker.com/v1/repositories/$namespace/$repo_name/images)

: ' DOWNLOAD LAYERS -------------------------------------------
 Each is a .tar.gz file, obtained from registry with curl
'

# For each image id, if it matches, then get the layer (call above)
echo $manifest | grep -Po '"id": "(.*?)"' | while read a; do

    # remove "id": and extra "'s
    image_id=`echo ${a/\"id\":/}`
    image_id=`echo ${image_id//\"/}`
    # Find the full image id for each tag, meaning everything up to the quote
    image_id=$(echo $repo_images | grep -o -P $image_id'.+?(?=\")')
    
    # If the image_id isn't empty, get the layer
    if [ ! -z $image_id ]; then

        # Obtain json (detailed manifest) about image
        url=$(echo https://cdn-registry-1.docker.io/v1/images/$image_id/json -H \'$token\')
        url=$(echo "$url"| tr -d '\r')  # get rid of ^M, eww
 
        # This needs to be fixed to get curl url from variable - having trouble with quotes
        echo $url > $image_id"_meta.url"
   
        # Pass the file into curl to get the result
        image_meta=$(cat $image_id"_meta.url" | xargs curl)

        # Now obtain image layer
        url=$(echo https://cdn-registry-1.docker.io/v1/images/$image_id/layer -H \'$token\')
        url=$(echo "$url"| tr -d '\r')
        echo $url > $image_id"_layer.url"
        echo "Downloading $image_id.tar.gz...\n"
        cat $image_id"_layer.url" | xargs curl -L >> $image_id.tar.gz # we will likely be redirected

        # clean up temporary files
        rm $image_id"_meta.url"
        rm $image_id"_layer.url"

        # Extract image
        echo "Extracting $image_id.tar.gz...\n"
        tar -xzf $image_id.tar.gz -C $SINGULARITY_ROOTFS
        rm $image_id.tar.gz

    fi

done


# Question - do something here (or in above loop) for permissions of extractions?

chmod 0755 -R "$SINGULARITY_ROOTFS/"

#TODO: save/do something with meta data from image? eg:
# {"container": "e6e4c4801f676bad4f31b86279b0766d00ed1030db6bdf4e92230c77e32e8cba", "parent": "51a9c7c1f8bb2fa19bcd09789a34e63f35abb80044bc10196e304f6634cc582c", "created": "2015-01-28T18:37:18.255519085Z", "config": {"Env": ["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"], "Hostname": "f4f502dce15c", "Entrypoint": null, "PortSpecs": null, "OnBuild": [], "OpenStdin": false, "MacAddress": "", "User": "", "VolumeDriver": "", "AttachStderr": false, "AttachStdout": false, "NetworkDisabled": false, "WorkingDir": "", "Cmd": ["/bin/bash"], "StdinOnce": false, "AttachStdin": false, "Volumes": null, "Tty": false, "Domainname": "", "Image": "51a9c7c1f8bb2fa19bcd09789a34e63f35abb80044bc10196e304f6634cc582c", "Labels": null, "ExposedPorts": null}, "container_config": {"Env": ["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"], "Hostname": "f4f502dce15c", "Entrypoint": null, "PortSpecs": null, "OnBuild": [], "OpenStdin": false, "MacAddress": "", "User": "", "VolumeDriver": "", "AttachStderr": false, "AttachStdout": false, "NetworkDisabled": false, "WorkingDir": "", "Cmd": ["/bin/sh", "-c", "#(nop) CMD [/bin/bash]"], "StdinOnce": false, "AttachStdin": false, "Volumes": null, "Tty": false, "Domainname": "", "Image": "51a9c7c1f8bb2fa19bcd09789a34e63f35abb80044bc10196e304f6634cc582c", "Labels": null, "ExposedPorts": null}, "architecture": "amd64", "docker_version": "1.4.1", "os": "linux", "id": "5ba9dab47459d81c0037ca3836a368a4f8ce5050505ce89720e1fb8839ea048a", "Size": 0}

# Likely we would want to use Cmd for runscript UNLESS the user has defined one. If I were bootstrapping a Docker image I would want (and
# expect) this to carry through.

# If we got here, exit...
exit 0
