'''

python: base template for making a connection to an API

'''

from logman import bot
import tempfile
import sys
import re
import os


try:
    from urllib.parse import urlencode, urlparse
    from urllib.request import urlopen, Request, unquote
    from urllib.error import HTTPError
except ImportError:
    from urllib import urlencode, unquote
    from urlparse import urlparse
    from urllib2 import urlopen, Request, HTTPError

if sys.version_info[0] < 3:
    from exceptions import OSError



class ApiConnection(object):

    def __init__(self, **kwargs):
 
        self.headers = dict()
        self.update_headers()
        

    def get_headers(self):
        return self.headers


    def _init_headers(self):
        return {'Accept': 'application/json','Content-Type':'application/json; charset=utf-8'}


    def update_headers(self,fields=None):
        '''get_headers will return a simple default header for a json
        post. This function will be adopted as needed.
        '''
        if len(self.headers) == 0:
            headers = self._init_headers()
        else:
            headers = self.headers

        if fields is not None:
            for key,value in fields.items():
                headers[key] = value

        header_names = ",".join(list(headers.keys()))
        bot.logger.debug("Headers found: %s",header_names)
        self.headers = headers


    def update_token(self,response=None):
        '''empty update_token to be defined by subclasses, if necessary
        '''
        return None


    def get(self,url,data=None,headers=None,default_headers=True,stream=None,return_response=False):
        '''get will use requests to get a particular url
        :param data: a dictionary of key:value items to add to the data args variable
        :param url: the url to get
        :param stream: The name of a file to stream the response to. If defined, will stream
        default is None (will not stream)
        :returns response: the requests response object, or stream
        '''
        bot.logger.debug("GET %s", url)

        # If we use default headers, start with client's
        request_headers = dict()
        if default_headers and len(self.headers) > 0:
            request_headers = self.headers

        if headers is not None:
            request_headers.update(headers)

        request = self.prepare_request(headers=request_headers,
                                       data=data,
                                       url=url)

        return self.submit_request(request,
                                   stream=stream,
                                   return_response=return_response)


    def submit_request(self,request,stream=None,return_response=False):
        '''submit_request will make the request, via a stream or not. If return
        response is True, the response is returned as is without further parsing.
        Given a 401 error, the update_token function is called to try the request 
        again, and only then the error returned.
        '''

        # Does the user want to stream a response?
        do_stream = False
        if stream is not None:
            do_stream = True

        try:
            response = urlopen(request)

        # If we have an HTTPError, try to follow the response
        except HTTPError as error:

            # Case 1: we have an http 401 error, and need to refresh token
            if error.code == 401:
                self.update_token(error)
                try:
                    response = urlopen(request)
                except HTTPError as error:    
                    return error
            else:
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
                except: # PermissionError
                    bot.logger.error("Cannot write to %s, exiting",stream)
                    sys.exit(1)

        return stream


    def prepare_request(self,url,data=None,headers=None):
        '''prepare the request object, determining if there is data (making it
        a POST) or if we should stream the result.
        '''
        if data is not None:
            args = urlencode(data)

            request = Request(url=url, 
                              data=args, 
                              headers=headers) 
        else:
            request = Request(url=url, 
                              headers=headers) 
        return request


    def download_atomically(self,url,file_name,headers=None):
        '''download stream atomically will stream to a temporary file, and
        rename only upon successful completion. This is to ensure that
        errored downloads are not found as complete in the cache
        :param file_name: the file name to stream to
        :param url: the url to stream from
        :param headers: additional headers to add to the get (default None)
        '''
        try:
            fd, tmp_file = tempfile.mkstemp(prefix=("%s.tmp." % file_name)) # file_name.tmp.XXXXXX
            os.close(fd)
            response = self.get(url,headers=headers,stream=tmp_file)
            if isinstance(response, HTTPError):
                bot.logger.error("Error downloading %s, exiting.", url)
                sys.exit(1)
            os.rename(tmp_file, file_name)
        except:
            download_folder = os.path.dirname(os.path.abspath(file_name))
            bot.logger.error("Error downloading %s. Do you have permission to write to %s?", url, download_folder)
            try:
                os.remove(tmp_file)
            except:
                pass
            sys.exit(1)
        return file_name
