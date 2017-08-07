'''

python: base template for making a connection to an API

Copyright (c) 2017, Vanessa Sochat. All rights reserved.

'''

from message import bot
from defaults import SINGULARITY_WORKERS
import multiprocessing
import itertools
import tempfile
import time
import signal
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


class MultiProcess(object):

    def __init__(self, workers=None):

        if workers is None:
            workers = SINGULARITY_WORKERS
        self.workers = workers
        bot.debug("Using %s workers for multiprocess." % (self.workers))

    def start(self):
        bot.debug("Starting multiprocess")
        self.start_time = time.time()

    def end(self):
        self.end_time = time.time()
        self.runtime = self.runtime = self.end_time - self.start_time
        bot.debug("Ending multiprocess, runtime: %s sec" % (self.runtime))

    def run(self, func, tasks, func2=None):
        '''run will send a list of tasks,
        a tuple with arguments, through a function.
        the arguments should be ordered correctly.
        :param func: the function to run with multiprocessing.pool
        :param tasks: a list of tasks, each a tuple
                      of arguments to process
        :param func2: filter function to run result
                      from func through (optional)
        '''

        # Keep track of some progress for the user
        progress = 1
        total = len(tasks)

        # If two functions are run per task, double total jobs
        if func2 is not None:
            total = total * 2

        finished = []
        level1 = []
        results = []

        try:
            prefix = "[%s/%s]" % (progress, total)
            bot.show_progress(0, total, length=35, prefix=prefix)
            pool = multiprocessing.Pool(self.workers, init_worker)

            self.start()
            for task in tasks:
                result = pool.apply_async(multi_wrapper,
                                          multi_package(func, [task]))
                results.append(result)
                level1.append(result._job)

            while len(results) > 0:
                result = results.pop()
                result.wait()
                bot.show_progress(progress, total, length=35, prefix=prefix)
                progress += 1
                prefix = "[%s/%s]" % (progress, total)

                # Pass the result through a second function?
                if func2 is not None and result._job in level1:
                    result = pool.apply_async(multi_wrapper,
                                              multi_package(func2,
                                                            [(result.get(),)]))
                    results.append(result)
                else:
                    finished.append(result.get())

            self.end()
            pool.close()
            pool.join()

        except (KeyboardInterrupt, SystemExit):
            bot.error("Keyboard interrupt detected, terminating workers!")
            pool.terminate()
            sys.exit(1)

        except Exception as e:
            bot.error(e)

        return finished


# Supporting functions for MultiProcess
def init_worker():
    signal.signal(signal.SIGINT, signal.SIG_IGN)


def multi_wrapper(func_args):
    function, args = func_args
    return function(*args)


def multi_package(func, args):
    return zip(itertools.repeat(func), args)


class ApiConnection(object):

    def __init__(self, **kwargs):
        self.headers = dict()
        self.update_headers()

    def get_headers(self):
        return self.headers

    def _init_headers(self):
        return {'Accept': 'application/json',
                'Content-Type': 'application/json; charset=utf-8'}

    def update_headers(self, fields=None):
        '''get_headers will return a simple default
           header for a json post.
           This function will be adopted as needed.
        '''
        if len(self.headers) == 0:
            headers = self._init_headers()
        else:
            headers = self.headers

        if fields is not None:
            for key, value in fields.items():
                headers[key] = value

        header_names = ",".join(list(headers.keys()))
        bot.debug("Headers found: %s" % (header_names))
        self.headers = headers

    def update_token(self, response=None):
        '''empty update_token to be defined
           by subclasses, if necessary
        '''
        return None

    def stream(self,
               url,
               file_name,
               data=None,
               headers=None,
               default_headers=True,
               show_progress=False):

        '''stream is a get that will stream to file_name
        :param data: a dictionary of key:value items
                     to add to the data args variable
        :param url: the url to get
        :param show_progress: if True, show a progress bar with the bot
        :returns response: the requests response object, or stream
        '''
        bot.debug("GET (stream) %s" % (url))

        # If we use default headers, start with client's
        request_headers = dict()
        if default_headers and len(self.headers) > 0:
            request_headers = self.headers

        if headers is not None:
            request_headers.update(headers)

        request = self.prepare_request(headers=request_headers,
                                       data=data,
                                       url=url)

        response = self.submit_request(request)

        # Keep user updated with Progress Bar?
        if show_progress:
            content_size = None
            not_error = response.code not in [400, 401]
            if 'Content-Length' in response.headers and not_error:
                progress = 0
                content_size = int(response.headers['Content-Length'])
                bot.show_progress(progress, content_size, length=35)

        chunk_size = 1 << 20
        with open(file_name, 'wb') as filey:
            while True:
                chunk = response.read(chunk_size)
                if not chunk:
                    break
                try:
                    filey.write(chunk)
                    if show_progress:
                        if content_size is not None:
                            progress += chunk_size
                            bot.show_progress(iteration=progress,
                                              total=content_size,
                                              length=35,
                                              carriage_return=False)
                except Exception as error:
                    msg = "Error writing to %s:" % (file_name)
                    msg += " %s exiting" % (error)
                    bot.error(msg)
                    sys.exit(1)

            # Newline to finish download
            if show_progress:
                sys.stdout.write('\n')

        return file_name

    def get(self,
            url,
            data=None,
            headers=None,
            default_headers=True,
            return_response=False):

        '''get will use requests to get a particular url
        :param data: a dictionary of key:value
                     items to add to the data args variable
        :param url: the url to get
        :returns response: the requests response object, or stream
        '''

        bot.debug("GET %s" % (url))

        # If we use default headers, start with client's
        request_headers = dict()
        if default_headers and len(self.headers) > 0:
            request_headers = self.headers

        if headers is not None:
            request_headers.update(headers)

        request = self.prepare_request(headers=request_headers,
                                       data=data,
                                       url=url)

        response = self.submit_request(request,
                                       return_response=return_response)

        if return_response is True:
            return response

        return response.read().decode('utf-8')

    def submit_request(self, request, return_response=False):
        '''submit_request will make the request,
        via a stream or not. If return response is True, the
        response is returned as is without further parsing.
        Given a 401 error, the update_token function is called
        to try the request again, and only then the error returned.
        '''

        try:
            response = urlopen(request)

        # If we have an HTTPError, try to follow the response
        except HTTPError as error:

            # Case 1: we have an http 401 error, and need to refresh token
            bot.debug('Http Error with code %s' % (error.code))

            if error.code == 401:
                self.update_token(response=error)
                try:
                    request = self.prepare_request(request.get_full_url(),
                                                   headers=self.headers)
                    response = urlopen(request)
                except HTTPError as error:
                    bot.debug('Http Error with code %s' % (error.code))
                    return error
            else:
                return error

        return response

    def prepare_request(self, url, data=None, headers=None):
        '''prepare the request object, determining
        if there is data (making it
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

    def download_atomically(self,
                            url,
                            file_name,
                            headers=None,
                            show_progress=False):

        '''download stream atomically will stream to a temporary file, and
        rename only upon successful completion. This is to ensure that
        errored downloads are not found as complete in the cache
        :param file_name: the file name to stream to
        :param url: the url to stream from
        :param headers: additional headers to add to the get (default None)
        '''
        try:
            tmp_file = "%s.%s" % (file_name,
                                  next(tempfile._get_candidate_names()))
            response = self.stream(url,
                                   file_name=tmp_file,
                                   headers=headers,
                                   show_progress=show_progress)
            os.rename(tmp_file, file_name)

        except Exception:

            download_folder = os.path.dirname(os.path.abspath(file_name))
            msg = "Error downloading %s. " % (url)
            msg += "Do you have permission to write to %s?" % (download_folder)
            bot.error(msg)
            try:
                os.remove(tmp_file)
            except Exception:
                pass
            sys.exit(1)

        return file_name
