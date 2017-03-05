#!/bin/sh

python3 -m unittest test_base
python3 -m unittest test_docker_import
python3 -m unittest test_docker_add
python3 -m unittest test_docker_api

python3 -m unittest test_shub_import
python3 -m unittest test_shub_pull
python3 -m unittest test_shub_add
python3 -m unittest test_shub

python3 -m unittest test_custom_cache
python3 -m unittest test_default_cache
python3 -m unittest test_disable_cache
