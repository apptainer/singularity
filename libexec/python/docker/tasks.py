'''

tasks.py: Tasks for the Docker API

Copyright (c) 2017, Vanessa Sochat. All rights reserved.


'''

import sys
import os
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__),  # noqa
                os.path.pardir)))  # noqa
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))) # noqa

from sutils import (
    add_http,
    get_cache,
    create_tar,
    change_tar_permissions,
    print_json,
    write_singularity_infos
)

from defaults import (
    DOCKER_NUMBER,
    DOCKER_PREFIX,
    ENV_BASE,
    LABELFILE,
    PLUGIN_FIXPERMS,
    METADATA_FOLDER_NAME,
    RUNSCRIPT_COMMAND_ASIS
)

from helpers.json.main import ADD
from message import bot
from templates import get_template

import json
import re


def download_layer(client, image_id, cache_base):
    '''download_layer is a function (external to the client)
    to download a layer and update the client's token after.
    This is intended to be used by the multiprocessing function.
    If return_tmp is True, the temporary file is returned
    (intended to be renamed later)'''
    targz = client.get_layer(image_id=image_id,
                             download_folder=cache_base,  # return tmp when
                             return_tmp=PLUGIN_FIXPERMS)  # fix permissions

    client.update_token()
    return targz


def change_permissions(tar_file, file_permission=None, folder_permission=None):
    '''change_permissions is a wrapper for change_tar_permissions,
    intended for use as a function for multiprocessing. To ensure atomic
    download and permission changes, the input file here is expected
    to have a temporary extension. This wrapper simply calls the function
    to change_tar_permissions, and then renames to the final file
    '''
    fixed_tar = change_tar_permissions(tar_file,
                                       file_permission=file_permission,
                                       folder_permission=folder_permission)

    final_tar = "%s.tar.gz" % fixed_tar.split('.tar.gz')[0]
    os.rename(fixed_tar, final_tar)
    return final_tar


def extract_runscript(manifest, includecmd=False):
    '''create_runscript will write a bash script with default "ENTRYPOINT"
    into the base_dir. If includecmd is True, CMD is used instead. For both.
    if the result is found empty, the other is tried, and then a default used.
    :param manifest: the manifest to use to get the runscript
    :param includecmd: overwrite default command (ENTRYPOINT) default is False
    '''
    cmd = None

    # Does the user want to use the CMD instead of ENTRYPOINT?
    commands = ["Entrypoint", "Cmd"]
    if includecmd is True:
        commands.reverse()
    configs = get_configs(manifest, commands)

    # Look for non "None" command
    for command in commands:
        if configs[command] is not None:
            cmd = configs[command]
            break

    if cmd is not None:
        bot.verbose3("Adding Docker %s as Singularity runscript..."
                     % command.upper())

        # If the command is a list, join. (eg ['/usr/bin/python','hello.py']
        if not isinstance(cmd, list):
            cmd = [cmd]

        cmd = " ".join(['"%s"' % x for x in cmd])

        if not RUNSCRIPT_COMMAND_ASIS:
            cmd = 'exec %s "$@"' % cmd
        cmd = "#!/bin/sh\n\n%s\n" % cmd
        return cmd

    bot.debug("CMD and ENTRYPOINT not found, skipping runscript.")
    return cmd


def extract_metadata_tar(manifest,
                         image_name,
                         include_env=True,
                         include_labels=True,
                         runscript=None):

    '''extract_metadata_tar will write a tarfile with the environment,
    labels, and runscript. include_env and include_labels should be booleans,
    and runscript should be None or a string to write to the runscript.
    '''
    tar_file = None
    files = []
    if include_env or include_labels:

        # Extract and add environment
        if include_env:
            environ = extract_env(manifest)
            if environ not in [None, ""]:
                bot.verbose3('Adding Docker environment to metadata tar')
                template = get_template('tarinfo')
                template['name'] = './%s/env/%s-%s.sh' % (METADATA_FOLDER_NAME,
                                                          DOCKER_NUMBER,
                                                          DOCKER_PREFIX)
                template['content'] = environ
                files.append(template)

        # Extract and add labels
        if include_labels:
            labels = extract_labels(manifest)
            if labels is not None:
                if isinstance(labels, dict):
                    labels = print_json(labels)
                bot.verbose3('Adding Docker labels to metadata tar')
                template = get_template('tarinfo')
                template['name'] = "./%s/labels.json" % METADATA_FOLDER_NAME
                template['content'] = labels
                files.append(template)

        if runscript is not None:
            bot.verbose3('Adding Docker runscript to metadata tar')
            template = get_template('tarinfo')
            template['name'] = "./%s/runscript" % METADATA_FOLDER_NAME
            template['content'] = runscript
            files.append(template)

    if len(files) > 0:
        output_folder = get_cache(subfolder="metadata", quiet=True)
        tar_file = create_tar(files, output_folder)
    else:
        bot.warning("No metadata will be included.")
    return tar_file


def extract_env(manifest):
    '''extract the environment from the manifest, or return None.
    Used by functions env_extract_image, and env_extract_tar
    '''
    environ = get_config(manifest, 'Env')
    if environ is not None:
        if not isinstance(environ, list):
            environ = [environ]

        lines = []
        for line in environ:
            line = re.findall("(?P<var_name>.+?)=(?P<var_value>.+)", line)
            line = ['export %s="%s"' % (x[0], x[1]) for x in line]
            lines = lines + line

        environ = "\n".join(lines)
        bot.verbose3("Found Docker container environment!")
    return environ


def env_extract_image(manifest):
    '''env_extract_image will write a file of key value pairs
    of the environment to export. The manner to export must
    be determined by the calling process depending on the OS type.
    :param manifest: the manifest to use
    '''
    environ = extract_env(manifest)
    if environ is not None:
        environ_file = write_singularity_infos(base_dir=ENV_BASE,
                                               prefix=DOCKER_PREFIX,
                                               start_number=DOCKER_NUMBER,
                                               content=environ,
                                               extension='sh')
    return environ


def extract_labels(manifest, labelfile=None, prefix=None):
    '''extract_labels will write a file of key value pairs including
    maintainer, and labels.
    :param manifest: the manifest to use
    :param labelfile: if defined, write to labelfile (json)
    :param prefix: an optional prefix to add to the names
    '''
    if prefix is None:
        prefix = ""

    labels = get_config(manifest, 'Labels')
    if labels is not None and len(labels) is not 0:
        bot.verbose3("Found Docker container labels!")
        if labelfile is not None:
            for key, value in labels.items():
                key = "%s%s" % (prefix, key)
                value = ADD(key, value, labelfile, force=True)
    return labels


def get_config(manifest, spec="Entrypoint", delim=None):
    '''get_config returns a particular spec (default is Entrypoint)
    from a VERSION 1 manifest obtained with get_manifest.
    :param manifest: the manifest obtained from get_manifest
    :param spec: the key of the spec to return, default is "Entrypoint"
    :param delim: Given a list, the delim to use to join the entries.
                  Default is newline
    '''
    cmd = None
    if "history" in manifest:
        history = manifest['history']
        for entry in manifest['history']:
            if 'v1Compatibility' in entry:
                entry = json.loads(entry['v1Compatibility'])
                if "config" in entry:
                    if spec in entry["config"]:
                        if entry["config"][spec] is not None:
                            cmd = entry["config"][spec]
                            break

    # Standard is to include commands like ['/bin/sh']
    if isinstance(cmd, list):
        if delim is not None:
            cmd = delim.join(cmd)
    bot.verbose3("Found Docker command (%s) %s" % (spec, cmd))

    return cmd


def get_configs(manifest, keys, delim=None):
    '''get_configs is a wrapper for get_config to return a dictionary
    with multiple config items.
    :param manifest: the complete manifest
    :param keys: the key to find
    :param delim: given a list, combine based on this delim
    '''
    configs = dict()
    if not isinstance(keys, list):
        keys = [keys]
    for key in keys:
        configs[key] = get_config(manifest, key, delim=delim)
    return configs
