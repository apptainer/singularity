SUBDIRS = yum arch debootstrap docker busybox

MAINTAINERCLEANFILES = Makefile.in 
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ *.la

AM_CFLAGS = -Wall -fpie
AM_LDFLAGS = -pie
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)

noinst_LTLIBRARIES = libbootstrapmodule.la

libbootstrapmodule_la_LIBADD = yum/libmoduleyum.la arch/libmodulearch.la debootstrap/libmoduledebootstrap.la docker/libmoduledocker.la busybox/libmodulebusybox
libbootstrapmodule_la_SOURCES = module.c

EXTRA_DIST = module.h
