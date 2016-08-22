SUBDIRS = bin etc libexec man src

MAINTAINERCLEANFILES = Makefile.in aclocal.m4 configure ltmain.sh depcomp install-sh missing config.* *.m4 singularity-*.tar.gz singularity-*.rpm m4/* test.sh
DISTCLEANFILES = Makefile test.sh
CLEANFILES = 
EXTRA_DIST = LICENSE README.md NEWS ChangeLog singularity.spec examples

maintainer-clean-local:
	rm -rf m4

distclean-local: maintainer-clean-local

test: test.sh
	sh ./test.sh

install-perms:
	@echo
	@echo "install-perms is no longer required"
	@echo

ACLOCAL_AMFLAGS = -I m4
