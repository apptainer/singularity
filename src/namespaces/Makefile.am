AM_CFLAGS = -Wall -fpie
AM_LDFLAGS = -pie
libns_pid_a_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)
libns_mnt_a_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)

noinst_LIBRARIES = libns_pid.a libns_mnt.a
libns_pid_a_SOURCES = ns_pid.c ../message.c ../config_parser.c
libns_mnt_a_SOURCES = ns_mnt.c ../message.c

EXTRA_DIST = ns_mnt.h ns_pid.h
