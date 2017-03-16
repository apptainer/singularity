environmentdir = $(libexecdir)/singularity/bootstrap-scripts/

dist_environment_SCRIPTS = environment.tar

# This isn't to be run via the build because it requires sudoi such that the
# permissions inside the tarball are correct. The resulting environment.tar
# output should be commited and updated as necessary and included within the
# Singularity source tree.
tar:
	sudo install -d -m 0755 newroot/.singularity
	sudo install -d -m 0755 newroot/.singularity/env
	sudo install -d -m 0755 newroot/.singularity/actions
	sudo install -d -m 0755 newroot/.singularity/labels
	sudo install -d -m 0755 newroot/dev
	sudo install -d -m 0755 newroot/etc
	sudo install -d -m 0755 newroot/home
	sudo install -d -m 0755 newroot/proc
	sudo install -d -m 0755 newroot/root
	sudo install -d -m 0755 newroot/sys
	sudo install -d -m 0755 newroot/tmp
	sudo install -d -m 0755 newroot/var
	sudo install -d -m 0755 newroot/var/tmp
	sudo install -m 0755 /dev/null newroot/etc/hosts
	sudo install -m 0755 /dev/null newroot/etc/resolv.conf
	sudo install -m 0755 /dev/null newroot/etc/passwd
	sudo install -m 0755 /dev/null newroot/etc/group
	sudo install -m 0755 runscript newroot/.singularity/
	sudo install -m 0755 run shell exec newroot/.singularity/actions/
	sudo install -m 0755 01-base.sh newroot/.singularity/env/
	sudo install -m 0755 99-environment.sh newroot/.singularity/env/
	sudo ln -sf .singularity/runscript newroot/singularity
	sudo ln -sf .singularity/env/99-environment.sh newroot/environment
	cd newroot; tar czf ../environment.tar .
	sudo rm -rf newroot

MAINTAINERCLEANFILES = Makefile.in
