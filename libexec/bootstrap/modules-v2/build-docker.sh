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
token=`echo ${token/X-Docker-Token:/}`
token=`echo 'Authorization: Token' $token`

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
repo_images=$(curl -si https://registry.hub.docker.com/v1/repositories/$namespace/$repo_name/$repo_tag/images)


### CODE NOT WRITTEN/FINISHED BELOW! going running be back later :)

# For each image id, if it matches, then get the layer (call above)
echo $manifest | grep -Po '"id": "(.*?)"' | while read a; do 
    # remove "id": and extra "'s
    image_id=`echo ${a/\"id\":/}`
    image_id=`echo ${image_id//\"/}`
    # Find the full image id for each tag, meaning everything up to the quote
    image_id=$(echo $repo_images | grep -o -P $image_id'.+?(?=\")')
    # If the image_id isn't empty, get the layer
    if [ -z "$image_id" ]; then

# THIS IS THE CALL THAT WORKS TO GET JSON - for complete image id that matches one in manifest above!
https://cdn-registry-1.docker.io/v1/images/a343823119db57543086463ae7da8aaadbcef25781c0c4d121397a2550a419a6/json -H 'Authorization: Token signature=f2488c0a82c984ea2ac04b86863af100e32cd025,repository="library/ubuntu",access=read'

# change to /layer to get image layer!

    fi
#url=$(echo https://cdn-registry-1.docker.io/v1/images/a343823119db57543086463ae7da8aaadbcef25781c0c4d121397a2550a419a6/json -H \'$token\')

#    url=$(echo https://registry-1.docker.io/v1/images/$image_id/json -H \'$token\')
#    curl -k $url
    #curl -k https://registry.hub.docker.com/v1/images/$image_id/layer

done



# Find image manifest
manifest=$(curl -k https://registry-1.docker.io/v2/$namespace/$repo_name)

/tags/$repo_tag


curl -k https://registry.hub.docker.com/v1/repositories/$repo_name/auth

><> 

511136ea3c5a64f264b78b5433614aec563103b4d4702f3ba7d4d2698e22c158/json -H 

# NOT USING BELOW THIS LINE... yet :)
# First obtain the list of image tags
#image_tags=$(curl -k https://registry.hub.docker.com/v1/repositories/$repo_name/tags)


# This will only match a tag directly, eg, 14.04.1 must be given and not 14.04
found_tag=$(echo $image_tags | grep -Po '"name": "(.*?)"' | while read a; do 

    # remove "name": and extra "'s
    contender_tag=`echo ${a/\"name\":/}`
    contender_tag=`echo ${contender_tag//\"/}`

    # Does the tag equal our specified repo tag?
    if [ $contender_tag == $repo_tag ]; then
       echo $contender_tag
    fi
done)

# Did we find a tag?
if [ -z "$found_tag" ]; then
    message ERROR "Docker tag $repo_name:$repo_tag not found with Docker Registry API v.1.0\n"
    exit 1
fi



# STOPPED HERE... work in progress


# Obtain the image manifest
/v2/<name>/manifests/<reference>
manifest=$(curl -k https://registry.hub.docker.com/v1/library/ubuntu:latest/manifests)

curl https://cdn-registry-1.docker.io/v1/images/511136ea3c5a64f264b78b5433614aec563103b4d4702f3ba7d4d2698e22c158/json -H $token

'Authorization: Token signature=01b8e3d3ef56515b33d9f68824134e3460de3a1a,repository="library/ubuntu",access=read'

{"id":"511136ea3c5a64f264b78b5433614aec563103b4d4702f3ba7d4d2698e22c158","comment":"Imported from -","created":"2013-06-13T14:03:50.821769-07:00","container_config":{"Hostname":"","User":"","Memory":0,"MemorySwap":0,"CpuShares":0,"AttachStdin":false,"AttachStdout":false,"AttachStderr":false,"PortSpecs":null,"Tty":false,"OpenStdin":false,"StdinOnce":false,"Env":null,"Cmd":null,"Dns":null,"Image":"","Volumes":null,"VolumesFrom":""},"docker_version":"0.4.0","architecture":"x86_64"}

List library repository images
GET /v1/repositories/(repo_name)/images

Get the images for a library repo.

Example Request:

    GET /v1/repositories/foobar/images HTTP/1.1
    Host: index.docker.io
    Accept: application/json
Parameters:

repo_name – the library name for the repo



# Get token for the Hub API

MIRROR=`singularity_key_get "MirrorURL" "$SINGULARITY_BUILDDEF"`
if [ -z "${MIRROR:-}" ]; then
    MIRROR="https://www.busybox.net/downloads/binaries/busybox-x86_64"
fi


mkdir -p -m 0755 "$SINGULARITY_ROOTFS/bin"
mkdir -p -m 0755 "$SINGULARITY_ROOTFS/etc"

echo "root:!:0:0:root:/root:/bin/sh" > "$SINGULARITY_ROOTFS/etc/passwd"
echo " root:x:0:" > "$SINGULARITY_ROOTFS/etc/group"
echo "127.0.0.1   localhost localhost.localdomain localhost4 localhost4.localdomain4" > "$SINGULARITY_ROOTFS/etc/hosts"

curl "$MIRROR" > "$SINGULARITY_ROOTFS/bin/busybox"

chmod 0755 "$SINGULARITY_ROOTFS/bin/busybox"

eval "$SINGULARITY_ROOTFS/bin/busybox" --install "$SINGULARITY_ROOTFS/bin/"


# If we got here, exit...
exit 0
