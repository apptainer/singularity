MAINTAINERCLEANFILES = Makefile.in config.h config.h.in
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ 
AM_CFLAGS = -Wall
sexec_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(NAMESPACE_DEFINES)
bootstrap_CPPFLAGS = -DLIBEXECDIR=\"$(libexecdir)\"
ftrace_CPPFLAGS = -DARCH_$(SINGULARITY_ARCH)

dist_suidPROGRAM_INSTALL = ${INSTALL} -m 640

install-exec-hook:
	@if test "$$UID" = "0"; then \
		echo " /bin/chown root:root $(DESTDIR)$(libexecdir)/singularity/sexec"; \
		/bin/chown root:root $(DESTDIR)$(libexecdir)/singularity/sexec; \
		echo " /bin/chmod 4755 $(DESTDIR)$(libexecdir)/singularity/sexec"; \
		/bin/chmod 4755 $(DESTDIR)$(libexecdir)/singularity/sexec; \
	fi

bindir = $(libexecdir)/singularity
bin_PROGRAMS = ftype sexec image-create image-expand image-mount image-bind

ftrace_SOURCES = ftrace.c file.c file.h util.c util.h
ftype_SOURCES = ftype.c util.c util.h file.c file.h
sexec_SOURCES = sexec.c util.c util.h loop-control.c loop-control.h mounts.c mounts.h user.c user.h file.c file.h image.c image.h
image_create_SOURCES = image-create.c file.c file.h util.c util.h image.c image.h
image_expand_SOURCES = image-expand.c file.c file.h util.c util.h image.c image.h
image_mount_SOURCES = image-mount.c util.c util.h loop-control.c loop-control.h mounts.c mounts.h file.c file.h image.c image.h
image_bind_SOURCES = image-bind.c util.c util.h loop-control.c loop-control.h mounts.c mounts.h file.c file.h image.c image.h

EXTRA_DIST = config.h 
