# go tool default build options
GO111MODULE := on
GO_TAGS := containers_image_openpgp sylog imgbuild_engine oci_engine singularity_engine fakeroot_engine
GO_TAGS_SUID := containers_image_openpgp sylog singularity_engine fakeroot_engine
GO_LDFLAGS :=
GO_BUILDMODE := -buildmode=default
GO_GCFLAGS := -gcflags=all=-trimpath=$(SOURCEDIR)
GO_ASMFLAGS := -asmflags=all=-trimpath=$(SOURCEDIR)
GO_MODFLAGS :=
GOFLAGS := -mod=vendor

export GOFLAGS GO111MODULE
