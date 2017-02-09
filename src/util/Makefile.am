MAINTAINERCLEANFILES = Makefile.in config.h config.h.in
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ *.la
AM_CFLAGS = -Wall -fpie -fPIC
AM_LDFLAGS = -pie
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)


EXTRA_DIST = config_parser.c \
			 file.c \
			 fork.c \
			 message.c \
			 privilege.c \
			 registry.c \
			 suid.c \
			 util.c \
			 config_defaults.h \
			 config_parser.h \
			 file.h \
			 fork.h \
			 message.h \
			 privilege.h \
			 registry.h \
			 suid.h \
			 util.h

