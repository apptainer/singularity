SUBDIRS = bin etc libexec man src

MAINTAINERCLEANFILES = Makefile.in aclocal.m4 configure ltmain.sh depcomp install-sh missing config.* *.m4 singularity-*.tar.gz singularity-*.rpm m4/*
DISTCLEANFILES = 
CLEANFILES =
EXTRA_DIST = LICENSE README.md NEWS ChangeLog singularity.spec examples

maintainer-clean-local:
	rm -rf m4

distclean-local: maintainer-clean-local

install-perms:
	@echo
	@echo "install-perms is no longer required"
	@echo

ACLOCAL_AMFLAGS = -I m4 -I $(top_srcdir)/src
