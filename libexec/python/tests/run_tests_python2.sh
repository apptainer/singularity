#!/bin/sh

python2 -m unittest test_base
python2 -m unittest test_docker_import
python2 -m unittest test_docker_add
python2 -m unittest test_docker_api

python2 -m unittest test_shub_import
python2 -m unittest test_shub_pull
python2 -m unittest test_shub_add
python2 -m unittest test_shub

python2 -m unittest test_custom_cache
python2 -m unittest test_default_cache
python2 -m unittest test_disable_cache
