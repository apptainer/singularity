#!/usr/bin/env python

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

import logging
import os
import re
import shutil
import subprocess
import sys
import tarfile
try:
    from urllib.parse import urlencode
    from urllib.request import urlopen, Request
    from urllib.error import HTTPError
except ImportError:
    from urllib import urlencode
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
    prefix = "https://"
    if use_https == False:
        prefix="http://"
    
    # Does the url have http?
    if re.search('^http*',url) == None:
        url = "%s%s" %(prefix,url)

    # Always remove extra slash
    url = url.strip('/')

    return url

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
           logging.error("Error parsing response for url %s, exiting.", url)        
           sys.exit(1)

       # If we have a next url
       if "next" in response:
           url = response["next"]
       else:
           done = True
 
       # Add new call to the results
       results = results + response['results']
   return results
    

def parse_headers(default_header,headers):
    '''parse_headers will return a completed header object, adding additional headers to some
    default header
    :param default_header: include default_header (above)
    :param headers: headers to add (default is None)
    '''
    
    # default header for all calls
    header = {'Accept': 'application/json','Content-Type':'application/json; charset=utf-8'}

    if default_header == True:
        if headers != None:
            headers.update(header)
        else:
            headers = header

    else:
        if headers == None:
            headers = dict() 

    return headers


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
        logging.error("HTTPError %s", error)        
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
            filey.write(chunk)

    return stream


############################################################################
## LOGGING #################################################################
############################################################################

def get_logging_level():
    '''setup_logging will configure a logger to standard out based on the user's
    selected level, which should be in an environment variable called MESSAGELEVEL.
    if MESSAGELEVEL is not set, the maximum level (5) is assumed (all messages).

    # http://elinux.org/Debugging_by_printing  
    KERN_EMERG	 "0"	Emergency messages, system is about to crash or is unstable	pr_emerg
    KERN_ALERT	 "1"	Something bad happened and action must be taken immediately	pr_alert
    KERN_CRIT	 "2"	A critical condition occurred like a serious hardware/software failure	pr_crit
    KERN_ERR	 "3"	An error condition, often used by drivers to indicate difficulties with the hardware	pr_err
    KERN_WARNING "4"	A warning, meaning nothing serious by itself but might indicate problems	pr_warning
    KERN_NOTICE	 "5"	Nothing serious, but notably nevertheless. Often used to report security events.	

    CRITICAL	50
    ERROR	40
    WARNING	30
    INFO	20
    DEBUG	10
    NOTSET	0
    # https://docs.python.org/2/library/logging.html
    '''
    MESSAGELEVEL = os.environ.get("MESSAGELEVEL", 5)
    if MESSAGELEVEL == 5:
        level = logging.INFO
    elif MESSAGELEVEL == 4:
        level = logging.WARNING
    elif MESSAGELEVEL == 3:
        level = logging.ERROR
    elif MESSAGELEVEL == 2:
        level = logging.CRITICAL
    elif MESSAGELEVEL == 1:
        level = logging.DEBUG
    else: # NOTSET defaults to parent loggers, then all messages
        level = logging.INFO
    return level


############################################################################
## COMMAND LINE OPERATIONS #################################################
############################################################################


def run_command(cmd):
    '''run_command uses subprocess to send a command to the terminal.
    :param cmd: the command to send, should be a list for subprocess
    '''
    try:
        logging.info("Running command %s with subprocess", " ".join(cmd))
        process = subprocess.Popen(cmd,stdout=subprocess.PIPE)
        output, err = process.communicate()
    except OSError as error:
        logging.error("Error with subprocess: %s, returning None",error)
        return None
    
    return output



############################################################################
## FILE OPERATIONS #########################################################
############################################################################

def change_permissions(path,permission="0755",recursive=True):
    '''change_permissions will use subprocess to change permissions of a file
    or directory. Recursive is default True
    :param path: path to change permissions for
    :param permission: the permission level (default is 0755)
    :param recursive: do recursively (default is True)
    '''
    if not isinstance(permission,str):
        logging.warning("Please provide permission as a string, not number! Skipping.")
    else:
        permission = str(permission)
        cmd = ["chmod",permission,"-R",path]
        if recursive == False:
           cmd = ["chmod",permission,path]
        logging.info("Changing permission of %s to %s with command %s",path,permission," ".join(cmd))
        return run_command(cmd)


def extract_tar(targz,output_folder):
    '''extract_tar will extract a tar.gz to a specified output folder
    :param targz: the tar.gz file to extract
    :param output_folder: the output folder to extract to
    '''
    # Just use command line, more succinct.
    command = ["tar","-xzf",targz,"-C",output_folder,"--exclude=dev/*"]
    logging.info("Extracting %s", " ".join(command))
    return run_command(command) 


def write_file(filename,content,mode="w"):
    '''write_file will open a file, "filename" and write content, "content"
    and properly close the file
    '''
    logging.info("Writing file %s with mode %s.",filename,mode)
    filey = open(filename,mode)
    filey.writelines(content)
    filey.close()
    return filename


def write_json(json_obj,filename,mode="w",print_pretty=True):
    '''write_json will (optionally,pretty print) a json object to file
    :param json_obj: the dict to print to json
    :param filename: the output file to write to
    :param pretty_print: if True, will use nicer formatting   
    '''
    logging.info("Writing json file %s with mode %s.",filename,mode)
    filey = open(filename,mode)
    if print_pretty == True:
        filey.writelines(simplejson.dumps(json_obj, indent=4, separators=(',', ': ')))
    else:
        filey.writelines(simplejson.dumps(json_obj))
    filey.close()
    return filename


def read_file(filename,mode="r"):
    '''write_file will open a file, "filename" and write content, "content"
    and properly close the file
    '''
    logging.info("Reading file %s with mode %s.",filename,mode)
    filey = open(filename,mode)
    content = filey.readlines()
    filey.close()
    return content
