MAINTAINERCLEANFILES = Makefile.in config.h config.h.in
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ *.la *.o *.lo
AM_CFLAGS = -Wall -fpie -fPIC
AM_LDFLAGS = -pie
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES)
distlibdir = $(libdir)/singularity
distincludedir = $(includedir)/singularity

noinst_LTLIBRARIES = libsigning.la
libsigning_la_SOURCES = crypt.c
libsigning_la_CFLAGS = $(AM_CFLAGS) # This fixes duplicate sources in library and progs

distinclude_HEADERS = crypt.h
distlib_LTLIBRARIES = libsingularity-signing.la

libsingularity_signing_la_SOURCES =
libsingularity_signing_la_LIBADD = $(noinst_LTLIBRARIES)
libsingularity_signing_la_LDFLAGS = -version-info 1:0:0
libsingularity_signing_la_CFLAGS = $(AM_CFLAGS)

# These are kludges so they don't remove the $(DEPDIR) in ../../util/ otherwise
# the clean will fail when other Makefile tries to remove those directories
distclean:
	-rm -rf ./$(DEPDIR)
	-rm -rf ./.libs
	-rm -f $(DISTCLEANFILES) $(CLEANFILES)

maintainer-clean:
	-rm -rf ./$(DEPDIR)
	-rm -rf ./.libs
	-rm -f $(MAINTAINERCLEANFILES) $(DISTCLEANFILES) $(CLEANFILES)
