SUBDIRS = runtime image
#SUBDIRS = ns rootfs action mount file image

MAINTAINERCLEANFILES = Makefile.in config.h config.h.in
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ *.la
AM_CFLAGS = -Wall -fpie -fPIC
AM_LDFLAGS = -pie
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES)
distlibdir = $(libdir)/singularity
distincludedir = $(includedir)/singularity

#noinst_LTLIBRARIES = libsingularity_internal.la
#libsingularity_internal_la_LIBADD = ns/libns.la rootfs/librootfs.la action/libaction.la mount/libmount.la file/libfile.la image/libimage.la
#libsingularity_internal_la_SOURCES = singularity.c privilege.c message.c util.c file.c sessiondir.c config_parser.c fork.c loop-control.c
#libsingularity_internal_la_CFLAGS = $(AM_CFLAGS) # This fixes duplicate sources in library and progs
#
#distinclude_HEADERS = singularity.h
#distlib_LTLIBRARIES = libsingularity.la
#
#libsingularity_la_SOURCES =
#libsingularity_la_LIBADD = libsingularity_internal.la
#libsingularity_la_LDFLAGS = -version-info 1:0:0 
#libsingularity_la_CFLAGS = $(AM_CFLAGS)
#
#EXTRA_DIST = config_parser.h fork.h loop-control.h message.h privilege.h sessiondir.h singularity.h config_defaults.h
EXTRA_DIST =
