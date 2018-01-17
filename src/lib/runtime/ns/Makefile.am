SUBDIRS = mnt pid ipc net

CLEANFILES = core.* *~ *.la
DISTCLEANFILES = Makefile
MAINTAINERCLEANFILES = Makefile.in

AM_CFLAGS = -Wall -fpie
AM_LDFLAGS = -pie
libns_pid_a_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES)
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES)

noinst_LTLIBRARIES = libinternal.la

libinternal_la_LIBADD = mnt/libinternal.la pid/libinternal.la ipc/libinternal.la net/libinternal.la
libinternal_la_SOURCES = ns.c

EXTRA_DIST = ns.h

# These are kludges so they don't remove the $(DEPDIR) in ../../util/ otherwise
# the clean will fail when other Makefile tries to remove those directories
distclean: distclean-recursive
	-rm -rf ./$(DEPDIR)
	-rm -rf ./.libs
	-rm -f $(DISTCLEANFILES) $(CLEANFILES)

maintainer-clean: maintainer-clean-recursive
	-rm -rf ./$(DEPDIR)
	-rm -rf ./.libs
	-rm -f $(MAINTAINERCLEANFILES) $(DISTCLEANFILES) $(CLEANFILES)

