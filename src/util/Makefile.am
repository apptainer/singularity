MAINTAINERCLEANFILES = Makefile.in config.h config.h.in
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ *.la *.o
AM_CFLAGS = -Wall -fpie -fPIC
AM_LDFLAGS = -pie
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES)


EXTRA_DIST = capability.c \
			 capability.h \
			 cleanupd.c \
			 cleanupd.h \
			 config_parser.c \
			 config_parser.h \
			 daemon.c \
			 daemon.h \
			 file.c \
			 file.h \
			 fork.c \
			 fork.h \
			 message.c \
			 message.h \
			 mount.c \
			 mount.h \
			 privilege.c \
			 privilege.h \
			 registry.c \
			 registry.h \
			 securebits.h \
			 sessiondir.c \
			 sessiondir.h \
			 suid.c \
			 suid.h \
			 util.c \
			 util.h \
			 setns.c \
			 setns.h \
			 signal.c \
			 signal.h \
			 binary.c \
			 binary.h \
			 config_defaults.h
