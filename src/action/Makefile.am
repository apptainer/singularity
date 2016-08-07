SUBDIRS = shell exec run

MAINTAINERCLEANFILES = Makefile.in
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~

AM_CFLAGS = -Wall -fpie
AM_LDFLAGS = -pie
libns_pid_a_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)

noinst_LTLIBRARIES = libaction.la

libaction_la_LIBADD = shell/libactionshell.la exec/libactionexec.la run/libactionrun.la
libaction_la_SOURCES = action.c
