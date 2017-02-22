basedir = $(libexecdir)/singularity/bootstrap-scripts/environment
singularitydir = $(basedir)/.singularity
actiondir = $(singularitydir)/actions
envdir = $(singularitydir)/env
labeldir = $(singularitydir)/labels

dist_singularity_SCRIPTS = runscript
dist_action_SCRIPTS = run shell exec
dist_env_DATA = 01-base.sh

install-data-local:
	install -d -m 0755 $(DESTDIR)/$(basedir)/dev
	install -d -m 0755 $(DESTDIR)/$(basedir)/etc
	install -d -m 0755 $(DESTDIR)/$(basedir)/home
	install -d -m 0755 $(DESTDIR)/$(basedir)/proc
	install -d -m 0755 $(DESTDIR)/$(basedir)/sys
	install -d -m 0755 $(DESTDIR)/$(basedir)/tmp
	install -d -m 0755 $(DESTDIR)/$(basedir)/var
	install -d -m 0755 $(DESTDIR)/$(basedir)/var/tmp
	install -d -m 0755 $(DESTDIR)/$(labeldir)
	install -m 0755 /dev/null $(DESTDIR)/$(basedir)/etc/hosts
	install -m 0755 /dev/null $(DESTDIR)/$(basedir)/etc/resolv.conf
	install -m 0755 /dev/null $(DESTDIR)/$(basedir)/etc/passwd
	install -m 0755 /dev/null $(DESTDIR)/$(basedir)/etc/group
	ln -sf .singularity/runscript $(DESTDIR)/$(basedir)/singularity
#	ln -sf .singularity/actions/run $(DESTDIR)/$(basedir)/.run
#	ln -sf .singularity/actions/shell $(DESTDIR)/$(basedir)/.shell
#	ln -sf .singularity/actions/exec $(DESTDIR)/$(basedir)/.exec

MAINTAINERCLEANFILES = Makefile.in
