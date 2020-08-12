    SUMMARY

    This is a build container that generates installable singularity packages
    for singularity v3.X.X. The container will outpt a tar.gz, deb and rpm
    in the current directory. A ~/rpmbuild directory is also created.


    KNOWN BUGS

    Some versions of singularity contain the character 'v', such as v3.0.0.
    The container will have to be rebuild with the following statement 
    commented out:

    curl -L -o singularity-${VERSION}.tar.gz https://github.com/sylabs/singularity/releases/download/v${VERSION}/singularity-${VERSION}.tar.gz

    If curl is removed and the tar.gz for those versions are present in the current directory,
    the container should see them and build them after failing to auto-
    matically download.


    BUILD

	sudo singularity build build-singularity.sif build-singularity.def


    USAGE

        ./build-singularity.sif {version}

	./build-singularity.sif 3.6.1
