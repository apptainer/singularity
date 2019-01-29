# This file contains installation rule for starter-suid binary. In order to 
#   include this file, Makefile_runtime.stub MUST first be included.

# starter suid
starter_suid_INSTALL := $(DESTDIR)$(LIBEXECDIR)/singularity/bin/starter-suid
$(starter_suid_INSTALL): $(starter)
	@echo " INSTALL SUID" $@
	$(V)install -d $(@D)
	$(V)install -m 4755 $(starter) $(starter_suid_INSTALL)

INSTALLFILES += $(starter_suid_INSTALL)
