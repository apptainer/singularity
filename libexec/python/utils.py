'''
utils.py: python helper for singularity command line tool

Copyright (c) 2016, Vanessa Sochat. All rights reserved. 

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

from defaults import SINGULARITY_CACHE
from logman import logger
import errno
import json
import os
import shutil
import subprocess
import stat
from stat import ST_MODE
import sys
import tempfile
import tarfile
import base64
try:
    from urllib.parse import urlencode, urlparse
    from urllib.request import urlopen, Request, unquote
    from urllib.error import HTTPError
except ImportError:
    from urllib import urlencode, unquote
    from urlparse import urlparse
    from urllib2 import urlopen, Request, HTTPError

# Python less than version 3 must import OSError
if sys.version_info[0] < 3:
    from exceptions import OSError


############################################################################
## HTTP OPERATIONS #########################################################
############################################################################

def add_http(url,use_https=True):
    '''add_http will add a http / https prefix to a url, in case the user didn't
    specify
    :param url: the url to add the prefix to
    :param use_https: should we default to https? default is True
    '''
    scheme = "https://"
    if use_https == False:
        scheme="http://"

    parsed = urlparse(url)
    # Returns tuple with(scheme,netloc,path,params,query,fragment)

    return "%s%s" %(scheme,"".join(parsed[1:]).rstrip('/'))


def api_get_pagination(url):
   '''api_pagination is a wrapper for "api_get" that will also handle pagination
   :param url: the url to retrieve:
   :param default_header: include default_header (above)
   :param headers: headers to add (default is None)
   '''
   done = False
   results = []
   while not done:
       response = api_get(url=url)
       try:
           response = json.loads(response)
       except:
           logger.error("Error parsing response for url %s, exiting.", url)        
           sys.exit(1)

       # If we have a next url
       if "next" in response:
           url = response["next"]
       else:
           done = True
 
       # Add new call to the results
       results = results + response['results']
   return results
    

def parse_headers(default_header,headers=None):
    '''parse_headers will return a completed header object, adding additional headers to some
    default header
    :param default_header: include default_header (above)
    :param headers: headers to add (default is None)
    '''
    
    # default header for all calls
    header = {'Accept': 'application/json','Content-Type':'application/json; charset=utf-8'}

    if default_header == True:
        if headers != None:
            final_headers = header.copy()
            final_headers.update(headers)
        else:
            final_headers = header

    else:
        final_headers = headers
        if headers == None:
            final_headers = dict() 

    return final_headers


def api_get(url,data=None,default_header=True,headers=None,stream=None,return_response=False):
    '''api_get gets a url to the api with appropriate headers, and any optional data
    :param data: a dictionary of key:value items to add to the data args variable
    :param url: the url to get
    :param stream: The name of a file to stream the response to. If defined, will stream
    default is None (will not stream)
    :returns response: the requests response object
    '''
    headers = parse_headers(default_header=default_header,
                            headers=headers)

    # Does the user want to stream a response?
    do_stream = False
    if stream != None:
        do_stream = True

    if data != None:
        args = urlencode(data)
        request = Request(url=url, 
                          data=args, 
                          headers=headers) 
    else:
        request = Request(url=url, 
                          headers=headers) 

    try:
        response = urlopen(request)

    # If we have an HTTPError, try to follow the response
    except HTTPError as error:
        return error

    # Does the call just want to return the response?
    if return_response == True:
        return response

    if do_stream == False:
        return response.read().decode('utf-8')
       
    chunk_size = 1 << 20
    with open(stream, 'wb') as filey:
        while True:
            chunk = response.read(chunk_size)
            if not chunk: 
                break
            try:
                filey.write(chunk)
            except: #PermissionError
                logger.error("Cannot write to %s, exiting",stream)
                sys.exit(1)

    return stream

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


############################################################################
## COMMAND LINE OPERATIONS #################################################
############################################################################


def run_command(cmd):
    '''run_command uses subprocess to send a command to the terminal.
    :param cmd: the command to send, should be a list for subprocess
    '''
    try:
        logger.info("Running command %s with subprocess", " ".join(cmd))
        process = subprocess.Popen(cmd,stdout=subprocess.PIPE)
        output, err = process.communicate()
    except OSError as error:
        logger.error("Error with subprocess: %s, returning None",error)
        return None
    
    return output



############################################################################
## PERMISSIONS #############################################################
############################################################################


def has_permission(file_path,permission=None):
    '''has_writability will check if a file has writability using
    bitwise operations
    :param file_path: the path to the file
    :param permission: the stat permission to check for
    is False)
    '''
    if permission == None:
        permission = stat.S_IWUSR
    st = os.stat(file_path)
    has_permission = st.st_mode & permission
    if has_permission > 0:
        return True
    return False


def change_permission(file_path,permission=None):
    '''change_permission changes a permission if the file does not have it
    :param file_path the path to the file
    :param permission: the stat permission to use
    '''
    if permission == None:
        permission = stat.S_IWUSR
    st = os.stat(file_path)
    has_perm = has_permission(file_path,permission)
    if not has_perm:
        logger.info("Fixing permission on: %s", file_path)
        try:
            os.chmod(file_path, st.st_mode | permission)
        except:
            print("ERROR: Couldn't change permission on ", file_path)
            sys.exit(1)
    return has_permission(file_path,permission)


def change_permissions(path,permission=None,recursive=True):
    '''change_permissions will change all permissions of files
    and directories. Recursive is default True, given a folder
    :param path: path to change permissions for
    :param permission: the permission from stat to add (default is stat.S_IWUSR)
    :param recursive: do recursively (default is True)
    '''
    # Default permission to change is adding write
    if permission == None:
        permission = stat.S_IWUSR

    # For a file, recursion is not relevant
    if os.path.isfile(path):
        logger.info("Changing permission of %s to %s",path,oct(permission))
        change_permission(path,permission)
    else:
        # If the user wants recursive, use os.walk
        logger.info("Changing permission of files and folders under %s to %s",path,oct(permission))
        for root, dirs, files in os.walk(path, topdown=False, followlinks=False):

            # Walking through directories
            for name in dirs:
                dir_path = os.path.join(root, name)
                # Make sure it's a valid dir
                if os.path.isdir(dir_path):
                    change_permission(dir_path, permission)

            # Walking through files (and checking for symbolic links)
            for name in files:
                file_path = os.path.join(root, name)
                # Make sure it's a valid file
                if os.path.isfile(file_path) and not os.path.islink(file_path):
                    change_permission(file_path, permission)
                

############################################################################
## FILE OPERATIONS #########################################################
############################################################################


def get_cache(cache_base=None,subfolder=None,disable_cache=False):
    '''get_cache will return the user's cache for singularity. If not specified
    via environmental variable, will be created in $HOME/.singularity
    :param cache_base: the cache base
    :param subfolder: a subfolder in the cache base to retrieve, specifically
    :param disable_cache: boolean, if True, will return temporary directory
    instead. The other functions are responsible for cleaning this up after use.
    for a particular kind of image cache (eg, docker, shub, etc.)
    '''
    # Obtain cache base from environment (1st priority, then variable)
    if disable_cache == True:
        return tempfile.mkdtemp()
    else:
        cache_base = os.environ.get("SINGULARITY_CACHEDIR", cache_base)

    # Default is set in defaults.py, $HOME/.singularity
    if cache_base == None:
        cache_base = SINGULARITY_CACHE

    # Clean up the path and create
    cache_base = clean_path(cache_base)

    # Does the user want to get a subfolder in cache base?
    if subfolder != None:
        cache_base = "%s/%s" %(cache_base,subfolder)
        
    # Create the cache folder(s), if don't exist
    create_folders(cache_base)

    print("Cache folder set to %s" %cache_base)
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
            logger.error("Error creating path %s, exiting.",path)
            sys.exit(1)


def extract_tar(archive,output_folder):
    '''extract_tar will extract a tar archive to a specified output folder
    :param archive: the archive file to extract
    :param output_folder: the output folder to extract to
    '''
    # If extension is .tar.gz, use -xzf
    args = '-xf'
    if archive.endswith(".tar.gz"):
        args = '-xzf'

    # Just use command line, more succinct.
    command = ["tar", args, archive, "-C", output_folder, "--exclude=dev/*"]
    print("Extracting %s" %(archive))

    retval = run_command(command)

    # Change permissions (default ensures writable)
    change_permissions(output_folder)

    # Should we return a list of extracted files? Current returns empty string
    return retval


def write_file(filename,content,mode="w"):
    '''write_file will open a file, "filename" and write content, "content"
    and properly close the file
    '''
    logger.info("Writing file %s with mode %s.",filename,mode)
    with open(filename,mode) as filey:
        filey.writelines(content)
    return filename


def write_json(json_obj,filename,mode="w",print_pretty=True):
    '''write_json will (optionally,pretty print) a json object to file
    :param json_obj: the dict to print to json
    :param filename: the output file to write to
    :param pretty_print: if True, will use nicer formatting   
    '''
    logger.info("Writing json file %s with mode %s.",filename,mode)
    with open(filename,mode) as filey:
        if print_pretty == True:
            filey.writelines(json.dumps(json_obj, indent=4, separators=(',', ': ')))
        else:
            filey.writelines(json.dumps(json_obj))
    return filename


def read_file(filename,mode="r"):
    '''write_file will open a file, "filename" and write content, "content"
    and properly close the file
    '''
    logger.info("Reading file %s with mode %s.",filename,mode)
    with open(filename,mode) as filey:
        content = filey.readlines()
    return content


def clean_path(path):
    '''clean_path will canonicalize the path
    :param path: the path to clean
    '''
    return os.path.realpath(path.strip(" "))


def get_fullpath(file_path,required=True):
    '''get_fullpath checks if a file exists, and returns the
    full path to it if it does. If required is true, an error is triggered.
    :param file_path: the path to check
    :param required: is the file required? If True, will exit with error
    '''
    file_path = os.path.abspath(file_path)
    if os.path.exists(file_path):
        return file_path

    # If file is required, we exit
    if required == True:
        logger.error("Cannot find file %s, exiting.",file_path)
        sys.exit(1)

    # If file isn't required and doesn't exist, return None
    logger.warning("Cannot find file %s",file_path)
    return None
