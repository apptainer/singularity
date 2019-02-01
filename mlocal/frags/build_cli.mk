# This file contains all of the rules for building the singularity CLI binary

# singularity build config
singularity_build_config := $(SOURCEDIR)/internal/pkg/buildcfg/config.go
$(singularity_build_config): $(BUILDDIR)/config.h
	$(V)rm -f $(singularity_build_config)
	$(V)export BUILDDIR=$(BUILDDIR_ABSPATH) && cd $(SOURCEDIR)/internal/pkg/buildcfg && go generate

CLEANFILES += $(singularity_build_config)

# singularity
singularity_SOURCE := $(shell $(SOURCEDIR)/makeit/gengodep $(SOURCEDIR)/cmd/singularity/cli.go)

singularity := $(BUILDDIR)/singularity
$(singularity): $(singularity_build_config) $(singularity_SOURCE)
	@echo " GO" $@ "\n    [+] GO_TAGS" \"$(GO_TAGS)\"
	$(V)go build $(GO_BUILDMODE) -tags "$(GO_TAGS)" $(GO_LDFLAGS) -o $(BUILDDIR)/singularity $(SOURCEDIR)/cmd/singularity/cli.go

singularity_INSTALL := $(DESTDIR)$(BINDIR)/singularity
$(singularity_INSTALL): $(singularity)
	@echo " INSTALL" $@
	$(V)install -d $(@D)
	$(V)install -m 0755 $(singularity) $(singularity_INSTALL) # set cp to install

CLEANFILES += $(singularity)
INSTALLFILES += $(singularity_INSTALL)
ALL += $(singularity)


# bash_completion file
bash_completion :=  $(BUILDDIR)/etc/bash_completion.d/singularity
$(bash_completion): $(singularity_build_config)
	@echo " GEN" $@
	$(V)rm -f $@
	$(V)mkdir -p $(@D)
	$(V)go run -tags "$(GO_TAGS)" $(SOURCEDIR)/cmd/bash_completion/bash_completion.go $@

bash_completion_INSTALL := $(DESTDIR)$(SYSCONFDIR)/bash_completion.d/singularity
$(bash_completion_INSTALL): $(bash_completion)
	@echo " INSTALL" $@
	$(V)install -d $(@D)
	$(V)install -m 0644 $< $@

CLEANFILES += $(bash_completion)
INSTALLFILES += $(bash_completion_INSTALL)
ALL += $(bash_completion)


# singularity.conf file
config := $(BUILDDIR)/singularity.conf
config_INSTALL := $(DESTDIR)$(SYSCONFDIR)/singularity/singularity.conf
# override this to empty to avoid merging old configuration settings
old_config := $(config_INSTALL)

$(config): $(singularity_build_config) $(SOURCEDIR)/etc/conf/gen.go $(SOURCEDIR)/internal/pkg/runtime/engines/singularity/config/data/singularity.conf $(SOURCEDIR)/internal/pkg/runtime/engines/singularity/config/config.go
	@echo " GEN $@`if [ -n "$(old_config)" ]; then echo " from $(old_config)"; fi`"
	$(V)go run $(SOURCEDIR)/etc/conf/gen.go \
		$(SOURCEDIR)/internal/pkg/runtime/engines/singularity/config/data/singularity.conf $(old_config) $(config)

$(config_INSTALL): $(config)
	@echo " INSTALL" $@
	$(V)install -d $(@D)
	$(V)install -m 0644 $< $@

INSTALLFILES += $(config_INSTALL)
ALL += $(config)
