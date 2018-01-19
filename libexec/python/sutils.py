# -*- coding: utf-8 -*-

'''
utils.py: python helper for singularity command line tool

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

from defaults import (
    SINGULARITY_CACHE,
    DISABLE_HTTPS
)
from message import bot
import datetime
import errno
import hashlib
import json
import os
import subprocess
import stat
import sys
import tempfile
import tarfile
import base64
import re

from io import (
    BytesIO,
    StringIO
)

# Python less than version 3 must import OSError
if sys.version_info[0] < 3:
    from exceptions import OSError


####################################################################
# HTTP OPERATIONS ##################################################
####################################################################

def add_http(url, use_https=True):
    '''add_http will add a http / https prefix to a url,
    in case the user didn't specify
    :param url: the url to add the prefix to
    :param use_https: should we default to https? default is True
    '''
    scheme = "https://"
    if use_https is False or DISABLE_HTTPS is True:
        scheme = "http://"

    # remove scheme from url
    # urlparse is buggy in Python 2.6
    # https://bugs.python.org/issue754016, use regex instead
    parsed = re.sub('.*//', '', url)

    return "%s%s" % (scheme, parsed.rstrip('/'))


def basic_auth_header(username, password):
    '''basic_auth_header will return a base64 encoded header object to
    generate a token
    :param username: the username
    :param password: the password
    '''
    s = "%s:%s" % (username, password)
    if sys.version_info[0] >= 3:
        s = bytes(s, 'utf-8')
        credentials = base64.b64encode(s).decode('utf-8')
    else:
        credentials = base64.b64encode(s)
    auth = {"Authorization": "Basic %s" % credentials}
    return auth


####################################################################
# COMMAND LINE OPERATIONS ##########################################
####################################################################


def run_command(cmd, env=None, quiet=False):
    '''run_command uses subprocess to send a command to the terminal.
    :param cmd: the command to send, should be a list for subprocess
    :param env: an optional environment to include, a dictionary of key/values
    '''
    try:
        if quiet is False:
            bot.verbose2("Running command %s with subprocess" % " ".join(cmd))
        if env is None:
            process = subprocess.Popen(cmd, stdout=subprocess.PIPE)
        else:
            process = subprocess.Popen(cmd, stdout=subprocess.PIPE, env=env)
    except OSError as error:
        bot.error("Error with subprocess: %s, returning None" % error)
        return None

    output = process.communicate()[0]
    if process.returncode != 0:
        return None

    return output


def clean_up(files):
    '''clean up will delete a list of files, only if they exist
    '''
    if not isinstance(files, list):
        files = [files]

    for f in files:
        if os.path.exists(f):
            bot.verbose3("Cleaning up %s" % f)
            os.remove(f)

############################################################################
# TAR/COMPRESSION ##########################################################
############################################################################


def extract_tar(archive, output_folder):
    '''extract a tar archive to a specified output folder
    :param archive: the archive file to extract
    :param output_folder: the output folder to extract to
    '''
    # If extension is .tar.gz, use -xzf
    args = '-xf'
    if archive.endswith(".tar.gz"):
        args = '-xzf'

    # Just use command line, more succinct.
    command = ["tar", args, archive, "-C", output_folder, "--exclude=dev/*"]
    if not bot.is_quiet():
        print("Extracting %s" % archive)

    return run_command(command)


def create_tar(files, output_folder=None):
    '''create_memory_tar will take a list of files (each a dictionary
    with name, permission, and content) and write the tarfile
    (a sha256 sum name is used) to the output_folder.
    If there is no output folde specified, the
    tar is written to a temporary folder.
    '''
    if output_folder is None:
        output_folder = tempfile.mkdtemp()

    finished_tar = None
    additions = []
    contents = []

    for entity in files:
        info = tarfile.TarInfo(name=entity['name'])
        info.mode = entity['mode']
        info.mtime = int(datetime.datetime.now().strftime('%s'))
        info.uid = entity["uid"]
        info.gid = entity["gid"]
        info.uname = entity["uname"]
        info.gname = entity["gname"]

        # Get size from stringIO write
        filey = StringIO()
        content = None
        try:  # python3
            info.size = filey.write(entity['content'])
            content = BytesIO(entity['content'].encode('utf8'))
        except Exception:  # python2
            info.size = int(filey.write(entity['content'].decode('utf-8')))
            content = BytesIO(entity['content'].encode('utf8'))
        pass

        if content is not None:
            addition = {'content': content,
                        'info': info}
            additions.append(addition)
            contents.append(content)

    # Now generate the sha256 name based on content
    if len(additions) > 0:
        hashy = get_content_hash(contents)
        finished_tar = "%s/sha256:%s.tar.gz" % (output_folder, hashy)

        # Warn the user if it already exists
        if os.path.exists(finished_tar):
            msg = "metadata file %s already exists " % finished_tar
            msg += "will over-write."
            bot.debug(msg)

        # Add all content objects to file
        tar = tarfile.open(finished_tar, "w:gz")
        for a in additions:
            tar.addfile(a["info"], a["content"])
        tar.close()

    else:
        msg = "No contents, environment or labels"
        msg += " for tarfile, will not generate."
        bot.debug(msg)

    return finished_tar


####################################################################
# HASHES/FORMAT ####################################################
####################################################################

def get_content_hash(contents):
    '''get_content_hash will return a hash for a list of content (bytes/other)
    '''
    hasher = hashlib.sha256()
    for content in contents:
        if isinstance(content, BytesIO):
            content = content.getvalue()
        if not isinstance(content, bytes):
            content = bytes(content)
        hasher.update(content)
    return hasher.hexdigest()


def get_image_format(image_file):
    '''
       get image format will use the image-format executable to return the kind
       of file type for the image
       Parameters
       ==========
       image_file: full path to the image file to inspect
       Returns
       =======
       GZIP, DIRECTORY, SQUASHFS, EXT3
    '''
    if image_file.endswith('gz'):
        bot.debug('Found compressed image')
        return "GZIP"

    here = os.path.abspath(os.path.dirname(__file__))
    sbin = here.replace("python", "bin/image-type")
    custom_env = os.environ.copy()
    custom_env["SINGULARITY_MESSAGELEVEL"] = "1"
    image_format = run_command([sbin, image_file], env=custom_env, quiet=True)
    if image_format is not None:
        if isinstance(image_format, bytes):
            image_format = image_format.decode('utf-8')
        image_format = str(image_format).strip('\n')
    bot.debug('Found %s image' % image_format)
    return image_format


####################################################################
# FOLDERS ##########################################################
####################################################################


def get_cache(subfolder=None, quiet=False):
    '''get_cache will return the user's cache for singularity.
    The path returned is generated at the start of the run,
    and returned optionally with a subfolder
    :param subfolder: a subfolder in the cache base to retrieve
    '''

    # Clean up the path and create
    cache_base = clean_path(SINGULARITY_CACHE)

    # Does the user want to get a subfolder in cache base?
    if subfolder is not None:
        cache_base = "%s/%s" % (cache_base, subfolder)

    # Create the cache folder(s), if don't exist
    create_folders(cache_base)

    if not quiet:
        bot.info("Cache folder set to %s" % cache_base)
    return cache_base


def create_folders(path):
    '''create_folders attempts to get the same functionality as mkdir -p
    :param path: the path to create.
    '''
    try:
        os.makedirs(path)
    except OSError as e:
        if e.errno == errno.EEXIST and os.path.isdir(path):
            pass
        else:
            bot.error("Error creating path %s, exiting." % path)
            sys.exit(1)


############################################################################
# PERMISSIONS ##############################################################
############################################################################


def has_permission(file_path, permission=None):
    '''has_writability will check if a file has writability using
    bitwise operations. file_path can be a tar member
    :param file_path: the path to the file, or tar member
    :param permission: the stat permission to check for
    is False)
    '''
    if permission is None:
        permission = stat.S_IWUSR
    if isinstance(file_path, tarfile.TarInfo):
        has_permission = file_path.mode & permission
    else:
        st = os.stat(file_path)
        has_permission = st.st_mode & permission
    if has_permission > 0:
        return True
    return False


def change_tar_permissions(tar_file,
                           file_permission=None,
                           folder_permission=None):

    '''change_tar_permissions changes a permission if
    any member in a tarfile file does not have it
    :param file_path the path to the file
    :param file_permission: stat permission to use for files
    :param folder_permission: stat permission to use for folders
    '''
    tar = tarfile.open(tar_file, "r:gz")

    # Owner read, write (o+rw)
    if file_permission is None:
        file_permission = stat.S_IRUSR | stat.S_IWUSR

    # Owner read, write execute (o+rwx)
    if folder_permission is None:
        folder_permission = stat.S_IRUSR | stat.S_IWUSR | stat.S_IXUSR

    # Add owner write permission to all, not symlinks
    members = tar.getmembers()

    if len(members) > 0:

        bot.verbose("Fixing permission for %s" % tar_file)

        # Add all content objects to file
        fd, tmp_tar = tempfile.mkstemp(prefix=("%s.fixperm." % tar_file))
        os.close(fd)
        fixed_tar = tarfile.open(tmp_tar, "w:gz")

        for member in members:

            # add o+rwx for directories
            if member.isdir() and not member.issym():
                member.mode = folder_permission | member.mode
                extracted = tar.extractfile(member)
                fixed_tar.addfile(member, extracted)

            # add o+rw for plain files
            elif member.isfile() and not member.issym():
                member.mode = file_permission | member.mode
                extracted = tar.extractfile(member)
                fixed_tar.addfile(member, extracted)
            else:
                fixed_tar.addfile(member)

        fixed_tar.close()
        tar.close()

        # Rename the fixed tar to be the old name
        os.rename(tmp_tar, tar_file)
    else:
        tar.close()
        bot.warning("Tar file %s is empty, skipping." % tar_file)

    return tar_file

############################################################################
# FILES ####################################################################
############################################################################


def write_file(filename, content, mode="w"):
    '''write_file will open a file, "filename"
    and write content, "content" and properly close the file
    '''
    bot.verbose2("Writing file %s with mode %s." % (filename, mode))
    with open(filename, mode) as filey:
        filey.writelines(content)
    return filename


def write_json(json_obj, filename, mode="w", print_pretty=True):
    '''write_json will (optionally,pretty print) a json object to file
    :param json_obj: the dict to print to json
    :param filename: the output file to write to
    :param pretty_print: if True, will use nicer formatting
    '''
    bot.verbose2("Writing json file %s with mode %s." % (filename, mode))
    with open(filename, mode) as filey:
        if print_pretty is True:
            filey.writelines(print_json(json_obj))
        else:
            filey.writelines(json.dumps(json_obj))
    return filename


def read_json(filename, mode='r'):
    '''read_json reads in a json file and returns
    the data structure as dict.
    '''
    with open(filename, mode) as filey:
        data = json.load(filey)
    return data


def read_file(filename, mode="r", readlines=True):
    '''read_file will open a file, "filename" and
    read content, "content" and properly close the file
    '''
    bot.verbose3("Reading file %s with mode %s." % (filename, mode))
    with open(filename, mode) as filey:
        if readlines:
            content = filey.readlines()
        else:
            content = filey.read()
    return content


def print_json(content, print_console=False):
    '''print_json is intended to pretty print a json
    :param content: the dictionary to print
    :param print_console: if False, return the dump as string (default)
    '''
    if print_console:
        print(json.dumps(content, indent=4, separators=(',', ': ')))
    else:
        return json.dumps(content, indent=4, separators=(',', ': '))


def clean_path(path):
    '''clean_path will canonicalize the path
    :param path: the path to clean
    '''
    return os.path.realpath(path.strip(" "))


def get_fullpath(file_path, required=True):
    '''get_fullpath checks if a file exists, and returns the
    full path to it if it does. If required is true, an error is triggered.
    :param file_path: the path to check
    :param required: is the file required? If True, will exit with error
    '''
    file_path = os.path.abspath(file_path)
    if os.path.exists(file_path):
        return file_path

    # If file is required, we exit
    if required is True:
        bot.error("Cannot find file %s, exiting." % file_path)
        sys.exit(1)

    # If file isn't required and doesn't exist, return None
    bot.warning("Cannot find file %s" % file_path)
    return None


def get_next_infos(base_dir, prefix, start_number, extension):
    '''get_next infos will browse some directory and return
    the next available file
    '''
    output_file = None
    counter = start_number
    found = False

    while not found:
        output_file = "%s/%s-%s%s" % (base_dir,
                                      counter,
                                      prefix,
                                      extension)
        if not os.path.exists(output_file):
            found = True
        counter += 1
    return output_file


def write_singularity_infos(base_dir,
                            prefix,
                            start_number,
                            content,
                            extension=None):

    '''write_singularity_infos will write some metadata object
    to a file in some base, starting at some default number. For example,
    we would want to write dockerN files with docker environment exports to
    some directory ENV_DIR and increase N until we find an available path
    :param base_dir: the directory base to write the file to
    :param prefix: the name of the file prefix (eg, docker)
    :param start_number: the number to start looking for available file at
    :param content: the content to write
    :param extension: the extension to use. If not defined, uses .sh
    '''
    if extension is None:
        extension = ""
    else:
        extension = ".%s" % extension

    # if the base directory doesn't exist, exit with error.
    if not os.path.exists(base_dir):
        msg = "Cannot find required metadata directory"
        msg = "%s %s. Exiting!" % (msg, base_dir)
        bot.warning(msg)
        sys.exit(1)

    # Get the next available number
    output_file = get_next_infos(base_dir,
                                 prefix,
                                 start_number,
                                 extension)
    write_file(output_file, content)
    return output_file
