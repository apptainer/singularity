'''

main.py: Docker helper functions for Singularity in Python

Copyright (c) 2016-2017, Vanessa Sochat. All rights reserved.

"Singularity" Copyright (c) 2016, The Regents of the University of California,
through Lawrence Berkeley National Laboratory (subject to receipt of any
required approvals from the U.S. Dept. of Energy).  All rights reserved.

This software is licensed under a customized 3-clause BSD license.  Please
consult LICENSE file distributed with the sources of this project regarding
your rights to use or distribute this software.

NOTICE.  This Software was developed under funding from the U.S. Department of
Energy and the U.S. Government consequently retains certain rights. As such,
the U.S. Government has been granted for itself and others acting on its
behalf a paid-up, nonexclusive, irrevocable, worldwide license in the Software
to reproduce, distribute copies to the public, prepare derivative works, and
perform publicly and display publicly, and to permit other to do so.

'''

import os
from defaults import (
    INCLUDE_CMD,
    PLUGIN_FIXPERMS
)

from base import MultiProcess

from sutils import (
    get_cache,
    write_file
)

from .api import (
    DockerApiConnection,
)

from .tasks import (
    download_layer,
    change_permissions,
    extract_runscript,
    extract_metadata_tar
)

from message import bot


def SIZE(image, auth=None, contentfile=None):
    '''size is intended to be run before an import, to return to the
    contentfile a list of sizes (one per layer) corresponding
    with the layers that will be downloaded for image
    '''
    bot.debug("Starting Docker SIZE, will get size from manifest")
    bot.verbose("Docker image: %s" % image)
    client = DockerApiConnection(image=image, auth=auth)
    size = client.get_size()
    if contentfile is not None:
        write_file(contentfile, str(size), mode="w")
    return size


def IMPORT(image, auth=None, layerfile=None):
    '''IMPORT is the main script that will obtain docker layers,
    runscript information (either entrypoint or cmd), and environment
    and return a list of tarballs to extract into the image
    :param auth: if needed, an authentication header (default None)
    :param layerfile: The file to write layers to extract into
    '''
    bot.debug("Starting Docker IMPORT, includes env, runscript, and metadata.")
    bot.verbose("Docker image: %s" % image)

    # Does the user want to override default of using ENTRYPOINT?
    if INCLUDE_CMD:
        bot.verbose2("Specified Docker CMD as %runscript.")
    else:
        bot.verbose2("Specified Docker ENTRYPOINT as %runscript.")

    # Input Parsing ----------------------------
    # Parse image name, repo name, and namespace
    client = DockerApiConnection(image=image, auth=auth)

    docker_image_uri = "Docker image path: %s" % client.assemble_uri("/")
    bot.info(docker_image_uri)

    # IMAGE METADATA -------------------------------------------
    # Use Docker Registry API (version 2.0) to get images ids, manifest

    images = client.get_images()

    #  DOWNLOAD LAYERS -------------------------------------------
    # Each is a .tar.gz file, obtained from registry with curl

    # Get the cache (or temporary one) for docker
    cache_base = get_cache(subfolder="docker")
    download_client = MultiProcess()

    # Generate a queue of tasks to run with MultiProcess
    layers = []
    tasks = []
    for ii in range(len(images)):
        image_id = images[ii]
        targz = "%s/%s.tar.gz" % (cache_base, image_id)
        if not os.path.exists(targz):
            tasks.append((client, image_id, cache_base))
        layers.append(targz)

    # Does the user want to change permissions of tar?
    func2 = None
    if PLUGIN_FIXPERMS:
        func2 = change_permissions

    if len(tasks) > 0:
        download_layers = download_client.run(func=download_layer,
                                              func2=func2,
                                              tasks=tasks)

    # Get Docker runscript
    runscript = extract_runscript(manifest=client.manifestv1,
                                  includecmd=INCLUDE_CMD)

    # Add the environment export
    tar_file = extract_metadata_tar(client.manifestv1,
                                    client.assemble_uri(),
                                    runscript=runscript)

    bot.verbose2('Tar file with Docker env and labels: %s' % tar_file)

    # Write all layers to the layerfile
    if layerfile is not None:
        bot.verbose3("Writing Docker layers files to %s" % layerfile)
        write_file(layerfile, "\n".join(layers), mode="w")
        if tar_file is not None:
            write_file(layerfile, "\n%s" % tar_file, mode="a")

    # Return additions dictionary
    additions = {"layers": layers,
                 "image": image,
                 "manifest": client.manifest,
                 "manifestv1": client.manifestv1,
                 "cache_base": cache_base,
                 "metadata": tar_file}

    bot.debug("*** FINISHING DOCKER IMPORT PYTHON PORTION ****\n")

    return additions
