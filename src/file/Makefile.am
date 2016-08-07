SUBDIRS = passwd group

MAINTAINERCLEANFILES = Makefile.in
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~

AM_CFLAGS = -Wall -fpie
AM_LDFLAGS = -pie
libns_pid_a_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)

noinst_LTLIBRARIES = libfile.la

libfile_la_LIBADD = passwd/libfilepasswd.la group/libfilegroup.la
libfile_la_SOURCES = file.c
