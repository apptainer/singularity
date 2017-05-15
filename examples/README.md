# Examples
The example bootstrap definition files, each called `Singularity`, are 
located in their respectively named folders in this directory. 
These files can be used to create new container images on a variety of 
Linux distributions or become the basis for customization to 
build reproducible containers for a specific purpose. While many of these
examples use core mirrors and OS distributions, keep in mind that you can
use a Docker bootstrap to create almost any of them.

## Contributing
If you have a specific scientific (or other) container, we suggest that you
consider [singularity hub](https://singularity-hub.org) to serve it. If you do
not intend to build or use the container, or want to provide a base template, 
then you might also want to send a pull request to add it here.

### contrib
If you wish to contribute a definition file that does not fall within one
of the folders here, it should go into [contrib](contrib). In this case,
please send a pull request and contribute it to the examples/contribs 
directory with the format being hyphen ('-') delimited of the following format:

    1. Base distribution name and version if applicable (e.g. centos7 or
        ubuntu_trusty)
    2. Target nomenclature that describes the container (e.g. tensorflow)
    3. Any relevant version strings to the application or work-flow
    4. Always end in .def

An example of this:

    examples/contrib/debian84-tensorflow-0.10.def

### base
If your contribution is more appropriate for one of the base or template distributions,
then please make a respective folder in the [examples](.) directory, and name
the definition file `Singularity`.
