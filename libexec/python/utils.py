#!/usr/bin/env python

'''
utils.py: python helper for singularity command line tool

'''

import os
import requests
import shutil
import subprocess
import sys
import tarfile

# Python less than version 3 must import OSError
if sys.version_info[0] < 3:
    from exceptions import OSError


############################################################################
## HTTP OPERATIONS #########################################################
############################################################################


# default header for all calls
header = {'Accept': 'application/json','Content-Type':'application/json; charset=utf-8'}


def api_post(url,data=None):
    '''api_post posts a url to the api with appropriate headers, and any optional data
    :param data: a dictionary of key:value items to add to the data args variable
    :param url: the url to post to
    :returns response: the requests response object
    '''

    if data != None:
        args = {"args":data}
        return requests.post(url, headers=headers, data=json.dumps(args))
    return requests.post(url, headers=headers)


def api_get(url,data=None,headers=None,stream=None):
    '''api_get gets a url to the api with appropriate headers, and any optional data
    :param data: a dictionary of key:value items to add to the data args variable
    :param url: the url to get
    :param stream: The name of a file to stream the response to. If defined, will stream
    default is None (will not stream)
    :returns response: the requests response object
    '''
    if headers != None:
        header.update(headers)

    # Does the user want to stream a response?
    do_stream = False
    if stream != None:
        do_stream = True

    if data != None:
        args = {"args":data}
        response = requests.get(url, 
                                headers=headers, 
                                data=json.dumps(args),
                                stream=do_stream)
    else:
        response = requests.get(url, 
                                headers=headers, 
                                stream=do_stream)

    if do_stream == False:
        return response

    # If do_stream is True, stream the response into a file
    with open(stream, 'wb') as filey:
        for chunk in response.iter_content(chunk_size=1024): 
            if chunk: # filter out keep-alive new chunks
                filey.write(chunk)
    return stream


############################################################################
## COMMAND LINE OPERATIONS #################################################
############################################################################


def run_command(cmd):
    '''run_command uses subprocess to send a command to the terminal.
    :param cmd: the command to send, should be a list for subprocess
    '''
    try:
        process = subprocess.Popen(cmd,stdout=subprocess.PIPE)
        output, err = process.communicate()
    except OSError as error:
        print(err)
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
        print("Please provide permission as a string, not number!")
    else:
        permission = str(permission)
        cmd = ["chmod",permission,"-R",path]
        if recursive == False:
           cmd = ["chmod",permission,path]
        print("Changing permission of %s to %s" %(path,permission))
        return run_command(cmd)


def extract_tar(targz,output_folder):
    '''extract_tar will extract a tar.gz to a specified output folder
    :param targz: the tar.gz file to extract
    :param output_folder: the output folder to extract to
    '''
    # Just use command line, more succinct.
    return run_command(["tar","-xzf",targz,"-C",output_folder]) 


def write_file(filename,content,mode="wb"):
    '''write_file will open a file, "filename" and write content, "content"
    and properly close the file
    '''
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
    filey = open(filename,mode)
    if print_pretty == True:
        filey.writelines(simplejson.dumps(json_obj, indent=4, separators=(',', ': ')))
    else:
        filey.writelines(simplejson.dumps(json_obj))
    filey.close()
    return filename


def read_file(filename,mode="rb"):
    '''write_file will open a file, "filename" and write content, "content"
    and properly close the file
    '''
    filey = open(filename,mode)
    content = filey.readlines()
    filey.close()
    return content
