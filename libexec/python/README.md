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
The following variables in [defaults.py](defaults.py) are a combination of static values, and variables that can be customized by the user via environment variables at runtime. 

#### Docker

**DOCKER_API_BASE** 
Set as `index.docker.io`, which is the name of the registry. In the first version of Singularity we parsed the Registry argument from the build spec file, however now this is removed because it can be obtained directly from the image name (eg, `registry/namespace/repo:tag`). If you don't specify a registry name for your image, this default is used.

**DOCKER_API_VERSION**
Is the version of the Docker Registry API currently being used, by default now is `v2`.

**DOCKER_OS**
This is exposed via the exported environment variable `SINGULARITY_DOCKER_OS` and pertains to images that reveal a version 2 manifest with a [manifest list](https://docs.docker.com/registry/spec/manifest-v2-2/#manifest-list). In the case that the list is present, we must choose an operating system (this variable) and an architecture (below). The default is `linux`.

**DOCKER_ARCHITECTURE**
This is exposed via the exported environment variable `SINGULARITY_DOCKER_ARCHITECTURE` and the same applies as for the `DOCKER_OS` with regards to being used in context of a list of manifests. In the case that the list is present, we must choose an architecture (this variable) and an os (above). The default is `amd64`, and other common ones include `arm`, `arm64`, `ppc64le`, `386`, and `s390x`.


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

**DISABLE_HTTPS**
If you export the variable `SINGULARITY_NOHTTPS` you can force the software to not use https when interacting with a Docker registry. This use case is typically for use of a local registry.


#### Singularity Hub

**SHUB_PREFIX**
Singularity images are imported in entirety (meaning no environmental data to parse) so we only need the prefix to write metadata for.

**SHUB_API_BASE**
The default base for the Singularity Hub API, which is `https://singularity-hub.org/api`

**SHUB_CONTAINERNAME**
The user is empowered to define a custom name for the singularity image downloaded. The first preference goes to specifying an `SHUB_CONTAINERNAME`. For example:

```bash
export SHUB_CONTAINERNAME="meatballs.img"
singularity pull shub://vsoch/singularity-images
...
Done. Container is at: ./meatballs.img
```

**SHUB_NAMEBYCOMMIT**
Second preference goes to naming the container by commit. If this variable is found in the environment, regardless of the value, it will be done!

```bash
unset SHUB_CONTAINERNAME
export SHUB_NAMEBYCOMMIT=yesplease
singularity pull shub://vsoch/singularity-images
Done. Container is at: ./7a75cd7a32192e5d50f267982e0c30aff794076b.img
```

**SHUB_NAMEBYHASH**
Finally, we can name the container based on the file hash.

```bash
unset SHUB_NAMEBYCOMMIT
export SHUB_NAMEBYHASH=yesplease
singularity pull shub://vsoch/singularity-images
Done. Container is at: ./a989bc72cb154d007aa47a5034978328.img
```

If none of the above are selected, the default is to use the username and reponame

```bash
unset SHUB_NAMEBYHASH
singularity pull shub://vsoch/singularity-images
Done. Container is at: ./vsoch-singularity-images-mongo.img
```

### Formatting
Formatting refers to the user interface for the command line tool.

**SINGULARITY_COLORIZE**
By default, debug messages (and other types) use ascii escape sequences to various commands with colors. This helps to distinguish them, and makes the user interface a bit more pleasant. If the output is not going to a terminal, or if the terminal does not support the ascii escape sequences, this variable is set to False. The user can always override this by setting this variable.


### Plugins
Singularity plugins are custom environment variables that can be set to turn bootstrap (and other building) customizations on and off. Currently, we just have one plugin that will, when turned on, have the Python API backend change permissions for the tarballs.

**SINGULARITY_FIX_PERMS**
If set to `True/true/1/yes`, the Python back end will parse through the tar files from Docker in memory, and fix permissions. This sets the variable `PLUGIN_FIXPERMS` in the script, and is by default False.


### General
**SINGULARITY_PYTHREADS**
The Python modules use threads (workers) to download layer files for Docker, and change permissions. By default, we will use 9 workers, unless the environment variable `SINGULARITY_PYTHREADS` is defined.


**SINGULARITY_COMMAND_ASIS**
By default, we want to make sure the container running process gets passed forward as the current process, so we want to prefix whatever the Docker command or entrypoint is with `exec`. We also want to make sure that following arguments get passed, so we append `"$@"`. Thus, some entrypoint or cmd might look like this:

     /usr/bin/python

and we would parse it into the runscript as:

     exec /usr/bin/python "$@"

However, it might be the case that the user does not want this. For this reason, we have the environmental variable `RUNSCRIPT_COMMAND_ASIS`. If defined as yes/y/1/True/true, etc., then the runscript will remain as `/usr/bin/python`.


**SINGULARITY_METADATA_FOLDER**
Goes into the variable `METADATA_BASE`, and is the directory location to write the metadata file structure. Specifically, this means folders for environmental variable and layers files. The default looks like this:

      `$SINGULARITY_ROOTFS`
           .singularity.d/
               env/
               labels.json


**SINGULARITY_ROOTFS**
This is the root file system location of the container. There are various checks in all calling functions so the script should never get to this point without having it defined.


If the environmental variable `$SINGULARITY_METADATA_FOLDER` is defined, the metadata folder doesn't even need to live in the container. This could be useful if the calling API wants to skip over it's generation, however care should be taken given that the files are some kind of dependency to produce `/environment`. If the variable isn't defined, then the default metadata folder is set to be `$SINGULARITY_ROOTFS/.singularity.d`. The variable is required, an extra precaution, but probably not necessary since a default is provided.



### Example Usage

#### Docker

##### Docker Import
The Docker commands include  `IMPORT`. Import means returning a layerfile with paths (separated by newlines) to a complete list of layers for import, one of those layers being a tarfile with Docker runscript, environment, and labels. For all Docker commands, by way of needing to use the Docker Registry, the user can optionally specifying a username and password for authentication:

    export SINGULARITY_DOCKER_USERNAME='mickeymouse' 
    export SINGULARITY_DOCKER_PASSWORD='cheeseftw'

The user and Singularity calling functions also have some control over the cache. Specifically:

 - `SINGULARITY_DISABLE_CACHE`: will write layers to a temporary directory. Note that since the functions aren't actually doing the import, they do not remove the layers (as was done in previous versions) 
 - `SINGULARITY_CACHE`: Is a specific path to the cache.

Since we now are also (potentially) parsing a runscript, the user has the choice to use `CMD` instead of `ENTRYPOINT` by way of the variable `SINGULARITY_INCLUDECMD` parsed from `Cmd` in the build spec file, and `SINGULARITY_COMMAND_ASIS` to not include `exec` and `$@`. As with ADD, the user can again specify a `SINGULARITY_DOCKER_USERNAME` and `SINGULARITY_DOCKER_PASSWORD` if authentication is needed.  The required environment exports are:

 - `SINGULARITY_CONTAINER`: (eg, docker://ubuntu:latest)
 - `SINGULARITY_CONTENTS`: the file to write the list of layers to.


The python function previously extracted the layers, but now to support user import without sudo, and consistency across import/shell/bootstrap, the calling function takes care of this. Thus, it is also important that the calling function write metadata to the `labels.json` and any of the user's preferences for the `runscript` after the layers are extracted, in the case that the user wants to overwrite something that came from the Docker dump.


An example use case is the following:

      #!/bin/bash

      # This is an example of the base usage for the docker/add.py command

      # We need, minimally, a docker container and rootfs defined
      export SINGULARITY_CONTAINER="docker://ubuntu:latest"
      export SINGULARITY_CONTENTS=/tmp/hello-kitty.txt

      cd libexec/python/tests
      python ../import.py

After the script runs, the file `/tmp/hello-kitty.txt` will contain the list of layers to import. Something like:


	/home/vanessa/.singularity/docker/sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4.tar.gz
	/home/vanessa/.singularity/docker/sha256:2508cbcde94b50cd53356e8730bb508ddb43c76664a35dc29e69bb8b56a0f531.tar.gz
	/home/vanessa/.singularity/docker/sha256:bc7277e579f03a13476b4d2dc6607124f7e67341dbd58f9d1cd6555bec086c04.tar.gz
	/home/vanessa/.singularity/docker/sha256:946d6c48c2a7d60cb2f4d1c4d3a8131086b412d11a9def59d0bcc0892428dde9.tar.gz
	/home/vanessa/.singularity/docker/sha256:695f074e24e392178d364af5ea2405dda7ab0035284001b49939afac5106c187.tar.gz
	/home/vanessa/.singularity/docker/sha256:8aec416115fdbd74102c9090bcfe03bfe8926876642d8846c8b917959ea9b552.tar.gz
	
The last of the layers will be the tarfile created by the python with the metadata.



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



## Utility Modules
Included in bootstrap, but not specifically for it, we have a set of utility modules, which do things like:

 - get, add, delete a key from a json file specified
 - get the size of a container

### Size
The size function will return the size of a container. Required environment variables are:

 - `SINGULARITY_CONTAINER`: the path (and uri) to the container (shub:// or docker://)
 - `SINGULARITY_CONTENTS` =  the layerfile to write the size to
        

Example usage is as follows:


    # Singularity Hub
    export SINGULARITY_CONTAINER=shub://vsoch/singularity-hello-world
    export SINGULARITY_CONTENTS=/tmp/hello-kitty.txt
    # from within tests, for example
    python ../size.py

    # Docker
    export SINGULARITY_CONTAINER=docker://ubuntu:latest
    export SINGULARITY_CONTENTS=/tmp/hello-kitty.txt
    python ../size.py


The size is obtained via reading the (version 2.0) manifest size of each layer, and adding them together. For example, I could use the API internally in Python as follows:

```
cd libexec/python
ipython

from docker.api import DockerApiConnection
client=DockerApiConnection(image='ubuntu')
DEBUG Headers found: Accept,Content-Type
VERBOSE Registry: index.docker.io
VERBOSE Namespace: library
VERBOSE Repo Name: ubuntu
VERBOSE Repo Tag: latest
VERBOSE Version: None
VERBOSE Obtaining tags: https://index.docker.io/v2/library/ubuntu/tags/list
DEBUG GET https://index.docker.io/v2/library/ubuntu/tags/list
DEBUG Http Error with code 401
DEBUG GET https://auth.docker.io/token?service=registry.docker.io&expires_in=9000&scope=repository:library/ubuntu:pull
DEBUG Headers found: Accept,Authorization,Content-Type

client.get_size()
VERBOSE Obtaining manifest: https://index.docker.io/v2/library/ubuntu/manifests/latest
DEBUG GET https://index.docker.io/v2/library/ubuntu/manifests/latest
Out[3]: 46795242
```

Given that the container does not have a version 2.0 manifest (not sure if this is possible, but it could be) or if the manifest is malformed in any way, a size of None is returned.


### Json
The json module serves two functions. First, it writes a key value store of labels specific to a container using supporting functions [add](helpers/json/add.py), [helpers/json/delete.py](helpers/json/delete.py) and [helpers/json/get.py](helpers/json/get.py). Second, it servers to return a [JSON API](http://jsonapi.org/) manifest with one or more metrics of interest.

#### Inspect
Inspect will return (print to the screen, or stdout) a json data structure with the fields asked for by the user, specifically which can be in the set defined in [inspect.help](../cli/inspect.help).

```
-l/--labels      Show the labels associated with the image (default)
-d/--deffile     Show the bootstrap definition file which was used
                 to generate this image
-r/--runscript   Show the runscript for this image
-t/--test        Show the test script for this image
-e/--environment Show the environment settings for this container
```

The bash client handles parsing these command line arguments into the following environment variables that are exported, and found by python:

```
SINGULARITY_MOUNTPOINT
SINGULARITY_INSPECT_LABELS
SINGULARITY_INSPECT_DEFFILE
SINGULARITY_INSPECT_RUNSCRIPT
SINGULARITY_INSPECT_TEST
SINGULARITY_INSPECT_ENVIRONMENT
```

Essentially, if any of the above for `SINGULARITY_INSPECT_*` are found to be defined, this is interpreted as "yes/True." Otherwise, if the environment variable is not defined, python finds it as `None` and doesn't do anything. In this manner, we can call the executable as follows (e.g., from [inspect.sh](../helpers/inspect.sh)).

```
eval_abort "$SINGULARITY_libexecdir/singularity/python/helpers/json/inspect.py"
```
and the python will look for the environmental variables specified above. This is different from the other json functions (below) that expect different parameters to be passed into the script call.


#### Json Functions
The functions `get`, `dump`, and `add` are primarily used to write and read `.json` files in the `SINGULARITY_METADATA/labels` folder, with each file mapping to it's source (eg, docker, shub, etc). Given that the calling (C) function has specified the label file (`SINGULARITY_LABELBASE`) The general use would be the following:


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

