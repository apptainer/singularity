#!/bin/sh

set -e

if test ! -e go.mod ; then
	echo "E: $PWD/go.mod not found. Abort."
	exit 1
fi

export GO111MODULE=on

# Make sure the index is updated
git update-index --refresh

if ! git diff-index --raw --exit-code HEAD ; then
	echo "E: Workspace is unexpectly dirty. Abort."
	exit 2
fi

if ! go mod download ; then
	echo "E: Failed to download Go modules. Abort"
	exit 3
fi

if ! go mod verify > /dev/null ; then
	echo "E: Invalid Go module state. Abort."
	exit 4
fi

if ! go mod tidy ; then
	echo "E: Failed to run go mod tidy. Abort."
	exit 5
fi

# The go mod tidy command above might have updated go.mod or go.sum.
# The next command tells git that things _might_ have been modified so
# it should check that.
git update-index --refresh

if ! git diff-index --raw --exit-code HEAD ; then
	echo "E: Workspace became dirty after runnig 'go mod tidy'. Abort."
	exit 6
fi

echo "I: go.mod OK."
exit 0
