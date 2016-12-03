
confdir = $(sysconfdir)/singularity/
completiondir = $(sysconfdir)/bash_completion.d/

singularity.conf: singularity.conf.in
	../src/configure_transform.py --defaults ../src/lib/config_defaults.h --infile singularity.conf.in --outfile singularity.conf

dist_conf_DATA = default-nsswitch.conf singularity.conf init
dist_completion_DATA =  bash_completion.d/singularity


MAINTAINERCLEANFILES = Makefile.in
