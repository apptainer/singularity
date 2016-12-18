SUBDIRS = files mount ns

MAINTAINERCLEANFILES = Makefile.in config.h config.h.in
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ *.la
AM_CFLAGS = -Wall -fpie -fPIC
AM_LDFLAGS = -pie
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)
distlibdir = $(libdir)/singularity
distincludedir = $(includedir)/singularity

noinst_LTLIBRARIES = libsingularity_runtime_internal.la
libsingularity_runtime_internal_la_LIBADD = ns/libns.la mount/libmount.la file/libfile.la
libsingularity_runtime_internal_la_SOURCES = runtime.c
libsingularity_runtime_internal_la_CFLAGS = $(AM_CFLAGS) # This fixes duplicate sources in library and progs

distinclude_HEADERS = runtime.h
distlib_LTLIBRARIES = libsingularity_runtime.la

libsingularity_runtime_la_SOURCES =
libsingularity_runtime_la_LIBADD = $(noinst_LTLIBRARIES)
libsingularity_runtime_la_LDFLAGS = -version-info 1:0:0 
libsingularity_runtime_la_CFLAGS = $(AM_CFLAGS)

EXTRA_DIST = 

