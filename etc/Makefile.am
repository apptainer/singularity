confdir = $(sysconfdir)/singularity/
completiondir = $(sysconfdir)/bash_completion.d/

all: singularity.conf

singularity.conf: singularity.conf.in ../src/util/config_defaults.h
	./configure_transform.py --defaults ../src/util/config_defaults.h --infile singularity.conf.in --outfile singularity.conf

dist_conf_DATA = default-nsswitch.conf singularity.conf init nvliblist.conf
dist_completion_DATA =  bash_completion.d/singularity

CLEANFILES = singularity.conf
MAINTAINERCLEANFILES = Makefile.in
EXTRA_DIST = singularity.conf.in configure_transform.py
