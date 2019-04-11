Singularity example plugin
==========================

This directory contains an example plugin for singularity. It
demonstrates how to add a command and flags, which are the only two
types of hooks currently supported. In the future as support for more
hooks is added, this plugin will be updated to demonstrate those as
well.

Building
--------

In order to build the plugin you need a copy of code matching the
version of singularity that you wish to use. You can find the commit
matching the singularity binary by running:

    $ singularity version
    3.1.1-723.g7998470e7

this means this version of singularity is _post_ 3.1.1 (but before the
next version after that one). The suffix .gXXXXXXXXX indicates the exact
commit in github.com/sylabs/singularity used to build this binary
(7998470e7 in this example).

Obtain a copy of the source code by running:

    git clone https://github.com/sylabs/singularity.git
    cd singularity
    git checkout 7998470e7

Still from within that directory, run:

	singularity plugin compile ./examples/plugins/test-plugin

This will produce a file `./examples/plugins/test-plugin/test-plugin.sif`.

Currently there's a limitation regarding the location of the plugin
code: it must reside somewhere _inside_ the singularity source code
tree.

Installing
----------

Once you have compiled the plugin into a SIF file, you can install it
into the correct singularity directory using the command:

	$ singularity plugin install ./examples/plugins/test-plugin/test-plugin.sif

Singularity will automatically load the plugin code from now on.

Other commands
--------------

You can query the list of installed plugins:

    $ singularity plugin list
    ENABLED  NAME
        yes  sylabs.io/test-plugin

Disable an installed plugin:

    $ singularity plugin disable sylabs.io/test-plugin

Enable a disabled plugin:

    $ singularity plugin enable sylabs.io/test-plugin

Uninstall an installed plugin:

    $ singularity plugin uninstall sylabs.io/test-plugin

And inspect a SIF file before installing:

    $ singularity plugin inspect examples/plugins/test-plugin/test-plugin.sif
    Name: sylabs.io/test-plugin
    Description: This is a short test plugin for Singularity
    Author: Michael Bauer
    Version: 0.0.1
