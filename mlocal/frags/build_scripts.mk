# This file contains generation rules for scripts

# go-test script
$(SOURCEDIR)/scripts/go-test: export GO := $(GO)
$(SOURCEDIR)/scripts/go-test: export GO111MODULE := $(GO111MODULE)
$(SOURCEDIR)/scripts/go-test: export GOFLAGS := $(GOFLAGS)
$(SOURCEDIR)/scripts/go-test: export GO_TAGS := $(GO_TAGS)
$(SOURCEDIR)/scripts/go-test: export SUDO_SCRIPT := $(SOURCEDIR)/scripts/test-sudo
$(SOURCEDIR)/scripts/go-test: $(SOURCEDIR)/scripts/go-test.in $(SOURCEDIR)/scripts/expand-env.go
	@echo ' GEN $@'
	$(V) $(GO) run $(GO_MODFLAGS) $(SOURCEDIR)/scripts/expand-env.go < $< > $@
	$(V) chmod +x $@

ALL += $(SOURCEDIR)/scripts/go-test

# go-generate script
$(SOURCEDIR)/scripts/go-generate: export BUILDDIR := $(BUILDDIR_ABSPATH)
$(SOURCEDIR)/scripts/go-generate: export GO := $(GO)
$(SOURCEDIR)/scripts/go-generate: export GO111MODULE := $(GO111MODULE)
$(SOURCEDIR)/scripts/go-generate: export GOFLAGS := $(GOFLAGS)
$(SOURCEDIR)/scripts/go-generate: export GO_TAGS := $(GO_TAGS)
$(SOURCEDIR)/scripts/go-generate: $(SOURCEDIR)/scripts/go-generate.in $(SOURCEDIR)/scripts/expand-env.go
	@echo ' GEN $@'
	$(V) $(GO) run $(GO_MODFLAGS) $(SOURCEDIR)/scripts/expand-env.go < $< > $@
	$(V) chmod +x $@

.PHONY: codegen
codegen: $(SOURCEDIR)/scripts/go-generate
	cd $(SOURCEDIR) && ./scripts/go-generate -x ./...

ALL += $(SOURCEDIR)/scripts/go-generate

