# Build Singularity


## Summary
    This is a build container that generates installable singularity packages 
    for singularity v3.X.X. The container will output a deb and rpm in the 
    current directory.

## Known Bugs
    Some versions of singularity contain the character 'v', such as v3.0.0.
    The container will have to be rebuilt with the following statement 
    modified:

    curl -L -o singularity-${VERSION}.tar.gz https://github.com/sylabs/singularity/releases/download/v${VERSION}/singularity-${VERSION}.tar.gz

## Usage

    sudo singularity build build-singularity.sif build-singularity.def

    ./build-singularity.sif {version}

    ./build-singularity.sif 3.6.2
