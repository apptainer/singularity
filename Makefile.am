SUBDIRS = bin src libexec

MAINTAINERCLEANFILES = Makefile.in aclocal.m4 configure ltmain.sh depcomp install-sh missing config.* *.m4 singularity-*.tar.gz singularity-*.rpm
DISTCLEANFILES = 
CLEANFILES =
EXTRA_DIST = LICENSE README NEWS ChangeLog singularity.spec


install-perms: install
	sudo chown root:root $(libexecdir)/singularity/sexec-prep
	sudo chown root:root $(libexecdir)/singularity/sexec
	sudo chmod +s $(libexecdir)/singularity/sexec

