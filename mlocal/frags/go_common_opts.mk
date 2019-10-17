# go tool default build options
GO111MODULE := on
GO_TAGS := containers_image_openpgp sylog imgbuild_engine oci_engine singularity_engine fakeroot_engine
GO_TAGS_SUID := containers_image_openpgp sylog singularity_engine fakeroot_engine
GO_LDFLAGS :=
GO_BUILDMODE := -buildmode=default
GO_GCFLAGS :=
GO_ASMFLAGS :=
GO_MODFLAGS :=
GOFLAGS := -mod=readonly -trimpath
GOPROXY := https://proxy.golang.org

export GOFLAGS GO111MODULE GOPROXY
