#!/bin/bash

cd core && ./mconfig -b builddir
make -C builddir && cp builddir/sif ../build

cd ../ && go build -o build/singularity cmd/cli/cli.go
