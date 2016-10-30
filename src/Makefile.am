SUBDIRS = lib util

MAINTAINERCLEANFILES = Makefile.in config.h config.h.in
DISTCLEANFILES = Makefile
CLEANFILES = core.* *~ *.la
AM_CFLAGS = -Wall -fpie -fPIC
AM_LDFLAGS = -pie
AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)

bindir = $(libexecdir)/singularity

bin_PROGRAMS = $(BUILD_SUID) sexec image-create image-expand image-mount image-bind get-section
EXTRA_PROGRAMS = sexec-suid

sexec_SOURCES = sexec.c util/util.c util/file.c
sexec_LDADD = lib/libsingularity.la

sexec_suid_SOURCES = sexec.c util/util.c util/file.c
sexec_suid_LDADD = lib/libsingularity.la
sexec_suid_LDFLAGS = -static
sexec_suid_CPPFLAGS = -DSINGULARITY_SUID $(AM_CPPFLAGS)

image_create_SOURCES = image-create.c util/file.c util/util.c
image_create_LDADD = lib/libsingularity.la

image_expand_SOURCES = image-expand.c util/file.c util/util.c
image_expand_LDADD = lib/libsingularity.la

image_mount_SOURCES = image-mount.c util/util.c util/file.c
image_mount_LDADD = lib/libsingularity.la

image_bind_SOURCES = image-bind.c util/util.c util/file.c
image_bind_LDADD = lib/libsingularity.la

get_section_SOURCES = get-section.c util/util.c util/file.c
get_section_LDADD = lib/libsingularity.la

plugindir = $(libdir)/slurm

if WITH_SLURM
plugin_LTLIBRARIES = singularity_spank.la
singularity_spank_la_SOURCES = slurm/singularity.c
singularity_spank_la_LIBADD = lib/libsingularity.la
singularity_spank_la_LDFLAGS = -module -no-undefined -avoid-version -export-symbols-regex '^slurm_spank_|^plugin_'
endif

#sexec_nosuid_CPPFLAGS = -DSINGULARITY_NOSUID=1 -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES) $(NO_SETNS)
#bootstrap_CPPFLAGS = -DLIBEXECDIR=\"$(libexecdir)\"
#ftrace_CPPFLAGS = -DARCH_$(SINGULARITY_ARCH)

#dist_suidPROGRAM_INSTALL = ${INSTALL} -m 640

install-data-hook: make_suid cleanup_plugin

make_suid:
	@if test `id -ru` = "0" -a "x$(BUILD_SUID)" = "xsexec-suid"; then \
		echo " /bin/chown root:root $(DESTDIR)$(libexecdir)/singularity/sexec-suid"; \
		/bin/chown root:root $(DESTDIR)$(libexecdir)/singularity/sexec-suid; \
		echo " /bin/chmod 4755 $(DESTDIR)$(libexecdir)/singularity/sexec-suid"; \
		/bin/chmod 4755 $(DESTDIR)$(libexecdir)/singularity/sexec-suid; \
	fi

cleanup_plugin:
	@if test -e "$(DESTDIR)$(plugindir)/singularity_spank.so"; then \
		echo "cp $(DESTDIR)$(plugindir)/singularity_spank.so $(DESTDIR)$(plugindir)/backup_singularity_spank"; \
		cp "$(DESTDIR)$(plugindir)/singularity_spank.so" "$(DESTDIR)$(plugindir)/backup_singularity_spank"; \
		echo "rm -f $(DESTDIR)$(plugindir)/singularity_spank.*"; \
		rm -f "$(DESTDIR)$(plugindir)/"singularity_spank.*; \
		echo "mv $(DESTDIR)$(plugindir)/backup_singularity_spank $(DESTDIR)$(plugindir)/singularity_spank.so"; \
		mv "$(DESTDIR)$(plugindir)/backup_singularity_spank" "$(DESTDIR)$(plugindir)/singularity.so"; \
	fi

EXTRA_DIST = config.h
