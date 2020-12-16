# go tool default build options
GO111MODULE := on
GO_TAGS := containers_image_openpgp sylog oci_engine singularity_engine fakeroot_engine
GO_TAGS_SUID := containers_image_openpgp sylog singularity_engine fakeroot_engine
GO_LDFLAGS :=
# Need to use non-pie build on ppc64le
# https://github.com/hpcng/singularity/issues/5762
uname_m := $(shell uname -m)
ifeq ($(uname_m),ppc64le)
GO_BUILDMODE := -buildmode=default
else
GO_BUILDMODE := -buildmode=pie
endif
GO_GCFLAGS := -gcflags=github.com/sylabs/singularity/...="-trimpath $(SOURCEDIR)=>github.com/sylabs/singularity@v0.0.0"
GO_ASMFLAGS := -asmflags=github.com/sylabs/singularity/...="-trimpath $(SOURCEDIR)=>github.com/sylabs/singularity@v0.0.0"
GO_MODFLAGS := $(if $(wildcard $(SOURCEDIR)/vendor/modules.txt),-mod=vendor,-mod=readonly)
GO_MODFILES := $(SOURCEDIR)/go.mod $(SOURCEDIR)/go.sum
GOFLAGS := $(GO_MODFLAGS) -trimpath
GOPROXY := https://proxy.golang.org

export GOFLAGS GO111MODULE GOPROXY
