SUBDIRS = bin etc libexec man src

MAINTAINERCLEANFILES = Makefile.in aclocal.m4 configure ltmain.sh compile depcomp install-sh missing config.* *.m4 singularity-*.tar.gz singularity-*.rpm m4/* test.sh
DISTCLEANFILES = Makefile test.sh
CLEANFILES = 
EXTRA_DIST = singularity.spec autogen.sh examples debian CONTRIBUTORS.md CONTRIBUTING.md COPYRIGHT.md INSTALL.md LICENSE-LBNL.md LICENSE.md README.md tests

maintainer-clean-local:
	rm -rf m4

distclean-local: maintainer-clean-local

test: test.sh
	sh ./test.sh

install-perms:
	@echo
	@echo "install-perms is no longer required"
	@echo

all-local:
if !FOUND_MKSQUASHFS
	@echo
	@echo "******"
	@echo "WARNING - mksquashfs was not found at configure time"
	@echo "mksquashfs is required at runtime for full functionality"
	@echo "Provided by distribution package squashfs-tools"
	@echo "******"
	@echo
endif

install-data-hook:
	install -d -m 0755 $(DESTDIR)$(CONTAINER_MOUNTDIR)
	install -d -m 0755 $(DESTDIR)$(CONTAINER_FINALDIR)
	install -d -m 0755 $(DESTDIR)$(CONTAINER_OVERLAY)
	install -d -m 0755 $(DESTDIR)$(SESSIONDIR)
	test -f $(DESTDIR)$(libexecdir)/singularity/sexec-suid && rm -f $(DESTDIR)$(libexecdir)/singularity/sexec-suid || :
	test -f $(DESTDIR)$(libexecdir)/singularity/bin/copy-suid && rm -f $(DESTDIR)$(libexecdir)/singularity/bin/copy-suid || :


ACLOCAL_AMFLAGS = -I m4
