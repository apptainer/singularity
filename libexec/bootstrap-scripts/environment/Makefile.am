environmentdir = $(libexecdir)/singularity/bootstrap-scripts/

CLEANFILES = environment.tar
EXTRA_DIST = 01-base.sh 90-environment.sh 95-apps.sh 99-base.sh exec run runscript shell test start startscript

dist_environment_DATA = environment.tar

.PHONY: environment.tar

environment.tar:
	install -d -m 0755 newroot
	install -d -m 0755 newroot/.singularity.d
	install -d -m 0755 newroot/.singularity.d/env
	install -d -m 0755 newroot/.singularity.d/actions
	install -d -m 0755 newroot/.singularity.d/libs
	install -d -m 0755 newroot/dev
	install -d -m 0755 newroot/etc
	install -d -m 0755 newroot/home
	install -d -m 0755 newroot/proc
	install -d -m 0755 newroot/root
	install -d -m 0755 newroot/sys
	install -d -m 0755 newroot/tmp
	install -d -m 0755 newroot/var
	install -d -m 0755 newroot/var/tmp
	install -m 0755 /dev/null newroot/etc/hosts
	install -m 0755 /dev/null newroot/etc/resolv.conf
	install -m 0755 runscript newroot/.singularity.d/
	install -m 0755 startscript newroot/.singularity.d/
	install -m 0755 exec newroot/.singularity.d/actions/
	install -m 0755 run newroot/.singularity.d/actions/
	install -m 0755 shell newroot/.singularity.d/actions/
	install -m 0755 test newroot/.singularity.d/actions/
	install -m 0755 start newroot/.singularity.d/actions/
	install -m 0755 01-base.sh newroot/.singularity.d/env/
	install -m 0755 90-environment.sh newroot/.singularity.d/env/
	install -m 0755 95-apps.sh newroot/.singularity.d/env/
	install -m 0755 99-base.sh newroot/.singularity.d/env/
	ln -sf .singularity.d/env/90-environment.sh newroot/environment
	ln -sf .singularity.d/runscript newroot/singularity
	ln -sf .singularity.d/actions/exec newroot/.exec
	ln -sf .singularity.d/actions/run newroot/.run
	ln -sf .singularity.d/actions/shell newroot/.shell
	ln -sf .singularity.d/actions/test newroot/.test
	cd newroot; tar czf ../environment.tar . --owner=0 --group=0
	rm -rf newroot

MAINTAINERCLEANFILES = Makefile.in
