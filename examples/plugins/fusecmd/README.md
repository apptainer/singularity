Singularity fusecmd mount plugin
================================

This plugin adds the --fusecmd option to singularity.   The parameter
is a string which is a command with plus its parameters to run inside the
container to implement a libfuse3-based filesystem. The last parameter is a
mountpoint that will be pre-mounted and replaced with a a /dev/fd/NN path to
the fuse file descriptor.

From the top level of a singularity source directory, run:

	singularity plugin compile examples/plugins/fusecmd

Installing
----------

Once you have compiled the plugin into a SIF file, you can install it
into the correct singularity directory using the command:

	$ sudo singularity plugin install examples/plugins/fusecmd/fusecmd.sif

Singularity will automatically load the plugin code from now on.
