#!/bin/bash

# Converting docker images to singularity images.

# Copyright 2016  Dave Love, University of Liverpool
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

# The basic technique is:
#   * Create a docker container from the image
#   * Figure out the singularity container size from that
#   * Create the singularity container
#   * docker export | singularity import
#   * Clean up
# It's partly obscured by error checking and recovery in verbose
# if blocks in Singularity style, and determining the size and possible
# /singularity contents is complex.

# Fixme: Check docker "Os" is "linux" [can it be anything else?]
#          Are there any guarantees of contents, like coreutils?
#        Maybe accept docker container instead of image?
#        Options to set extra, tarfactor in import.exec (messy because
#          they're probably specific to the source)
#        Convention for singularity container metadata that could be
#          extracted from docker in this case
#        Can it be sped up somehow?  This takes ~1 min with the minimal
#          fedora image and ~75s with one of ~1GB.
#        Treatment of entrypoint/cmd likely still doesn't match docker
#          properly, in particular "shell" v. "exec"
#        Consider the locale, e.g. for use of =~

docker_cleanup() {
    if [[ -n $id ]]; then
        message 1 "Cleaning up Docker container...\n"
        sudo docker rm -f $id >/dev/null
    fi
    rm -f "$tmp"
}

# If we've started a container, we want to remove it on exit.
trap docker_cleanup 0

if [[ -z ${FILE:-} ]]; then
    message ERROR "No Docker image specified (with --file)\n"
    exit 1
fi
dock=$FILE
sing=${1:-}

if [[ -f $sing ]]; then
    message ERROR "$sing exists -- not over-written\n"
    exit 1
fi

if ! [[ null = $(docker inspect --format='{{json .State}}' "$dock") ]]; then
    message ERROR "Docker image required, not container\n"
    exit 1
fi

# We need to have the default entrypoint to run df, and we'd have to
# generate /singularity differently with a non-default one.
if ! entry=$(docker inspect --format='{{json .Config.Entrypoint}}' "$dock"); then
    # Assume any error will give an obvious "not found" message
    exit 1
fi

# Create a container from the image and stash its id.  There's
# probably no advantage to generating a name for the container and
# using that.
# Give it a command to provide the root size when we start it.  Assume
# we can run df.  (There doesn't seem to be any useful information
# available with inspect on either the docker image or container;
# VirtualSize is the sum of all the layers and Size may be zero,
# however it's defined.)  If that won't work we have to fall back to
# measuring the export stream before exporting it again for real and
# guessing on the basis of that.  Note that the filesystem is
# typically significantly bigger (~20% in cases I've looked at,
# surprisingly) than the tar stream.  Also the two containers may use
# different filesystem types, which will affect their relative sizes.

message 1 "Patience...\nCreating Docker container...\n"
if ! id=$(docker create "$dock" df -k -P /); then
    message ERROR "docker create failed\n"
    exit 1
fi

# Fixme: These should come from import.exec
extra=${extra:-50}
tarfactor=${tarfactor:-1.5}

# Estimate the size of the running root.

# Add an arbitrary $extra headroom for now in case of alterations or
# as a possible fiddle factor for the filesystem, and hope $tarfactor
# is a big enough factor by which to expand the tar stream for
# singularity, else try again.  I.e. the initial attempt for the
# singularity size is estimate*tarfactor+extra.

# Starting docker with the cmd we gave it is presumably no good with
# non-default entrypoint.  (We could actually use --entrypoint, though
# it looks as if that has to be single word.)
if [[ $entry = null ]]; then
    size=$(docker start -a $id 2>/dev/null |
           # Try to verify proper df output (checking the first line) to
           # see if running df worked, and calculate the size.
           awk -v extra="$extra" 'NR==1 && !/^Filesystem/ {exit 1}
                                  NR==2 {print int($3/1024+extra)}')
fi
if [[ $? -ne 0 || $entry = null || ! $size =~ [0-9]+ ]]; then
    # We have to fall back to guessing from the tar stream.
    # Use awk as we may not have bc for the floating point calc.
    if size=$(docker export $id | wc -c); then
        size=$(awk "END {print int($tarfactor*$size/1024/1024+$extra)}" </dev/null)
    else false
    fi
    if [[ $? -ne 0 ]]; then
        message ERROR "Can't extract container size\n"
        exit 1
    fi
fi

message 1 "Creating $sing...\n"
# redirect because -q doesn't work
if ! singularity -q create -s $size "$sing" >/dev/null; then
    message ERROR "failed: singularity create -s $size $sing\n"
    exit 1
fi

## Export

message 1 "Exporting/importing...\n"
if ! docker export $id; then
    message ERROR "docker export failed\n"
    exit 1
fi |
  # If the import fails (presumably for lack of space), we expand the
  # image and try again.  Sink stdout because of tar v and sink stderr
  # in case of not-enough-space errors.
  if ! singularity import "$sing" >/dev/null 2>&1; then
      if ! singularity -q expand "$sing"; then
          message ERROR "singularity expand failed\n"
          exit 1
      elif ! docker export $id | singularity import "$sing" >/dev/null; then
          message ERROR "singularity import failed\n"
          exit 1
      fi
  fi

# Populate /singularity to reflect Docker semantics.  Quoting is a pain.
# This generates a script stream to feed to singularity exec, avoiding a
# scratch file, but perhaps the file would be better.
# Note that the Docker semantics for the entrypoint and cmd aren't too
# clear <https://docs.docker.com/engine/reference/builder/>.

if ! cmd=$(docker inspect --format='{{json .Config.Cmd}}' "$dock"); then
    message ERROR "docker inspect failed\n"
    exit 1
fi

if [[ $cmd != null ]]; then
    cmd=$(IFS='[],'; echo $cmd)
    cmd=${cmd:1}            # no leading blank
fi
if [[ $entry != null ]]; then
    entry=$(IFS='[],'; echo $entry)
    entry=${entry:1}
fi

if [[ $cmd != null || $entry != null ]]; then
    # It's difficult to avoid this by piping into singularity exec.
    tmp=$(mktemp)
    message 1 "Populating /singularity...\n"
    if [[ $entry = null ]]; then
        # Since the default entrypoint is /bin/sh -c, just inline the
        # command.  Docker semantics seem to say it should be
        # overridden by the command line.
        cat <<EOF >"$tmp"
#!/bin/sh
if [ -n "\$1" ]; then
    "\$@"
else
    $cmd
fi
EOF
    else
        # cmd is the single argument of the entrypoint, so we need an
        # extra level of quoting.
        cmd=${cmd//\\/\\\\}     # double \
        cmd=${cmd//\"/\\\"}     # quote "
        # Fixme: Expansion of $* isn't right
        # Fixme: Is /bin/sh guaranteed to be there?  It probably needs
        #        to be for singularity.
        cat <<EOF >"$tmp"
#!/bin/sh
if [ -z "\$1" ]; then
    $entry "$cmd"
else
    $entry "\$*"
fi
EOF
    fi
    chmod 0755 "$tmp"
    if ! singularity -q copy "$sing" -p "$tmp" /singularity; then
       message ERROR "singularity copy failed\n"
       exit 1
   fi
fi

# trap will clean up for us
