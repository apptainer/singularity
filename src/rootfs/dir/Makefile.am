AM_CFLAGS = -Wall -fpie
AM_LDFLAGS = -pie
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)

noinst_LIBRARIES = librootfsdir.a
librootfsdir_a_SOURCES = dir.c ../../message.c ../../config_parser.c

EXTRA_DIST = dir.h
