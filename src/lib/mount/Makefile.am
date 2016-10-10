SUBDIRS = kernelfs binds home tmp hostfs userbinds dev cwd scratch

MAINTAINERCLEANFILES = Makefile.in
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ *.la

AM_CFLAGS = -Wall -fpie
AM_LDFLAGS = -pie
libns_pid_a_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)

noinst_LTLIBRARIES = libmount.la

libmount_la_LIBADD = kernelfs/libmountkernelfs.la binds/libmountbinds.la home/libmounthome.la tmp/libmounttmp.la hostfs/libmounthostfs.la userbinds/libmountuserbinds.la dev/libmountdev.la cwd/libmountcwd.la scratch/libmountscratch.la
libmount_la_SOURCES = mount.c mount-util.c

EXTRA_DIST = mount-util.c mount-util.h
