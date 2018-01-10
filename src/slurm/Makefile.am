AM_CPPFLAGS = -DSYSCONFDIR=\"$(sysconfdir)\" -DLOCALSTATEDIR=\"$(localstatedir)\" -DLIBEXECDIR=\"$(libexecdir)\" $(SINGULARITY_DEFINES)

plugindir = $(libdir)/slurm


if WITH_SLURM
plugin_LTLIBRARIES = singularity_spank.la
singularity_spank_la_SOURCES = singularity.c ../util/sessiondir.c ../action-lib/ready.c ../util/cleanupd.c
singularity_spank_la_LIBADD = ../lib/image/libsingularity-image.la ../lib/runtime/libsingularity-runtime.la
singularity_spank_la_LDFLAGS = -module -no-undefined -avoid-version -export-symbols-regex '^slurm_spank_|^plugin_'
endif

install-data-hook: cleanup_plugin

cleanup_plugin:
	@if test -e "$(DESTDIR)$(plugindir)/singularity_spank.so"; then \
		echo "cp $(DESTDIR)$(plugindir)/singularity_spank.so $(DESTDIR)$(plugindir)/backup_singularity_spank"; \
		cp "$(DESTDIR)$(plugindir)/singularity_spank.so" "$(DESTDIR)$(plugindir)/backup_singularity_spank"; \
		echo "rm -f $(DESTDIR)$(plugindir)/singularity_spank.*"; \
		rm -f "$(DESTDIR)$(plugindir)/"singularity_spank.*; \
		echo "mv $(DESTDIR)$(plugindir)/backup_singularity_spank $(DESTDIR)$(plugindir)/singularity_spank.so"; \
		mv "$(DESTDIR)$(plugindir)/backup_singularity_spank" "$(DESTDIR)$(plugindir)/singularity.so"; \
	fi


