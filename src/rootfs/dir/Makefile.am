AM_CFLAGS = -Wall -fpie
AM_LDFLAGS = -pie
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)

noinst_LTLIBRARIES = librootfsdir.la
librootfsdir_la_SOURCES = dir.c
#librootfsdir_la_SOURCES = dir.c ../../file.c ../../util.c ../../message.c ../../privilege.c

EXTRA_DIST = dir.h
