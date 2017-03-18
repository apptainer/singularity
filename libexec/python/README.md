# Python Developer Documentation

This document explains how to use the Python functions from any calling Singularity (C) process. For the first version we used a client [cli.py](https://github.com/singularityware/singularity/blob/6433ddd67b6abfdf1faa3eaa7b0338aa8fe55b31/libexec/python/cli.py), with optparse, and this strategy made sense given that the python module had a limited functionality to import Docker layers. When the need arose to support more than one uri, and different optional imports, it was decided that a more modular strategy was needed, and one that mirrored how the core Singularity software worked. Specifically:

 - variables are passed around via the environment
 - commands are simple and modular, named intuitively

In the current version, this old client was removed in favor of a modular, environment variable-based approach. The following sections are about the python api endpoints available, including bootstrap modules, and general utilities. For each, the following standards are used for parsing environmental variables (the source of the inputs for the functions):


# Table of Contents
1. [Environment Varialbes](#environment-variables)
2. [Cache](#cache)
3. [Bootstrap API](#bootstrap-modules)
4. [Util Functions API](#utility-modules)
5. [Future Additions](#future-additions)


## Environment Variables
All environmental variables are parsed in [defaults.py](defaults.py), which is a gateway between variables defined at runtime, and defaults. By way of import from the file, variables set at runtime do not change if re-imported. This was done intentionally to prevent changes during the execution, and could be changed if needed. For all variables, the order of operations works as follows:
  
  1. First preference goes to environment variable set at runtime
  2. Second preference goes to default defined in this file
  3. Then, if neither is found, null is returned except in the case that `required=True`. A `required=True` variable not found will system exit with an error.
  4. Variables that should not be dispayed in debug logger are set with `silent=True`, and are only reported to be defined.

For boolean variables, the following are acceptable for True, with any kind of capitalization or not:

      ("yes", "true", "t", "1","y")


## Cache
The location and usage of the cache is also determined by environment variables. 

**SINGULARITY_DISABLE_CACHE**
If the user wants to disable the cache, all this means is that the layers are written to a temporary directory. The python functions do nothing to actually remove images, as they are needed by the calling process. It should be responsibility of the calling process to remove layers given that `SINGULARITY_DISABLE_CACHE` is set to any true/yes value. By default, the cache is not disabled.

**SINGULARITY_CACHE**
Is the base folder for caching layers and singularity hub images. If not defined, it uses default of `$HOME/.singularity`, and subfolders for docker layers are `$HOME` If defined, the defined location is used instead. If `DISABLE_CACHE` is set to True, this value is ignored in favor of a temporary directory. For specific subtypes of things to cache, subdirectories are created (by python), including `$SINGULARITY_CACHE/docker` for docker layers and `$SINGULARITY_CACHE/shub` for Singularity Hub images. If the cache is not created, the Python script creates it.

**SINGULARITY_CONTENT**
The layerfile is important for both docker ADD and IMPORT, as it is the file where .tar.gz layer files are written for the calling process to extract. If `SINGULARITY_CONTENT` is not defined, then it will be generated as 
`$SINGULARITY_METADATA_BASE/.layers`. 

**SINGULARITY_ENVBASE**
The environment base folder is the folder name within the metadata folder to hold environment variable files to be sourced. If not defined, it defaults to `$SINGULARITY_METADATA_BASE/.env`, and python carries it around in the variable `ENV_BASE`.

**SINGULARITY_LABELBASE**
The label base is akin to the `ENV_BASE`, except it is for labels from the docker image. If not defined, it defaults to `$SINGULARITY_METADATA_BASE/labels`


**SINGULARITY_PULLFOLDER**
By default, images are pulled to the present working directory. The user can change this variable to change that. Currently, the "pull" command is only relevant for Singularity Hub.



## Bootstrap Modules
A boostrap module is a set of functions that allow importing contents, metadata, and environment variables from other containers.

In the current version, the old client was removed, and each module (currently we have support for [shub](shub) and [docker](docker) has a set of functions not exposed for command line use (typically in a `main.py` and `api.py` module, and then a main client (an executable script) that is meant to be called from the command line. For example: 

 - `libexec/python/import.py`
 - `libexec/python/pull.py`

For each of the above, the python takes car of parsing the uri, meaning that a uri of `docker://` or `shub://` can be passed to `python/import.py` and it will be directed to the correct module to handle it. This basic structure is meant to put more responsibility on python for parsing and handling uris, with possibility of easily adding other endpoints in the future. This means that the Singularity (calling process), given that required environmental variables are defined, can call a function like:

      eval $SINGULARITY_libexecdir/singularity/python/import.py 

and the environment might hold and image uri for `docker://` or `shub://`.

For each, the details of required arguments are detailed in the scripts, and discussed below. First, we will review the environmental variables.


### Defaults
The following variables in [defaults.py](defaults.py) are static values that do not change. You probably don't care much about these, but they are included for reference.


#### Docker

**API_BASE** 
Set as `index.docker.io`, which is the name of the registry. In the first version of Singularity we parsed the Registry argument from the build spec file, however now this is removed because it can be obtained directly from the image name (eg, `registry/namespace/repo:tag`)

**API_VERSION**
Is the version of the Docker Registry API currently being used, by default now is `v2`.

**DOCKER_PREFIX**
Whenever a new Docker container is imported, it brings its environment. This means that we must write the environmental variables to a file where they can be preserved. To keep a record of Docker imports, we generate a file starting with `DOCKER_PREFIX` in the environment metadata folder (see environment variable `ENV_BASE`) (default is `docker`). 

**DOCKER_NUMBER**
To support multiple imports, we must number this file (eg, `10-docker.sh`). The `DOCKER_NUMBER` is the starting count for this file, with default `10` to allow more important environment variables to come first. A note of caution to the calling script - this would mean we source them in reverse, otherwise higher numbers (which should be lower priority) overwrite. We probably should import in reverse always, but maintain 1..N as priority ordering so it is intuitive. 

**NAMESPACE**
Is the default namespace, `library`.

**RUNSCRIPT_COMMAND** 
Is not obtained from the environment, but is a hard coded default (`"/bin/bash"`). This is the fallback command used in the case that the docker image does not have a `CMD` or `ENTRYPOINT`.

**TAG**
Is the default tag, `latest`.


#### Singularity Hub

**SHUB_PREFIX**
Singularity images are imported in entirety (meaning no environmental data to parse) so we only need the prefix to write metadata for.

**SHUB_API_BASE**
The default base for the Singularity Hub API, which is `https://singularity-hub.org/api`



### General

**SINGULARITY_COMMAND_ASIS**
By default, we want to make sure the container running process gets passed forward as the current process, so we want to prefix whatever the Docker command or entrypoint is with `exec`. We also want to make sure that following arguments get passed, so we append `"$@"`. Thus, some entrypoint or cmd might look like this:

     /usr/bin/python

and we would parse it into the runscript as:

     exec /usr/bin/python "$@"

However, it might be the case that the user does not want this. For this reason, we have the environmental variable `RUNSCRIPT_COMMAND_ASIS`. If defined as yes/y/1/True/true, etc., then the runscript will remain as `/usr/bin/python`.


**SINGULARITY_METADATA_FOLDER**
Goes into the variable `METADATA_BASE`, and is the directory location to write the metadata file structure. Specifically, this means folders for environmental variable and layers files. The default looks like this:

      `$SINGULARITY_ROOTFS`
           .singularity
               env
               labels


**SINGULARITY_ROOTFS**
This is the root file system location of the container. There are various checks in all calling functions so the script should never get to this point without having it defined.


If the environmental variable `$SINGULARITY_METADATA_FOLDER` is defined, the metadata folder doesn't even need to live in the container. This could be useful if the calling API wants to skip over it's generation, however care should be taken given that the files are some kind of dependency to produce `/environment`. If the variable isn't defined, then the default metadata folder is set to be `$SINGULARITY_ROOTFS/.singularity`. The variable is required, an extra precaution, but probably not necessary since a default is provided.



### Example Usage

#### Docker
The Docker commands include `ADD` and `IMPORT`. Import means returning a layerfile with paths (separated by newlines) to a complete list of layers for import, along with metadata written to the directory structure inside the image. Add means only generating the layerfile without metadata. For all Docker commands, by way of needing to use the Docker Registry, the user can optionally specifying a username and password for authentication:

    export SINGULARITY_DOCKER_USERNAME='mickeymouse' 
    export SINGULARITY_DOCKER_PASSWORD='cheeseftw'

The user and Singularity calling functions also have some control over the cache. Specifically:

 - `SINGULARITY_DISABLE_CACHE`: will write layers to a temporary directory. Note that since the functions aren't actually doing the import, they do not remove the layers (as was done in previous versions) 
 - `SINGULARITY_CACHE`: Is a specific path to the cache.


##### Docker Add

The [docker/add.py](docker/add.py) is akin to an import, but without any environment or metadata variables (e.g., only the .layers file is written). It does not attempt to create an image - it simply writes a list of layers to some layerfile folder. The minimum required environmental variables are:

 - `SINGULARITY_CONTAINER`: (eg, docker://ubuntu:latest)
 - `SINGULARITY_ROOTFS`: the folder where the container is being built

The `SINGULARITY_ROOTFS` and the metadata folder, default value as `$SINGULARITY_ROOTFS/.singularity` MUST exist for the function to run.

An example use case is the following:

      #!/bin/bash

      # This is an example of the base usage for the docker/add.py command

      # We need, minimally, a docker container and rootfs defined
      export SINGULARITY_CONTAINER="docker://ubuntu:latest"
      export SINGULARITY_ROOTFS=/tmp/hello-kitty
      mkdir -p $SINGULARITY_ROOTFS

      # For the rootfs, given an add, the metadata folder must exist
      mkdir -p $SINGULARITY_ROOTFS/.singularity # see defaults.py
      cd libexec/python/tests
      python ../add.py

After the script runs, the file `/tmp/hello-kitty/.layers` will contain the list of layers to import. Something like:


	/home/vanessa/.singularity/docker/sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4.tar.gz
	/home/vanessa/.singularity/docker/sha256:2508cbcde94b50cd53356e8730bb508ddb43c76664a35dc29e69bb8b56a0f531.tar.gz
	/home/vanessa/.singularity/docker/sha256:bc7277e579f03a13476b4d2dc6607124f7e67341dbd58f9d1cd6555bec086c04.tar.gz
	/home/vanessa/.singularity/docker/sha256:946d6c48c2a7d60cb2f4d1c4d3a8131086b412d11a9def59d0bcc0892428dde9.tar.gz
	/home/vanessa/.singularity/docker/sha256:695f074e24e392178d364af5ea2405dda7ab0035284001b49939afac5106c187.tar.gz
	/home/vanessa/.singularity/docker/sha256:8aec416115fdbd74102c9090bcfe03bfe8926876642d8846c8b917959ea9b552.tar.gz


Notice that the `.layers` is written inside the metadata folder, which means that it will remain with the image. We have two options here (@gmkurtzer looking for your feedback on this). We can either write it somewhere else (eg, to tmp) or we can keep the file there, overwrite if the process is done again, and (optionally) change the user cache directory so it doesn't live with the image. It might be cleaner to write to tmp to begin with, which we would do as follows:

      export SINGULARITY_CONTAINER="docker://ubuntu:latest"
      export SINGULARITY_ROOTFS=/tmp/hello-kitty
      export SINGULARITY_LAYERFILE=/tmp/.layers 
      mkdir -p $SINGULARITY_ROOTFS
      python ../add.py

Note that for the above, because this is running an add (that doesn't save any environmental variables) I didn't need to create the metadata folder. This is because it isn't used.


##### Docker Import
Import is the more robust version of add, and works as it did before, meaning we extract layers into the rootfs, and don't need to return or use a layerfile (as with add). Additionally, environment variables and labels are written to the metadata folder. Again, we MUST have the following, otherwise will return error:

 - `SINGULARITY_CONTAINER`: (eg, docker://ubuntu:latest)
 - `SINGULARITY_ROOTFS`: the folder where the container is being built

and the default metadata folder (`$SINGULARITY_ROOTFS/.singularity`) or the user defined `$SINGULARITY_METADATA_BASE` along with the `$SINGULARITY_ENVBASE` and `$SINGULARITY_LABELBASE` must also exist. Since we now are also (potentially) parsing a runscript, the user has the choice to use `CMD` instead of `ENTRYPOINT` by way of the variable `SINGULARITY_DOCKER_INCLUDE_CMD` parsed from `Cmd` in the build spec file, and `SINGULARITY_COMMAND_ASIS` to not include `exec` and `$@`. As with ADD, the user can again specify a `SINGULARITY_DOCKER_USERNAME` and `SINGULARITY_DOCKER_PASSWORD` if authentication is needed. And again, the `SINGULARITY_ROOTFS` and the metadata folder, default value as `$SINGULARITY_ROOTFS/.singularity` MUST exist for the function to run.


An example use case is the following:

      #!/bin/bash

      # This is an example of the base usage for the docker/import.py command
      # run from within libexec/python/tests

      cd libexec/python/tests
      # We need, minimally, a docker container and rootfs defined
      export SINGULARITY_CONTAINER="docker://ubuntu:latest"
      export SINGULARITY_ROOTFS=/tmp/hello-kitty
      mkdir -p $SINGULARITY_ROOTFS
      mkdir -p $SINGULARITY_ROOTFS/.singularity # see defaults.py
      mkdir -p $SINGULARITY_ROOTFS/.singularity/env
      mkdir -p $SINGULARITY_ROOTFS/.singularity/labels
      python ../import.py

After the script runs, the folder `/tmp/hello-kitty` will contain the full image, along with `.singularity` that contains `env` and `labels`.


#### Singularity Hub
The Singularity Hub python functions include IMPORT, and PULL.

 - PULL: is the most basic of the three, pulling an image from Singularity Hub to the cache (default) or if defined, the `SINGULARITY_PULLFOLDER`. It names the image with the format `username-repo-tag.img.gz`, where the tag is optional. The path to the image is returned to the calling process via the `SINGULARITY_LAYERFILE` environmental variable.
 - IMPORT: is the most robust, doing the same as PULL, but additionally extracting metadata about the image to the `SINGULARITY_LABELDIR` folder. 

Examples are included below.


##### PULL
Pull must minimally have a container defined in `SINGULARITY_CONTAINER`

      #!/bin/bash

      cd libexec/python/tests
      # We need, minimally, a singularity hub container, default pulls to cache
      export SINGULARITY_CONTAINER="shub://vsoch/singularity-images"
      python ../pull.py

      # If we specify a different folder, we will specify it
      export SINGULARITY_HUB_PULL_FOLDER=$PWD
      python ../pull.py



##### IMPORT
Finally, IMPORT also writes to the `labels` folder, and depending on if `SINGULARITY_ROOTFS` is defined or not, will either just return layers written to `SINGULARITY_CONTENTS` or do an entire extraction.

      #!/bin/bash

      cd libexec/python/tests
      SINGULARITY_CONTAINER="shub://vsoch/singularity-images"
      
      # This would be doing a full import
      SINGULARITY_ROOTFS=/tmp/hello-kitty
      mkdir -p $SINGULARITY_ROOTFS
      mkdir -p $SINGULARITY_ROOTFS/.singularity # see defaults.py
      mkdir -p $SINGULARITY_ROOTFS/.singularity/labels
      python ../import.py

      # This is just the layerfile writing
      SINGULARITY_CONTENTS=/tmp/hello-kitty.layers
      python ../import.py


## Utility Modules
Included in bootstrap, but not specifically for it, we have a set of utility modules, which do things like:

 - get, add, delete a key from a json file specified

### Json
The json module is (so far) primarily intended to write a key value store of labels specific to a container. This comes down to `.json` files in the `SINGULARITY_METADATA/labels` folder, with each file mapping to it's source (eg, docker, shub, etc). Given that the calling (C) function has specified the label file (`SINGULARITY_LABELBASE`) The general use would be the following:


	# Add a key value to labelfile. The key must not exist
	exec $SINGULARITY_libexecdir/singularity/python/utils/json/add.py --key $KEY --value $VALUE --file $LABELFILE

	# If it exists, you can force add
	exec $SINGULARITY_libexecdir/singularity/python/utils/json/add.py --key $KEY --value $VALUE --file $LABELFILE -f

	# Remove a key from labelfile. If the file is empty after, it is removed
	exec $SINGULARITY_libexecdir/singularity/python/utils/json/delete.py --key $KEY --file $LABELFILE

	# Get a stream / list of all labels (in single file, one per line, separated by :)
	exec $SINGULARITY_libexecdir/singularity/python/utils/json/dump.py --file $LABELFILE

	# Get a single key from label file, returns empty and exists if not defined
	exec $SINGULARITY_libexecdir/singularity/python/utils/json/get.py --key $KEY



## Future Additions

#### Python Internal API URIs
The internal python modules, in the case of returning a `SINGULARITY_CONTENTS` file with a list of contents to be parsed by the calling function, will prefix each content (line in the file) with a uri to tell the calling script how to manage it. Currently, we have the following defined:

- URI_IMAGE: img:// - intended for Singularity Hub images, which are downloaded as .img.gz, but returned after decompression.
- URI_TAR: tar:// - for a tar (not compressed)
- URI_TARGZ: targz:// - for a compressed tarball

For each of the above, a path would add an extra slash (e.g. `img:///home/vanessa/image.img.gz`)

