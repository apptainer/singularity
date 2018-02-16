SUBDIRS = files mounts ns enter environment overlayfs autofs

MAINTAINERCLEANFILES = Makefile.in config.h config.h.in
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ *.la
AM_CFLAGS = -Wall -fpie -fPIC
AM_LDFLAGS = -pie
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES)
distlibdir = $(libdir)/singularity
distincludedir = $(includedir)/singularity

noinst_LTLIBRARIES = libinternal.la
libinternal_la_LIBADD = ns/libinternal.la mounts/libinternal.la files/libinternal.la enter/libinternal.la overlayfs/libinternal.la environment/libinternal.la autofs/libinternal.la
libinternal_la_SOURCES = runtime.c ../../util/fork.c ../../util/registry.c ../../util/message.c ../../util/config_parser.c ../../util/privilege.c ../../util/util.c ../../util/file.c ../../util/setns.c ../../util/mount.c
libinternal_la_CFLAGS = $(AM_CFLAGS) # This fixes duplicate sources in library and progs

distinclude_HEADERS = runtime.h
distlib_LTLIBRARIES = libsingularity-runtime.la

libsingularity_runtime_la_SOURCES =
libsingularity_runtime_la_LIBADD = $(noinst_LTLIBRARIES)
libsingularity_runtime_la_LDFLAGS = -version-info 1:0:0 
libsingularity_runtime_la_CFLAGS = $(AM_CFLAGS)

EXTRA_DIST = 


# These are kludges so they don't remove the $(DEPDIR) in ../../util/ otherwise
# the clean will fail when other Makefile tries to remove those directories
distclean: distclean-recursive
	-rm ./$(DEPDIR)
	-rm -f Makefile

maintainer-clean: maintainer-clean-recursive
	-rm ./$(DEPDIR)
	-rm -f Makefile

