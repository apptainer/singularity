#!/bin/sh

echo "suid"
./cli suid /tmp/testing.simg /bin/ls /
echo "userns"
./cli userns /tmp/testing /bin/ls /
