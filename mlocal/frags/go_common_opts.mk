# go tool default build options
GO111MODULE := on
GO_TAGS := containers_image_openpgp sylog
GO_LDFLAGS :=
GO_BUILDMODE := -buildmode=default
GO_GCFLAGS := -gcflags=all=-trimpath=$(SOURCEDIR)
GO_ASMFLAGS := -asmflags=all=-trimpath=$(SOURCEDIR)
GO_MODFLAGS :=
GOFLAGS := -mod=vendor

export GOFLAGS GO111MODULE
