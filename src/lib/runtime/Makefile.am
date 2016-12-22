SUBDIRS = files mount
#SUBDIRS = files mount ns

MAINTAINERCLEANFILES = Makefile.in config.h config.h.in
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ *.la
AM_CFLAGS = -Wall -fpie -fPIC
AM_LDFLAGS = -pie
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)
distlibdir = $(libdir)/singularity
distincludedir = $(includedir)/singularity

noinst_LTLIBRARIES = libinternal.la
libinternal_la_LIBADD = ns/libinternal.la mount/libinternal.la files/libinternal.la
libinternal_la_SOURCES = runtime.c
libinternal_la_CFLAGS = $(AM_CFLAGS) # This fixes duplicate sources in library and progs

distinclude_HEADERS = runtime.h
distlib_LTLIBRARIES = libruntime.la

libruntime_la_SOURCES =
libruntime_la_LIBADD = $(noinst_LTLIBRARIES)
libruntime_la_LDFLAGS = -version-info 1:0:0 
libruntime_la_CFLAGS = $(AM_CFLAGS)

EXTRA_DIST = 

